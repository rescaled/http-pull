package hook

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	pkghook "http-pull/pkg/hook"
)

// MoveHook moves the downloaded file to another destination after a pull.
// It attempts an atomic rename first and falls back to copy+delete for
// cross-device moves.
type MoveHook struct {
	destination string
	logger      *slog.Logger
}

// NewMoveHook creates a MoveHook that moves to the given destination.
func NewMoveHook(destination string, logger *slog.Logger) *MoveHook {
	return &MoveHook{
		destination: destination,
		logger:      logger,
	}
}

// Execute moves the pulled file to the configured destination.
func (h *MoveHook) Execute(ctx context.Context, result pkghook.Result) error {
	h.logger.Debug("executing move hook", "src", result.FilePath, "dst", h.destination)

	err := os.Rename(result.FilePath, h.destination)
	if err == nil {
		return nil
	}

	// Rename failed — likely a cross-device link. Fall back to copy+delete.
	h.logger.Debug("rename failed, falling back to copy", "error", err)
	if err := copyFile(result.FilePath, h.destination); err != nil {
		return fmt.Errorf("move hook copy failed: %w", err)
	}
	if err := os.Remove(result.FilePath); err != nil {
		return fmt.Errorf("move hook: removing source after copy: %w", err)
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	if err := out.Close(); err != nil {
		return err
	}

	// Preserve file ownership across the copy.
	return preserveOwnership(info, dst)
}
