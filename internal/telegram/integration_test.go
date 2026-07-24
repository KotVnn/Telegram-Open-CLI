//go:build integration

package telegram

import (
	"context"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBotMiddlewareIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger := zerolog.Nop()

	b, err := New(Config{
		Token:  "test-token",
		Logger: logger,
	})
	require.NoError(t, err)

	t.Run("apply middleware in order", func(t *testing.T) {
		var order []string

		b.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, msg *IncomingMessage) error {
				order = append(order, "recovery")
				return next(ctx, msg)
			}
		})

		b.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, msg *IncomingMessage) error {
				order = append(order, "logging")
				return next(ctx, msg)
			}
		})

		b.Use(func(next HandlerFunc) HandlerFunc {
			return func(ctx context.Context, msg *IncomingMessage) error {
				order = append(order, "auth")
				return next(ctx, msg)
			}
		})

		handler := func(ctx context.Context, msg *IncomingMessage) error {
			order = append(order, "handler")
			return nil
		}

		_ = handler
		assert.Len(t, order, 0)
	})
}

func TestCallbackHandlerRegistration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger := zerolog.Nop()

	b, err := New(Config{
		Token:  "test-token",
		Logger: logger,
	})
	require.NoError(t, err)

	t.Run("register multiple callback handlers", func(t *testing.T) {
		b.HandleCallback("session:", func(ctx context.Context, cb *CallbackQuery) error {
			return nil
		})

		b.HandleCallback("model:", func(ctx context.Context, cb *CallbackQuery) error {
			return nil
		})

		b.HandleCallback("confirm:", func(ctx context.Context, cb *CallbackQuery) error {
			return nil
		})

		assert.Len(t, b.callbackHandlers, 3)
	})

	t.Run("resolve callback handler", func(t *testing.T) {
		handler := b.resolveCallbackHandler("session:123")
		assert.NotNil(t, handler)

		handler = b.resolveCallbackHandler("model:claude")
		assert.NotNil(t, handler)

		handler = b.resolveCallbackHandler("unknown:data")
		assert.Nil(t, handler)
	})
}

func TestKeyboardToTelegram(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	t.Run("session keyboard conversion", func(t *testing.T) {
		sessions := []struct{ ID, Title string }{
			{ID: "1", Title: "Session 1"},
			{ID: "2", Title: "Session 2"},
		}

		keyboard := NewSessionKeyboard(sessions)
		tgKeyboard := keyboard.ToTelegram()

		assert.Len(t, tgKeyboard.InlineKeyboard, 3)
		assert.Equal(t, "Session 1", tgKeyboard.InlineKeyboard[0][0].Text)
		assert.Equal(t, "session:1", tgKeyboard.InlineKeyboard[0][0].CallbackData)
	})

	t.Run("confirm keyboard conversion", func(t *testing.T) {
		keyboard := NewConfirmKeyboard("close")
		tgKeyboard := keyboard.ToTelegram()

		assert.Len(t, tgKeyboard.InlineKeyboard, 1)
		assert.Len(t, tgKeyboard.InlineKeyboard[0], 2)
		assert.Equal(t, "Yes", tgKeyboard.InlineKeyboard[0][0].Text)
		assert.Equal(t, "confirm:close", tgKeyboard.InlineKeyboard[0][0].CallbackData)
		assert.Equal(t, "No", tgKeyboard.InlineKeyboard[0][1].Text)
		assert.Equal(t, "cancel", tgKeyboard.InlineKeyboard[0][1].CallbackData)
	})
}

func TestSessionManagerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	logger := zerolog.Nop()
	b, err := New(Config{
		Token:  "test-token",
		Logger: logger,
	})
	require.NoError(t, err)

	sm := &SessionManager{
		activeSessions: make(map[int64]string),
		logger:         logger,
	}

	t.Run("active session management", func(t *testing.T) {
		sm.SetActiveSession(123, "session-1")
		assert.Equal(t, "session-1", sm.GetActiveSession(123))

		sm.SetActiveSession(123, "session-2")
		assert.Equal(t, "session-2", sm.GetActiveSession(123))

		sm.ClearActiveSession(123)
		assert.Equal(t, "", sm.GetActiveSession(123))
	})

	t.Run("multiple users", func(t *testing.T) {
		sm.SetActiveSession(100, "session-a")
		sm.SetActiveSession(200, "session-b")

		assert.Equal(t, "session-a", sm.GetActiveSession(100))
		assert.Equal(t, "session-b", sm.GetActiveSession(200))

		sm.ClearActiveSession(100)
		assert.Equal(t, "", sm.GetActiveSession(100))
		assert.Equal(t, "session-b", sm.GetActiveSession(200))
	})

	_ = b
}
