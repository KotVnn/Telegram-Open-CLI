package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// Manager handles user operations.
type Manager struct {
	storage storage.Storage
}

// NewManager creates a new user manager.
func NewManager(storage storage.Storage) *Manager {
	return &Manager{storage: storage}
}

// GetOrCreate retrieves a user or creates a new one.
func (m *Manager) GetOrCreate(ctx context.Context, id int64, username, firstName, lastName string) (*storage.User, error) {
	user, err := m.storage.GetUser(ctx, id)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			user = &storage.User{
				ID:        id,
				Username:  username,
				FirstName: firstName,
				LastName:  lastName,
				Role:      storage.UserRoleUser,
				IsActive:  true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := m.storage.SaveUser(ctx, user); err != nil {
				return nil, fmt.Errorf("create user: %w", err)
			}
			return user, nil
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	user.Username = username
	user.FirstName = firstName
	user.LastName = lastName
	user.UpdatedAt = time.Now()

	if err := m.storage.SaveUser(ctx, user); err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	return user, nil
}

// IsAuthorized checks if a user is authorized.
func (m *Manager) IsAuthorized(ctx context.Context, userID int64) bool {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return false
	}
	return user.IsActive
}

// IsAdmin checks if a user is an admin.
func (m *Manager) IsAdmin(ctx context.Context, userID int64) bool {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return false
	}
	return user.Role == storage.UserRoleAdmin
}

// SetRole sets a user's role.
func (m *Manager) SetRole(ctx context.Context, userID int64, role storage.UserRole) error {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	user.Role = role
	user.UpdatedAt = time.Now()

	if err := m.storage.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

// List returns all users.
func (m *Manager) List(ctx context.Context) ([]*storage.User, error) {
	users, err := m.storage.ListUsers(ctx)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

// SetActive enables or disables a user.
func (m *Manager) SetActive(ctx context.Context, userID int64, active bool) error {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return fmt.Errorf("get user: %w", err)
	}

	user.IsActive = active
	user.UpdatedAt = time.Now()

	if err := m.storage.SaveUser(ctx, user); err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

// Delete removes a user.
func (m *Manager) Delete(ctx context.Context, userID int64) error {
	if err := m.storage.DeleteUser(ctx, userID); err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}
