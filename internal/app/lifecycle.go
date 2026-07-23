package app

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"

	"github.com/KotVnn/Telegram-Open-CLI/internal/config"
)

// Lifecycle manages application startup and shutdown.
type Lifecycle struct {
	app    *App
	logger zerolog.Logger
}

// NewLifecycle creates a new Lifecycle.
func NewLifecycle(cfg *config.Config, logger zerolog.Logger) *Lifecycle {
	return &Lifecycle{
		app:    New(cfg, logger),
		logger: logger,
	}
}

// Run starts the application and handles shutdown signals.
func (l *Lifecycle) Run(ctx context.Context) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	l.logger.Info().Msg("starting application")

	if err := l.app.Run(ctx); err != nil {
		l.logger.Error().Err(err).Msg("application error")
		return err
	}

	l.logger.Info().Msg("application stopped")
	return nil
}
