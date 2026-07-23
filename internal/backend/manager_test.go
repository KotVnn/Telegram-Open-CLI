package backend

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockBackend is a minimal stub implementing the Backend interface.
type mockBackend struct {
	name           string
	initErr        error
	startErr       error
	stopErr        error
	healthErr      error
	initialized    bool
	started        bool
	stopped        bool
	initCalledWith *BackendConfig
}

func (m *mockBackend) Name() string        { return m.name }
func (m *mockBackend) Description() string { return "mock backend" }
func (m *mockBackend) Version() string     { return "0.0.1" }

func (m *mockBackend) Initialize(_ context.Context, config BackendConfig) error {
	if m.initErr != nil {
		return m.initErr
	}
	m.initialized = true
	m.initCalledWith = &config
	return nil
}

func (m *mockBackend) Start(_ context.Context) error {
	if m.startErr != nil {
		return m.startErr
	}
	m.started = true
	return nil
}

func (m *mockBackend) Stop(_ context.Context) error {
	if m.stopErr != nil {
		return m.stopErr
	}
	m.stopped = true
	return nil
}

func (m *mockBackend) Health(_ context.Context) error { return m.healthErr }

func (m *mockBackend) CreateSession(_ context.Context, _ SessionOpts) (*Session, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) GetSession(_ context.Context, _ string) (*Session, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) ListSessions(_ context.Context, _ string) ([]*Session, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) DeleteSession(_ context.Context, _ string) error {
	return errors.New("not implemented")
}

func (m *mockBackend) SendMessage(_ context.Context, _ *SendMessageRequest) (*SendMessageResponse, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) StreamMessage(_ context.Context, _ *SendMessageRequest) (<-chan StreamChunk, error) {
	return nil, errors.New("not implemented")
}

func (m *mockBackend) Capabilities() *Capabilities {
	return &Capabilities{}
}

func TestNewManager(t *testing.T) {
	m := NewManager()
	require.NotNil(t, m)
	assert.Empty(t, m.registry)
	assert.Empty(t, m.instances)
}

func TestRegister(t *testing.T) {
	m := NewManager()

	m.Register("opencode", func() Backend { return &mockBackend{name: "opencode"} })
	m.Register("claude", func() Backend { return &mockBackend{name: "claude"} })

	names := m.List()
	assert.Len(t, names, 2)
	assert.Contains(t, names, "opencode")
	assert.Contains(t, names, "claude")
}

func TestRegister_Overwrite(t *testing.T) {
	m := NewManager()

	m.Register("opencode", func() Backend { return &mockBackend{name: "v1"} })
	m.Register("opencode", func() Backend { return &mockBackend{name: "v2"} })

	backend, err := m.Get("opencode")
	require.NoError(t, err)
	assert.Equal(t, "v2", backend.Name())
}

func TestGet(t *testing.T) {
	m := NewManager()
	factoryCalled := 0

	m.Register("opencode", func() Backend {
		factoryCalled++
		return &mockBackend{name: "opencode"}
	})

	b1, err := m.Get("opencode")
	require.NoError(t, err)
	assert.Equal(t, "opencode", b1.Name())
	assert.Equal(t, 1, factoryCalled)

	// Second call should return the same instance without calling factory again.
	b2, err := m.Get("opencode")
	require.NoError(t, err)
	assert.Same(t, b1, b2)
	assert.Equal(t, 1, factoryCalled)
}

func TestGet_NotFound(t *testing.T) {
	m := NewManager()

	b, err := m.Get("nonexistent")
	require.Nil(t, b)
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackendNotFound))
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestInitializeAll(t *testing.T) {
	m := NewManager()

	opencode := &mockBackend{name: "opencode"}
	claude := &mockBackend{name: "claude"}
	gemini := &mockBackend{name: "gemini"}

	m.Register("opencode", func() Backend { return opencode })
	m.Register("claude", func() Backend { return claude })
	m.Register("gemini", func() Backend { return gemini })

	configs := map[string]BackendConfig{
		"opencode": {Enabled: true, Command: "opencode"},
		"claude":   {Enabled: false, Command: "claude"},
		"gemini":   {Enabled: true, Command: "gemini"},
	}

	err := m.InitializeAll(context.Background(), configs)
	require.NoError(t, err)

	assert.True(t, opencode.initialized)
	assert.False(t, claude.initialized)
	assert.True(t, gemini.initialized)
	assert.Equal(t, "opencode", opencode.initCalledWith.Command)
}

func TestInitializeAll_Error(t *testing.T) {
	m := NewManager()

	m.Register("opencode", func() Backend {
		return &mockBackend{name: "opencode", initErr: errors.New("init failed")}
	})
	m.Register("claude", func() Backend {
		return &mockBackend{name: "claude"}
	})

	configs := map[string]BackendConfig{
		"opencode": {Enabled: true},
		"claude":   {Enabled: true},
	}

	err := m.InitializeAll(context.Background(), configs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "init failed")
}

func TestInitialize(t *testing.T) {
	m := NewManager()

	expected := &mockBackend{name: "opencode"}
	m.Register("opencode", func() Backend { return expected })

	config := BackendConfig{Enabled: true, Command: "opencode"}
	err := m.Initialize(context.Background(), "opencode", config)
	require.NoError(t, err)
	assert.True(t, expected.initialized)
	assert.Equal(t, config, *expected.initCalledWith)
}

func TestInitialize_NotFound(t *testing.T) {
	m := NewManager()

	err := m.Initialize(context.Background(), "nonexistent", BackendConfig{})
	require.Error(t, err)
	assert.True(t, errors.Is(err, ErrBackendNotFound))
}

func TestStart(t *testing.T) {
	m := NewManager()

	b1 := &mockBackend{name: "opencode"}
	b2 := &mockBackend{name: "claude"}

	// Pre-populate instances via factory registration and Get.
	m.Register("opencode", func() Backend { return b1 })
	m.Register("claude", func() Backend { return b2 })
	_, _ = m.Get("opencode")
	_, _ = m.Get("claude")

	err := m.Start(context.Background())
	require.NoError(t, err)
	assert.True(t, b1.started)
	assert.True(t, b2.started)
}

func TestStart_Error(t *testing.T) {
	m := NewManager()

	b1 := &mockBackend{name: "opencode", startErr: errors.New("start failed")}
	b2 := &mockBackend{name: "claude"}

	m.Register("opencode", func() Backend { return b1 })
	m.Register("claude", func() Backend { return b2 })
	_, _ = m.Get("opencode")
	_, _ = m.Get("claude")

	err := m.Start(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "start failed")
}

func TestStart_NoInstances(t *testing.T) {
	m := NewManager()

	err := m.Start(context.Background())
	require.NoError(t, err)
}

func TestStop(t *testing.T) {
	m := NewManager()

	b1 := &mockBackend{name: "opencode"}
	b2 := &mockBackend{name: "claude"}

	m.Register("opencode", func() Backend { return b1 })
	m.Register("claude", func() Backend { return b2 })
	_, _ = m.Get("opencode")
	_, _ = m.Get("claude")

	err := m.Stop(context.Background())
	require.NoError(t, err)
	assert.True(t, b1.stopped)
	assert.True(t, b2.stopped)
}

func TestStop_ContinuesOnError(t *testing.T) {
	m := NewManager()

	b1 := &mockBackend{name: "opencode", stopErr: errors.New("stop failed")}
	b2 := &mockBackend{name: "claude"}

	m.Register("opencode", func() Backend { return b1 })
	m.Register("claude", func() Backend { return b2 })
	_, _ = m.Get("opencode")
	_, _ = m.Get("claude")

	err := m.Stop(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "stop failed")
	// claude should still be stopped even though opencode failed.
	assert.True(t, b2.stopped)
}

func TestStop_NoInstances(t *testing.T) {
	m := NewManager()

	err := m.Stop(context.Background())
	require.NoError(t, err)
}

func TestList(t *testing.T) {
	m := NewManager()

	assert.Empty(t, m.List())

	m.Register("a", func() Backend { return &mockBackend{name: "a"} })
	m.Register("b", func() Backend { return &mockBackend{name: "b"} })
	m.Register("c", func() Backend { return &mockBackend{name: "c"} })

	names := m.List()
	assert.Len(t, names, 3)
	assert.Contains(t, names, "a")
	assert.Contains(t, names, "b")
	assert.Contains(t, names, "c")
}

func TestHealth(t *testing.T) {
	m := NewManager()

	b1 := &mockBackend{name: "opencode"}
	b2 := &mockBackend{name: "claude", healthErr: fmt.Errorf("unhealthy")}

	m.Register("opencode", func() Backend { return b1 })
	m.Register("claude", func() Backend { return b2 })
	_, _ = m.Get("opencode")
	_, _ = m.Get("claude")

	health := m.Health(context.Background())
	require.Len(t, health, 2)
	assert.NoError(t, health["opencode"])
	assert.Error(t, health["claude"])
	assert.Contains(t, health["claude"].Error(), "unhealthy")
}

func TestHealth_NoInstances(t *testing.T) {
	m := NewManager()

	health := m.Health(context.Background())
	assert.Empty(t, health)
}

func TestMustGet(t *testing.T) {
	m := NewManager()
	m.Register("opencode", func() Backend { return &mockBackend{name: "opencode"} })

	b := m.MustGet("opencode")
	assert.Equal(t, "opencode", b.Name())
}

func TestMustGet_Panic(t *testing.T) {
	m := NewManager()

	require.Panics(t, func() {
		m.MustGet("nonexistent")
	})
}
