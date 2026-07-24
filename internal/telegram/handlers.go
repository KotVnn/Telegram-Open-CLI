package telegram

import (
	"context"
	"fmt"

	"github.com/KotVnn/Telegram-Open-CLI/pkg/version"
)

// HandleStart handles the /start command.
func HandleStart(adapter Adapter) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		welcome := fmt.Sprintf("Welcome to TOC!\n\nVersion: %s\n\nUse /help to see available commands.", version.Version)
		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: welcome,
		})
	}
}

// HandleHelp handles the /help command.
func HandleHelp(adapter Adapter) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		help := `Available commands:

/start - Start the bot
/help - Show this help message
/version - Show version info
/me - Show your user info

Session commands:
/new [name] - Create a new session
/sessions - List all sessions
/switch <id> - Switch to a session
/close [id] - Close current or specified session
/status - Show current session status

Project commands:
/project - Show project help
/project new <name> [path] - Create a project
/project list - List projects
/project switch <id> - Switch project
/project delete <id> - Delete project

Admin commands:
/users - List all users (admin)
/ban <user_id> - Ban a user (admin)
/unban <user_id> - Unban a user (admin)
/role <user_id> <role> - Set user role (admin)`

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: help,
		})
	}
}

// HandleVersion handles the /version command.
func HandleVersion(adapter Adapter) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: version.Info(),
		})
	}
}
