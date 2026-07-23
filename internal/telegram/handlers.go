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

Session commands:
/new [name] - Create a new session
/sessions - List all sessions
/switch <id> - Switch to a session
/close [id] - Close current or specified session
/status - Show current session status`

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
