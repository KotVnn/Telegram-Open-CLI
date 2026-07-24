package example

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/plugin"
)

// Plugin is an example plugin that demonstrates the plugin interface.
type Plugin struct {
	config    plugin.PluginConfig
	startTime time.Time
}

// New creates a new example plugin.
func New() *Plugin {
	return &Plugin{}
}

// Name returns the plugin name.
func (p *Plugin) Name() string {
	return "example"
}

// Description returns the plugin description.
func (p *Plugin) Description() string {
	return "An example plugin demonstrating TOC plugin capabilities"
}

// Version returns the plugin version.
func (p *Plugin) Version() string {
	return "1.0.0"
}

// Initialize initializes the plugin.
func (p *Plugin) Initialize(ctx context.Context, config plugin.PluginConfig) error {
	p.config = config
	return nil
}

// Start starts the plugin.
func (p *Plugin) Start(ctx context.Context) error {
	p.startTime = time.Now()
	return nil
}

// Stop stops the plugin.
func (p *Plugin) Stop(ctx context.Context) error {
	return nil
}

// Capabilities returns the plugin capabilities.
func (p *Plugin) Capabilities() *plugin.PluginCapabilities {
	return &plugin.PluginCapabilities{
		SupportsPreMessage:  true,
		SupportsPostMessage: true,
		SupportsPreCommand:  true,
		SupportsPostCommand: true,
		CustomCommands:      []string{"ping", "uptime"},
		CustomMiddleware:    false,
	}
}

// HandlePreMessage handles the pre-message hook.
func (p *Plugin) HandlePreMessage(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	// Example: Log all messages
	fmt.Printf("[example] Pre-message from user %d: %s\n", hookCtx.UserID, hookCtx.Message)

	return &plugin.HookResult{
		Continue: true,
	}, nil
}

// HandlePostMessage handles the post-message hook.
func (p *Plugin) HandlePostMessage(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	// Example: Could add analytics here
	return &plugin.HookResult{
		Continue: true,
	}, nil
}

// HandlePreCommand handles the pre-command hook.
func (p *Plugin) HandlePreCommand(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	// Example: Block certain commands for specific users
	blockedUsers := []int64{12345678}
	for _, uid := range blockedUsers {
		if hookCtx.UserID == uid {
			return &plugin.HookResult{
				Continue: false,
				Response: "You are not allowed to use this command.",
			}, nil
		}
	}

	return &plugin.HookResult{
		Continue: true,
	}, nil
}

// HandlePostCommand handles the post-command hook.
func (p *Plugin) HandlePostCommand(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	// Example: Log command execution
	fmt.Printf("[example] Post-command: %s\n", hookCtx.Command)

	return &plugin.HookResult{
		Continue: true,
	}, nil
}

// HandlePing handles the /ping command.
func (p *Plugin) HandlePing(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	return &plugin.HookResult{
		Continue: true,
		Response: "Pong! 🏓",
	}, nil
}

// HandleUptime handles the /uptime command.
func (p *Plugin) HandleUptime(ctx context.Context, hookCtx *plugin.HookContext) (*plugin.HookResult, error) {
	uptime := time.Since(p.startTime)
	hours := int(uptime.Hours())
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60

	response := fmt.Sprintf("Uptime: %dh %dm %ds", hours, minutes, seconds)

	return &plugin.HookResult{
		Continue: true,
		Response: response,
	}, nil
}

// Register registers all hooks and commands with the plugin manager.
func (p *Plugin) Register(m *plugin.Manager) error {
	// Register hooks
	m.RegisterHook(plugin.HookPreMessage, p.Name(), p.HandlePreMessage)
	m.RegisterHook(plugin.HookPostMessage, p.Name(), p.HandlePostMessage)
	m.RegisterHook(plugin.HookPreCommand, p.Name(), p.HandlePreCommand)
	m.RegisterHook(plugin.HookPostCommand, p.Name(), p.HandlePostCommand)

	// Custom commands are registered separately in the Telegram handler
	// This is just a demonstration of the pattern

	return nil
}

// IsCommand checks if a message is a specific command.
func IsCommand(message, command string) bool {
	return strings.HasPrefix(message, "/"+command)
}
