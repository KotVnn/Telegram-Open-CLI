package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

// Event is a single event parsed from the SSE stream.
type Event struct {
	Type      string          `json:"type"`
	SessionID string          `json:"sessionID"`
	Raw       json.RawMessage `json:"-"`
}

// parseEvent extracts the event type and session id from a raw SSE payload.
// The payload can be a bare event (with its own type/sessionID) or a
// GlobalEvent wrapper that nests the payload.
func parseEvent(data []byte) (Event, error) {
	var wrapper struct {
		Type      string          `json:"type"`
		SessionID string          `json:"sessionID"`
		Payload   json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return Event{}, err
	}

	ev := Event{Type: wrapper.Type, SessionID: wrapper.SessionID, Raw: data}
	if len(wrapper.Payload) > 0 {
		var inner struct {
			Type      string `json:"type"`
			SessionID string `json:"sessionID"`
		}
		if err := json.Unmarshal(wrapper.Payload, &inner); err == nil {
			if ev.Type == "" {
				ev.Type = inner.Type
			}
			if ev.SessionID == "" {
				ev.SessionID = inner.SessionID
			}
		}
	}
	return ev, nil
}

// Hub routes events to per-session subscribers.
type Hub struct {
	mu     sync.Mutex
	subs   map[string]map[chan Event]struct{}
	logger zerolog.Logger
}

// NewHub creates an event hub.
func NewHub(logger zerolog.Logger) *Hub {
	return &Hub{subs: map[string]map[chan Event]struct{}{}, logger: logger}
}

// Subscribe registers a channel for a session and returns an unsubscribe func.
func (h *Hub) Subscribe(sessionID string) (<-chan Event, func()) {
	ch := make(chan Event, 128)
	h.mu.Lock()
	if h.subs[sessionID] == nil {
		h.subs[sessionID] = map[chan Event]struct{}{}
	}
	h.subs[sessionID][ch] = struct{}{}
	h.mu.Unlock()

	var once sync.Once
	unsub := func() {
		once.Do(func() {
			h.mu.Lock()
			delete(h.subs[sessionID], ch)
			if len(h.subs[sessionID]) == 0 {
				delete(h.subs, sessionID)
			}
			h.mu.Unlock()
			close(ch)
		})
	}
	return ch, unsub
}

// Publish delivers an event to all subscribers of its session.
func (h *Hub) Publish(ev Event) {
	if ev.SessionID == "" {
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[ev.SessionID] {
		select {
		case ch <- ev:
		default:
			// Slow subscriber: drop rather than block the SSE loop.
		}
	}
}

// streamSSE opens a GET /event stream and calls handler for each event.
// It blocks until ctx is cancelled or the stream fails.
func streamSSE(ctx context.Context, baseURL, password string, logger zerolog.Logger, handler func(Event)) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL+"/event", nil)
	if err != nil {
		logger.Error().Err(err).Msg("sse: build request")
		return
	}
	if password != "" {
		req.SetBasicAuth("admin", password)
	}
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msg("sse: connect")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		logger.Error().Msgf("sse: status %d: %s", resp.StatusCode, raw)
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			logger.Error().Err(err).Msg("sse: read")
			return
		}
		line = trimSuffix(line, "\n")
		if !hasPrefix(line, "data: ") {
			continue
		}
		data := []byte(line[len("data: "):])
		ev, err := parseEvent(data)
		if err != nil {
			logger.Debug().Err(err).Msg("sse: parse event")
			continue
		}
		handler(ev)
	}
}

func trimSuffix(s, suffix string) string {
	if len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix {
		return s[:len(s)-len(suffix)]
	}
	return s
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// startEventLoop runs the SSE subscription in a goroutine with reconnect
// backoff. It stops when ctx is cancelled.
func startEventLoop(ctx context.Context, baseURL, password string, logger zerolog.Logger, hub *Hub) {
	go func() {
		backoff := time.Second
		for {
			if ctx.Err() != nil {
				return
			}
			streamSSE(ctx, baseURL, password, logger, func(ev Event) {
				hub.Publish(ev)
			})
			if ctx.Err() != nil {
				return
			}
			logger.Warn().Dur("backoff", backoff).Msg("sse: reconnecting")
			select {
			case <-ctx.Done():
				return
			case <-time.After(backoff):
			}
			if backoff < 15*time.Second {
				backoff *= 2
			}
		}
	}()
}
