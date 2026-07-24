//go:build integration

package app

import (
	"testing"

	"github.com/KotVnn/Telegram-Open-CLI/internal/config"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
)

func TestAppInitializationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger := zerolog.Nop()

	t.Run("app creation", func(t *testing.T) {
		cfg := &config.Config{
			Version:        "1",
			DefaultBackend: "opencode",
			Telegram: config.TelegramConfig{
				Token: "test-token",
			},
			Storage: config.StorageConfig{
				Path: ":memory:",
			},
			Backends: map[string]config.Backend{
				"opencode": {
					Enabled: true,
					Command: "opencode",
					Args:    []string{"run"},
				},
			},
		}

		app := New(cfg, logger)
		assert.NotNil(t, app)
	})
}

func TestBackendSelectionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger := zerolog.Nop()

	t.Run("default backend selection", func(t *testing.T) {
		cfg := &config.Config{
			Version:        "1",
			DefaultBackend: "opencode",
			Backends: map[string]config.Backend{
				"opencode": {Enabled: true, Command: "opencode"},
				"claude":   {Enabled: true, Command: "claude"},
			},
		}

		app := New(cfg, logger)
		assert.NotNil(t, app)
		assert.Equal(t, "opencode", app.config.DefaultBackend)
	})
}
