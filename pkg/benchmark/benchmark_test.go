package benchmark

import (
	"context"
	"testing"
	"time"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
	"github.com/KotVnn/Telegram-Open-CLI/internal/backend/opencode"
)

func BenchmarkSendMessage(b *testing.B) {
	bk := opencode.New()

	if err := bk.Initialize(context.Background(), backend.BackendConfig{
		Command: "opencode",
		Args:    []string{"run", "--format", "json"},
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = bk.SendMessage(context.Background(), &backend.SendMessageRequest{
			Content: "Hello",
		})
	}
}

func BenchmarkSessionCreation(b *testing.B) {
	bk := opencode.New()

	if err := bk.Initialize(context.Background(), backend.BackendConfig{
		Command: "opencode",
		Args:    []string{"run", "--format", "json"},
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sess, _ := bk.CreateSession(context.Background(), backend.SessionOpts{})
		if sess != nil {
			_ = bk.DeleteSession(context.Background(), sess.ID)
		}
	}
}

func BenchmarkBackendHealth(b *testing.B) {
	bk := opencode.New()

	if err := bk.Initialize(context.Background(), backend.BackendConfig{
		Command: "opencode",
		Args:    []string{"run", "--format", "json"},
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = bk.Health(context.Background())
	}
}

func BenchmarkConcurrentSessions(b *testing.B) {
	bk := opencode.New()

	if err := bk.Initialize(context.Background(), backend.BackendConfig{
		Command: "opencode",
		Args:    []string{"run", "--format", "json"},
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		sessions := make([]*backend.Session, 10)
		for j := 0; j < 10; j++ {
			sess, _ := bk.CreateSession(context.Background(), backend.SessionOpts{})
			sessions[j] = sess
		}

		for _, sess := range sessions {
			if sess != nil {
				_ = bk.DeleteSession(context.Background(), sess.ID)
			}
		}
	}
}

func BenchmarkResponseTime(b *testing.B) {
	bk := opencode.New()

	if err := bk.Initialize(context.Background(), backend.BackendConfig{
		Command: "opencode",
		Args:    []string{"run", "--format", "json"},
		Timeout: 30 * time.Second,
	}); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		start := time.Now()
		_, _ = bk.SendMessage(context.Background(), &backend.SendMessageRequest{
			Content: "Hello",
		})
		_ = time.Since(start)
	}
}
