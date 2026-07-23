package project

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// Manager handles project operations.
type Manager struct {
	storage storage.Storage
}

// NewManager creates a new project manager.
func NewManager(storage storage.Storage) *Manager {
	return &Manager{storage: storage}
}

// Create creates a new project.
func (m *Manager) Create(ctx context.Context, name, path, description, defaultBackend string, ownerID int64) (*storage.Project, error) {
	project := &storage.Project{
		ID:             uuid.New().String(),
		Name:           name,
		Path:           path,
		Description:    description,
		DefaultBackend: defaultBackend,
		OwnerID:        ownerID,
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := m.storage.SaveProject(ctx, project); err != nil {
		return nil, fmt.Errorf("save project: %w", err)
	}

	return project, nil
}

// Get retrieves a project by ID.
func (m *Manager) Get(ctx context.Context, id string) (*storage.Project, error) {
	return m.storage.GetProject(ctx, id)
}

// List returns projects for a user.
func (m *Manager) List(ctx context.Context, userID int64) ([]*storage.Project, error) {
	return m.storage.ListProjects(ctx, userID)
}

// Update updates a project.
func (m *Manager) Update(ctx context.Context, project *storage.Project) error {
	project.UpdatedAt = time.Now()
	if err := m.storage.SaveProject(ctx, project); err != nil {
		return fmt.Errorf("save project: %w", err)
	}
	return nil
}

// Delete removes a project.
func (m *Manager) Delete(ctx context.Context, id string) error {
	if err := m.storage.DeleteProject(ctx, id); err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}
