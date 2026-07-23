package telegram

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

const maxTelegramMessageLength = 4096

func splitMessage(text string, limit int) []string {
	if len(text) <= limit {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= limit {
			parts = append(parts, text)
			break
		}

		splitIdx := strings.LastIndex(text[:limit], "\n")
		if splitIdx <= 0 {
			splitIdx = limit
		}

		parts = append(parts, text[:splitIdx])
		text = strings.TrimLeft(text[splitIdx:], "\n")
	}
	return parts
}

// HandleMessage handles default text messages by forwarding to the active session's backend.
func HandleMessage(adapter Adapter, sm *SessionManager) HandlerFunc {
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

		if session.Status == storage.SessionStatusClosed {
			sm.ClearActiveSession(msg.FromID)
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Session is closed. Use /new to create a new one.",
			})
		}

		userMsg := &storage.Message{
			ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
			SessionID: sessionID,
			Role:      storage.MessageRoleUser,
			Content:   msg.Text,
			CreatedAt: time.Now(),
		}
		if err := sm.storage.SaveMessage(ctx, userMsg); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to save message. Please try again.",
			})
		}

		processingMsg, err := adapter.SendMessageWithResult(ctx, msg.ChatID, OutgoingMessage{
			Text: "Processing...",
		})
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to send message. Please try again.",
			})
		}

		stream, err := sm.backend.StreamMessage(ctx, &backend.SendMessageRequest{
			SessionID: sessionID,
			Content:   msg.Text,
		})
		if err != nil {
			return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
				"Backend error. Please try again.")
		}

		var responseBuilder strings.Builder
		lastEdit := time.Now()
		editCooldown := 500 * time.Millisecond

		for chunk := range stream {
			if chunk.Error != nil {
				fmt.Fprintf(os.Stderr, "stream error: %v\n", chunk.Error)
				return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
					"Backend error. Please try again.")
			}

			if responseBuilder.Len() < maxTelegramMessageLength {
				responseBuilder.WriteString(chunk.Content)
			}

			if time.Since(lastEdit) >= editCooldown {
				editText := responseBuilder.String()
				if len(editText) > maxTelegramMessageLength {
					editText = editText[:maxTelegramMessageLength]
				}
				if err := adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
					editText); err != nil {
					continue
				}
				lastEdit = time.Now()
			}
		}

		finalResponse := responseBuilder.String()
		if finalResponse == "" {
			finalResponse = "No response from backend."
		}

		parts := splitMessage(finalResponse, maxTelegramMessageLength)

		if err := adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID, parts[0]); err != nil {
			fmt.Fprintf(os.Stderr, "edit final message: %v\n", err)
		}

		for _, part := range parts[1:] {
			if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: part,
			}); err != nil {
				fmt.Fprintf(os.Stderr, "send overflow message: %v\n", err)
			}
		}

		assistantMsg := &storage.Message{
			ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
			SessionID: sessionID,
			Role:      storage.MessageRoleAssistant,
			Content:   finalResponse,
			CreatedAt: time.Now(),
		}
		if err := sm.storage.SaveMessage(ctx, assistantMsg); err != nil {
			fmt.Fprintf(os.Stderr, "save assistant message: %v\n", err)
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to save response. Please try again.",
			})
		}

		return nil
	}
}
