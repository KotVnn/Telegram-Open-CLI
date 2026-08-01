package opencode

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

func newTestClient(t *testing.T, handler http.Handler) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	return NewClient(server.URL, "", zerolog.Nop())
}

func TestCreateSession(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/session", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		_, _ = w.Write([]byte(`{"id":"ses_test","title":"Hello"}`))
	}))

	id, err := client.CreateSession(context.Background(), "Hello", "", "")
	require.NoError(t, err)
	assert.Equal(t, "ses_test", id)
}

func TestCreateSessionError(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`bad request`))
	}))

	_, err := client.CreateSession(context.Background(), "Hello", "", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "create session")
}

func TestSendMessage(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/session/ses_1/message", r.URL.Path)
		body, _ := json.Marshal(r.Body)
		_ = body
		_, _ = w.Write([]byte(`{
			"info": {"id":"msg_1","role":"assistant","sessionID":"ses_1","time":{"created":1785586000000}},
			"parts": [{"id":"prt_1","type":"text","sessionID":"ses_1","messageID":"msg_1","text":"Hello world"}]
		}`))
	}))

	resp, err := client.SendMessage(context.Background(), "ses_1", map[string]any{"parts": []map[string]any{{"type": "text", "text": "hi"}}})
	require.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Parts, 1)
	assert.Equal(t, "Hello world", resp.Parts[0].Text)
}

func TestListMessages(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/session/ses_1/message", r.URL.Path)
		_, _ = w.Write([]byte(`[
			{"info":{"id":"msg_1","role":"user","time":{"created":1785585000000}},
			 "parts":[{"type":"text","text":"hello"}]},
			{"info":{"id":"msg_2","role":"assistant","time":{"created":1785586000000}},
			 "parts":[{"type":"step-start"},{"type":"text","text":"Hi"},{"type":"step-finish"}]}
		]`))
	}))

	msgs, err := client.ListMessages(context.Background(), "ses_1", 10, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 2)
	assert.Equal(t, "Hi", msgs[0].DisplayText())
	assert.Equal(t, "assistant", msgs[0].Type)
	assert.Equal(t, "hello", msgs[1].DisplayText())
}

func TestAbort(t *testing.T) {
	var path string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	err := client.Abort(context.Background(), "ses_1")
	require.NoError(t, err)
	assert.Equal(t, "/session/ses_1/abort", path)
}

func TestRespondPermission(t *testing.T) {
	var path, body string
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		body = string(raw)
		w.WriteHeader(http.StatusOK)
	}))
	err := client.RespondPermission(context.Background(), "ses_1", "per_1", "once")
	require.NoError(t, err)
	assert.Equal(t, "/session/ses_1/permissions/per_1", path)
	assert.Contains(t, body, `"response":"once"`)
}

func TestHealth(t *testing.T) {
	client := newTestClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/app", r.URL.Path)
		_, _ = w.Write([]byte(`{"version":"1.18.10"}`))
	}))
	version, err := client.Health(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "1.18.10", version)
}

func TestParseModel(t *testing.T) {
	m, err := parseModel("openai/gpt-4o")
	require.NoError(t, err)
	assert.Equal(t, "openai", m.provider)
	assert.Equal(t, "gpt-4o", m.id)

	_, err = parseModel("gpt-4o")
	require.Error(t, err)
}

func TestMessageBody(t *testing.T) {
	body := messageBody(&backend.SendMessageRequest{
		SessionID: "s1",
		Content:   "hello",
		Model:     "openai/gpt-4o",
		Agent:     "build",
	})
	assert.Equal(t, "build", body["agent"])
	model := body["model"].(map[string]string)
	assert.Equal(t, "openai", model["providerID"])
	assert.Equal(t, "gpt-4o", model["modelID"])
}

func TestPartsText(t *testing.T) {
	parts := []Part{
		{ID: "p1", Type: "text", Text: "line one"},
		{ID: "p2", Type: "reasoning", Text: "hidden"},
		{ID: "p3", Type: "tool", Tool: "bash"},
	}
	text := partsText(parts)
	assert.Equal(t, "line one", text)
}

func TestExtractPermission(t *testing.T) {
	raw := json.RawMessage(`{
		"id":"evt_1","type":"permission.asked",
		"properties":{"id":"per_1","sessionID":"ses_1","permission":"edit","patterns":["**"]}
	}`)
	req := extractPermission(raw)
	require.NotNil(t, req)
	assert.Equal(t, "per_1", req.PermissionID)
	assert.Equal(t, "edit", req.Permission)
	assert.Equal(t, []string{"**"}, req.Patterns)
}

func TestExtractPermissionWrapped(t *testing.T) {
	raw := json.RawMessage(`{
		"type":"event","payload":{"type":"permission.asked","properties":{"id":"per_2","permission":"bash","patterns":["*"]}}
	}`)
	req := extractPermission(raw)
	require.NotNil(t, req)
	assert.Equal(t, "per_2", req.PermissionID)
	assert.Equal(t, "bash", req.Permission)
}

func TestExtractPart(t *testing.T) {
	raw := json.RawMessage(`{
		"id":"evt_1","type":"message.part.updated",
		"properties":{"sessionID":"ses_1","part":{"id":"prt_1","type":"text","sessionID":"ses_1","messageID":"msg_1","text":"abc"}}
	}`)
	part := extractPart(raw)
	require.NotNil(t, part)
	assert.Equal(t, "prt_1", part.ID)
	assert.Equal(t, "abc", part.Text)
}

func TestPartStreamerDedup(t *testing.T) {
	streamCh := make(chan backend.StreamChunk, 16)
	s := newPartStreamer(streamCh)

	// Simulate SSE events carrying the full (growing) part text.
	s.handleEvent(context.Background(), nil, &sessionRec{externalID: "ses_1"}, Event{
		Type: "message.part.updated",
		Raw:  json.RawMessage(`{"part":{"id":"prt_1","type":"text","text":"Hel"}}`),
	})
	s.handleEvent(context.Background(), nil, &sessionRec{externalID: "ses_1"}, Event{
		Type: "message.part.updated",
		Raw:  json.RawMessage(`{"part":{"id":"prt_1","type":"text","text":"Hello"}}`),
	})

	first := <-streamCh
	second := <-streamCh
	assert.Equal(t, "Hel", first.Content)
	assert.Equal(t, "lo", second.Content)

	// Flushing the final part must not duplicate text already streamed.
	s.flushParts([]Part{{ID: "prt_1", Type: "text", Text: "Hello"}})
	select {
	case dup := <-streamCh:
		t.Fatalf("unexpected duplicate chunk: %q", dup.Content)
	default:
	}

	// A part that was never streamed should be flushed once.
	s.flushParts([]Part{{ID: "prt_2", Type: "text", Text: "fresh"}})
	assert.Equal(t, "fresh", (<-streamCh).Content)
}
