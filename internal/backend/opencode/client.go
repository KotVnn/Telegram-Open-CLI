package opencode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
)

// Client is a thin HTTP client for the opencode serve API.
type Client struct {
	baseURL    string
	password   string
	httpClient *http.Client
	logger     zerolog.Logger
}

// NewClient creates a client for the given base URL (e.g. http://127.0.0.1:4096).
func NewClient(baseURL, password string, logger zerolog.Logger) *Client {
	timeout := 10 * time.Second
	if baseURL == "" {
		baseURL = "http://127.0.0.1:4096"
	}
	return &Client{
		baseURL:    baseURL,
		password:   password,
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
	}
}

// Session is the payload returned by POST /session and GET /session.
type Session struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Agent       string            `json:"agent"`
	Model       *ModelInfo        `json:"model"`
	ParentID    string            `json:"parentID"`
	WorkspaceID string            `json:"workspaceID"`
	Time        *TimeInfo         `json:"time"`
	Info        map[string]string `json:"info"`
}

// ModelInfo is the model payload inside a session or agent.
type ModelInfo struct {
	ID         string `json:"id"`
	ProviderID string `json:"providerID"`
	Model      string `json:"model"`
}

// TimeInfo holds session/message timestamps (epoch milliseconds).
type TimeInfo struct {
	Created   int64 `json:"created"`
	Updated   int64 `json:"updated"`
	Completed int64 `json:"completed"`
}

// Time converts a created timestamp to a time.Time.
func (t TimeInfo) Time() time.Time {
	if t.Created > 0 {
		return time.UnixMilli(t.Created)
	}
	return time.Time{}
}

// Message is the payload returned by POST /session/{id}/message.
type Message struct {
	ID        string   `json:"id"`
	Role      string   `json:"role"`
	SessionID string   `json:"sessionID"`
	Time      TimeInfo `json:"time"`
	Parts     []Part   `json:"parts"`
	Error     string   `json:"error,omitempty"`
}

// Part is a single content part of a message.
type Part struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	SessionID string         `json:"sessionID"`
	MessageID string         `json:"messageID"`
	Text      string         `json:"text"`
	State     string         `json:"state,omitempty"`
	Path      string         `json:"path,omitempty"`
	Tool      string         `json:"tool,omitempty"`
	ToolInput map[string]any `json:"toolInput,omitempty"`
	Title     string         `json:"title,omitempty"`
}

// PartText extracts the visible text of a part for user display.
func (p Part) PartText() string {
	switch p.Type {
	case "text", "reasoning":
		return p.Text
	case "tool":
		title := p.Tool
		if p.Title != "" {
			title = p.Title
		}
		if title != "" {
			return fmt.Sprintf("tool %s %s", title, p.State)
		}
		return ""
	case "file":
		return fmt.Sprintf("file %s", p.Path)
	case "subtask":
		return p.Title
	default:
		return ""
	}
}

// MessageResponse is the response envelope of a message send.
type MessageResponse struct {
	Info  Message `json:"info"`
	Parts []Part  `json:"parts"`
}

// Health checks that the server is up and returns its version if available.
func (c *Client) Health(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/app", nil)
	if err != nil {
		return "", err
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", fmt.Errorf("health check: status %d: %s", resp.StatusCode, body)
	}

	var info struct {
		Version string `json:"version"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&info)
	return info.Version, nil
}

// CreateSession creates a new session and returns its id.
func (c *Client) CreateSession(ctx context.Context, title, agent, model string) (string, error) {
	body := map[string]any{}
	if title != "" {
		body["title"] = title
	}
	if agent != "" {
		body["agent"] = agent
	}
	if model != "" {
		if m, err := parseModel(model); err == nil {
			body["model"] = map[string]string{"id": m.id, "providerID": m.provider}
		}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/session", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", fmt.Errorf("create session: status %d: %s", resp.StatusCode, body)
	}

	var sess Session
	if err := json.NewDecoder(resp.Body).Decode(&sess); err != nil {
		return "", fmt.Errorf("create session: decode: %w", err)
	}
	if sess.ID == "" {
		return "", fmt.Errorf("create session: empty session id")
	}
	return sess.ID, nil
}

// SendMessage sends a message to a session and returns the full response.
// It blocks until the run completes.
func (c *Client) SendMessage(ctx context.Context, sessionID string, body map[string]any) (*MessageResponse, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/session/"+url.PathEscape(sessionID)+"/message", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("send message: status %d: %s", resp.StatusCode, raw)
	}

	var out MessageResponse
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("send message: decode: %w", err)
	}
	return &out, nil
}

// Abort stops the currently running message in a session.
func (c *Client) Abort(ctx context.Context, sessionID string) error {
	return c.postNoContent(ctx, "/session/"+url.PathEscape(sessionID)+"/abort")
}

// RevertMessage reverts a message inside a session.
func (c *Client) RevertMessage(ctx context.Context, sessionID, messageID string) error {
	body := map[string]any{"messageID": messageID}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/session/"+url.PathEscape(sessionID)+"/revert", payload)
}

// SessionMessage is a message in the v2 list-messages response.
type SessionMessage struct {
	ID      string        `json:"id"`
	Type    string        `json:"type"`
	Time    TimeInfo      `json:"time"`
	Text    string        `json:"text"`
	Command string        `json:"command"`
	Output  string        `json:"output"`
	Content []ContentItem `json:"content"`
}

// ContentItem is a single piece of an assistant message's content.
type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// DisplayText returns a human readable representation of the message.
func (m SessionMessage) DisplayText() string {
	if m.Text != "" {
		return m.Text
	}
	var sb strings.Builder
	for _, c := range m.Content {
		if c.Type == "text" && c.Text != "" {
			sb.WriteString(c.Text)
			sb.WriteString("\n")
		}
	}
	if sb.Len() > 0 {
		return strings.TrimSpace(sb.String())
	}
	if m.Command != "" {
		return fmt.Sprintf("$ %s\n%s", m.Command, m.Output)
	}
	return ""
}

// ListMessages returns the messages of a session, newest first by default.
func (c *Client) ListMessages(ctx context.Context, sessionID string, limit int, offset int) ([]SessionMessage, error) {
	q := url.Values{}
	if limit > 0 {
		q.Set("limit", strconv.Itoa(limit))
	}
	if offset > 0 {
		q.Set("offset", strconv.Itoa(offset))
	}
	q.Set("order", "desc")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/session/"+url.PathEscape(sessionID)+"/message?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("list messages: status %d: %s", resp.StatusCode, body)
	}

	var out struct {
		Data []SessionMessage `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// Provider describes an LLM provider.
type Provider struct {
	ID     string            `json:"id"`
	Name   string            `json:"name"`
	Models map[string]*Model `json:"models"`
}

// Model describes a model exposed by a provider.
type Model struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	ProviderID string   `json:"providerID"`
	Reasoning  bool     `json:"reasoning"`
	Cost       *Cost    `json:"cost,omitempty"`
	Cap        []string `json:"cap,omitempty"`
}

// Cost describes model pricing.
type Cost struct {
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

// Agent describes a coding agent.
type Agent struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Mode        string    `json:"mode"`
	Hidden      bool      `json:"hidden"`
	Model       ModelInfo `json:"model"`
}

// ListProviders returns the configured providers and their models.
func (c *Client) ListProviders(ctx context.Context) ([]Provider, []string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/provider", nil)
	if err != nil {
		return nil, nil, err
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, nil, fmt.Errorf("list providers: status %d: %s", resp.StatusCode, body)
	}

	var out struct {
		All       []Provider `json:"all"`
		Connected []string   `json:"connected"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, nil, err
	}
	return out.All, out.Connected, nil
}

// ListAgents returns the configured agents.
func (c *Client) ListAgents(ctx context.Context) ([]Agent, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/agent", nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("list agents: status %d: %s", resp.StatusCode, body)
	}

	var out []Agent
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out, nil
}

// ListFiles lists directory entries in the workspace.
func (c *Client) ListFiles(ctx context.Context, dir string) ([]FileInfo, error) {
	q := url.Values{}
	if dir != "" {
		q.Set("path", dir)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/fs/list?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("list files: status %d: %s", resp.StatusCode, body)
	}

	var out struct {
		Data []FileInfo `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Data, nil
}

// FileInfo describes a file or directory entry returned by /find.
type FileInfo struct {
	Path string `json:"path"`
	Type string `json:"type"`
	Size int64  `json:"size,omitempty"`
}

// PermissionRequestInfo is the payload of a permission.ask event.
type PermissionRequestInfo struct {
	ID         string   `json:"id"`
	Permission string   `json:"permission"`
	Patterns   []string `json:"patterns"`
}

// RespondPermission answers a pending permission request.
func (c *Client) RespondPermission(ctx context.Context, sessionID, permissionID, response string) error {
	body := map[string]any{"response": response}
	payload, err := json.Marshal(body)
	if err != nil {
		return err
	}
	return c.postNoContent(ctx, "/session/"+url.PathEscape(sessionID)+"/permissions/"+url.PathEscape(permissionID), payload)
}

func (c *Client) postNoContent(ctx context.Context, path string, body ...[]byte) error {
	var payload []byte
	if len(body) > 0 {
		payload = body[0]
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	if len(payload) > 0 {
		req.Header.Set("Content-Type", "application/json")
	}
	c.auth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("post %s: status %d: %s", path, resp.StatusCode, raw)
	}
	return nil
}

func (c *Client) auth(req *http.Request) {
	if c.password != "" {
		req.SetBasicAuth("admin", c.password)
	}
}

// parseModel splits "providerID/modelID" into its parts.
func parseModel(s string) (struct{ id, provider string }, error) {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == '/' {
			return struct{ id, provider string }{id: s[i+1:], provider: s[:i]}, nil
		}
	}
	return struct{ id, provider string }{}, fmt.Errorf("model %q: missing provider", s)
}
