package config

import (
	"errors"
	"fmt"
)

var ErrMissingConfig = errors.New("missing config")

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

	return nil
}
