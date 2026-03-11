package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/spf13/pflag"

	"http-pull/internal/config"
	"http-pull/internal/logging"
	"http-pull/internal/runner"
)

func main() {
	configPath := pflag.String("config", "/etc/http-pull/config.yaml", "path to configuration file")
	pflag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error loading config: %v\n", err)
		os.Exit(1)
	}

	logger, logCleanup, err := logging.Setup(cfg.Log)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error setting up logging: %v\n", err)
		os.Exit(1)
	}
	defer logCleanup()

	logger.Info("http-pull starting", "config", *configPath)

	r := runner.New(logger)
	if err := r.ApplyConfig(cfg); err != nil {
		logger.Error("error applying config", "error", err)
		os.Exit(1)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGHUP:
			logger.Info("received SIGHUP, reloading configuration")
			newCfg, err := config.Load(*configPath)
			if err != nil {
				logger.Error("failed to reload config, keeping current", "error", err)
				continue
			}

			newLogger, newCleanup, err := logging.Setup(newCfg.Log)
			if err != nil {
				logger.Error("failed to setup new logging, keeping current", "error", err)
				continue
			}

			logCleanup()
			logCleanup = newCleanup
			logger = newLogger
			r.SetLogger(logger)

			if err := r.ApplyConfig(newCfg); err != nil {
				logger.Error("failed to apply new config", "error", err)
			} else {
				logger.Info("configuration reloaded successfully")
			}

		case syscall.SIGINT, syscall.SIGTERM:
			logger.Info("received shutdown signal", "signal", sig)
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			r.Shutdown(ctx)
			cancel()
			logger.Info("http-pull stopped")
			return
		}
	}
}
