package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_ValidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	t.Setenv("TOC_CONFIG", configFile)

	cfg, err := loadConfig(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "1", cfg.Version)
	assert.Equal(t, "opencode", cfg.DefaultBackend)
	assert.Equal(t, "test-token", cfg.Telegram.Token)
	assert.True(t, cfg.Security.RequireAuth)
	assert.Equal(t, 5, cfg.Security.MaxSessionsPerUser)

	backend, ok := cfg.Backends["opencode"]
	assert.True(t, ok)
	assert.True(t, backend.Enabled)
	assert.Equal(t, "opencode", backend.Command)
}

func TestLoad_MissingToken(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = ""

[backends.opencode]
enabled = true
command = "opencode"

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", configFile)
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrMissingConfig)
}

func TestLoad_MissingDefaultBackend(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = ""

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", configFile)
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrMissingConfig)
}

func TestLoad_BackendNotFound(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "nonexistent"

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", configFile)
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, cfg)
	assert.ErrorIs(t, err, ErrMissingConfig)
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr error
	}{
		{
			name: "valid config",
			config: &Config{
				DefaultBackend: "opencode",
				Telegram: TelegramConfig{
					Token: "test-token",
				},
				Backends: map[string]Backend{
					"opencode": {Enabled: true},
				},
			},
			wantErr: nil,
		},
		{
			name: "missing token",
			config: &Config{
				DefaultBackend: "opencode",
				Telegram: TelegramConfig{
					Token: "",
				},
				Backends: map[string]Backend{
					"opencode": {Enabled: true},
				},
			},
			wantErr: ErrMissingConfig,
		},
		{
			name: "missing default backend",
			config: &Config{
				DefaultBackend: "",
				Telegram: TelegramConfig{
					Token: "test-token",
				},
				Backends: map[string]Backend{
					"opencode": {Enabled: true},
				},
			},
			wantErr: ErrMissingConfig,
		},
		{
			name: "backend not found",
			config: &Config{
				DefaultBackend: "nonexistent",
				Telegram: TelegramConfig{
					Token: "test-token",
				},
				Backends: map[string]Backend{
					"opencode": {Enabled: true},
				},
			},
			wantErr: ErrMissingConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validate(tt.config)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestLoad_InvalidConfigFile(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `invalid toml content {{`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", configFile)
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_NoConfigFile(t *testing.T) {
	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", "/nonexistent/path/config.toml")
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	assert.Error(t, err)
	assert.Nil(t, cfg)
}

func TestLoad_EnvOverride(t *testing.T) {
	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", "/nonexistent/path/config.toml")
	defer os.Setenv("TOC_CONFIG", origVal)

	os.Setenv("TOC_TELEGRAM_TOKEN", "env-token")
	defer os.Unsetenv("TOC_TELEGRAM_TOKEN")

	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = "file-token"

[backends.opencode]
enabled = true
command = "opencode"

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	os.Setenv("TOC_CONFIG", configFile)

	cfg, err := Load(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "env-token", cfg.Telegram.Token)
}

func TestLoad_Defaults(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	origVal := os.Getenv("TOC_CONFIG")
	os.Setenv("TOC_CONFIG", configFile)
	defer os.Setenv("TOC_CONFIG", origVal)

	cfg, err := Load(context.Background())
	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "1", cfg.Version)
	assert.Equal(t, "opencode", cfg.DefaultBackend)
	assert.Equal(t, 5, cfg.Security.MaxSessionsPerUser)
}

func TestViper_Backends(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")

	err = v.ReadInConfig()
	require.NoError(t, err)

	var cfg Config
	err = v.Unmarshal(&cfg)
	require.NoError(t, err)

	backend, ok := cfg.Backends["opencode"]
	assert.True(t, ok)
	assert.True(t, backend.Enabled)
	assert.Equal(t, "opencode", backend.Command)
}

func TestViper_BackendsWithAutomaticEnv(t *testing.T) {
	tempDir := t.TempDir()
	configFile := filepath.Join(tempDir, "config.toml")

	configContent := `version = "1"
default_backend = "opencode"

[telegram]
token = "test-token"

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]

[storage]
path = ""

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "console"
output = "stdout"
`

	err := os.WriteFile(configFile, []byte(configContent), 0644)
	require.NoError(t, err)

	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("toml")
	v.SetEnvPrefix("TOC")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	err = v.ReadInConfig()
	require.NoError(t, err)

	var cfg Config
	err = v.Unmarshal(&cfg)
	require.NoError(t, err)

	t.Logf("default_backend from Viper: %q", v.GetString("default_backend"))
	t.Logf("Config.DefaultBackend: %q", cfg.DefaultBackend)
	t.Logf("All keys: %v", v.AllKeys())
}
