//go:build integration

package opencode

import (
	"context"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	return ln.Addr().(*net.TCPAddr).Port
}

func newIntegrationAdapter(t *testing.T, port int, dir string) *Adapter {
	t.Helper()
	adapter := New(WithLogger(zerolog.Nop()))
	err := adapter.Initialize(context.Background(), backend.BackendConfig{
		Enabled:     true,
		Command:     "opencode",
		WorkingDir:  dir,
		Port:        port,
		AutoRestart: true,
		Permission:  "auto",
	})
	require.NoError(t, err)
	return adapter
}

func TestServeAdapterEndToEnd(t *testing.T) {
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode binary not found")
	}
	if os.Getenv("TOC_OPENCODE_INTEGRATION") == "" {
		t.Skip("set TOC_OPENCODE_INTEGRATION=1 to run; requires a configured model provider")
	}

	ctx := context.Background()
	port := freePort(t)
	dir := t.TempDir()
	if err := os.WriteFile(dir+"/README.md", []byte("integration workspace"), 0o644); err != nil {
		t.Fatal(err)
	}

	adapter := newIntegrationAdapter(t, port, dir)

	t.Run("start", func(t *testing.T) {
		require.NoError(t, adapter.Start(ctx))
		require.NoError(t, adapter.Health(ctx))
	})

	t.Run("list models and agents", func(t *testing.T) {
		models, agents, err := adapter.ListModels(ctx)
		require.NoError(t, err)
		t.Logf("models: %d, agents: %v", len(models), agents)
		assert.NotEmpty(t, models)
		assert.NotEmpty(t, agents)
	})

	t.Run("create and message", func(t *testing.T) {
		session, err := adapter.CreateSession(ctx, backend.SessionOpts{
			Title:      "Integration",
			WorkingDir: dir,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, session.ID)
		assert.NotEmpty(t, session.ExternalID, "external session should be created eagerly")

		stream, err := adapter.StreamMessage(ctx, &backend.SendMessageRequest{
			SessionID: session.ID,
			Content:   "Reply with exactly the word PONG and nothing else.",
		})
		require.NoError(t, err)

		var sb strings.Builder
		for chunk := range stream {
			if chunk.Error != nil {
				t.Fatalf("stream error: %v", chunk.Error)
			}
			sb.WriteString(chunk.Content)
		}
		require.NotEmpty(t, sb.String(), "expected a response")
		t.Logf("response: %s", sb.String())
	})

	t.Run("list files", func(t *testing.T) {
		entries, err := adapter.ListFiles(ctx, "")
		require.NoError(t, err)
		found := false
		for _, e := range entries {
			if e.Name == "README.md" {
				found = true
			}
		}
		assert.True(t, found, "workspace README.md should be listed")
	})

	t.Run("stop", func(t *testing.T) {
		require.NoError(t, adapter.Stop(ctx))
	})
}

func TestAdapterLifecycleAttach(t *testing.T) {
	if _, err := exec.LookPath("opencode"); err != nil {
		t.Skip("opencode binary not found")
	}

	port := freePort(t)
	dir := t.TempDir()

	// Start one adapter, keep serve running.
	a1 := newIntegrationAdapter(t, port, dir)
	require.NoError(t, a1.Start(context.Background()))
	externalID, err := a1.client.CreateSession(context.Background(), "persist-me", "", "")
	require.NoError(t, err)
	require.NoError(t, a1.Stop(context.Background()))

	// A second adapter should attach to the same server and find the session.
	a2 := newIntegrationAdapter(t, port, dir)
	require.NoError(t, a2.Start(context.Background()))
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = a2.server.Stop(ctx)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	msgs, err := a2.client.ListMessages(ctx, externalID, 5, 0)
	require.NoError(t, err)
	t.Logf("messages in persisted session: %d", len(msgs))
	require.NoError(t, a2.Stop(context.Background()))
}
