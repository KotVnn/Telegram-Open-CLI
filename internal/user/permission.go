package user

import (
	"context"
	"errors"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// Permission represents a specific action permission.
type Permission string

const (
	PermissionCreateSession Permission = "session:create"
	PermissionDeleteSession Permission = "session:delete"
	PermissionCreateProject Permission = "project:create"
	PermissionDeleteProject Permission = "project:delete"
	PermissionManageUsers   Permission = "users:manage"
	PermissionManageRoles   Permission = "roles:manage"
	PermissionUseBackend    Permission = "backend:use"
)

// rolePermissions maps roles to their allowed permissions.
var rolePermissions = map[storage.UserRole][]Permission{
	storage.UserRoleAdmin: {
		PermissionCreateSession,
		PermissionDeleteSession,
		PermissionCreateProject,
		PermissionDeleteProject,
		PermissionManageUsers,
		PermissionManageRoles,
		PermissionUseBackend,
	},
	storage.UserRoleUser: {
		PermissionCreateSession,
		PermissionDeleteSession,
		PermissionCreateProject,
		PermissionDeleteProject,
		PermissionUseBackend,
	},
	storage.UserRoleViewer: {
		PermissionUseBackend,
	},
}

// HasPermission checks if a user has a specific permission.
func (m *Manager) HasPermission(ctx context.Context, userID int64, perm Permission) bool {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		return false
	}

	if !user.IsActive {
		return false
	}

	perms, ok := rolePermissions[user.Role]
	if !ok {
		return false
	}

	for _, p := range perms {
		if p == perm {
			return true
		}
	}

	return false
}

// GetPermissions returns all permissions for a user.
func (m *Manager) GetPermissions(ctx context.Context, userID int64) ([]Permission, error) {
	user, err := m.storage.GetUser(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrUserNotFound) {
			return nil, nil
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, nil
	}

	perms, ok := rolePermissions[user.Role]
	if !ok {
		return nil, nil
	}

	return perms, nil
}

// CheckPermission returns an error if user lacks permission.
func (m *Manager) CheckPermission(ctx context.Context, userID int64, perm Permission) error {
	if !m.HasPermission(ctx, userID, perm) {
		return ErrInsufficientPermissions
	}
	return nil
}

// ErrInsufficientPermissions is returned when a user lacks required permission.
var ErrInsufficientPermissions = errors.New("insufficient permissions")
