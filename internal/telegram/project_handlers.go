package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/KotVnn/Telegram-Open-CLI/internal/project"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// ProjectManager manages active projects per user (similar to SessionManager).
type ProjectManager struct {
	manager        *project.Manager
	storage        storage.Storage
	mu             sync.RWMutex
	activeProjects map[int64]string // userID -> projectID
}

// NewProjectManager creates a new project manager for Telegram.
func NewProjectManager(manager *project.Manager, storage storage.Storage) *ProjectManager {
	return &ProjectManager{
		manager:        manager,
		storage:        storage,
		activeProjects: make(map[int64]string),
	}
}

// GetActiveProject returns the active project ID for a user.
func (m *ProjectManager) GetActiveProject(userID int64) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.activeProjects[userID]
}

// SetActiveProject sets the active project ID for a user.
func (m *ProjectManager) SetActiveProject(userID int64, projectID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.activeProjects[userID] = projectID
}

// ClearActiveProject removes the active project for a user.
func (m *ProjectManager) ClearActiveProject(userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.activeProjects, userID)
}

// HandleProjectNew handles the /project new <name> command.
func HandleProjectNew(adapter Adapter, pm *ProjectManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if len(msg.Args) < 2 || msg.Args[0] != "new" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /project new <name> [path]",
			})
		}

		name := msg.Args[1]
		path := ""
		if len(msg.Args) > 2 {
			path = msg.Args[2]
		}

		project, err := pm.manager.Create(ctx, name, path, "", "opencode", msg.FromID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to create project: %v", err),
			})
		}

		pm.SetActiveProject(msg.FromID, project.ID)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Project created!\n\nID: %s\nName: %s\nPath: %s",
				project.ID, project.Name, project.Path),
		})
	}
}

// HandleProjectList handles the /project list command.
func HandleProjectList(adapter Adapter, pm *ProjectManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		projects, err := pm.manager.List(ctx, msg.FromID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to list projects.",
			})
		}

		if len(projects) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No projects found. Use /project new <name> to create one.",
			})
		}

		activeProject := pm.GetActiveProject(msg.FromID)

		text := "Projects:\n\n"
		for _, p := range projects {
			status := ""
			if p.ID == activeProject {
				status = " (active)"
			}
			text += fmt.Sprintf("ID: %s\nName: %s\nPath: %s%s\n\n",
				p.ID, p.Name, p.Path, status)
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: text,
		})
	}
}

// HandleProjectSwitch handles the /project switch <id> command.
func HandleProjectSwitch(adapter Adapter, pm *ProjectManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if len(msg.Args) < 2 || msg.Args[0] != "switch" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /project switch <project_id>",
			})
		}

		projectID := msg.Args[1]

		project, err := pm.manager.Get(ctx, projectID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Project not found: %s", projectID),
			})
		}

		if project.OwnerID != msg.FromID {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "You can only switch to your own projects.",
			})
		}

		pm.SetActiveProject(msg.FromID, project.ID)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Switched to project: %s\n\nPath: %s", project.Name, project.Path),
		})
	}
}

// HandleProjectDelete handles the /project delete <id> command.
func HandleProjectDelete(adapter Adapter, pm *ProjectManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if len(msg.Args) < 2 || msg.Args[0] != "delete" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /project delete <project_id>",
			})
		}

		projectID := msg.Args[1]

		project, err := pm.manager.Get(ctx, projectID)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Project not found: %s", projectID),
			})
		}

		if project.OwnerID != msg.FromID {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Only the project owner can delete it.",
			})
		}

		if err := pm.manager.Delete(ctx, projectID); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to delete project: %v", err),
			})
		}

		if pm.GetActiveProject(msg.FromID) == projectID {
			pm.ClearActiveProject(msg.FromID)
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Project deleted: %s", project.Name),
		})
	}
}

// HandleProject handles the /project command dispatcher.
func HandleProject(adapter Adapter, pm *ProjectManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if len(msg.Args) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: `Project commands:

/project new <name> [path] - Create a new project
/project list - List all projects
/project switch <id> - Switch to a project
/project delete <id> - Delete a project`,
			})
		}

		subcmd := strings.ToLower(msg.Args[0])
		switch subcmd {
		case "new":
			return HandleProjectNew(adapter, pm)(ctx, msg)
		case "list":
			return HandleProjectList(adapter, pm)(ctx, msg)
		case "switch":
			return HandleProjectSwitch(adapter, pm)(ctx, msg)
		case "delete":
			return HandleProjectDelete(adapter, pm)(ctx, msg)
		default:
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Unknown project subcommand: %s", subcmd),
			})
		}
	}
}
