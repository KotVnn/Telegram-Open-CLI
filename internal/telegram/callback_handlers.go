package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
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
		models, _, err := listModelsAgents(ctx, sm.backend)
		if err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Failed to list models")
		}

		keyboard := NewModelKeyboard(models)

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

		sm.mu.Lock()
		sm.pendingModels[cb.FromID] = model
		sm.mu.Unlock()

		_, agents, err := listModelsAgents(ctx, sm.backend)
		if err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Failed to list agents")
		}

		keyboard := NewAgentKeyboard(agents)

		if err := adapter.SendMessage(ctx, cb.ChatID, OutgoingMessage{
			Text:        "Model: " + model + "\nSelect agent:",
			ReplyMarkup: keyboard.ToTelegram(),
		}); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "")
	}
}

// listModelsAgents returns available models and agents, falling back to the
// backend's static capabilities when it does not support dynamic listing.
func listModelsAgents(ctx context.Context, b backend.Backend) ([]string, []string, error) {
	if lister, ok := b.(backend.ModelLister); ok {
		models, agents, err := lister.ListModels(ctx)
		if err == nil && len(models) > 0 {
			return models, agents, nil
		}
	}
	caps := b.Capabilities()
	return caps.SupportedModels, caps.SupportedAgents, nil
}

// HandleAgentCallback handles agent selection callback.
func HandleAgentCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		agent := strings.TrimPrefix(cb.Data, "agent:")

		projectID := sm.projectManager.GetActiveProject(cb.FromID)
		if projectID == "" {
			return adapter.AnswerCallback(ctx, cb.ID, "No active project. Use /project new first.")
		}

		project, err := sm.projectManager.manager.Get(ctx, projectID)
		if err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Active project not found")
		}

		sm.mu.Lock()
		model := sm.pendingModels[cb.FromID]
		delete(sm.pendingModels, cb.FromID)
		sm.mu.Unlock()

		session, err := sm.backend.CreateSession(ctx, backend.SessionOpts{
			Title:      "New Session",
			Model:      model,
			Agent:      agent,
			ProjectID:  projectID,
			WorkingDir: project.Path,
		})
		if err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Failed to create session")
		}

		storageSession := &storage.Session{
			ID:         session.ID,
			Backend:    session.Backend,
			ProjectID:  session.ProjectID,
			Title:      session.Title,
			Status:     storage.SessionStatus(session.Status),
			Model:      session.Model,
			Agent:      session.Agent,
			WorkingDir: session.WorkingDir,
			ExternalID: session.ExternalID,
			CreatedAt:  session.CreatedAt,
			UpdatedAt:  session.UpdatedAt,
		}

		if err := sm.storage.SaveSession(ctx, storageSession); err != nil {
			return adapter.AnswerCallback(ctx, cb.ID, "Failed to save session")
		}

		sm.SetActiveSession(cb.FromID, session.ID)

		if err := adapter.EditMessage(ctx, cb.ChatID, cb.MessageID,
			fmt.Sprintf("Session created!\n\nID: %s\nModel: %s\nAgent: %s\n\nSend a message to start.", session.ID, model, agent)); err != nil {
			return err
		}

		return adapter.AnswerCallback(ctx, cb.ID, "Session created")
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
