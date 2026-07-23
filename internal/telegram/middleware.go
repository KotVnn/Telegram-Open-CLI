package telegram

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// RecoveryMiddleware catches panics and prevents bot crashes.
func RecoveryMiddleware(logger zerolog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error().
						Interface("panic", r).
						Msg("panic recovered in handler")

					err = nil
				}
			}()
			return next(ctx, msg)
		}
	}
}

// LoggingMiddleware logs all messages and their processing time.
func LoggingMiddleware(logger zerolog.Logger) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			start := time.Now()

			logger.Info().
				Int64("user_id", msg.FromID).
				Str("username", msg.FromUsername).
				Str("text", msg.Text).
				Int64("chat_id", msg.ChatID).
				Msg("message received")

			err := next(ctx, msg)

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

// ErrUnauthorized is returned when a user is not authorized.
var ErrUnauthorized = errors.New("unauthorized")

// AuthMiddleware checks if users are authorized.
func AuthMiddleware(adapter Adapter, allowedUsers []int64, allowedChats []int64) Middleware {
	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			if len(allowedUsers) > 0 && !contains(allowedUsers, msg.FromID) {
				_ = adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
					Text: "You are not authorized to use this bot.",
				})
				return ErrUnauthorized
			}

			if msg.ChatID < 0 && len(allowedChats) > 0 && !contains(allowedChats, msg.ChatID) {
				_ = adapter.SendMessage(ctx, msg.ChatID, OutgoingMessage{
					Text: "This chat is not authorized to use this bot.",
				})
				return ErrUnauthorized
			}

			return next(ctx, msg)
		}
	}
}

// RateLimitMiddleware limits request frequency.
func RateLimitMiddleware(requestsPerSecond float64, burst int) Middleware {
	var (
		mu             sync.Mutex
		lastExecutions = make(map[int64]time.Time)
	)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			mu.Lock()
			cutoff := time.Now().Add(-1 * time.Hour)
			for id, t := range lastExecutions {
				if t.Before(cutoff) {
					delete(lastExecutions, id)
				}
			}
			mu.Unlock()
		}
	}()

	return func(next HandlerFunc) HandlerFunc {
		return func(ctx context.Context, msg *IncomingMessage) error {
			mu.Lock()
			lastExec, ok := lastExecutions[msg.FromID]
			shouldLimit := ok && time.Since(lastExec) < time.Duration(float64(time.Second)/requestsPerSecond)
			if !shouldLimit {
				lastExecutions[msg.FromID] = time.Now()
			}
			mu.Unlock()

			if shouldLimit {
				return nil
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
