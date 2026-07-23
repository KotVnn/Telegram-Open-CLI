package config

import (
	"context"
	"time"
)

// Config holds the application configuration.
type Config struct {
	Version        string            `mapstructure:"version" toml:"version"`
	DefaultBackend string            `mapstructure:"default_backend" toml:"default_backend"`
	Telegram       TelegramConfig    `mapstructure:"telegram" toml:"telegram"`
	Storage        StorageConfig     `mapstructure:"storage" toml:"storage"`
	Backends       map[string]Backend `mapstructure:"backends" toml:"backends"`
	Security       SecurityConfig    `mapstructure:"security" toml:"security"`
	Logging        LoggingConfig     `mapstructure:"logging" toml:"logging"`
}

// TelegramConfig holds Telegram bot configuration.
type TelegramConfig struct {
	Token         string  `mapstructure:"token" toml:"token"`
	Mode          string  `mapstructure:"mode" toml:"mode"`
	Timeout       int     `mapstructure:"timeout" toml:"timeout"`
	AllowedUsers  []int64 `mapstructure:"allowed_users" toml:"allowed_users"`
	AllowedGroups []int64 `mapstructure:"allowed_groups" toml:"allowed_groups"`
}

// StorageConfig holds storage configuration.
type StorageConfig struct {
	Path string `mapstructure:"path" toml:"path"`
}

// Backend holds backend-specific configuration.
type Backend struct {
	Enabled     bool              `mapstructure:"enabled" toml:"enabled"`
	Command     string            `mapstructure:"command" toml:"command"`
	Args        []string          `mapstructure:"args" toml:"args"`
	WorkingDir  string            `mapstructure:"working_dir" toml:"working_dir"`
	Environment map[string]string `mapstructure:"environment" toml:"environment"`
	Timeout     time.Duration     `mapstructure:"timeout" toml:"timeout"`
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
	RequireAuth       bool `mapstructure:"require_auth" toml:"require_auth"`
	MaxSessionsPerUser int  `mapstructure:"max_sessions_per_user" toml:"max_sessions_per_user"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `mapstructure:"level" toml:"level"`
	Format string `mapstructure:"format" toml:"format"`
	Output string `mapstructure:"output" toml:"output"`
}

// Load loads configuration from file, environment variables, and flags.
func Load(ctx context.Context) (*Config, error) {
	return loadConfig(ctx)
}
