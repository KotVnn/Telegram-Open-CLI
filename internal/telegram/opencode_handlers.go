package telegram

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

const permissionWaitTimeout = 2 * time.Minute

type pendingPermission struct {
	ch     chan backend.PermissionDecision
	chatID int64
}

// PermissionHandler returns a backend.PermissionHandler that routes permission
// requests to the user over Telegram via an inline keyboard.
func (m *SessionManager) PermissionHandler(adapter Adapter, chatID int64, userID int64) backend.PermissionHandler {
	return func(ctx context.Context, req backend.PermissionRequest) backend.PermissionDecision {
		return m.askPermission(ctx, adapter, req, chatID, userID)
	}
}

func (m *SessionManager) askPermission(ctx context.Context, adapter Adapter, req backend.PermissionRequest, chatID int64, userID int64) backend.PermissionDecision {
	key := req.PermissionID
	if key == "" {
		key = "perm-" + fmt.Sprintf("%d", time.Now().UnixNano())
	}

	ch := make(chan backend.PermissionDecision, 1)
	m.mu.Lock()
	m.permRequests[key] = &pendingPermission{ch: ch, chatID: chatID}
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		delete(m.permRequests, key)
		m.mu.Unlock()
	}()

	text := fmt.Sprintf("🔐 Permission required\n\nAction: %s\nPattern: %s",
		req.Permission, strings.Join(req.Patterns, ", "))

	kb := NewKeyboard().
		AddButton("Allow once", "perm:"+key+":allow").
		AddButton("Always", "perm:"+key+":always").
		AddButton("Deny", "perm:"+key+":deny").
		Build()

	if err := adapter.SendMessage(ctx, chatID, OutgoingMessage{
		Text:        text,
		ReplyMarkup: kb.ToTelegram(),
	}); err != nil {
		return backend.PermissionDecision{Response: "deny"}
	}

	timer := time.NewTimer(permissionWaitTimeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return backend.PermissionDecision{Response: "deny"}
	case decision := <-ch:
		return decision
	case <-timer.C:
		return backend.PermissionDecision{Response: "deny"}
	}
}

// HandlePermissionCallback handles permission decision callbacks.
func HandlePermissionCallback(adapter Adapter, sm *SessionManager) CallbackHandlerFunc {
	return func(ctx context.Context, cb *CallbackQuery) error {
		payload := strings.TrimPrefix(cb.Data, "perm:")
		parts := strings.Split(payload, ":")
		if len(parts) < 2 {
			return adapter.AnswerCallback(ctx, cb.ID, "Invalid request")
		}
		key, action := parts[0], parts[1]

		sm.mu.Lock()
		pending, ok := sm.permRequests[key]
		sm.mu.Unlock()

		if !ok {
			return adapter.AnswerCallback(ctx, cb.ID, "Permission request expired")
		}

		var decision backend.PermissionDecision
		switch action {
		case "allow":
			decision = backend.PermissionDecision{Response: "allow"}
		case "always":
			decision = backend.PermissionDecision{Response: "allow", Remember: true}
		case "deny":
			decision = backend.PermissionDecision{Response: "deny"}
		default:
			return adapter.AnswerCallback(ctx, cb.ID, "Unknown action")
		}

		select {
		case pending.ch <- decision:
		default:
		}

		label := "✅ Allowed"
		if action == "always" {
			label = "✅ Allowed (always)"
		} else if action == "deny" {
			label = "⛔ Denied"
		}

		_ = adapter.EditMessage(ctx, cb.ChatID, cb.MessageID, label)
		return adapter.AnswerCallback(ctx, cb.ID, "Decision sent")
	}
}

// HandleAbort handles the /abort command.
func HandleAbort(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		sessionID := sm.GetActiveSession(msg.FromID)
		if sessionID == "" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active session.",
			})
		}

		aborter, ok := sm.backend.(backend.Aborter)
		if !ok {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Backend does not support abort.",
			})
		}

		if err := aborter.Abort(ctx, sessionID); err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Abort failed: %v", err),
			})
		}

		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text: "Aborted current task.",
		})
	}
}

// HandleMessages handles the /messages command.
func HandleMessages(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		sessionID := sm.GetActiveSession(msg.FromID)
		if sessionID == "" {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No active session.",
			})
		}

		lister, ok := sm.backend.(backend.MessageLister)
		if !ok {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Backend does not support message listing.",
			})
		}

		messages, err := lister.ListMessages(ctx, sessionID, 10)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to list messages: %v", err),
			})
		}

		if len(messages) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "No messages yet.",
			})
		}

		var sb strings.Builder
		for _, m := range messages {
			role := m.Role
			content := m.Content
			if len(content) > 200 {
				content = content[:200] + "…"
			}
			sb.WriteString(fmt.Sprintf("<%s> %s\n\n", role, content))
		}

		for _, part := range splitMessage(sb.String(), maxTelegramMessageLength) {
			if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{Text: part}); err != nil {
				return err
			}
		}
		return nil
	}
}

// HandleLS handles the /ls command.
func HandleLS(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		dir := ""
		if len(msg.Args) > 0 {
			dir = msg.Args[0]
		} else {
			sessionID := sm.GetActiveSession(msg.FromID)
			if sessionID == "" {
				return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
					Text: "No active session.",
				})
			}
			session, err := sm.storage.GetSession(ctx, sessionID)
			if err == nil && session.WorkingDir != "" {
				dir = session.WorkingDir
			}
		}

		browser, ok := sm.backend.(backend.FileBrowser)
		if !ok {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Backend does not support file browsing.",
			})
		}

		entries, err := browser.ListFiles(ctx, dir)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to list files: %v", err),
			})
		}

		if len(entries) == 0 {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: "Empty directory.",
			})
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("📁 %s\n", dir))
		for _, e := range entries {
			marker := "📄"
			if e.IsDir {
				marker = "📁"
			}
			size := ""
			if !e.IsDir {
				size = fmt.Sprintf(" (%d B)", e.Size)
			}
			sb.WriteString(fmt.Sprintf("%s %s%s\n", marker, e.Name, size))
		}

		for _, part := range splitMessage(sb.String(), maxTelegramMessageLength) {
			if err := adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{Text: part}); err != nil {
				return err
			}
		}
		return nil
	}
}

// HandleModels handles the /models command by showing the model picker.
func HandleModels(adapter Adapter, sm *SessionManager) HandlerFunc {
	return func(ctx context.Context, msg *IncomingMessage) error {
		models, _, err := listModelsAgents(ctx, sm.backend)
		if err != nil {
			return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
				Text: fmt.Sprintf("Failed to list models: %v", err),
			})
		}
		return adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
			Text:        "Select model:",
			ReplyMarkup: NewModelKeyboard(models).ToTelegram(),
		})
	}
}
