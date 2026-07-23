# Middleware Guide

This guide explains how to use and create middleware in TOC.

## Table of Contents

- [Overview](#overview)
- [Middleware Chain](#middleware-chain)
- [Built-in Middleware](#built-in-middleware)
- [Creating Custom Middleware](#creating-custom-middleware)
- [Middleware Examples](#middleware-examples)
- [Testing Middleware](#testing-middleware)

---

## Overview

Middleware provides a way to process messages before and after handlers. Cross-cutting concerns like authentication, logging, and rate limiting are implemented as middleware.

```
┌─────────────────────────────────────────────────────────────┐
│                    Middleware Chain                          │
│                                                             │
│  Incoming Message                                           │
│       │                                                     │
│       ▼                                                     │
│  ┌─────────┐   ┌─────────┐   ┌─────────┐   ┌─────────┐    │
│  │ Recovery │ → │ Logging │ → │   Auth  │ → │ Handler │    │
│  └─────────┘   └─────────┘   └─────────┘   └─────────┘    │
│       │             │             │             │           │
│       ◀─────────────┴─────────────┴─────────────┘           │
│                    Response flows back                      │
└─────────────────────────────────────────────────────────────┘
```

---

## Middleware Chain

### Definition

```go
// Middleware is a function that wraps a handler.
type Middleware func(next HandlerFunc) HandlerFunc

// HandlerFunc handles incoming messages.
type HandlerFunc func(ctx context.Context, msg *IncomingMessage) error
```

### Execution Order

Middleware executes in the order it's added:

```go
adapter.Use(
    RecoveryMiddleware(),    // 1. First
    LoggingMiddleware(),     // 2. Second
    AuthMiddleware(),        // 3. Third
    RateLimitMiddleware(),   // 4. Fourth
)
```

Execution flow:
1. RecoveryMiddleware receives message
2. Calls LoggingMiddleware
3. Calls AuthMiddleware
4. Calls RateLimitMiddleware
5. Calls actual Handler
6. Response flows back through chain

### Context Propagation

Middleware can add values to context:

```go
func AuthMiddleware() Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            // Add user to context
            ctx = context.WithValue(ctx, userKey, user)
            return next(ctx, msg)
        }
    }
}

// Later in handler
func handleCommand(ctx context.Context, msg *IncomingMessage) error {
    user := ctx.Value(userKey).(*User)
    // Use user...
}
```

---

## Built-in Middleware

### Recovery Middleware

Catches panics and prevents bot crashes:

```go
// RecoveryMiddleware catches panics and logs them.
func RecoveryMiddleware() Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) (err error) {
            defer func() {
                if r := recover(); r != nil {
                    // Log panic with stack trace
                    log.Error().
                        Interface("panic", r).
                        Str("stack", string(debug.Stack())).
                        Msg("panic recovered in handler")

                    // Send error message to user
                    if msg != nil {
                        _ = sendMessage(ctx, msg.ChatID, OutgoingMessage{
                            Text: "❌ An internal error occurred. Please try again.",
                        })
                    }

                    err = fmt.Errorf("panic recovered: %v", r)
                }
            }()
            return next(ctx, msg)
        }
    }
}
```

### Logging Middleware

Logs all messages and their processing time:

```go
// LoggingMiddleware logs message details and processing time.
func LoggingMiddleware(logger *zerolog.Logger) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            start := time.Now()

            // Log incoming message
            logger.Info().
                Int64("user_id", msg.FromID).
                Str("username", msg.FromUsername).
                Str("text", msg.Text).
                Int64("chat_id", msg.ChatID).
                Msg("message received")

            // Process message
            err := next(ctx, msg)

            // Log result
            logger.Info().
                Int64("user_id", msg.FromID).
                Str("text", msg.Text).
                Dur("duration", time.Since(start)).
                Err(err).
                Msg("message processed")

            return err
        }
    }
}
```

### Authentication Middleware

Checks if users are authorized:

```go
// AuthMiddleware verifies user authorization.
func AuthMiddleware(allowedUsers []int64, allowedGroups []int64) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            // Check user authorization
            if len(allowedUsers) > 0 && !contains(allowedUsers, msg.FromID) {
                return sendMessage(ctx, msg.ChatID, OutgoingMessage{
                    Text: "🚫 You are not authorized to use this bot.",
                })
            }

            // Check group authorization (for group chats)
            if msg.ChatID < 0 && len(allowedGroups) > 0 && !contains(allowedGroups, msg.ChatID) {
                return sendMessage(ctx, msg.ChatID, OutgoingMessage{
                    Text: "🚫 This bot is not allowed in this group.",
                })
            }

            return next(ctx, msg)
        }
    }
}

func contains(slice []int64, item int64) bool {
    for _, v := range slice {
        if v == item {
            return true
        }
    }
    return false
}
```

### Rate Limit Middleware

Limits request frequency:

```go
// RateLimitMiddleware limits requests per user.
func RateLimitMiddleware(requestsPerSecond float64, burst int) Middleware {
    limiter := rate.NewLimiter(rate.Limit(requestsPerSecond), burst)

    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            if !limiter.Allow() {
                return sendMessage(ctx, msg.ChatID, OutgoingMessage{
                    Text: "⏳ Too many requests. Please slow down.",
                })
            }
            return next(ctx, msg)
        }
    }
}
```

### Per-User Rate Limiter

Separate rate limit per user:

```go
// PerUserRateLimitMiddleware limits requests per user individually.
func PerUserRateLimitMiddleware(requestsPerMinute int) Middleware {
    type userLimiter struct {
        limiter  *rate.Limiter
        lastSeen time.Time
    }

    var (
        limiters = make(map[int64]*userLimiter)
        mu       sync.Mutex
    )

    // Cleanup goroutine
    go func() {
        for {
            time.Sleep(time.Minute)
            mu.Lock()
            for id, ul := range limiters {
                if time.Since(ul.lastSeen) > 5*time.Minute {
                    delete(limiters, id)
                }
            }
            mu.Unlock()
        }
    }()

    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            mu.Lock()
            ul, ok := limiters[msg.FromID]
            if !ok {
                ul = &userLimiter{
                    limiter: rate.NewLimiter(
                        rate.Limit(requestsPerMinute)/60,
                        requestsPerMinute,
                    ),
                }
                limiters[msg.FromID] = ul
            }
            ul.lastSeen = time.Now()
            mu.Unlock()

            if !ul.limiter.Allow() {
                return sendMessage(ctx, msg.ChatID, OutgoingMessage{
                    Text: "⏳ Rate limit exceeded. Please wait.",
                })
            }
            return next(ctx, msg)
        }
    }
}
```

### Session Context Middleware

Adds current session to context:

```go
// SessionContextMiddleware adds the active session to context.
func SessionContextMiddleware(sessionManager SessionManager) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            // Get user's active session
            session, err := sessionManager.GetActiveSession(msg.FromID)
            if err == nil {
                ctx = context.WithValue(ctx, sessionKey, session)
            }

            // Always proceed, even if no session
            return next(ctx, msg)
        }
    }
}
```

### Telemetry Middleware

Collects metrics:

```go
// TelemetryMiddleware collects metrics about message processing.
func TelemetryMiddleware(metrics MetricsService) Middleware {
    return func(next HandlerFunc) HandlerFunc {
        return func(ctx context.Context, msg *IncomingMessage) error {
            // Increment counter
            metrics.IncrementCounter("messages_received", map[string]string{
                "command": msg.Command,
            })

            // Record latency
            start := time.Now()

            err := next(ctx, msg)

            // Record latency
            metrics.RecordHistogram("message_duration", time.Since(start).Seconds())

            // Record errors
            if err != nil {
                metrics.IncrementCounter("messages_failed", map[string]string{
                    "command": msg.Command,
                    "error":   err.Error(),
                })
            } else {
                metrics.IncrementCounter("messages_processed", map[string]string{
                    "command": msg.Command,
                })
            }

            return err
        }
    }
}
```

---

## Creating Custom Middleware

### Template

```go
package middleware

import (
    "context"
    "github.com/KotVnn/Telegram-Open-CLI/internal/telegram"
)

// MyMiddleware does something useful.
func MyMiddleware(config Config) telegram.Middleware {
    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            // Pre-processing
            // ...

            // Call next handler
            err := next(ctx, msg)

            // Post-processing
            // ...

            return err
        }
    }
}
```

### Configuration Example

```go
// Config holds middleware configuration.
type Config struct {
    Enabled     bool
    Debug       bool
    // ... other config
}

// MyMiddleware with configuration
func MyMiddleware(cfg Config) telegram.Middleware {
    if !cfg.Enabled {
        // Return passthrough middleware
        return func(next telegram.HandlerFunc) telegram.HandlerFunc {
            return next
        }
    }

    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            // Middleware logic
            return next(ctx, msg)
        }
    }
}
```

---

## Middleware Examples

### Command Whitelist Middleware

Restrict commands to specific user roles:

```go
func CommandWhitelistMiddleware(allowedCommands map[string][]string) telegram.Middleware {
    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            if msg.Command == "" {
                return next(ctx, msg)
            }

            user := ctx.Value(userKey).(*User)
            allowed := allowedCommands[msg.Command]

            if !containsString(allowed, string(user.Role)) {
                return sendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
                    Text: "🚫 You don't have permission to use this command.",
                })
            }

            return next(ctx, msg)
        }
    }
}

// Usage
allowedCommands := map[string][]string{
    "delete":  {"admin"},
    "config":  {"admin", "user"},
    "sessions": {"admin", "user", "viewer"},
}

adapter.Use(CommandWhitelistMiddleware(allowedCommands))
```

### Cooldown Middleware

Add cooldown between operations:

```go
func CooldownMiddleware(cooldown time.Duration) telegram.Middleware {
    var (
        lastExecutions = make(map[int64]time.Time)
        mu             sync.Mutex
    )

    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            mu.Lock()
            lastExec, ok := lastExecutions[msg.FromID]
            if ok && time.Since(lastExec) < cooldown {
                mu.Unlock()
                remaining := cooldown - time.Since(lastExec)
                return sendMessage(ctx, msg.ChatID, telegram.OutgoingMessage{
                    Text: fmt.Sprintf("⏳ Please wait %s before next command.", remaining.Round(time.Second)),
                })
            }
            lastExecutions[msg.FromID] = time.Now()
            mu.Unlock()

            return next(ctx, msg)
        }
    }
}

// Usage
adapter.Use(CooldownMiddleware(5 * time.Second))
```

### Audit Log Middleware

Log all user actions for audit:

```go
func AuditLogMiddleware(auditLogger AuditLogger) telegram.Middleware {
    return func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            start := time.Now()

            err := next(ctx, msg)

            // Log audit event
            auditLogger.Log(AuditEvent{
                UserID:    msg.FromID,
                Username:  msg.FromUsername,
                Command:   msg.Command,
                Text:      msg.Text,
                ChatID:    msg.ChatID,
                Success:   err == nil,
                Error:     err,
                Duration:  time.Since(start),
                Timestamp: start,
            })

            return err
        }
    }
}
```

---

## Testing Middleware

### Unit Test Template

```go
package middleware

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/KotVnn/Telegram-Open-CLI/internal/telegram"
)

func TestMyMiddleware_CallsNext(t *testing.T) {
    called := false
    handler := MyMiddleware()(func(ctx context.Context, msg *telegram.IncomingMessage) error {
        called = true
        return nil
    })

    err := handler(context.Background(), &telegram.IncomingMessage{
        FromID: 12345,
        Text:   "test",
    })

    assert.NoError(t, err)
    assert.True(t, called)
}

func TestMyMiddleware_SkipsOnCondition(t *testing.T) {
    called := false
    handler := MyMiddleware()(func(ctx context.Context, msg *telegram.IncomingMessage) error {
        called = true
        return nil
    })

    err := handler(context.Background(), &telegram.IncomingMessage{
        FromID: 99999, // Should skip
        Text:   "test",
    })

    assert.NoError(t, err)
    assert.False(t, called)
}

func TestAuthMiddleware_Unauthorized(t *testing.T) {
    handler := AuthMiddleware([]int64{12345}, nil)(func(ctx context.Context, msg *telegram.IncomingMessage) error {
        t.Fatal("handler should not be called")
        return nil
    })

    err := handler(context.Background(), &telegram.IncomingMessage{
        FromID: 99999,
    })

    assert.Error(t, err)
}

func TestRateLimitMiddleware_Exceeded(t *testing.T) {
    handler := RateLimitMiddleware(1, 1)(func(ctx context.Context, msg *telegram.IncomingMessage) error {
        return nil
    })

    ctx := context.Background()
    msg := &telegram.IncomingMessage{FromID: 12345}

    // First request should pass
    err := handler(ctx, msg)
    assert.NoError(t, err)

    // Second request should be rate limited
    err = handler(ctx, msg)
    assert.Error(t, err)
}
```

### Integration Test

```go
func TestMiddlewareChain_ExecutionOrder(t *testing.T) {
    var order []string

    m1 := func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            order = append(order, "m1-before")
            err := next(ctx, msg)
            order = append(order, "m1-after")
            return err
        }
    }

    m2 := func(next telegram.HandlerFunc) telegram.HandlerFunc {
        return func(ctx context.Context, msg *telegram.IncomingMessage) error {
            order = append(order, "m2-before")
            err := next(ctx, msg)
            order = append(order, "m2-after")
            return err
        }
    }

    handler := func(ctx context.Context, msg *telegram.IncomingMessage) error {
        order = append(order, "handler")
        return nil
    }

    // Build chain
    var chain telegram.HandlerFunc = handler
    chain = m2(chain)
    chain = m1(chain)

    // Execute
    err := chain(context.Background(), &telegram.IncomingMessage{})
    assert.NoError(t, err)

    // Verify order
    assert.Equal(t, []string{
        "m1-before",
        "m2-before",
        "handler",
        "m2-after",
        "m1-after",
    }, order)
}
```
