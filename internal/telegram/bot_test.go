package telegram

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockAdapter struct {
	sentMessages []OutgoingMessage
}

func (m *mockAdapter) Start(ctx context.Context) error                          { return nil }
func (m *mockAdapter) Stop(ctx context.Context) error                           { return nil }
func (m *mockAdapter) SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error {
	m.sentMessages = append(m.sentMessages, msg)
	return nil
}
func (m *mockAdapter) SendMessageWithResult(ctx context.Context, chatID int64, msg OutgoingMessage) (*SendResult, error) {
	m.sentMessages = append(m.sentMessages, msg)
	return &SendResult{MessageID: 1}, nil
}
func (m *mockAdapter) EditMessage(ctx context.Context, chatID int64, messageID int, text string) error {
	return nil
}
func (m *mockAdapter) SendDocument(ctx context.Context, chatID int64, doc Document) error {
	return nil
}
func (m *mockAdapter) AnswerCallback(ctx context.Context, callbackID string, text string) error {
	return nil
}
func (m *mockAdapter) HandleCommand(cmd string, handler HandlerFunc)    {}
func (m *mockAdapter) HandleMessage(pattern string, handler HandlerFunc) {}
func (m *mockAdapter) HandleCallback(pattern string, handler CallbackHandlerFunc) {}
func (m *mockAdapter) HandleDefault(handler HandlerFunc)                {}
func (m *mockAdapter) Use(middlewares ...Middleware)                     {}

func newTestBot() *Bot {
	return &Bot{
		commandHandlers: make(map[string]HandlerFunc),
	}
}

func TestNewBotInvalidToken(t *testing.T) {
	_, err := New(Config{Token: ""})
	assert.Error(t, err)

	_, err = New(Config{Token: "invalid"})
	assert.Error(t, err)
}

func TestBotHandleCommand(t *testing.T) {
	b := newTestBot()

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	b.HandleCommand("start", handler)

	assert.Len(t, b.commandHandlers, 1)
	assert.NotNil(t, b.commandHandlers["start"])

	err := b.commandHandlers["start"](context.Background(), &IncomingMessage{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestBotHandleCommandOverwrite(t *testing.T) {
	b := newTestBot()

	handler1 := func(ctx context.Context, msg *IncomingMessage) error {
		return fmt.Errorf("handler1")
	}
	handler2 := func(ctx context.Context, msg *IncomingMessage) error {
		return fmt.Errorf("handler2")
	}

	b.HandleCommand("test", handler1)
	b.HandleCommand("test", handler2)

	assert.Len(t, b.commandHandlers, 1)

	err := b.commandHandlers["test"](context.Background(), &IncomingMessage{})
	assert.EqualError(t, err, "handler2")
}

func TestBotHandleMessage(t *testing.T) {
	b := newTestBot()

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		return nil
	}

	b.HandleMessage("hello", handler)

	assert.Len(t, b.messageHandlers, 1)
	assert.Equal(t, "hello", b.messageHandlers[0].pattern)
}

func TestBotHandleMessageMultiple(t *testing.T) {
	b := newTestBot()

	b.HandleMessage("pattern1", func(ctx context.Context, msg *IncomingMessage) error { return nil })
	b.HandleMessage("pattern2", func(ctx context.Context, msg *IncomingMessage) error { return nil })
	b.HandleMessage("pattern3", func(ctx context.Context, msg *IncomingMessage) error { return nil })

	assert.Len(t, b.messageHandlers, 3)
}

func TestBotHandleDefault(t *testing.T) {
	b := newTestBot()
	assert.Nil(t, b.defaultHandler)

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		return nil
	}

	b.HandleDefault(handler)
	assert.NotNil(t, b.defaultHandler)
}

func TestBotUse(t *testing.T) {
	b := newTestBot()
	assert.Empty(t, b.middlewares)

	mw := func(next HandlerFunc) HandlerFunc {
		return next
	}

	b.Use(mw)
	assert.Len(t, b.middlewares, 1)

	b.Use(mw, mw)
	assert.Len(t, b.middlewares, 3)
}

func TestMiddlewareChainExecutionOrder(t *testing.T) {
	b := newTestBot()
	var order []string

	mw1 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			order = append(order, "mw1-before")
			err := next(ctx, msg)
			order = append(order, "mw1-after")
			return err
		}
	}

	mw2 := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			order = append(order, "mw2-before")
			err := next(ctx, msg)
			order = append(order, "mw2-after")
			return err
		}
	}

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		order = append(order, "handler")
		return nil
	}

	b.Use(mw1, mw2)
	b.HandleDefault(handler)

	msg := &IncomingMessage{Text: "test"}
	resolved := b.resolveHandler(msg)

	chain := resolved
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		chain = b.middlewares[i](chain)
	}

	err := chain(context.Background(), msg)
	require.NoError(t, err)

	expected := []string{"mw1-before", "mw2-before", "handler", "mw2-after", "mw1-after"}
	assert.Equal(t, expected, order)
}

func TestMiddlewareCanShortCircuit(t *testing.T) {
	b := newTestBot()

	handlerCalled := false
	blockingMW := func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			return nil
		}
	}

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		handlerCalled = true
		return nil
	}

	b.Use(blockingMW)
	b.HandleDefault(handler)

	msg := &IncomingMessage{Text: "test"}
	resolved := b.resolveHandler(msg)

	chain := resolved
	for i := len(b.middlewares) - 1; i >= 0; i-- {
		chain = b.middlewares[i](chain)
	}

	_ = chain(context.Background(), msg)
	assert.False(t, handlerCalled)
}

func TestParseIncomingMessageTextOnly(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   100,
			Chat: models.Chat{ID: 200},
			From: &models.User{
				ID:        300,
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
			},
			Text: "hello world",
		},
	}

	msg := parseIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, 100, msg.MessageID)
	assert.Equal(t, int64(200), msg.ChatID)
	assert.Equal(t, int64(300), msg.FromID)
	assert.Equal(t, "testuser", msg.FromUsername)
	assert.Equal(t, "Test", msg.FromFirstName)
	assert.Equal(t, "User", msg.FromLastName)
	assert.Equal(t, "hello world", msg.Text)
	assert.Empty(t, msg.Command)
	assert.Nil(t, msg.Document)
}

func TestParseIncomingMessageWithCommand(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   100,
			Chat: models.Chat{ID: 200},
			From: &models.User{ID: 300, Username: "user"},
			Text: "/start arg1 arg2",
			Entities: []models.MessageEntity{
				{
					Type:   "bot_command",
					Offset: 0,
					Length: 6,
				},
			},
		},
	}

	msg := parseIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, "start", msg.Command)
	assert.Equal(t, []string{"arg1", "arg2"}, msg.Args)
}

func TestParseIncomingMessageWithDocument(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   100,
			Chat: models.Chat{ID: 200},
			From: &models.User{ID: 300},
			Text: "check this file",
			Document: &models.Document{
				FileID:   "file_abc",
				FileName: "test.go",
				MimeType: "text/plain",
				FileSize: 1024,
			},
		},
	}

	msg := parseIncomingMessage(update)
	require.NotNil(t, msg)
	require.NotNil(t, msg.Document)
	assert.Equal(t, "file_abc", msg.Document.FileID)
	assert.Equal(t, "test.go", msg.Document.FileName)
	assert.Equal(t, "text/plain", msg.Document.MimeType)
	assert.Equal(t, int64(1024), msg.Document.FileSize)
}

func TestParseIncomingMessageWithReply(t *testing.T) {
	update := &models.Update{
		Message: &models.Message{
			ID:   100,
			Chat: models.Chat{ID: 200},
			From: &models.User{ID: 300},
			Text: "reply text",
			ReplyToMessage: &models.Message{
				ID: 500,
			},
		},
	}

	msg := parseIncomingMessage(update)
	require.NotNil(t, msg)
	assert.Equal(t, 500, msg.ReplyToMessageID)
}

func TestParseIncomingMessageNilUpdate(t *testing.T) {
	update := &models.Update{}
	msg := parseIncomingMessage(update)
	assert.Nil(t, msg)
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{"empty", "", nil},
		{"whitespace", "   ", nil},
		{"single", "arg1", []string{"arg1"}},
		{"multiple", "arg1 arg2 arg3", []string{"arg1", "arg2", "arg3"}},
		{"extra spaces", "  arg1   arg2  ", []string{"arg1", "arg2"}},
		{"tabs", "arg1\targ2", []string{"arg1", "arg2"}},
		{"mixed", " arg1 \n arg2 \t arg3 ", []string{"arg1", "arg2", "arg3"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseArgs(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestResolveHandlerCommandPriority(t *testing.T) {
	b := newTestBot()

	commandCalled := false
	messageCalled := false
	defaultCalled := false

	b.HandleCommand("start", func(ctx context.Context, msg *IncomingMessage) error {
		commandCalled = true
		return nil
	})
	b.HandleMessage("start", func(ctx context.Context, msg *IncomingMessage) error {
		messageCalled = true
		return nil
	})
	b.HandleDefault(func(ctx context.Context, msg *IncomingMessage) error {
		defaultCalled = true
		return nil
	})

	msg := &IncomingMessage{
		Command: "start",
		Text:    "/start",
	}

	handler := b.resolveHandler(msg)
	err := handler(context.Background(), msg)
	require.NoError(t, err)

	assert.True(t, commandCalled)
	assert.False(t, messageCalled)
	assert.False(t, defaultCalled)
}

func TestResolveHandlerMessagePattern(t *testing.T) {
	b := newTestBot()

	messageCalled := false
	defaultCalled := false

	b.HandleMessage("error", func(ctx context.Context, msg *IncomingMessage) error {
		messageCalled = true
		return nil
	})
	b.HandleDefault(func(ctx context.Context, msg *IncomingMessage) error {
		defaultCalled = true
		return nil
	})

	msg := &IncomingMessage{
		Text: "something error occurred",
	}

	handler := b.resolveHandler(msg)
	err := handler(context.Background(), msg)
	require.NoError(t, err)

	assert.True(t, messageCalled)
	assert.False(t, defaultCalled)
}

func TestResolveHandlerMessagePatternNoMatch(t *testing.T) {
	b := newTestBot()

	messageCalled := false
	defaultCalled := false

	b.HandleMessage("error", func(ctx context.Context, msg *IncomingMessage) error {
		messageCalled = true
		return nil
	})
	b.HandleDefault(func(ctx context.Context, msg *IncomingMessage) error {
		defaultCalled = true
		return nil
	})

	msg := &IncomingMessage{
		Text: "something happened",
	}

	handler := b.resolveHandler(msg)
	err := handler(context.Background(), msg)
	require.NoError(t, err)

	assert.False(t, messageCalled)
	assert.True(t, defaultCalled)
}

func TestResolveHandlerDefaultFallback(t *testing.T) {
	b := newTestBot()

	defaultCalled := false

	b.HandleDefault(func(ctx context.Context, msg *IncomingMessage) error {
		defaultCalled = true
		return nil
	})

	msg := &IncomingMessage{Text: "random text"}
	handler := b.resolveHandler(msg)
	err := handler(context.Background(), msg)
	require.NoError(t, err)

	assert.True(t, defaultCalled)
}

func TestResolveHandlerNoHandlerReturnsNoop(t *testing.T) {
	b := newTestBot()

	msg := &IncomingMessage{Text: "nothing matches"}
	handler := b.resolveHandler(msg)

	err := handler(context.Background(), msg)
	assert.NoError(t, err)
}

func TestResolveHandlerUnknownCommand(t *testing.T) {
	b := newTestBot()

	defaultCalled := false

	b.HandleCommand("start", func(ctx context.Context, msg *IncomingMessage) error {
		return nil
	})
	b.HandleDefault(func(ctx context.Context, msg *IncomingMessage) error {
		defaultCalled = true
		return nil
	})

	msg := &IncomingMessage{
		Command: "unknown",
		Text:    "/unknown",
	}

	handler := b.resolveHandler(msg)
	_ = handler(context.Background(), msg)

	assert.True(t, defaultCalled)
}

func TestResolveHandlerFirstMessagePatternWins(t *testing.T) {
	b := newTestBot()

	var matched string

	b.HandleMessage("error", func(ctx context.Context, msg *IncomingMessage) error {
		matched = "error"
		return nil
	})
	b.HandleMessage("critical", func(ctx context.Context, msg *IncomingMessage) error {
		matched = "critical"
		return nil
	})

	msg := &IncomingMessage{Text: "critical error occurred"}
	handler := b.resolveHandler(msg)
	_ = handler(context.Background(), msg)

	assert.Equal(t, "error", matched)
}

func TestSessionManager(t *testing.T) {
	sm := &SessionManager{
		activeSessions: make(map[int64]string),
	}

	t.Run("GetActiveSession Empty", func(t *testing.T) {
		sessionID := sm.GetActiveSession(100)
		assert.Empty(t, sessionID)
	})

	t.Run("SetActiveSession", func(t *testing.T) {
		sm.SetActiveSession(100, "session_abc")
		assert.Equal(t, "session_abc", sm.GetActiveSession(100))
	})

	t.Run("SetActiveSession Overwrite", func(t *testing.T) {
		sm.SetActiveSession(100, "session_xyz")
		assert.Equal(t, "session_xyz", sm.GetActiveSession(100))
	})

	t.Run("ClearActiveSession", func(t *testing.T) {
		sm.ClearActiveSession(100)
		assert.Empty(t, sm.GetActiveSession(100))
	})

	t.Run("ClearActiveSession NonExistent", func(t *testing.T) {
		sm.ClearActiveSession(999)
		assert.Empty(t, sm.GetActiveSession(999))
	})

	t.Run("MultipleUsersIndependent", func(t *testing.T) {
		sm.SetActiveSession(1, "s1")
		sm.SetActiveSession(2, "s2")
		sm.SetActiveSession(3, "s3")

		assert.Equal(t, "s1", sm.GetActiveSession(1))
		assert.Equal(t, "s2", sm.GetActiveSession(2))
		assert.Equal(t, "s3", sm.GetActiveSession(3))

		sm.ClearActiveSession(2)

		assert.Equal(t, "s1", sm.GetActiveSession(1))
		assert.Empty(t, sm.GetActiveSession(2))
		assert.Equal(t, "s3", sm.GetActiveSession(3))
	})
}

func TestRecoveryMiddleware(t *testing.T) {
	logger := zerolog.Nop()
	mw := RecoveryMiddleware(logger)

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{ChatID: 1})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRecoveryMiddlewareCatchesPanic(t *testing.T) {
	logger := zerolog.Nop()
	mw := RecoveryMiddleware(logger)

	panicCount := 0
	panickingHandler := func(ctx context.Context, msg *IncomingMessage) error {
		panicCount++
		panic("test panic")
	}

	wrapped := mw(panickingHandler)

	err := wrapped(context.Background(), &IncomingMessage{ChatID: 1, Text: "/start"})
	assert.NoError(t, err)
	assert.Equal(t, 1, panicCount)
}

func TestLoggingMiddleware(t *testing.T) {
	var buf strings.Builder
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	mw := LoggingMiddleware(logger)

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{
		FromID:       100,
		FromUsername: "user",
		Text:         "hello",
		ChatID:       200,
	})
	require.NoError(t, err)
	assert.True(t, called)

	output := buf.String()
	assert.Contains(t, output, "message received")
	assert.Contains(t, output, "message processed")
}

func TestLoggingMiddlewarePassesError(t *testing.T) {
	var buf strings.Builder
	logger := zerolog.New(&buf).With().Timestamp().Logger()

	mw := LoggingMiddleware(logger)

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		return fmt.Errorf("handler error")
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{
		FromID:       100,
		FromUsername: "user",
		Text:         "test",
		ChatID:       1,
	})
	assert.EqualError(t, err, "handler error")
}

func TestAuthMiddlewareAllowedUser(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, []int64{100, 200}, nil)

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 100, ChatID: 1})
	require.NoError(t, err)
	assert.True(t, called)
	assert.Empty(t, adapter.sentMessages)
}

func TestAuthMiddlewareBlockedUser(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, []int64{100, 200}, nil)

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 999, ChatID: 1})
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.False(t, called)
	require.Len(t, adapter.sentMessages, 1)
	assert.Equal(t, "You are not authorized to use this bot.", adapter.sentMessages[0].Text)
}

func TestAuthMiddlewareNoAllowedUsersPassesAll(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, nil, nil)

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 999})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAuthMiddlewareGroupChatAllowed(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, nil, []int64{-100123})

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 100, ChatID: -100123})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAuthMiddlewareGroupChatBlocked(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, nil, []int64{-100123})

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 100, ChatID: -100456})
	require.ErrorIs(t, err, ErrUnauthorized)
	assert.False(t, called)
	require.Len(t, adapter.sentMessages, 1)
	assert.Equal(t, "This chat is not authorized to use this bot.", adapter.sentMessages[0].Text)
}

func TestAuthMiddlewarePrivateChatSkipsChatCheck(t *testing.T) {
	adapter := &mockAdapter{}
	mw := AuthMiddleware(adapter, nil, []int64{-100123})

	called := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		called = true
		return nil
	}

	wrapped := mw(handler)
	err := wrapped(context.Background(), &IncomingMessage{FromID: 100, ChatID: 100})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestRateLimitMiddleware(t *testing.T) {
	mw := RateLimitMiddleware(10, 1)

	handler := func(ctx context.Context, msg *IncomingMessage) error {
		return nil
	}

	wrapped := mw(handler)
	msg := &IncomingMessage{FromID: 100}

	err := wrapped(context.Background(), msg)
	assert.NoError(t, err)
}

func TestRateLimitMiddlewareBlocks(t *testing.T) {
	mw := RateLimitMiddleware(1, 1)

	handlerCalled := false
	handler := func(ctx context.Context, msg *IncomingMessage) error {
		handlerCalled = true
		return nil
	}

	wrapped := mw(handler)
	msg := &IncomingMessage{FromID: 100}

	_ = wrapped(context.Background(), msg)
	assert.True(t, handlerCalled)

	handlerCalled = false
	_ = wrapped(context.Background(), msg)
	assert.False(t, handlerCalled)
}

func TestContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []int64
		item     int64
		expected bool
	}{
		{"found", []int64{1, 2, 3}, 2, true},
		{"not found", []int64{1, 2, 3}, 4, false},
		{"empty", []int64{}, 1, false},
		{"first", []int64{1, 2, 3}, 1, true},
		{"last", []int64{1, 2, 3}, 3, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestKeyboardBuilder(t *testing.T) {
	kb := NewKeyboard().
		AddButton("Yes", "confirm:yes").
		AddButton("No", "cancel").
		NewRow().
		AddButton("Maybe", "confirm:maybe").
		Build()

	assert.Len(t, kb.InlineKeyboard, 2)
	assert.Len(t, kb.InlineKeyboard[0], 2)
	assert.Len(t, kb.InlineKeyboard[1], 1)

	assert.Equal(t, "Yes", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "confirm:yes", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "No", kb.InlineKeyboard[0][1].Text)
	assert.Equal(t, "cancel", kb.InlineKeyboard[0][1].CallbackData)
	assert.Equal(t, "Maybe", kb.InlineKeyboard[1][0].Text)
	assert.Equal(t, "confirm:maybe", kb.InlineKeyboard[1][0].CallbackData)
}

func TestNewSessionKeyboard(t *testing.T) {
	sessions := []struct {
		ID, Title string
	}{
		{"s1", "Session 1"},
		{"s2", "Session 2"},
	}

	kb := NewSessionKeyboard(sessions)

	assert.Len(t, kb.InlineKeyboard, 3)
	assert.Equal(t, "Session 1", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "session:s1", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "Session 2", kb.InlineKeyboard[1][0].Text)
	assert.Equal(t, "session:s2", kb.InlineKeyboard[1][0].CallbackData)
	assert.Equal(t, "+ New Session", kb.InlineKeyboard[2][0].Text)
	assert.Equal(t, "new_session", kb.InlineKeyboard[2][0].CallbackData)
}

func TestNewConfirmKeyboard(t *testing.T) {
	kb := NewConfirmKeyboard("delete")

	require.Len(t, kb.InlineKeyboard, 1)
	require.Len(t, kb.InlineKeyboard[0], 2)
	assert.Equal(t, "Yes", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "confirm:delete", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "No", kb.InlineKeyboard[0][1].Text)
	assert.Equal(t, "cancel", kb.InlineKeyboard[0][1].CallbackData)
}

func TestNewModelKeyboard(t *testing.T) {
	kb := NewModelKeyboard([]string{"gpt-4", "claude-3"})

	require.Len(t, kb.InlineKeyboard, 2)
	assert.Equal(t, "gpt-4", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "model:gpt-4", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "claude-3", kb.InlineKeyboard[1][0].Text)
	assert.Equal(t, "model:claude-3", kb.InlineKeyboard[1][0].CallbackData)
}

func TestNewAgentKeyboard(t *testing.T) {
	kb := NewAgentKeyboard([]string{"coder", "reviewer"})

	require.Len(t, kb.InlineKeyboard, 2)
	assert.Equal(t, "coder", kb.InlineKeyboard[0][0].Text)
	assert.Equal(t, "agent:coder", kb.InlineKeyboard[0][0].CallbackData)
	assert.Equal(t, "reviewer", kb.InlineKeyboard[1][0].Text)
	assert.Equal(t, "agent:reviewer", kb.InlineKeyboard[1][0].CallbackData)
}

func TestBotStop(t *testing.T) {
	b := newTestBot()
	err := b.Stop(context.Background())
	assert.NoError(t, err)
}
