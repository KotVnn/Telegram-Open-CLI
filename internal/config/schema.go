package config

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrMissingConfig    = errors.New("missing config")
	ErrInvalidConfig    = errors.New("invalid config")
)

func validate(cfg *Config) error {
	if cfg.Telegram.Token == "" {
		return fmt.Errorf("telegram token is required: %w", ErrMissingConfig)
	}

	if cfg.DefaultBackend == "" {
		return fmt.Errorf("default_backend is required: %w", ErrMissingConfig)
	}

	if _, ok := cfg.Backends[cfg.DefaultBackend]; !ok {
		return fmt.Errorf("default_backend %q not found in backends: %w", cfg.DefaultBackend, ErrMissingConfig)
	}

	if cfg.Storage.Path == "" {
		return fmt.Errorf("storage path is required: %w", ErrMissingConfig)
	}

	for name, backendCfg := range cfg.Backends {
		if !backendCfg.Enabled {
			continue
		}
		if backendCfg.Command == "" {
			return fmt.Errorf("backend %q: command is required: %w", name, ErrInvalidConfig)
		}
		if backendCfg.Timeout < 0 {
			return fmt.Errorf("backend %q: timeout must be non-negative: %w", name, ErrInvalidConfig)
		}
		if backendCfg.Timeout == 0 {
			backendCfg.Timeout = 30 * time.Minute
			cfg.Backends[name] = backendCfg
		}
	}

	return nil
}
