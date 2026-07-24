package telegram

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// SessionManager manages active sessions per user.
type SessionManager struct {
	storage        storage.Storage
	backend        backend.Backend
	projectManager *ProjectManager
	logger         zerolog.Logger
	botToken       string
	mu             sync.RWMutex
	activeSessions map[int64]string
	pendingModels  map[int64]string
}

// NewSessionManager creates a new session manager.
func NewSessionManager(storage storage.Storage, backend backend.Backend, pm *ProjectManager, logger zerolog.Logger, botToken string) *SessionManager {
	return &SessionManager{
		storage:        storage,
		backend:        backend,
		projectManager: pm,
		logger:         logger,
		botToken:       botToken,
		activeSessions: make(map[int64]string),
		pendingModels:  make(map[int64]string),
	}
}

// GetActiveSession returns the active session ID for a user.
func (m *SessionManager) GetActiveSession(userID int64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeSessions[userID]
}

// SetActiveSession sets the active session ID for a user.
func (m *SessionManager) SetActiveSession(userID int64, sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeSessions[userID] = sessionID
}

// ClearActiveSession removes the active session for a user.
func (m *SessionManager) ClearActiveSession(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeSessions, userID)
}

// HandleNew handles the /new [name] command.
func HandleNew(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		projectID := sm.projectManager.GetActiveProject(msg.FromID)
		if projectID == "" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active project. Use /project new <name> <path> to create one first.",
			})
		}

		project, err := sm.projectManager.manager.Get(ctx, projectID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Active project not found. Use /project list to see available projects.",
			})
		}

		name := "Untitled Session"
		if len(msg.Args) > 0 {
			name = msg.Args[0]
		}

		session, err := sm.backend.CreateSession(ctx, backend.SessionOpts{
			Title:      name,
			ProjectID:  projectID,
			WorkingDir: project.Path,
		})
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to create session. Please try again.",
			})
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
			CreatedAt:  session.CreatedAt,
			UpdatedAt:  session.UpdatedAt,
		}

		if err := sm.storage.SaveSession(ctx, storageSession); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to save session. Please try again.",
			})
		}

		sm.SetActiveSession(msg.FromID, session.ID)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Session created!\n\nID: %s\nName: %s\n\nUse /switch %s to switch to this session.", session.ID, name, session.ID),
		})
	}
}

// HandleSessions handles the /sessions command.
func HandleSessions(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		sessions, err := sm.storage.ListSessions(ctx, storage.SessionFilter{})
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to list sessions. Please try again.",
			})
		}

		if len(sessions) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No sessions found. Use /new to create one.",
			})
		}

		activeSession := sm.GetActiveSession(msg.FromID)

		text := "Sessions:\n\n"
		for _, s := range sessions {
			status := string(s.Status)
			if s.ID == activeSession {
				status += " (active)"
			}
			projectInfo := ""
			if s.ProjectID != "" {
				p, err := sm.storage.GetProject(ctx, s.ProjectID)
				if err == nil {
					projectInfo = fmt.Sprintf("\nProject: %s (%s)", p.Name, p.Path)
				}
			}
			text += fmt.Sprintf("ID: %s\nName: %s\nStatus: %s%s\n\n", s.ID, s.Title, status, projectInfo)
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: text,
		})
	}
}

// HandleSwitch handles the /switch [id] command.
func HandleSwitch(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if len(msg.Args) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /switch <session_id>",
			})
		}

		sessionID := msg.Args[0]

		session, err := sm.storage.GetSession(ctx, sessionID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Session not found: %s", sessionID),
			})
		}

		if session.Status == storage.SessionStatusClosed {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Session is closed.",
			})
		}

		sm.SetActiveSession(msg.FromID, sessionID)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Switched to session: %s\n\nName: %s", sessionID, session.Title),
		})
	}
}

// HandleClose handles the /close [id] command.
func HandleClose(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		sessionID := sm.GetActiveSession(msg.FromID)
		if len(msg.Args) > 0 {
			sessionID = msg.Args[0]
		}

		if sessionID == "" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active session. Use /close <session_id> to close a specific session.",
			})
		}

		session, err := sm.storage.GetSession(ctx, sessionID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Session not found: %s", sessionID),
			})
		}

		session.Status = storage.SessionStatusClosed
		session.UpdatedAt = time.Now()

		if err := sm.storage.UpdateSession(ctx, session); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Error closing session: %v", err),
			})
		}

		sm.ClearActiveSession(msg.FromID)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Session closed: %s\n\nName: %s", sessionID, session.Title),
		})
	}
}

// HandleStatus handles the /status command.
func HandleStatus(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		sessionID := sm.GetActiveSession(msg.FromID)
		if sessionID == "" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active session. Use /new to create one or /sessions to list existing ones.",
			})
		}

		session, err := sm.storage.GetSession(ctx, sessionID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Session not found: %s", sessionID),
			})
		}

		text := fmt.Sprintf("Active Session:\n\nID: %s\nName: %s\nStatus: %s\nBackend: %s\nModel: %s\nAgent: %s\nCreated: %s\nUpdated: %s",
			session.ID, session.Title, session.Status, session.Backend, session.Model, session.Agent, session.CreatedAt.Format(time.RFC3339), session.UpdatedAt.Format(time.RFC3339))

		if session.ProjectID != "" {
			p, err := sm.storage.GetProject(ctx, session.ProjectID)
			if err == nil {
				text += fmt.Sprintf("\nProject: %s (%s)", p.Name, p.Path)
			}
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: text,
		})
	}
}
