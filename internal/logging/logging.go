// Package logging sets up structured logging via slog.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"http-pull/internal/config"
)

// Setup creates a slog.Logger from the given configuration.
// It returns the logger, a cleanup function that should be called on
// shutdown (e.g. to close the log file), and any error encountered.
func Setup(cfg config.LogConfig) (*slog.Logger, func(), error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, nil, err
	}

	opts := &slog.HandlerOptions{Level: level}

	switch strings.ToLower(cfg.Target) {
	case "stdout", "":
		handler := slog.NewJSONHandler(os.Stdout, opts)
		return slog.New(handler), func() {}, nil

	case "file":
		f, err := os.OpenFile(cfg.File, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, nil, fmt.Errorf("opening log file: %w", err)
		}
		handler := slog.NewJSONHandler(f, opts)
		cleanup := func() { f.Close() }
		return slog.New(handler), cleanup, nil

	default:
		return nil, nil, fmt.Errorf("unknown log target %q", cfg.Target)
	}
}

func parseLevel(s string) (slog.Level, error) {
	switch strings.ToUpper(s) {
	case "DEBUG":
		return slog.LevelDebug, nil
	case "INFO", "":
		return slog.LevelInfo, nil
	case "WARN":
		return slog.LevelWarn, nil
	case "ERROR":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q", s)
	}
}
