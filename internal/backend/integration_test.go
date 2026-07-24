//go:build integration

package backend

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBackendManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	manager := NewManager()

	t.Run("register and initialize backends", func(t *testing.T) {
		manager.Register("test-backend-1", func() Backend {
			return &integrationMockBackend{name: "test-backend-1"}
		})

		manager.Register("test-backend-2", func() Backend {
			return &integrationMockBackend{name: "test-backend-2"}
		})

		configs := map[string]BackendConfig{
			"test-backend-1": {Enabled: true, Command: "test1"},
			"test-backend-2": {Enabled: true, Command: "test2"},
		}

		err := manager.InitializeAll(ctx, configs)
		require.NoError(t, err)
	})

	t.Run("list registered backends", func(t *testing.T) {
		names := manager.List()
		assert.Len(t, names, 2)
		assert.Contains(t, names, "test-backend-1")
		assert.Contains(t, names, "test-backend-2")
	})

	t.Run("get backend instance", func(t *testing.T) {
		b, err := manager.Get("test-backend-1")
		require.NoError(t, err)
		assert.Equal(t, "test-backend-1", b.Name())
	})

	t.Run("start all backends", func(t *testing.T) {
		err := manager.Start(ctx)
		require.NoError(t, err)
	})

	t.Run("health check", func(t *testing.T) {
		health := manager.Health(ctx)
		assert.Len(t, health, 2)
		for name, err := range health {
			assert.NoError(t, err, "backend %s should be healthy", name)
		}
	})

	t.Run("stop all backends", func(t *testing.T) {
		err := manager.Stop(ctx)
		require.NoError(t, err)
	})
}

func TestBackendSessionFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	ctx := context.Background()
	b := &integrationMockBackend{name: "test"}

	err := b.Initialize(ctx, BackendConfig{Enabled: true, Command: "test"})
	require.NoError(t, err)

	err = b.Start(ctx)
	require.NoError(t, err)

	t.Run("create session", func(t *testing.T) {
		session, err := b.CreateSession(ctx, SessionOpts{
			Title:      "Integration Test",
			Model:      "test-model",
			WorkingDir: "/tmp",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "test", session.Backend)
	})

	t.Run("list sessions", func(t *testing.T) {
		sessions, err := b.ListSessions(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, sessions)
	})

	err = b.Stop(ctx)
	require.NoError(t, err)
}

type integrationMockBackend struct {
	name        string
	initialized bool
	sessions    map[string]*Session
}

func (m *integrationMockBackend) Name() string        { return m.name }
func (m *integrationMockBackend) Description() string { return "Mock Backend" }
func (m *integrationMockBackend) Version() string     { return "1.0.0" }

func (m *integrationMockBackend) Initialize(ctx context.Context, config BackendConfig) error {
	m.initialized = true
	m.sessions = make(map[string]*Session)
	return nil
}

func (m *integrationMockBackend) Start(ctx context.Context) error { return nil }
func (m *integrationMockBackend) Stop(ctx context.Context) error  { return nil }
func (m *integrationMockBackend) Health(ctx context.Context) error { return nil }

func (m *integrationMockBackend) CreateSession(ctx context.Context, opts SessionOpts) (*Session, error) {
	session := &Session{
		ID:      "test-session",
		Backend: m.name,
		Title:   opts.Title,
		Status:  SessionStatusActive,
	}
	m.sessions[session.ID] = session
	return session, nil
}

func (m *integrationMockBackend) GetSession(ctx context.Context, id string) (*Session, error) {
	if s, ok := m.sessions[id]; ok {
		return s, nil
	}
	return nil, fmt.Errorf("session not found")
}

func (m *integrationMockBackend) ListSessions(ctx context.Context, projectID string) ([]*Session, error) {
	var sessions []*Session
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions, nil
}

func (m *integrationMockBackend) DeleteSession(ctx context.Context, id string) error {
	delete(m.sessions, id)
	return nil
}

func (m *integrationMockBackend) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
	return &SendMessageResponse{
		ID:      "response-1",
		Content: "Mock response",
	}, nil
}

func (m *integrationMockBackend) StreamMessage(ctx context.Context, req *SendMessageRequest) (<-chan StreamChunk, error) {
	ch := make(chan StreamChunk, 1)
	go func() {
		defer close(ch)
		ch <- StreamChunk{Content: "Mock ", Done: false}
		ch <- StreamChunk{Content: "response", Done: false}
		ch <- StreamChunk{Done: true}
	}()
	return ch, nil
}

func (m *integrationMockBackend) Capabilities() *Capabilities {
	return &Capabilities{
		SupportsStreaming: true,
		SupportsFiles:     true,
		MaxTokens:         100000,
	}
}
