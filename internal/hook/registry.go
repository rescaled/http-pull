package hook

import (
	"fmt"
	"log/slog"

	"http-pull/internal/config"
	pkghook "http-pull/pkg/hook"
)

// Entry pairs a Hook implementation with the triggers that should fire it.
type Entry struct {
	Hook     pkghook.Hook
	Triggers []pkghook.Trigger
}

// FromConfig builds hook entries from configuration. It returns an error
// if any hook type is unknown.
func FromConfig(cfgs []config.HookConfig, logger *slog.Logger) ([]Entry, error) {
	entries := make([]Entry, 0, len(cfgs))

	for i, cfg := range cfgs {
		triggers, err := parseTriggers(cfg.On)
		if err != nil {
			return nil, fmt.Errorf("hook %d: %w", i, err)
		}

		var h pkghook.Hook
		switch cfg.Type {
		case "shell":
			h = NewShellHook(cfg.Command, logger)
		case "move":
			h = NewMoveHook(cfg.Destination, logger)
		default:
			return nil, fmt.Errorf("hook %d: unknown type %q", i, cfg.Type)
		}

		entries = append(entries, Entry{Hook: h, Triggers: triggers})
	}

	return entries, nil
}

func parseTriggers(on []string) ([]pkghook.Trigger, error) {
	triggers := make([]pkghook.Trigger, 0, len(on))
	for _, s := range on {
		switch s {
		case "success":
			triggers = append(triggers, pkghook.TriggerSuccess)
		case "failure":
			triggers = append(triggers, pkghook.TriggerFailure)
		default:
			return nil, fmt.Errorf("unknown trigger %q", s)
		}
	}
	return triggers, nil
}
