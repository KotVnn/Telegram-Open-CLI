package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

const maxTelegramMessageLength = 4096
const maxFileSizeWarning = 10 * 1024 * 1024 // 10MB

func splitMessage(text string, limit int) []string {
	runes := []rune(text)
	if len(runes) <= limit {
		return []string{text}
	}

	var parts []string
	for len(runes) > 0 {
		if len(runes) <= limit {
			parts = append(parts, string(runes))
			break
		}

		splitIdx := -1
		for i := limit - 1; i >= 0; i-- {
			if runes[i] == '\n' {
				splitIdx = i
				break
			}
		}
		if splitIdx < 0 {
			splitIdx = limit
		}

		parts = append(parts, string(runes[:splitIdx]))
		runes = runes[splitIdx:]
		for len(runes) > 0 && runes[0] == '\n' {
			runes = runes[1:]
		}
	}
	return parts
}

// HandleMessage handles default text messages by forwarding to the active session's backend.
func HandleMessage(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if msg.Document != nil {
			return handleDocumentMessage(ctx, adapter, sm, msg)
		}

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
				sm.logger.Error().
					Err(chunk.Error).
					Int64("user_id", msg.FromID).
					Str("session_id", sessionID).
					Str("text", msg.Text).
					Msg("stream error")
				return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
					fmt.Sprintf("Backend error: %v", chunk.Error))
			}

			responseBuilder.WriteString(chunk.Content)

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
			sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("edit final message")
		}

		for _, part := range parts[1:] {
			if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: part,
			}); err != nil {
				sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("send overflow message")
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
			sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("save assistant message")
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to save response. Please try again.",
			})
		}

		return nil
	}
}

func handleDocumentMessage(ctx context.Context, adapter Adapter, sm *SessionManager, msg *IncomingMessage) error {
	sessionID := sm.GetActiveSession(msg.FromID)
	if sessionID == "" {
		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: "No active session. Use /new to create one or /sessions to switch.",
		})
	}

	if msg.Document.FileSize > maxFileSizeWarning {
		_ = adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: fmt.Sprintf("Warning: File size is %dMB. Large files may take longer to process or fail.", msg.Document.FileSize/(1024*1024)),
		})
	}

	if err := sm.storage.SaveMessage(ctx, &storage.Message{
		ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
		SessionID: sessionID,
		Role:      storage.MessageRoleUser,
		Content:   fmt.Sprintf("[File: %s]", msg.Document.FileName),
		CreatedAt: time.Now(),
	}); err != nil {
		sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("save document message")
	}

	processingMsg, err := adapter.SendMessageWithResult(ctx, msg.ChatID, OutgoingMessage{
		Text: "Processing file: " + msg.Document.FileName + "...",
	})
	if err != nil {
		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: "Failed to send message. Please try again.",
		})
	}

	content, err := downloadFile(ctx, sm.botToken, msg.Document.FileID)
	if err != nil {
		sm.logger.Error().Err(err).Str("file_id", msg.Document.FileID).Msg("download file")
		return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
			"Failed to download file. Please try again.")
	}

	req := &backend.SendMessageRequest{
		SessionID: sessionID,
		Content:   fmt.Sprintf("Analyze this file: %s", msg.Document.FileName),
		Files: []backend.FileAttachment{
			{
				Name:     msg.Document.FileName,
				Content:  content,
				MIMEType: msg.Document.MimeType,
			},
		},
	}

	streamCh, err := sm.backend.StreamMessage(ctx, req)
	if err != nil {
		return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
			"Backend error. Please try again.")
	}

	var response strings.Builder
	for chunk := range streamCh {
		if chunk.Error != nil {
			sm.logger.Error().
				Err(chunk.Error).
				Int64("user_id", msg.FromID).
				Str("session_id", sessionID).
				Str("file", msg.Document.FileName).
				Msg("stream error")
			return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
				fmt.Sprintf("Backend error: %v", chunk.Error))
		}
		response.WriteString(chunk.Content)
	}

	result := response.String()
	if result == "" {
		result = "File processed successfully."
	}

	parts := splitMessage(result, 4000)
	if err := adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID, parts[0]); err != nil {
		sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("edit final message")
	}

	for _, part := range parts[1:] {
		if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: part,
		}); err != nil {
			sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("send overflow message")
		}
	}

	assistantMsg := &storage.Message{
		ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
		SessionID: sessionID,
		Role:      storage.MessageRoleAssistant,
		Content:   result,
		CreatedAt: time.Now(),
	}
	if err := sm.storage.SaveMessage(ctx, assistantMsg); err != nil {
		sm.logger.Error().Err(err).Int64("user_id", msg.FromID).Msg("save assistant message")
	}

	return nil
}
