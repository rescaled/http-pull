// Package hook provides built-in hook implementations for http-pull.
package hook

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"

	pkghook "http-pull/pkg/hook"
)

// ShellHook executes a shell command after a pull completes.
type ShellHook struct {
	command string
	logger  *slog.Logger
}

// NewShellHook creates a ShellHook that runs the given command via /bin/sh.
func NewShellHook(command string, logger *slog.Logger) *ShellHook {
	return &ShellHook{
		command: command,
		logger:  logger,
	}
}

// Execute runs the shell command. The pull result is available but not
// directly passed to the command; callers filter by trigger before invoking.
func (h *ShellHook) Execute(ctx context.Context, result pkghook.Result) error {
	h.logger.Debug("executing shell hook", "command", h.command)

	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", h.command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shell hook failed: %w (output: %s)", err, output)
	}

	h.logger.Debug("shell hook completed", "output", string(output))
	return nil
}
