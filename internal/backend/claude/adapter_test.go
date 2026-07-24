package claude

import (
	"context"
	"testing"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdapter_Name(t *testing.T) {
	a := New()
	assert.Equal(t, "claude", a.Name())
}

func TestAdapter_Description(t *testing.T) {
	a := New()
	assert.NotEmpty(t, a.Description())
}

func TestAdapter_Version(t *testing.T) {
	a := New()
	assert.NotEmpty(t, a.Version())
}

func TestAdapter_Initialize(t *testing.T) {
	a := New()
	ctx := context.Background()

	t.Run("disabled backend", func(t *testing.T) {
		err := a.Initialize(ctx, backend.BackendConfig{Enabled: false})
		require.ErrorIs(t, err, ErrBackendDisabled)
	})

	t.Run("missing command", func(t *testing.T) {
		err := a.Initialize(ctx, backend.BackendConfig{Enabled: true})
		require.ErrorIs(t, err, ErrMissingCommand)
	})

	t.Run("valid config", func(t *testing.T) {
		err := a.Initialize(ctx, backend.BackendConfig{
			Enabled: true,
			Command: "claude",
			Args:    []string{"-p"},
		})
		require.NoError(t, err)
		assert.True(t, a.initialized)
	})
}

func TestAdapter_StartStop(t *testing.T) {
	a := New()
	ctx := context.Background()

	t.Run("start without init", func(t *testing.T) {
		err := a.Start(ctx)
		require.ErrorIs(t, err, ErrNotInitialized)
	})

	require.NoError(t, a.Initialize(ctx, backend.BackendConfig{
		Enabled: true,
		Command: "claude",
	}))

	require.NoError(t, a.Start(ctx))

	t.Run("stop clears sessions", func(t *testing.T) {
		_, err := a.CreateSession(ctx, backend.SessionOpts{Title: "test"})
		require.NoError(t, err)

		require.NoError(t, a.Stop(ctx))
		sessions, err := a.ListSessions(ctx, "")
		require.NoError(t, err)
		assert.Empty(t, sessions)
		assert.False(t, a.initialized)
	})
}

func TestAdapter_SessionManagement(t *testing.T) {
	a := New()
	ctx := context.Background()

	require.NoError(t, a.Initialize(ctx, backend.BackendConfig{
		Enabled: true,
		Command: "claude",
	}))

	t.Run("create session", func(t *testing.T) {
		session, err := a.CreateSession(ctx, backend.SessionOpts{
			Title:     "Test Session",
			Model:     "claude-sonnet-4-20250514",
			WorkingDir: "/tmp",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.Equal(t, "claude", session.Backend)
		assert.Equal(t, "Test Session", session.Title)
		assert.Equal(t, backend.SessionStatusActive, session.Status)
	})

	t.Run("get session", func(t *testing.T) {
		session, err := a.CreateSession(ctx, backend.SessionOpts{Title: "Get Test"})
		require.NoError(t, err)

		got, err := a.GetSession(ctx, session.ID)
		require.NoError(t, err)
		assert.Equal(t, session.ID, got.ID)
	})

	t.Run("get nonexistent session", func(t *testing.T) {
		_, err := a.GetSession(ctx, "nonexistent")
		require.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("list sessions", func(t *testing.T) {
		sessions, err := a.ListSessions(ctx, "")
		require.NoError(t, err)
		assert.NotEmpty(t, sessions)
	})

	t.Run("delete session", func(t *testing.T) {
		session, err := a.CreateSession(ctx, backend.SessionOpts{Title: "Delete Test"})
		require.NoError(t, err)

		require.NoError(t, a.DeleteSession(ctx, session.ID))
		_, err = a.GetSession(ctx, session.ID)
		require.ErrorIs(t, err, ErrSessionNotFound)
	})

	t.Run("delete nonexistent session", func(t *testing.T) {
		err := a.DeleteSession(ctx, "nonexistent")
		require.ErrorIs(t, err, ErrSessionNotFound)
	})
}

func TestAdapter_Capabilities(t *testing.T) {
	a := New()
	caps := a.Capabilities()

	assert.True(t, caps.SupportsStreaming)
	assert.True(t, caps.SupportsFiles)
	assert.True(t, caps.SupportsMultiModal)
	assert.True(t, caps.SupportsToolCalling)
	assert.Greater(t, caps.MaxTokens, 0)
	assert.NotEmpty(t, caps.SupportedModels)
}

func TestAdapter_BuildArgs(t *testing.T) {
	a := New()
	_ = a.Initialize(context.Background(), backend.BackendConfig{
		Enabled: true,
		Command: "claude",
		Args:    []string{"-p"},
	})

	session := &backend.Session{ID: "test-session"}
	req := &backend.SendMessageRequest{
		Content: "hello",
		Model:   "claude-sonnet-4-20250514",
	}

	args := a.buildArgs(session, req)

	assert.Contains(t, args, "-p")
	assert.Contains(t, args, "--session")
	assert.Contains(t, args, "test-session")
	assert.Contains(t, args, "--model")
	assert.Contains(t, args, "claude-sonnet-4-20250514")
	assert.Contains(t, args, "hello")
}

func TestAdapter_Health(t *testing.T) {
	a := New()
	ctx := context.Background()

	t.Run("not initialized", func(t *testing.T) {
		err := a.Health(ctx)
		require.ErrorIs(t, err, ErrNotInitialized)
	})

	require.NoError(t, a.Initialize(ctx, backend.BackendConfig{
		Enabled: true,
		Command: "nonexistent-command-12345",
	}))

	t.Run("command not found", func(t *testing.T) {
		err := a.Health(ctx)
		require.Error(t, err)
	})
}

func TestAdapter_SessionMetadata(t *testing.T) {
	a := New()
	ctx := context.Background()

	require.NoError(t, a.Initialize(ctx, backend.BackendConfig{
		Enabled: true,
		Command: "claude",
	}))

	session, err := a.CreateSession(ctx, backend.SessionOpts{
		Title: "Metadata Test",
		Metadata: map[string]interface{}{
			"key": "value",
		},
	})
	require.NoError(t, err)
	assert.Equal(t, "value", session.Metadata["key"])
	assert.WithinDuration(t, time.Now(), session.CreatedAt, time.Second)
}
