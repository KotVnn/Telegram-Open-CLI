package aider

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

var (
	ErrBackendDisabled = errors.New("backend disabled")
	ErrMissingCommand  = errors.New("backend requires a command")
	ErrNotInitialized  = errors.New("backend not initialized")
	ErrSessionNotFound = errors.New("session not found")
)

// Adapter implements the backend.Backend interface for Aider.
type Adapter struct {
	config      backend.BackendConfig
	sessions    map[string]*backend.Session
	mu          sync.RWMutex
	initialized bool
}

// New creates a new Adapter instance.
func New() *Adapter {
	return &Adapter{
		sessions: make(map[string]*backend.Session),
	}
}

func (a *Adapter) Name() string        { return "aider" }
func (a *Adapter) Description() string  { return "Aider AI Coding Agent" }
func (a *Adapter) Version() string      { return "1.0.0" }

func (a *Adapter) Initialize(ctx context.Context, config backend.BackendConfig) error {
	if !config.Enabled {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrBackendDisabled)
	}
	if config.Command == "" {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrMissingCommand)
	}
	a.config = config
	a.initialized = true
	return nil
}

func (a *Adapter) Start(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrNotInitialized)
	}
	return nil
}

func (a *Adapter) Stop(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sessions = make(map[string]*backend.Session)
	a.initialized = false
	return nil
}

func (a *Adapter) Health(ctx context.Context) error {
	if !a.initialized {
		return fmt.Errorf("backend %s: %w", a.Name(), ErrNotInitialized)
	}
	_, err := exec.LookPath(a.config.Command)
	if err != nil {
		return fmt.Errorf("command %s not found: %w", a.config.Command, err)
	}
	return nil
}

func (a *Adapter) CreateSession(ctx context.Context, opts backend.SessionOpts) (*backend.Session, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	session := &backend.Session{
		ID:         uuid.New().String(),
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
	a.sessions[session.ID] = session
	return session, nil
}

func (a *Adapter) GetSession(ctx context.Context, id string) (*backend.Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	session, ok := a.sessions[id]
	if !ok {
		return nil, fmt.Errorf("session %s: %w", id, ErrSessionNotFound)
	}
	return session, nil
}

func (a *Adapter) ListSessions(ctx context.Context, projectID string) ([]*backend.Session, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	var sessions []*backend.Session
	for _, s := range a.sessions {
		if projectID == "" || s.ProjectID == projectID {
			sessions = append(sessions, s)
		}
	}
	return sessions, nil
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

func (a *Adapter) SendMessage(ctx context.Context, req *backend.SendMessageRequest) (*backend.SendMessageResponse, error) {
	session, err := a.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	args := a.buildArgs(session, req)
	output, err := a.executeCommand(ctx, args)
	if err != nil {
		return nil, fmt.Errorf("execute command: %w", err)
	}
	return &backend.SendMessageResponse{
		ID:       uuid.New().String(),
		Content:  output,
		Metadata: make(map[string]interface{}),
	}, nil
}

func (a *Adapter) StreamMessage(ctx context.Context, req *backend.SendMessageRequest) (<-chan backend.StreamChunk, error) {
	session, err := a.GetSession(ctx, req.SessionID)
	if err != nil {
		return nil, err
	}
	ch := make(chan backend.StreamChunk, 100)
	go func() {
		defer close(ch)
		args := a.buildArgs(session, req)
		if err := a.executeStreaming(ctx, args, ch); err != nil {
			ch <- backend.StreamChunk{Error: fmt.Errorf("streaming failed: %w", err), Done: true}
		}
	}()
	return ch, nil
}

func (a *Adapter) Capabilities() *backend.Capabilities {
	return &backend.Capabilities{
		SupportsStreaming:   true,
		SupportsFiles:       false,
		SupportsMultiModal:  false,
		SupportsToolCalling: false,
		MaxTokens:           128000,
		SupportedModels:     []string{"default"},
		SupportedAgents:     []string{"default"},
	}
}

func (a *Adapter) executeCommand(ctx context.Context, args []string) (string, error) {
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, a.config.Command, args...)
	if a.config.WorkingDir != "" {
		cmd.Dir = a.config.WorkingDir
	}
	env := os.Environ()
	for k, v := range a.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
	}
	return stdout.String(), nil
}

func (a *Adapter) executeStreaming(ctx context.Context, args []string, ch chan<- backend.StreamChunk) error {
	if a.config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, a.config.Timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, a.config.Command, args...)
	if a.config.WorkingDir != "" {
		cmd.Dir = a.config.WorkingDir
	}
	env := os.Environ()
	for k, v := range a.config.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	cmd.Env = env
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("get stdout pipe: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start command: %w", err)
	}
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // 1MB max line size
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			_ = cmd.Wait()
			return ctx.Err()
		case ch <- backend.StreamChunk{Content: scanner.Text() + "\n", Done: false}:
		}
	}
	if err := scanner.Err(); err != nil {
		_ = cmd.Wait()
		return fmt.Errorf("scanner error: %w", err)
	}
	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("wait command: %w", err)
	}
	ch <- backend.StreamChunk{Done: true}
	return nil
}

func (a *Adapter) buildArgs(session *backend.Session, req *backend.SendMessageRequest) []string {
	args := make([]string, len(a.config.Args))
	copy(args, a.config.Args)
	if req.Model != "" {
		args = append(args, "--model", req.Model)
	}
	args = append(args, "--message", req.Content)
	return args
}
