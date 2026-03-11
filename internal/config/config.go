// Package config handles loading and validating http-pull configuration.
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config is the root configuration structure.
type Config struct {
	Log     LogConfig      `mapstructure:"log"`
	Targets []TargetConfig `mapstructure:"targets"`
}

// LogConfig controls logging behaviour.
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Target string `mapstructure:"target"`
	File   string `mapstructure:"file"`
}

// TargetConfig describes a single HTTP pull target.
type TargetConfig struct {
	Name        string             `mapstructure:"name"`
	URL         string             `mapstructure:"url"`
	Interval    time.Duration      `mapstructure:"interval"`
	Destination string             `mapstructure:"destination"`
	HTTPRequest *HTTPRequestConfig `mapstructure:"http_request"`
	Hooks       []HookConfig       `mapstructure:"hooks"`
}

// HTTPRequestConfig holds optional HTTP request settings.
type HTTPRequestConfig struct {
	Method          string     `mapstructure:"method"`
	Headers         []Header   `mapstructure:"headers"`
	BasicAuth       *BasicAuth `mapstructure:"basic_auth"`
	FollowRedirects *bool      `mapstructure:"follow_redirects"`
}

// Header is a single HTTP header key-value pair.
type Header struct {
	Name  string `mapstructure:"name"`
	Value string `mapstructure:"value"`
}

// BasicAuth holds HTTP basic authentication credentials.
type BasicAuth struct {
	Username string `mapstructure:"username"`
	Password string `mapstructure:"password"`
}

// HookConfig describes a hook to execute after a pull.
type HookConfig struct {
	Type        string   `mapstructure:"type"`
	Command     string   `mapstructure:"command"`
	Destination string   `mapstructure:"destination"`
	On          []string `mapstructure:"on"`
}

// Load reads and validates a configuration file at the given path.
func Load(path string) (*Config, error) {
	v := viper.New()
	v.SetConfigFile(path)

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshalling config: %w", err)
	}

	applyDefaults(&cfg)

	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("validating config: %w", err)
	}

	return &cfg, nil
}

func applyDefaults(cfg *Config) {
	if cfg.Log.Level == "" {
		cfg.Log.Level = "INFO"
	}
	if cfg.Log.Target == "" {
		cfg.Log.Target = "stdout"
	}

	for i := range cfg.Targets {
		t := &cfg.Targets[i]
		if t.HTTPRequest != nil {
			if t.HTTPRequest.Method == "" {
				t.HTTPRequest.Method = "GET"
			}
			if t.HTTPRequest.FollowRedirects == nil {
				v := true
				t.HTTPRequest.FollowRedirects = &v
			}
		}
		for j := range t.Hooks {
			if len(t.Hooks[j].On) == 0 {
				t.Hooks[j].On = []string{"success"}
			}
		}
	}
}

func validate(cfg *Config) error {
	// Validate log level.
	switch strings.ToUpper(cfg.Log.Level) {
	case "DEBUG", "INFO", "WARN", "ERROR":
	default:
		return fmt.Errorf("invalid log level %q", cfg.Log.Level)
	}

	// Validate log target.
	switch strings.ToLower(cfg.Log.Target) {
	case "stdout":
	case "file":
		if cfg.Log.File == "" {
			return fmt.Errorf("log target is 'file' but no file path specified")
		}
	default:
		return fmt.Errorf("invalid log target %q", cfg.Log.Target)
	}

	// Validate targets.
	if len(cfg.Targets) == 0 {
		return fmt.Errorf("no targets configured")
	}

	seen := make(map[string]bool)
	for i, t := range cfg.Targets {
		if t.Name == "" {
			return fmt.Errorf("target %d: name is required", i)
		}
		if seen[t.Name] {
			return fmt.Errorf("duplicate target name %q", t.Name)
		}
		seen[t.Name] = true

		if t.URL == "" {
			return fmt.Errorf("target %q: url is required", t.Name)
		}
		if t.Interval <= 0 {
			return fmt.Errorf("target %q: interval must be positive", t.Name)
		}
		if t.Destination == "" {
			return fmt.Errorf("target %q: destination is required", t.Name)
		}

		for j, h := range t.Hooks {
			if err := validateHook(t.Name, j, h); err != nil {
				return err
			}
		}
	}

	return nil
}

func validateHook(targetName string, idx int, h HookConfig) error {
	switch h.Type {
	case "shell":
		if h.Command == "" {
			return fmt.Errorf("target %q hook %d: shell hook requires a command", targetName, idx)
		}
	case "move":
		if h.Destination == "" {
			return fmt.Errorf("target %q hook %d: move hook requires a destination", targetName, idx)
		}
	default:
		return fmt.Errorf("target %q hook %d: unknown hook type %q", targetName, idx, h.Type)
	}

	for _, on := range h.On {
		switch on {
		case "success", "failure":
		default:
			return fmt.Errorf("target %q hook %d: invalid on value %q", targetName, idx, on)
		}
	}

	return nil
}
