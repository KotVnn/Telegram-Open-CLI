package telegram

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
	"github.com/KotVnn/Telegram-Open-CLI/internal/user"
)

// HandleUsers handles the /users command (admin only).
func HandleUsers(adapter Adapter, um *user.Manager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if !um.IsAdmin(ctx, msg.FromID) {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Admin access required.",
			})
		}

		users, err := um.List(ctx)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to list users.",
			})
		}

		if len(users) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No users found.",
			})
		}

		text := "Users:\n\n"
		for _, u := range users {
			status := "active"
			if !u.IsActive {
				status = "inactive"
			}
			text += fmt.Sprintf("ID: %d\nUsername: @%s\nName: %s %s\nRole: %s\nStatus: %s\n\n",
				u.ID, u.Username, u.FirstName, u.LastName, u.Role, status)
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: text,
		})
	}
}

// HandleBan handles the /ban <user_id> command (admin only).
func HandleBan(adapter Adapter, um *user.Manager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if !um.IsAdmin(ctx, msg.FromID) {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Admin access required.",
			})
		}

		if len(msg.Args) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /ban <user_id>",
			})
		}

		targetID, err := strconv.ParseInt(msg.Args[0], 10, 64)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Invalid user ID.",
			})
		}

		if targetID == msg.FromID {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "You cannot ban yourself.",
			})
		}

		if um.IsAdmin(ctx, targetID) {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Cannot ban an admin user.",
			})
		}

		if err := um.SetActive(ctx, targetID, false); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to ban user: %v", err),
			})
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("User %d has been banned.", targetID),
		})
	}
}

// HandleUnban handles the /unban <user_id> command (admin only).
func HandleUnban(adapter Adapter, um *user.Manager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if !um.IsAdmin(ctx, msg.FromID) {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Admin access required.",
			})
		}

		if len(msg.Args) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /unban <user_id>",
			})
		}

		targetID, err := strconv.ParseInt(msg.Args[0], 10, 64)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Invalid user ID.",
			})
		}

		if err := um.SetActive(ctx, targetID, true); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to unban user: %v", err),
			})
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("User %d has been unbanned.", targetID),
		})
	}
}

// HandleRole handles the /role <user_id> <role> command (admin only).
func HandleRole(adapter Adapter, um *user.Manager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if !um.IsAdmin(ctx, msg.FromID) {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Admin access required.",
			})
		}

		if len(msg.Args) < 2 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Usage: /role <user_id> <admin|user|viewer>",
			})
		}

		targetID, err := strconv.ParseInt(msg.Args[0], 10, 64)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Invalid user ID.",
			})
		}

		if targetID == msg.FromID {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "You cannot change your own role.",
			})
		}

		role := strings.ToLower(msg.Args[1])
		switch role {
		case "admin", "user", "viewer":
		default:
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Invalid role. Use: admin, user, or viewer.",
			})
		}

		if err := um.SetRole(ctx, targetID, storage.UserRole(role)); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to set role: %v", err),
			})
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("User %d role set to %s.", targetID, role),
		})
	}
}

// HandleMe handles the /me command - shows current user info.
func HandleMe(adapter Adapter, um *user.Manager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		u, err := um.GetOrCreate(ctx, msg.FromID, msg.FromUsername, msg.FromFirstName, msg.FromLastName)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to get user info.",
			})
		}

		text := fmt.Sprintf("Your Info:\n\nID: %d\nUsername: @%s\nName: %s %s\nRole: %s\nActive: %v",
			u.ID, u.Username, u.FirstName, u.LastName, u.Role, u.IsActive)

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: text,
		})
	}
}
