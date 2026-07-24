package plugin

import (
	"context"
)

// Plugin defines the interface for TOC plugins.
type Plugin interface {
	// Metadata
	Name() string
	Description() string
	Version() string

	// Lifecycle
	Initialize(ctx context.Context, config PluginConfig) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// Capabilities
	Capabilities() *PluginCapabilities
}

// PluginConfig holds plugin-specific configuration.
type PluginConfig struct {
	Enabled bool
	Options map[string]interface{}
}

// PluginCapabilities describes what the plugin provides.
type PluginCapabilities struct {
	// Hook types this plugin supports
	SupportsPreMessage  bool
	SupportsPostMessage bool
	SupportsPreCommand  bool
	SupportsPostCommand bool

	// Custom commands
	CustomCommands []string

	// Custom middleware
	CustomMiddleware bool
}

// Hook represents a point where plugins can intercept processing.
type Hook string

const (
	HookPreMessage  Hook = "pre_message"
	HookPostMessage Hook = "post_message"
	HookPreCommand  Hook = "pre_command"
	HookPostCommand Hook = "post_command"
	HookOnStart     Hook = "on_start"
	HookOnStop      Hook = "on_stop"
)

// HookContext provides context to hook handlers.
type HookContext struct {
	Hook    Hook
	UserID  int64
	ChatID  int64
	Command string
	Message string
	Data    map[string]interface{}
}

// HookResult determines if processing should continue.
type HookResult struct {
	Continue bool
	Response string
	Data     map[string]interface{}
}

// Handler is a function that handles a hook.
type Handler func(ctx context.Context, hookCtx *HookContext) (*HookResult, error)
