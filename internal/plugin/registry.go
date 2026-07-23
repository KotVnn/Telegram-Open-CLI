package plugin

import (
	"context"
	"fmt"
	"sync"

	"github.com/rs/zerolog"
)

// Manager manages plugin lifecycle and hook execution.
type Manager struct {
	plugins map[string]Plugin
	hooks   map[Hook][]handlerEntry
	mu      sync.RWMutex
	logger  zerolog.Logger
}

type handlerEntry struct {
	pluginName string
	handler    Handler
}

// NewManager creates a new plugin manager.
func NewManager(logger zerolog.Logger) *Manager {
	return &Manager{
		plugins: make(map[string]Plugin),
		hooks:   make(map[Hook][]handlerEntry),
		logger:  logger,
	}
}

// Register registers a plugin with the manager.
func (m *Manager) Register(plugin Plugin) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := plugin.Name()
	if _, exists := m.plugins[name]; exists {
		return fmt.Errorf("plugin %s already registered", name)
	}

	m.plugins[name] = plugin
	m.logger.Info().Str("plugin", name).Msg("plugin registered")

	return nil
}

// Initialize initializes all registered plugins.
func (m *Manager) Initialize(ctx context.Context, configs map[string]PluginConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, plugin := range m.plugins {
		cfg, exists := configs[name]
		if !exists {
			cfg = PluginConfig{Enabled: true}
		}

		if !cfg.Enabled {
			m.logger.Info().Str("plugin", name).Msg("plugin disabled")
			continue
		}

		if err := plugin.Initialize(ctx, cfg); err != nil {
			return fmt.Errorf("initialize plugin %s: %w", name, err)
		}

		m.logger.Info().Str("plugin", name).Msg("plugin initialized")
	}

	return nil
}

// Start starts all initialized plugins.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, plugin := range m.plugins {
		if err := plugin.Start(ctx); err != nil {
			return fmt.Errorf("start plugin %s: %w", name, err)
		}
		m.logger.Info().Str("plugin", name).Msg("plugin started")
	}

	return nil
}

// Stop stops all running plugins.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, plugin := range m.plugins {
		if err := plugin.Stop(ctx); err != nil {
			m.logger.Error().Err(err).Str("plugin", name).Msg("failed to stop plugin")
		}
	}

	return nil
}

// RegisterHook registers a handler for a specific hook.
func (m *Manager) RegisterHook(hook Hook, pluginName string, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks[hook] = append(m.hooks[hook], handlerEntry{
		pluginName: pluginName,
		handler:    handler,
	})
}

// ExecuteHooks executes all handlers for a specific hook.
func (m *Manager) ExecuteHooks(ctx context.Context, hook Hook, hookCtx *HookContext) (*HookResult, error) {
	m.mu.RLock()
	handlers := m.hooks[hook]
	m.mu.RUnlock()

	result := &HookResult{Continue: true}

	for _, entry := range handlers {
		res, err := entry.handler(ctx, hookCtx)
		if err != nil {
			m.logger.Error().Err(err).
				Str("plugin", entry.pluginName).
				Str("hook", string(hook)).
				Msg("hook handler error")
			continue
		}

		if !res.Continue {
			return res, nil
		}

		// Merge data
		if res.Data != nil {
			if result.Data == nil {
				result.Data = make(map[string]interface{})
			}
			for k, v := range res.Data {
				result.Data[k] = v
			}
		}

		if res.Response != "" {
			result.Response = res.Response
		}
	}

	return result, nil
}

// GetPlugin returns a plugin by name.
func (m *Manager) GetPlugin(name string) (Plugin, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	plugin, exists := m.plugins[name]
	return plugin, exists
}

// ListPlugins returns all registered plugin names.
func (m *Manager) ListPlugins() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.plugins))
	for name := range m.plugins {
		names = append(names, name)
	}
	return names
}
