package telegram

import (
	"context"
	"fmt"

	"github.com/KotVnn/Telegram-Open-CLI/internal/user"
)

// RequirePermission creates a middleware that checks user permission.
func RequirePermission(um *user.Manager, perm user.Permission) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			if !um.HasPermission(ctx, msg.FromID, perm) {
				return fmt.Errorf("permission denied: %s", perm)
			}
			return next(ctx, msg)
		}
	}
}

// RequireAdmin creates a middleware that requires admin role.
func RequireAdmin(um *user.Manager) Middleware {
	return RequirePermission(um, user.PermissionManageUsers)
}

// RequireRole creates a middleware that requires specific role or higher.
func RequireRole(um *user.Manager, minRole string) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			user, err := um.GetOrCreate(ctx, msg.FromID, msg.FromUsername, msg.FromFirstName, msg.FromLastName)
			if err != nil {
				return fmt.Errorf("get user: %w", err)
			}

			if !hasMinRole(string(user.Role), minRole) {
				return fmt.Errorf("requires role %s or higher", minRole)
			}

			return next(ctx, msg)
		}
	}
}

// hasMinRole checks if the user's role meets the minimum requirement.
func hasMinRole(userRole, minRole string) bool {
	roleHierarchy := map[string]int{
		"viewer": 1,
		"user":   2,
		"admin":  3,
	}

	userLevel, ok := roleHierarchy[string(userRole)]
	if !ok {
		return false
	}

	minLevel, ok := roleHierarchy[minRole]
	if !ok {
		return false
	}

	return userLevel >= minLevel
}
