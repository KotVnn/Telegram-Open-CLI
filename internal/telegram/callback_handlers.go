package telegram

import (
	"context"
	"strings"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// HandleSessionCallback handles session selection callback.
func HandleSessionCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		sessionID := strings.TrimPrefix(cb.Data, "session:")

		session, err := sm.storage.GetSession(ctx, sessionID)
		if err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Session not found")
		}

		sm.mu.Lock()
		sm.activeSessions[cb.FromID] = session.ID
		sm.mu.Unlock()

		text := "Switched to session: " + session.Title
		if err := adapter.EditMessage(ctx, cb.ChatID, cb.MessageID, text); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "")
	}
}

// HandleNewSessionCallback handles new session callback.
func HandleNewSessionCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		backend := sm.backend
		caps := backend.Capabilities()

		keyboard := NewModelKeyboard(caps.SupportedModels)

		if err := adapter.SendMessage(ctx, cb.ChatID, OutgoingMessage{
			Text:        "Select model:",
			ReplyMarkup: keyboard.ToTelegram(),
		}); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "")
	}
}

// HandleConfirmCallback handles confirmation callback.
func HandleConfirmCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		action := strings.TrimPrefix(cb.Data, "confirm:")

		switch action {
		case "close":
			userID := cb.FromID
			sm.mu.RLock()
			sessionID, ok := sm.activeSessions[userID]
			sm.mu.RUnlock()

			if !ok {
				return adapter.AnswerCallback(ctx, cb.ID, "No active session")
			}

			session, err := sm.storage.GetSession(ctx, sessionID)
			if err != nil {
				return adapter.AnswerCallback(ctx, cb.ID, "Session not found")
			}

			session.Status = storage.SessionStatusClosed
			session.UpdatedAt = time.Now()

			if err := sm.storage.UpdateSession(ctx, session); err != nil {
				return adapter.AnswerCallback(ctx, cb.ID, "Failed to close session")
			}

			sm.mu.Lock()
			delete(sm.activeSessions, userID)
			sm.mu.Unlock()

			if err := adapter.EditMessage(ctx, cb.ChatID, cb.MessageID,
				"Session closed."); err != nil {
				return err
			}

			return adapter.AnswerCallback(ctx, cb.ID, "Session closed")
		default:
			return adapter.AnswerCallback(ctx, cb.ID, "Unknown action")
		}
	}
}

// HandleModelCallback handles model selection callback.
func HandleModelCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		model := strings.TrimPrefix(cb.Data, "model:")

		backend := sm.backend
		caps := backend.Capabilities()

		keyboard := NewAgentKeyboard(caps.SupportedAgents)

		if err := adapter.SendMessage(ctx, cb.ChatID, OutgoingMessage{
			Text:        "Model: " + model + "\nSelect agent:",
			ReplyMarkup: keyboard.ToTelegram(),
		}); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "")
	}
}

// HandleAgentCallback handles agent selection callback.
func HandleAgentCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		agent := strings.TrimPrefix(cb.Data, "agent:")

		if err := adapter.EditMessage(ctx, cb.ChatID, cb.MessageID,
			"Agent: "+agent+"\n\nSession ready! Send a message to start."); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "")
	}
}

// HandleCancelCallback handles cancel callback.
func HandleCancelCallback(adapter Adapter) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		if err := adapter.EditMessage(ctx, cb.ChatID, cb.MessageID,
			"Cancelled."); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "Cancelled")
	}
}
