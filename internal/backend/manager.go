package backend

import (
	"context"
	"errors"
	"fmt"
	"sync"
)

// Manager manages backend instances.
type Manager struct {
	registry  map[string]func() Backend
	instances map[string]Backend
	mu        sync.RWMutex
}

// NewManager creates a new backend manager.
func NewManager() *Manager {
	return &Manager{
		registry:  make(map[string]func() Backend),
		instances: make(map[string]Backend),
	}
}

// Register registers a backend factory.
func (m *Manager) Register(name string, factory func() Backend) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.registry[name] = factory
}

// Get returns a backend instance by name.
func (m *Manager) Get(name string) (Backend, error) {
	m.mu.RLock()
	if backend, ok := m.instances[name]; ok {
		m.mu.RUnlock()
		return backend, nil
	}
	m.mu.RUnlock()

	m.mu.Lock()
	defer m.mu.Unlock()

	if backend, ok := m.instances[name]; ok {
		return backend, nil
	}

	factory, ok := m.registry[name]
	if !ok {
		return nil, fmt.Errorf("%w: backend %q", ErrBackendNotFound, name)
	}

	backend := factory()
	m.instances[name] = backend
	return backend, nil
}

// Initialize initializes a backend with the given configuration.
func (m *Manager) Initialize(ctx context.Context, name string, config BackendConfig) error {
	backend, err := m.Get(name)
	if err != nil {
		return err
	}

	return backend.Initialize(ctx, config)
}

// InitializeAll initializes all registered backends.
func (m *Manager) InitializeAll(ctx context.Context, configs map[string]BackendConfig) error {
	for name := range m.registry {
		if config, ok := configs[name]; ok && config.Enabled {
			if err := m.Initialize(ctx, name, config); err != nil {
				return fmt.Errorf("initialize backend %q: %w", name, err)
			}
		}
	}
	return nil
}

// Start starts all initialized backends.
func (m *Manager) Start(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, backend := range m.instances {
		if err := backend.Start(ctx); err != nil {
			return fmt.Errorf("start backend %q: %w", name, err)
		}
	}
	return nil
}

// Stop stops all running backends.
func (m *Manager) Stop(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var errs []error
	for name, backend := range m.instances {
		if err := backend.Stop(ctx); err != nil {
			errs = append(errs, fmt.Errorf("stop backend %q: %w", name, err))
		}
	}
	return errors.Join(errs...)
}

// List returns all registered backend names.
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.registry))
	for name := range m.registry {
		names = append(names, name)
	}
	return names
}

// Health checks the health of all running backends.
func (m *Manager) Health(ctx context.Context) map[string]error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health := make(map[string]error)
	for name, backend := range m.instances {
		health[name] = backend.Health(ctx)
	}
	return health
}

// ErrBackendNotFound is returned when a backend is not registered.
var ErrBackendNotFound = errors.New("backend not found")

// MustGet is like Get but panics if the backend is not found.
func (m *Manager) MustGet(name string) Backend {
	backend, err := m.Get(name)
	if err != nil {
		panic(fmt.Sprintf("backend %q not found", name))
	}
	return backend
}
