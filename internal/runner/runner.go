// Package runner orchestrates pull workers with reload and graceful shutdown.
package runner

import (
	"context"
	"log/slog"
	"reflect"
	"sync"
	"time"

	"http-pull/internal/config"
	"http-pull/internal/hook"
	"http-pull/internal/puller"
	pkghook "http-pull/pkg/hook"
)

// Runner manages a set of pull workers based on configuration.
type Runner struct {
	logger  *slog.Logger
	mu      sync.Mutex
	workers map[string]*worker
	wg      sync.WaitGroup
}

// New creates a Runner with the given logger.
func New(logger *slog.Logger) *Runner {
	return &Runner{
		logger:  logger,
		workers: make(map[string]*worker),
	}
}

// SetLogger replaces the runner's logger, used after config reload.
func (r *Runner) SetLogger(logger *slog.Logger) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.logger = logger
}

// ApplyConfig reconciles the running workers with the provided configuration.
// It stops workers for removed or changed targets and starts new ones.
func (r *Runner) ApplyConfig(cfg *config.Config) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	desired := make(map[string]config.TargetConfig, len(cfg.Targets))
	for _, t := range cfg.Targets {
		desired[t.Name] = t
	}

	// Stop workers that were removed or whose config changed.
	for name, w := range r.workers {
		newTarget, exists := desired[name]
		if !exists || !reflect.DeepEqual(w.target, newTarget) {
			r.logger.Info("stopping worker", "target", name)
			w.stop()
			delete(r.workers, name)
		}
	}

	// Start workers for new or changed targets.
	for name, target := range desired {
		if _, running := r.workers[name]; running {
			continue
		}

		hookEntries, err := hook.FromConfig(target.Hooks, r.logger)
		if err != nil {
			return err
		}

		w := &worker{
			target:  target,
			hooks:   hookEntries,
			puller:  puller.New(target, r.logger),
			logger:  r.logger,
			stopCh:  make(chan struct{}),
			stopped: make(chan struct{}),
		}
		r.workers[name] = w
		r.wg.Add(1)
		go func() {
			defer r.wg.Done()
			w.run()
		}()
		r.logger.Info("started worker", "target", name, "interval", target.Interval)
	}

	return nil
}

// Shutdown gracefully stops all workers and waits for them to finish
// within the given context's deadline.
func (r *Runner) Shutdown(ctx context.Context) {
	r.mu.Lock()
	for name, w := range r.workers {
		r.logger.Info("stopping worker for shutdown", "target", name)
		w.stop()
		delete(r.workers, name)
	}
	r.mu.Unlock()

	done := make(chan struct{})
	go func() {
		r.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		r.logger.Info("all workers stopped")
	case <-ctx.Done():
		r.logger.Warn("shutdown timed out, some workers may not have completed")
	}
}

type worker struct {
	target  config.TargetConfig
	hooks   []hook.Entry
	puller  *puller.Puller
	logger  *slog.Logger
	stopCh  chan struct{}
	stopped chan struct{}
	once    sync.Once
}

func (w *worker) stop() {
	w.once.Do(func() { close(w.stopCh) })
	<-w.stopped
}

func (w *worker) run() {
	defer close(w.stopped)

	// Execute immediately on start.
	w.execute()

	ticker := time.NewTicker(w.target.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.execute()
		}
	}
}

func (w *worker) execute() {
	// Use a background context for the actual pull so that in-flight
	// operations can complete even when the worker is being stopped.
	ctx := context.Background()

	err := w.puller.Pull(ctx)
	result := pkghook.Result{
		Success:  err == nil,
		FilePath: w.target.Destination,
		Err:      err,
	}

	if err != nil {
		w.logger.Error("pull failed", "target", w.target.Name, "error", err)
	}

	w.runHooks(ctx, result)
}

func (w *worker) runHooks(ctx context.Context, result pkghook.Result) {
	trigger := pkghook.TriggerSuccess
	if !result.Success {
		trigger = pkghook.TriggerFailure
	}

	for _, entry := range w.hooks {
		if !hasTrigger(entry.Triggers, trigger) {
			continue
		}
		if err := entry.Hook.Execute(ctx, result); err != nil {
			w.logger.Error("hook failed",
				"target", w.target.Name,
				"error", err,
			)
		}
	}
}

func hasTrigger(triggers []pkghook.Trigger, t pkghook.Trigger) bool {
	for _, tr := range triggers {
		if tr == t {
			return true
		}
	}
	return false
}
