package telegram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// HandleDocument handles document messages (file uploads).
func HandleDocument(adapter Adapter, sm *SessionManager, token string) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		if msg.Document == nil {
			return nil
		}

		userID := msg.FromID
		sm.mu.RLock()
		sessionID, ok := sm.activeSessions[userID]
		sm.mu.RUnlock()

		if !ok {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active session. Use /new to create one or /sessions to switch.",
			})
		}

		if err := sm.storage.SaveMessage(ctx, &storage.Message{
			ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
			SessionID: sessionID,
			Role:      "user",
			Content:   fmt.Sprintf("[File: %s]", msg.Document.FileName),
			CreatedAt: time.Now(),
		}); err != nil {
			sm.logger.Error().Err(err).Int64("user_id", userID).Msg("save document message")
		}

		content, err := downloadFile(ctx, token, msg.Document.FileID)
		if err != nil {
			sm.logger.Error().Err(err).Str("file_id", msg.Document.FileID).Msg("download file")
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Failed to download file. Please try again.",
			})
		}

		processingMsg, err := adapter.SendMessageWithResult(ctx, msg.ChatID, OutgoingMessage{
			Text: "Processing file: " + msg.Document.FileName + "...",
		})
		if err != nil {
			return err
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
				sm.logger.Error().Err(chunk.Error).Int64("user_id", userID).Msg("stream error")
				return adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID,
					"Backend error. Please try again.")
			}
			response.WriteString(chunk.Content)
		}

		result := response.String()
		if result == "" {
			result = "File processed successfully."
		}

		parts := splitMessage(result, 4000)
		if err := adapter.EditMessage(ctx, msg.ChatID, processingMsg.MessageID, parts[0]); err != nil {
			sm.logger.Error().Err(err).Int64("user_id", userID).Msg("edit final message")
		}

		for _, part := range parts[1:] {
			if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: part,
			}); err != nil {
				sm.logger.Error().Err(err).Int64("user_id", userID).Msg("send overflow message")
			}
		}

		assistantMsg := &storage.Message{
			ID:        fmt.Sprintf("msg-%s", uuid.New().String()),
			SessionID: sessionID,
			Role:      "assistant",
			Content:   result,
			CreatedAt: time.Now(),
		}
		if err := sm.storage.SaveMessage(ctx, assistantMsg); err != nil {
			sm.logger.Error().Err(err).Int64("user_id", userID).Msg("save assistant message")
		}

		return nil
	}
}

func downloadFile(ctx context.Context, token, fileID string) ([]byte, error) {
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", token, fileID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	const maxFileSize = 50 * 1024 * 1024 // 50MB
	limitedReader := io.LimitReader(resp.Body, maxFileSize+1)
	content, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if int64(len(content)) > maxFileSize {
		return nil, fmt.Errorf("file too large: %d bytes (max %d)", len(content), maxFileSize)
	}

	return content, nil
}
