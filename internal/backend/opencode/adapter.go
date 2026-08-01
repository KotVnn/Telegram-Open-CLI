package opencode

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

var (
	ErrBackendDisabled   = errors.New("backend disabled")
	ErrMissingCommand    = errors.New("backend requires a command")
	ErrNotInitialized    = errors.New("backend not initialized")
	ErrSessionNotFound   = errors.New("session not found")
	ErrServerUnreachable = errors.New("opencode serve unreachable")
)

// sessionRec tracks the mapping between a TOC session and an external
// opencode serve session.
type sessionRec struct {
	sessionID  string
	externalID string
	model      string
	agent      string
	title      string
	workingDir string
	created    time.Time
}

// Adapter implements the backend.Backend interface against an `opencode serve`
// HTTP server. It manages the serve subprocess, a streaming event loop and a
// registry of external session ids.
type Adapter struct {
	logger     zerolog.Logger
	config     backend.BackendConfig
	server     *Server
	client     *Client
	hub        *Hub
	permission string

	mu       sync.RWMutex
	sessions map[string]*sessionRec
	inflight map[string]context.CancelFunc
	handlers map[string]backend.PermissionHandler

	initialized bool
	started     atomic.Bool
	eventCancel context.CancelFunc
}

// Option configures an Adapter.
type Option func(*Adapter)

// WithLogger sets the adapter's logger.
func WithLogger(logger zerolog.Logger) Option {
	return func(a *Adapter) {
		a.logger = logger
	}
}

// New creates a new Adapter instance.
func New(opts ...Option) *Adapter {
	a := &Adapter{
		logger:     zerolog.Nop(),
		sessions:   make(map[string]*sessionRec),
		inflight:   make(map[string]context.CancelFunc),
		handlers:   make(map[string]backend.PermissionHandler),
		permission: "auto",
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *Adapter) Name() string {
	return "opencode"
}

func (a *Adapter) Description() string {
	return "OpenCode AI Coding Agent (serve mode)"
}

func (a *Adapter) Version() string {
	return "2.0.0"
}

func (a *Adapter) Initialize(ctx context.Context, config backend.BackendConfig) error {
	if !config.Enabled {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrBackendDisabled)
	}
	if config.Command == "" {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrMissingCommand)
	}

	hostname := config.Hostname
	if hostname == "" {
		hostname = "127.0.0.1"
	}
	port := config.Port
	if port == 0 {
		port = 4096
	}
	permission := config.Permission
	if permission == "" {
		permission = "auto"
	}

	a.config = config
	a.permission = permission
	a.client = NewClient(fmt.Sprintf("http://%s:%d", hostname, port), config.Password, a.logger)
	a.server = NewServer(config.Command, hostname, port, config.WorkingDir, config.Password, envSlice(config.Environment), a.logger)
	a.hub = NewHub(a.logger)
	a.initialized = true
	return nil
}

func (a *Adapter) Start(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrNotInitialized)
	}

	if _, err := exec.LookPath(a.config.Command); err != nil {
		return fmt.Errorf("backend %s: command %q not found: %w", a.Name(), a.config.Command, err)
	}

	// Attach to an already running server if possible, otherwise spawn one.
	if version, err := a.client.Health(ctx); err == nil {
		a.logger.Info().Str("version", version).Msg("attaching to existing opencode serve")
	} else {
		if !a.config.AutoRestart {
			return fmt.Errorf("backend %s: %w: %v", a.Name(), ErrServerUnreachable, err)
		}
		if err := a.server.Start(ctx); err != nil {
			return err
		}
		if err := a.waitReady(ctx); err != nil {
			return err
		}
	}

	eventCtx, cancel := context.WithCancel(context.Background())
	a.eventCancel = cancel
	startEventLoop(eventCtx, a.client.baseURL, a.config.Password, a.logger, a.hub)

	a.started.Store(true)
	return nil
}

func (a *Adapter) waitReady(ctx context.Context) error {
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if version, err := a.client.Health(ctx); err == nil {
			a.logger.Info().Str("version", version).Msg("opencode serve ready")
			return nil
		}
		time.Sleep(300 * time.Millisecond)
	}
	return fmt.Errorf("backend %s: %w: timed out waiting for server", a.Name(), ErrServerUnreachable)
}

func (a *Adapter) Stop(ctx context.Context) error {
	if a.eventCancel != nil {
		a.eventCancel()
	}

	a.mu.Lock()
	for _, cancel := range a.inflight {
		cancel()
	}
	a.inflight = make(map[string]context.CancelFunc)
	a.sessions = make(map[string]*sessionRec)
	a.mu.Unlock()

	// Do not kill the serve process on shutdown so sessions survive a restart;
	// the next Start() attaches to it.
	a.started.Store(false)
	return nil
}

func (a *Adapter) Health(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrNotInitialized)
	}
	if _, err := a.client.Health(ctx); err != nil {
		return fmt.Errorf("backend %s: %w: %v", a.Name(), ErrServerUnreachable, err)
	}
	return nil
}

// getOrCreateRec returns the external session record for a TOC session,
// creating it (and its external counterpart) on first use.
func (a *Adapter) getOrCreateRec(ctx context.Context, sessionID, externalID, title, model, agent, workingDir string) (*sessionRec, error) {
	a.mu.RLock()
	rec := a.sessions[sessionID]
	a.mu.RUnlock()

	if rec == nil {
		rec = &sessionRec{
			sessionID:  sessionID,
			title:      title,
			model:      model,
			agent:      agent,
			workingDir: workingDir,
		}
		a.mu.Lock()
		if existing := a.sessions[sessionID]; existing != nil {
			rec = existing
		} else {
			a.sessions[sessionID] = rec
		}
		a.mu.Unlock()
	}

	// Adopt an externally supplied external id (e.g. restored from storage).
	if rec.externalID == "" && externalID != "" {
		a.mu.Lock()
		rec.externalID = externalID
		a.mu.Unlock()
	}

	if rec.externalID == "" {
		id, err := a.client.CreateSession(ctx, rec.title, rec.agent, rec.model)
		if err != nil {
			return nil, fmt.Errorf("create external session: %w", err)
		}
		a.mu.Lock()
		rec.externalID = id
		a.mu.Unlock()
	}

	return rec, nil
}

// ensureExternalSession verifies the external session still exists on the
// server, recreating it if it was lost (e.g. after a server restart).
func (a *Adapter) ensureExternalSession(ctx context.Context, rec *sessionRec) error {
	if rec.externalID == "" {
		return nil
	}
	if _, err := a.client.ListMessages(ctx, rec.externalID, 1, 0); err == nil {
		return nil
	}

	a.logger.Warn().Str("session_id", rec.sessionID).Str("external_id", rec.externalID).
		Msg("external session no longer exists; recreating")
	id, err := a.client.CreateSession(ctx, rec.title, rec.agent, rec.model)
	if err != nil {
		return fmt.Errorf("recreate external session: %w", err)
	}
	a.mu.Lock()
	rec.externalID = id
	a.mu.Unlock()
	return nil
}

func (a *Adapter) CreateSession(ctx context.Context, opts backend.SessionOpts) (*backend.Session, error) {
	session := &backend.Session{
		ID:         "ses-" + uuid.NewString(),
		Backend:    a.Name(),
		ProjectID:  opts.ProjectID,
		Title:      opts.Title,
		Status:     backend.SessionStatusActive,
		Model:      opts.Model,
		Agent:      opts.Agent,
		WorkingDir: opts.WorkingDir,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
		Metadata:   opts.Metadata,
	}

	rec := &sessionRec{
		sessionID:  session.ID,
		title:      opts.Title,
		model:      opts.Model,
		agent:      opts.Agent,
		workingDir: opts.WorkingDir,
		created:    session.CreatedAt,
	}

	a.mu.Lock()
	a.sessions[session.ID] = rec
	a.mu.Unlock()

	if a.started.Load() {
		externalID, err := a.client.CreateSession(ctx, rec.title, rec.agent, rec.model)
		if err != nil {
			// Non-fatal: the external session is created lazily on first message.
			a.logger.Warn().Err(err).Str("session_id", session.ID).Msg("create external session failed")
		} else {
			a.mu.Lock()
			rec.externalID = externalID
			a.mu.Unlock()
			session.ExternalID = externalID
		}
	}

	return session, nil
}

func (a *Adapter) GetSession(ctx context.Context, id string) (*backend.Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	rec, ok := a.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s: %w", id, ErrSessionNotFound)
	}
	return rec.toBackendSession(a.Name()), nil
}

func (a *Adapter) ListSessions(ctx context.Context, projectID string) ([]*backend.Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var out []*backend.Session
	for _, rec := range a.sessions {
		out = append(out, rec.toBackendSession(a.Name()))
	}
	return out, nil
}

func (a *Adapter) DeleteSession(ctx context.Context, id string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if _, ok := a.sessions[id]; !ok {
		return fmt.Errorf("session %s: %w", id, ErrSessionNotFound)
	}
	delete(a.sessions, id)
	return nil
}

func (rec *sessionRec) toBackendSession(backendName string) *backend.Session {
	return &backend.Session{
		ID:         rec.sessionID,
		Backend:    backendName,
		Title:      rec.title,
		Status:     backend.SessionStatusActive,
		Model:      rec.model,
		Agent:      rec.agent,
		WorkingDir: rec.workingDir,
		ExternalID: rec.externalID,
		CreatedAt:  rec.created,
		UpdatedAt:  rec.created,
	}
}

func (a *Adapter) SendMessage(ctx context.Context, req *backend.SendMessageRequest) (*backend.SendMessageResponse, error) {
	rec, err := a.getOrCreateRec(ctx, req.SessionID, req.ExternalID, "", req.Model, req.Agent, req.WorkingDir)
	if err != nil {
		return nil, err
	}
	if err := a.ensureExternalSession(ctx, rec); err != nil {
		return nil, err
	}

	resp, err := a.client.SendMessage(ctx, rec.externalID, messageBody(req))
	if err != nil {
		return nil, err
	}

	content := partsText(resp.Parts)
	return &backend.SendMessageResponse{
		ID:       resp.Info.ID,
		Content:  content,
		Metadata: map[string]interface{}{"externalID": rec.externalID},
	}, nil
}

func (a *Adapter) StreamMessage(ctx context.Context, req *backend.SendMessageRequest) (<-chan backend.StreamChunk, error) {
	rec, err := a.getOrCreateRec(ctx, req.SessionID, req.ExternalID, "", req.Model, req.Agent, req.WorkingDir)
	if err != nil {
		return nil, err
	}
	if err := a.ensureExternalSession(ctx, rec); err != nil {
		return nil, err
	}

	ch := make(chan backend.StreamChunk, 256)

	reqCtx, cancel := context.WithCancel(ctx)
	if a.config.Timeout > 0 {
		reqCtx, cancel = context.WithTimeout(reqCtx, a.config.Timeout)
	}

	a.mu.Lock()
	a.inflight[req.SessionID] = cancel
	a.handlers[rec.externalID] = req.PermissionHandler
	a.mu.Unlock()

	go func() {
		defer close(ch)
		defer func() {
			a.mu.Lock()
			delete(a.inflight, req.SessionID)
			delete(a.handlers, rec.externalID)
			a.mu.Unlock()
			cancel()
		}()

		sub, unsub := a.hub.Subscribe(rec.externalID)
		defer unsub()

		streamer := newPartStreamer(ch)

		// The SSE reader forwards events to the streamer while the blocking
		// message call below runs.
		eventDone := make(chan struct{})
		go func() {
			defer close(eventDone)
			for {
				select {
				case <-reqCtx.Done():
					return
				case ev, ok := <-sub:
					if !ok {
						return
					}
					streamer.handleEvent(reqCtx, a, rec, ev)
				}
			}
		}()

		resp, sendErr := a.client.SendMessage(reqCtx, rec.externalID, messageBody(req))

		// Close the subscription so the SSE reader goroutine can exit before
		// we flush the final parts.
		unsub()
		<-eventDone

		if sendErr != nil {
			if reqCtx.Err() != nil {
				select {
				case ch <- backend.StreamChunk{Error: fmt.Errorf("message cancelled: %w", reqCtx.Err()), Done: true}:
				case <-ctx.Done():
				}
				return
			}
			select {
			case ch <- backend.StreamChunk{Error: sendErr, Done: true}:
			case <-ctx.Done():
			}
			return
		}

		// Flush any text parts that were not already streamed (e.g. when the
		// SSE loop was down).
		streamer.flushParts(resp.Parts)

		select {
		case ch <- backend.StreamChunk{Done: true}:
		case <-ctx.Done():
		}
	}()

	return ch, nil
}

func (a *Adapter) Capabilities() *backend.Capabilities {
	return &backend.Capabilities{
		SupportsStreaming:   true,
		SupportsFiles:       true,
		SupportsMultiModal:  true,
		SupportsToolCalling: true,
		MaxTokens:           200000,
		SupportedModels:     []string{},
		SupportedAgents:     []string{"build", "plan"},
	}
}

// Abort cancels the in-flight message in a session and asks the server to stop.
func (a *Adapter) Abort(ctx context.Context, sessionID string) error {
	a.mu.RLock()
	cancel := a.inflight[sessionID]
	rec := a.sessions[sessionID]
	a.mu.RUnlock()

	if cancel != nil {
		cancel()
	}
	if rec == nil {
		return fmt.Errorf("session %s: %w", sessionID, ErrSessionNotFound)
	}
	return a.client.Abort(ctx, rec.externalID)
}

// ExternalSessionID returns the external serve session id for a TOC session.
func (a *Adapter) ExternalSessionID(ctx context.Context, sessionID string) (string, error) {
	a.mu.RLock()
	rec, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok {
		return "", fmt.Errorf("session %s: %w", sessionID, ErrSessionNotFound)
	}
	return rec.externalID, nil
}

// ListMessages lists the last `limit` messages of a session.
func (a *Adapter) ListMessages(ctx context.Context, sessionID string, limit int) ([]*backend.MessageInfo, error) {
	a.mu.RLock()
	rec, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session %s: %w", sessionID, ErrSessionNotFound)
	}

	msgs, err := a.client.ListMessages(ctx, rec.externalID, limit, 0)
	if err != nil {
		return nil, err
	}

	out := make([]*backend.MessageInfo, 0, len(msgs))
	for _, m := range msgs {
		out = append(out, &backend.MessageInfo{
			ID:      m.ID,
			Role:    m.Type,
			Content: m.DisplayText(),
			Time:    m.Time.Time(),
		})
	}
	return out, nil
}

// RevertMessage reverts a message inside a session.
func (a *Adapter) RevertMessage(ctx context.Context, sessionID, messageID string) error {
	a.mu.RLock()
	rec, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s: %w", sessionID, ErrSessionNotFound)
	}
	return a.client.RevertMessage(ctx, rec.externalID, messageID)
}

// ListFiles lists files and directories in the workspace.
func (a *Adapter) ListFiles(ctx context.Context, dir string) ([]*backend.FileEntry, error) {
	files, err := a.client.ListFiles(ctx, dir)
	if err != nil {
		return nil, err
	}
	out := make([]*backend.FileEntry, 0, len(files))
	for _, f := range files {
		entry := &backend.FileEntry{
			Path: f.Path,
			Size: f.Size,
		}
		if f.Type == "directory" {
			entry.IsDir = true
		}
		parts := strings.Split(strings.TrimSuffix(f.Path, "/"), "/")
		if len(parts) > 0 {
			entry.Name = parts[len(parts)-1]
		}
		out = append(out, entry)
	}
	return out, nil
}

// ListModels returns the models and agents available on the serve server.
func (a *Adapter) ListModels(ctx context.Context) ([]string, []string, error) {
	providers, connected, err := a.client.ListProviders(ctx)
	if err != nil {
		return nil, nil, err
	}

	connSet := map[string]bool{}
	for _, id := range connected {
		connSet[id] = true
	}

	seen := map[string]bool{}
	var models []string
	for _, p := range providers {
		if len(connSet) > 0 && !connSet[p.ID] {
			continue
		}
		for _, m := range p.Models {
			id := p.ID + "/" + m.ID
			if seen[id] {
				continue
			}
			seen[id] = true
			models = append(models, id)
		}
	}

	agents, err := a.client.ListAgents(ctx)
	if err != nil {
		return models, nil, nil
	}
	var names []string
	for _, ag := range agents {
		if ag.Hidden {
			continue
		}
		names = append(names, ag.Name)
	}
	return models, names, nil
}

// RespondPermission answers a pending permission request.
func (a *Adapter) RespondPermission(ctx context.Context, sessionID, permissionID string, decision backend.PermissionDecision) error {
	a.mu.RLock()
	rec, ok := a.sessions[sessionID]
	a.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s: %w", sessionID, ErrSessionNotFound)
	}

	response := "reject"
	if decision.Response == "allow" {
		response = "once"
		if decision.Remember {
			response = "always"
		}
	}
	return a.client.RespondPermission(ctx, rec.externalID, permissionID, response)
}

// ---- helpers ----

func messageBody(req *backend.SendMessageRequest) map[string]any {
	body := map[string]any{
		"parts": []map[string]any{{"type": "text", "text": req.Content}},
	}
	if req.Model != "" {
		if m, err := parseModel(req.Model); err == nil {
			body["model"] = map[string]string{"providerID": m.provider, "modelID": m.id}
		}
	}
	if req.Agent != "" {
		body["agent"] = req.Agent
	}
	if len(req.Files) > 0 {
		parts := body["parts"].([]map[string]any)
		for _, f := range req.Files {
			parts = append(parts, map[string]any{
				"type": "text",
				"text": fmt.Sprintf("[File: %s]\n%s", f.Name, string(f.Content)),
			})
		}
		body["parts"] = parts
	}
	return body
}

func partsText(parts []Part) string {
	var sb strings.Builder
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			sb.WriteString(p.Text)
			sb.WriteString("\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

func envSlice(env map[string]string) []string {
	var out []string
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	return out
}

// partStreamer converts SSE events and final parts into StreamChunks,
// tracking which part ids have already been emitted so the blocking response
// does not duplicate live streamed text.
type partStreamer struct {
	ch     chan<- backend.StreamChunk
	mu     sync.Mutex
	seen   map[string]bool
	length map[string]int
}

func newPartStreamer(ch chan<- backend.StreamChunk) *partStreamer {
	return &partStreamer{
		ch:     ch,
		seen:   make(map[string]bool),
		length: make(map[string]int),
	}
}

func (s *partStreamer) handleEvent(ctx context.Context, a *Adapter, rec *sessionRec, ev Event) {
	switch ev.Type {
	case "message.part.updated":
		part := extractPart(ev.Raw)
		if part == nil || part.Type != "text" {
			return
		}
		s.mu.Lock()
		last := s.length[part.ID]
		s.seen[part.ID] = true
		s.mu.Unlock()

		if part.Text == "" || len(part.Text) <= last {
			return
		}
		delta := part.Text[last:]
		s.mu.Lock()
		s.length[part.ID] = len(part.Text)
		s.mu.Unlock()

		select {
		case s.ch <- backend.StreamChunk{Content: delta}:
		case <-ctx.Done():
		}
	case "message.part.removed":
		part := extractPart(ev.Raw)
		if part == nil {
			return
		}
		s.mu.Lock()
		delete(s.seen, part.ID)
		delete(s.length, part.ID)
		s.mu.Unlock()
	case "permission.asked":
		req := extractPermission(ev.Raw)
		if req == nil {
			return
		}
		decision := a.permissionDecision(ctx, rec, req)
		response := "reject"
		if decision.Response == "allow" {
			response = "once"
			if decision.Remember {
				response = "always"
			}
		}
		select {
		case s.ch <- backend.StreamChunk{Status: fmt.Sprintf("🔐 %s %s", req.Permission, strings.Join(req.Patterns, ", "))}:
		default:
		}
		if err := a.client.RespondPermission(ctx, rec.externalID, req.PermissionID, response); err != nil {
			a.logger.Error().Err(err).Str("permission", req.Permission).Msg("respond permission")
		}
	}
}

func (s *partStreamer) flushParts(parts []Part) {
	for _, p := range parts {
		if p.Type != "text" || p.Text == "" {
			continue
		}
		s.mu.Lock()
		alreadyStreamed := s.seen[p.ID]
		if !alreadyStreamed {
			s.seen[p.ID] = true
		}
		last := s.length[p.ID]
		s.mu.Unlock()

		delta := p.Text[last:]
		if len(delta) == 0 {
			continue
		}
		s.ch <- backend.StreamChunk{Content: delta}
	}
}

// permissionDecision resolves the permission policy for a request. In "ask"
// mode the per-message handler (if any) is consulted; it may block while a
// human answers.
func (a *Adapter) permissionDecision(ctx context.Context, rec *sessionRec, req *permissionReq) backend.PermissionDecision {
	switch a.permission {
	case "deny":
		return backend.PermissionDecision{Response: "deny"}
	case "ask":
		a.mu.RLock()
		handler := a.handlers[rec.externalID]
		a.mu.RUnlock()
		if handler == nil {
			return backend.PermissionDecision{Response: "deny"}
		}
		return handler(ctx, backend.PermissionRequest{
			SessionID:    rec.sessionID,
			PermissionID: req.PermissionID,
			Permission:   req.Permission,
			Patterns:     req.Patterns,
		})
	case "auto":
		return backend.PermissionDecision{Response: "allow", Remember: true}
	default:
		return backend.PermissionDecision{Response: "deny"}
	}
}

type permissionReq struct {
	PermissionID string
	Permission   string
	Patterns     []string
}

// extractPermission parses a permission.asked event payload, accepting both
// the bare and the GlobalEvent-wrapped shape. The permission id is the "per_"
// request id nested under "properties".
func extractPermission(raw json.RawMessage) *permissionReq {
	var envelope struct {
		Properties *struct {
			ID         string   `json:"id"`
			Permission string   `json:"permission"`
			Patterns   []string `json:"patterns"`
			Action     string   `json:"action"`
			Resources  []string `json:"resources"`
		} `json:"properties"`
		ID         string          `json:"id"`
		Permission string          `json:"permission"`
		Patterns   []string        `json:"patterns"`
		Action     string          `json:"action"`
		Resources  []string        `json:"resources"`
		Payload    json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}

	// Nested properties carry the per_ request id.
	if envelope.Properties != nil && envelope.Properties.ID != "" {
		return buildPermissionReq(envelope.Properties.ID, envelope.Properties.Permission, envelope.Properties.Patterns, envelope.Properties.Action, envelope.Properties.Resources)
	}
	// Recurse into a GlobalEvent payload.
	if len(envelope.Payload) > 0 {
		return extractPermission(envelope.Payload)
	}
	// Fall back to top-level fields.
	if envelope.ID != "" {
		return buildPermissionReq(envelope.ID, envelope.Permission, envelope.Patterns, envelope.Action, envelope.Resources)
	}
	return nil
}

func buildPermissionReq(id, permission string, patterns []string, action string, resources []string) *permissionReq {
	if permission == "" {
		permission = action
	}
	if len(patterns) == 0 {
		patterns = resources
	}
	return &permissionReq{PermissionID: id, Permission: permission, Patterns: patterns}
}

// extractPart parses the part payload of a message.part.* event, which is
// nested under "properties".
func extractPart(raw json.RawMessage) *Part {
	var envelope struct {
		Properties *struct {
			Part *Part `json:"part"`
		} `json:"properties"`
		Part    *Part           `json:"part"`
		Payload json.RawMessage `json:"payload"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil
	}
	if envelope.Properties != nil && envelope.Properties.Part != nil {
		return envelope.Properties.Part
	}
	if envelope.Part != nil {
		return envelope.Part
	}
	if len(envelope.Payload) > 0 {
		var inner struct {
			Properties *struct {
				Part *Part `json:"part"`
			} `json:"properties"`
			Part *Part `json:"part"`
		}
		if err := json.Unmarshal(envelope.Payload, &inner); err == nil {
			if inner.Properties != nil && inner.Properties.Part != nil {
				return inner.Properties.Part
			}
			if inner.Part != nil {
				return inner.Part
			}
		}
	}
	return nil
}
