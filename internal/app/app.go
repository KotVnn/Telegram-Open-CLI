package app

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/backend/claude"
	"github.com/KotVnn/Telegram-Open-CLI/internal/backend/opencode"
	"github.com/KotVnn/Telegram-Open-CLI/internal/config"
	"github.com/KotVnn/Telegram-Open-CLI/internal/project"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
	"github.com/KotVnn/Telegram-Open-CLI/internal/telegram"
	"github.com/KotVnn/Telegram-Open-CLI/internal/user"
)

// App is the main application struct.
type App struct {
	config   *config.Config
	storage  storage.Storage
	bot      telegram.Adapter
	backends *backend.Manager
	users    *user.Manager
	projects *project.Manager
	logger   zerolog.Logger
}

// New creates a new App instance.
func New(cfg *config.Config, logger zerolog.Logger) *App {
	return &App{
		config: cfg,
		logger: logger,
	}
}

// Run starts the application.
func (a *App) Run(ctx context.Context) error {
	if err := a.initStorage(ctx); err != nil {
		return err
	}

	if err := a.initBackends(ctx); err != nil {
		return err
	}

	if err := a.initTelegram(ctx); err != nil {
		return err
	}

	if err := a.backends.Start(ctx); err != nil {
		return err
	}

	if err := a.bot.Start(ctx); err != nil {
		return err
	}

	<-ctx.Done()

	return a.Shutdown(ctx)
}

// Shutdown gracefully shuts down the application.
func (a *App) Shutdown(ctx context.Context) error {
	if err := a.bot.Stop(ctx); err != nil {
		a.logger.Error().Err(err).Msg("failed to stop telegram bot")
	}

	if err := a.backends.Stop(ctx); err != nil {
		a.logger.Error().Err(err).Msg("failed to stop backends")
	}

	if err := a.storage.Close(); err != nil {
		a.logger.Error().Err(err).Msg("failed to close storage")
	}

	return nil
}

func (a *App) initStorage(ctx context.Context) error {
	db, err := storage.NewSQLiteDB(a.config.Storage.Path)
	if err != nil {
		return err
	}

	a.storage = storage.NewSQLite(db)

	if err := a.storage.Migrate(ctx); err != nil {
		return err
	}

	a.users = user.NewManager(a.storage)
	a.projects = project.NewManager(a.storage)

	return nil
}

func (a *App) initBackends(ctx context.Context) error {
	a.backends = backend.NewManager()

	a.backends.Register("opencode", func() backend.Backend {
		return opencode.New()
	})

	a.backends.Register("claude", func() backend.Backend {
		return claude.New()
	})

	configs := make(map[string]backend.BackendConfig)
	for name, cfg := range a.config.Backends {
		configs[name] = backend.BackendConfig{
			Enabled:     cfg.Enabled,
			Command:     cfg.Command,
			Args:        cfg.Args,
			WorkingDir:  cfg.WorkingDir,
			Environment: cfg.Environment,
			Timeout:     cfg.Timeout,
		}
	}

	return a.backends.InitializeAll(ctx, configs)
}

func (a *App) initTelegram(ctx context.Context) error {
	bot, err := telegram.New(telegram.Config{
		Token:  a.config.Telegram.Token,
		Logger: a.logger,
	})
	if err != nil {
		return err
	}

	a.bot = bot

	defaultBackend, err := a.backends.Get(a.config.DefaultBackend)
	if err != nil {
		return fmt.Errorf("get default backend: %w", err)
	}

	sessionManager := telegram.NewSessionManager(a.storage, defaultBackend, a.logger, a.config.Telegram.Token)

	bot.HandleCommand("start", telegram.HandleStart(a.bot))
	bot.HandleCommand("help", telegram.HandleHelp(a.bot))
	bot.HandleCommand("version", telegram.HandleVersion(a.bot))
	bot.HandleCommand("new", telegram.HandleNew(a.bot, sessionManager))
	bot.HandleCommand("sessions", telegram.HandleSessions(a.bot, sessionManager))
	bot.HandleCommand("switch", telegram.HandleSwitch(a.bot, sessionManager))
	bot.HandleCommand("close", telegram.HandleClose(a.bot, sessionManager))
	bot.HandleCommand("status", telegram.HandleStatus(a.bot, sessionManager))

	bot.HandleDefault(telegram.HandleMessage(a.bot, sessionManager))

	bot.HandleCallback("session:", telegram.HandleSessionCallback(a.bot, sessionManager))
	bot.HandleCallback("new_session", telegram.HandleNewSessionCallback(a.bot, sessionManager))
	bot.HandleCallback("confirm:", telegram.HandleConfirmCallback(a.bot, sessionManager))
	bot.HandleCallback("model:", telegram.HandleModelCallback(a.bot, sessionManager))
	bot.HandleCallback("agent:", telegram.HandleAgentCallback(a.bot, sessionManager))
	bot.HandleCallback("cancel", telegram.HandleCancelCallback(a.bot))

	bot.Use(
		telegram.RecoveryMiddleware(a.logger),
		telegram.LoggingMiddleware(a.logger),
	)

	if a.config.Security.RequireAuth {
		bot.Use(telegram.AuthMiddleware(bot,
			a.config.Telegram.AllowedUsers,
			a.config.Telegram.AllowedGroups,
		))
	}

	bot.Use(telegram.RateLimitMiddleware(10, 5))

	return nil
}
