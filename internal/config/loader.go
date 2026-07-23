package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

func loadConfig(ctx context.Context) (*Config, error) {
	v := viper.New()

	v.SetDefault("version", "1")
	v.SetDefault("default_backend", "opencode")
	v.SetDefault("telegram.mode", "polling")
	v.SetDefault("telegram.timeout", 60)
	v.SetDefault("storage.path", defaultStoragePath())
	v.SetDefault("security.require_auth", true)
	v.SetDefault("security.max_sessions_per_user", 5)
	v.SetDefault("logging.level", "info")
	v.SetDefault("logging.format", "json")
	v.SetDefault("logging.output", "stdout")

	configPath := os.Getenv("TOC_CONFIG")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(home, ".toc", "config.toml")
		}
	}

	v.SetConfigFile(configPath)
	v.SetConfigType("toml")

	v.SetEnvPrefix("TOC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	if err := v.BindEnv("telegram.token", "TOC_TELEGRAM_TOKEN"); err != nil {
		return nil, fmt.Errorf("bind env TOC_TELEGRAM_TOKEN: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	if err := validate(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func defaultStoragePath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "toc.db"
	}
	return filepath.Join(home, ".toc", "toc.db")
}
