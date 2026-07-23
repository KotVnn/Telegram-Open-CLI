# Telegram Adapter Guide

This guide explains how to implement and use the Telegram adapter in TOC.

## Table of Contents

- [Overview](#overview)
- [Adapter Interface](#adapter-interface)
- [Initialization](#initialization)
- [Message Handling](#message-handling)
- [Keyboard Builders](#keyboard-builders)
- [Streaming Responses](#streaming-responses)
- [File Handling](#file-handling)
- [Error Handling](#error-handling)
- [Webhook Mode](#webhook-mode)

---

## Overview

The Telegram adapter wraps the `go-telegram/bot` library and provides a clean interface for sending and receiving messages.

```
┌─────────────────────────────────────────────────────────────┐
│                   Telegram Adapter                          │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │   Bot API   │    │   Handlers  │    │  Middleware  │     │
│  │   Client    │    │             │    │             │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
│                                                             │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────┐     │
│  │  Keyboards  │    │   Streaming │    │   File      │     │
│  │  Builder    │    │   Manager   │    │   Manager   │     │
│  └─────────────┘    └─────────────┘    └─────────────┘     │
└─────────────────────────────────────────────────────────────┘
```

---

## Adapter Interface

```go
type Adapter interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // Message Operations
    SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error
    EditMessage(ctx context.Context, chatID int64, messageID int, text string) error
    SendDocument(ctx context.Context, chatID int64, doc Document) error
    AnswerCallback(ctx context.Context, callbackID string, text string) error

    // Handler Registration
    HandleCommand(cmd string, handler HandlerFunc)
    HandleMessage(pattern string, handler HandlerFunc)
    HandleCallback(pattern string, handler CallbackHandlerFunc)
    HandleDefault(handler HandlerFunc)

    // Middleware
    Use(middlewares ...Middleware)
}
```

See [interfaces.md](interfaces.md) for complete interface documentation.

---

## Initialization

### Basic Setup

```go
package main

import (
    "context"
    "log"

    "github.com/KotVnn/Telegram-Open-CLI/internal/telegram"
    "github.com/KotVnn/Telegram-Open-CLI/internal/config"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Create adapter
    adapter, err := telegram.New(telegram.Config{
        Token: cfg.Telegram.Token,
    })
    if err != nil {
        log.Fatal(err)
    }

    // Register handlers
    adapter.HandleCommand("start", handleStart)
    adapter.HandleCommand("help", handleHelp)
    adapter.HandleCommand("new", handleNewSession)

    // Add middleware
    adapter.Use(
        telegram.RecoveryMiddleware(),
        telegram.LoggingMiddleware(logger),
        telegram.RateLimitMiddleware(10, 20),
    )

    // Start bot
    ctx := context.Background()
    if err := adapter.Start(ctx); err != nil {
        log.Fatal(err)
    }
}
```

### Configuration

```go
type Config struct {
    // Token is the Telegram bot token.
    Token string

    // Mode is "polling" or "webhook".
    Mode string

    // WebhookURL is the webhook URL (required for webhook mode).
    WebhookURL string

    // WebhookSecret is the webhook secret token.
    WebhookSecret string

    // AllowedUsers is a list of allowed user IDs (empty = allow all).
    AllowedUsers []int64

    // AllowedGroups is a list of allowed group IDs (empty = allow all).
    AllowedGroups []int64
}
```

---

## Message Handling

### Command Handlers

Commands are messages starting with `/`. The adapter parses them automatically.

```go
// Simple command handler
adapter.HandleCommand("start", func(ctx context.Context, msg *telegram.IncomingMessage) error {
    welcome := fmt.Sprintf("Welcome, %s!", msg.FromFirstName)
    return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
        Text: welcome,
    })
})

// Command with arguments
adapter.HandleCommand("new", func(ctx context.Context, msg *telegram.IncomingMessage) error {
    if len(msg.Args) == 0 {
        return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
            Text: "Usage: /new <session-name>",
        })
    }

    name := msg.Args[0]
    // Create session...
    return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
        Text: fmt.Sprintf("Created session: %s", name),
    })
})
```

### Message Handlers

Handle non-command messages with pattern matching:

```go
// Handle all messages
adapter.HandleDefault(func(ctx context.Context, msg *telegram.IncomingMessage) error {
    // Forward to current session
    return nil
})

// Handle messages matching a pattern
adapter.HandleMessage("fix.*bug", func(ctx context.Context, msg *telegram.IncomingMessage) error {
    // Handle bug fix request
    return nil
})
```

### Callback Handlers

Handle inline button callbacks:

```go
// Handle callbacks with prefix
adapter.HandleCallback("session:", func(ctx context.Context, cb *telegram.CallbackQuery) error {
    // Parse session ID from callback data
    sessionID := strings.TrimPrefix(cb.Data, "session:")

    // Process callback
    return adapter.AnswerCallback(ctx, cb.ID, fmt.Sprintf("Switched to session %s", sessionID))
})
```

---

## Keyboard Builders

### Inline Keyboards

```go
// Create inline keyboard for session list
func buildSessionKeyboard(sessions []*Session) telegram.InlineKeyboard {
    var rows [][]telegram.InlineKeyboardButton

    for _, session := range sessions {
        rows = append(rows, []telegram.InlineKeyboardButton{
            {
                Text:         session.Title,
                CallbackData: fmt.Sprintf("session:%s", session.ID),
            },
        })
    }

    // Add "New Session" button
    rows = append(rows, []telegram.InlineKeyboardButton{
        {
            Text:         "+ New Session",
            CallbackData: "new_session",
        },
    })

    return telegram.InlineKeyboard{
        InlineKeyboard: rows,
    }
}

// Send message with inline keyboard
adapter.SendMessage(ctx, chatID, telegram.OutgoingMessage{
    Text:       "Select a session:",
    ReplyMarkup: buildSessionKeyboard(sessions),
})
```

### Reply Keyboards

```go
// Create reply keyboard
func buildReplyKeyboard() telegram.ReplyKeyboard {
    return telegram.ReplyKeyboard{
        Keyboard: [][]telegram.KeyboardButton{
            {
                {Text: "/sessions"},
                {Text: "/new"},
            },
            {
                {Text: "/status"},
                {Text: "/help"},
            },
        },
        ResizeKeyboard:  true,
        OneTimeKeyboard: false,
    }
}

// Send message with reply keyboard
adapter.SendMessage(ctx, chatID, telegram.OutgoingMessage{
    Text:       "Available commands:",
    ReplyMarkup: buildReplyKeyboard(),
})
```

### Keyboard Builder Helper

```go
// KeyboardBuilder provides a fluent API for building keyboards
type KeyboardBuilder struct {
    rows [][]telegram.InlineKeyboardButton
}

func NewKeyboard() *KeyboardBuilder {
    return &KeyboardBuilder{}
}

func (b *KeyboardBuilder) AddButton(text, data string) *KeyboardBuilder {
    if len(b.rows) == 0 {
        b.rows = append(b.rows, []telegram.InlineKeyboardButton{})
    }
    b.rows[len(b.rows)-1] = append(b.rows[len(b.rows)-1], telegram.InlineKeyboardButton{
        Text:         text,
        CallbackData: data,
    })
    return b
}

func (b *KeyboardBuilder) NewRow() *KeyboardBuilder {
    b.rows = append(b.rows, []telegram.InlineKeyboardButton{})
    return b
}

func (b *KeyboardBuilder) Build() telegram.InlineKeyboard {
    return telegram.InlineKeyboard{InlineKeyboard: b.rows}
}

// Usage
keyboard := NewKeyboard().
    AddButton("Session 1", "session:1").
    AddButton("Session 2", "session:2").
    NewRow().
    AddButton("+ New", "new_session").
    Build()
```

---

## Streaming Responses

Stream AI responses by editing the message progressively.

### Basic Streaming

```go
func streamResponse(adapter telegram.Adapter, chatID int64, stream <-chan backend.StreamChunk) error {
    // Send initial message
    msgID, err := adapter.SendMessage(ctx, chatID, telegram.OutgoingMessage{
        Text: "Processing...",
    })
    if err != nil {
        return err
    }

    var content strings.Builder
    lastUpdate := time.Now()
    updateInterval := 500 * time.Millisecond

    for chunk := range stream {
        if chunk.Error != nil {
            return chunk.Error
        }

        content.WriteString(chunk.Content)

        // Rate limit edits to Telegram (max 30/sec)
        if time.Since(lastUpdate) >= updateInterval || chunk.Done {
            err := adapter.EditMessage(ctx, chatID, msgID, content.String())
            if err != nil {
                // Log error but continue
                log.Printf("failed to edit message: %v", err)
            }
            lastUpdate = time.Now()
        }
    }

    return nil
}
```

### Advanced Streaming with Progress

```go
func streamWithProgress(adapter telegram.Adapter, chatID int64, stream <-chan backend.StreamChunk) error {
    msgID, err := adapter.SendMessage(ctx, chatID, telegram.OutgoingMessage{
        Text: "🔄 Starting...",
    })
    if err != nil {
        return err
    }

    var content strings.Builder
    var chunkCount int
    lastUpdate := time.Now()

    for chunk := range stream {
        if chunk.Error != nil {
            // Send error message
            adapter.EditMessage(ctx, chatID, msgID, fmt.Sprintf("❌ Error: %v", chunk.Error))
            return chunk.Error
        }

        content.WriteString(chunk.Content)
        chunkCount++

        // Update display
        if time.Since(lastUpdate) >= 500*time.Millisecond || chunk.Done {
            display := content.String()
            if !chunk.Done {
                display += "\n\n⏳ Processing..."
            }

            adapter.EditMessage(ctx, chatID, msgID, display)
            lastUpdate = time.Now()
        }
    }

    return nil
}
```

---

## File Handling

### Receiving Files

```go
adapter.HandleDefault(func(ctx context.Context, msg *telegram.IncomingMessage) error {
    // Check for attached document
    if msg.Document != nil {
        return handleDocument(adapter, ctx, msg)
    }

    // Handle text message
    return handleTextMessage(adapter, ctx, msg)
})

func handleDocument(adapter telegram.Adapter, ctx context.Context, msg *telegram.IncomingMessage) error {
    doc := msg.Document

    // Validate file size (max 20MB)
    if doc.FileSize > 20*1024*1024 {
        return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
            Text: "❌ File too large. Maximum size is 20MB.",
        })
    }

    // Note: File downloads require direct bot API access
    // The adapter wraps go-telegram/bot which provides bot.GetFile()
    // Implementation depends on the specific backend requirements

    return nil
}
```

### Sending Files

```go
// Send file to user
func sendFile(adapter telegram.Adapter, ctx context.Context, chatID int64, filePath string) error {
    content, err := os.ReadFile(filePath)
    if err != nil {
        return err
    }

    return adapter.SendDocument(ctx, chatID, telegram.Document{
        FileName: filepath.Base(filePath),
        Content:  content,
        Caption:  fmt.Sprintf("📄 %s", filepath.Base(filePath)),
    })
}
```

---

## Error Handling

### User-Friendly Error Messages

```go
func handleError(adapter telegram.Adapter, ctx context.Context, chatID int64, err error) error {
    userMsg := formatUserError(err)
    return adapter.SendMessage(ctx, chatID, telegram.OutgoingMessage{
        Text:      userMsg,
        ParseMode: "HTML",
    })
}

func formatUserError(err error) string {
    switch {
    case errors.Is(err, backend.ErrSessionNotFound):
        return "⚠️ Session not found. Use /sessions to see available sessions."
    case errors.Is(err, backend.ErrUnauthorized):
        return "🚫 You are not authorized to use this bot."
    case errors.Is(err, backend.ErrBackendUnavailable):
        return "⚠️ AI backend is temporarily unavailable. Please try again later."
    case errors.Is(err, backend.ErrRateLimited):
        return "⏳ Too many requests. Please wait a moment."
    default:
        return "❌ An error occurred. Please try again."
    }
}
```

### Panic Recovery

```go
func RecoveryMiddleware() telegram.Middleware {
    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) (err error) {
            defer func() {
                if r := recover(); r != nil {
                    log.Printf("panic recovered: %v\n%s", r, debug.Stack())
                    err = fmt.Errorf("internal error")
                }
            }()
            return next(ctx, msg)
        }
    }
}
```

---

## Webhook Mode

### Setup

```go
adapter, err := telegram.New(telegram.Config{
    Token:        cfg.Telegram.Token,
    Mode:         "webhook",
    WebhookURL:   "https://yourdomain.com/webhook",
    WebhookSecret: "your-secret-token",
})
```

### With Custom HTTP Server

```go
mux := http.NewServeMux()

// Health check endpoint
mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(http.StatusOK)
})

// Webhook endpoint - use the bot's built-in handler
mux.HandleFunc("/webhook", bot.WebhookHandler())

// Start server
server := &http.Server{
    Addr:    ":8080",
    Handler: mux,
}

go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatal(err)
    }
}()
```

### Nginx Configuration

```nginx
server {
    listen 443 ssl;
    server_name yourdomain.com;

    ssl_certificate /etc/letsencrypt/live/yourdomain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/yourdomain.com/privkey.pem;

    location /webhook {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /health {
        proxy_pass http://127.0.0.1:8080;
    }
}
```

---

## Rate Limiting

### Built-in Rate Limiter

```go
func RateLimitMiddleware(requestsPerSecond float64, burst int) telegram.Middleware {
    limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            if !limiter.Allow() {
                return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
                    Text: "⏳ Too many requests. Please slow down.",
                })
            }
            return next(ctx, msg)
        }
    }
}
```

### Per-User Rate Limiter

```go
func PerUserRateLimitMiddleware(requestsPerMinute int) telegram.Middleware {
    limiters := make(map[int64]*rate.Limiter)
    var mu sync.Mutex

    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            mu.Lock()
            limiter, ok := limiters[msg.FromID]
            if !ok {
                limiter = rate.NewLimiter(
                    rate.Limit(requestsPerMinute)/60,
                    requestsPerMinute,
                )
                limiters[msg.FromID] = limiter
            }
            mu.Unlock()

            if !limiter.Allow() {
                return adapter.SendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
                    Text: "⏳ Rate limit exceeded. Please wait.",
                })
            }
            return next(ctx, msg)
        }
    }
}
```
