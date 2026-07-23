# Backend Adapter Guide

This guide explains how to implement and register AI backend adapters in TOC.

## Table of Contents

- [Overview](#overview)
- [Backend Interface](#backend-interface)
- [Creating an Adapter](#creating-an-adapter)
- [Adapter Examples](#adapter-examples)
- [Registration](#registration)
- [Testing](#testing)

---

## Overview

TOC supports multiple AI coding agents through a plugin-like adapter system. Each adapter implements the `Backend` interface and wraps a specific AI coding agent.

```
┌─────────────────────────────────────────────────────────────┐
│                   Backend Manager                           │
│  ┌─────────────────────────────────────────────────────────┐│
│  │                    Backend Interface                     ││
│  └─────────────────────────────────────────────────────────┘│
│       │              │              │              │         │
│  ┌────┴────┐    ┌────┴────┐    ┌────┴────┐    ┌────┴────┐  │
│  │OpenCode │    │ Claude  │    │  Aider  │    │ Gemini  │  │
│  │ Adapter │    │ Adapter │    │ Adapter │    │ Adapter │  │
│  └─────────┘    └─────────┘    └─────────┘    └─────────┘  │
└─────────────────────────────────────────────────────────────┘
```

---

## Backend Interface

Every adapter must implement this interface:

```go
type Backend interface {
    Name() string
    Description() string
    Version() string
    Initialize(ctx context.Context, config BackendConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error
    CreateSession(ctx context.Context, opts SessionOpts) (*Session, error)
    GetSession(ctx context.Context, id string) (*Session, error)
    ListSessions(ctx context.Context, projectID string) ([]*Session, error)
    DeleteSession(ctx context.Context, id string) error
    SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
    StreamMessage(ctx context.Context, req *SendMessageRequest) (<-chan StreamChunk, error)
    Capabilities() *Capabilities
}
```

See [interfaces.md](interfaces.md) for complete interface documentation.

---

## Creating an Adapter

### Step 1: Create Package Structure

Create a new directory under `internal/backend/`:

```
internal/backend/mybackend/
├── adapter.go          # Main adapter implementation
├── adapter_test.go     # Unit tests
├── session.go          # Session management (optional)
├── stream.go           # Streaming implementation (optional)
└── command.go          # Command execution (optional)
```

### Step 2: Define Adapter Struct

```go
package mybackend

import (
    "context"
    "sync"
    "time"

    "github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

// Adapter implements the backend.Backend interface for MyBackend.
type Adapter struct {
    config    backend.BackendConfig
    sessions  map[string]*backend.Session
    mu        sync.RWMutex
    initialized bool
}

// New creates a new Adapter instance.
func New() *Adapter {
    return &Adapter{
        sessions: make(map[string]*backend.Session),
    }
}
```

### Step 3: Implement Metadata Methods

```go
// Name returns the backend identifier.
func (a *Adapter) Name() string {
    return "mybackend"
}

// Description returns a human-readable description.
func (a *Adapter) Description() string {
    return "MyBackend AI Coding Agent"
}

// Version returns the backend version.
func (a *Adapter) Version() string {
    return "1.0.0"
}
```

### Step 4: Implement Lifecycle Methods

```go
// Initialize sets up the backend with configuration.
func (a *Adapter) Initialize(ctx context.Context, config backend.BackendConfig) error {
    if !config.Enabled {
        return fmt.Errorf("backend %s is disabled", a.Name())
    }

    if config.Command == "" {
        return fmt.Errorf("backend %s requires a command", a.Name())
    }

    a.config = config
    a.initialized = true
    return nil
}

// Start activates the backend.
func (a *Adapter) Start(ctx context.Context) error {
    if !a.initialized {
        return fmt.Errorf("backend %s not initialized", a.Name())
    }
    // Start any background processes if needed
    return nil
}

// Stop gracefully shuts down the backend.
func (a *Adapter) Stop(ctx context.Context) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    // Close all active sessions
    for id, session := range a.sessions {
        if session.Status == backend.SessionStatusActive {
            session.Status = backend.SessionStatusClosed
            delete(a.sessions, id)
        }
    }

    a.initialized = false
    return nil
}

// Health checks if the backend is available.
func (a *Adapter) Health(ctx context.Context) error {
    if !a.initialized {
        return fmt.Errorf("backend %s not initialized", a.Name())
    }

    // Check if the command is available
    _, err := exec.LookPath(a.config.Command)
    if err != nil {
        return fmt.Errorf("command %s not found: %w", a.config.Command, err)
    }

    return nil
}
```

### Step 5: Implement Session Methods

```go
// CreateSession creates a new coding session.
func (a *Adapter) CreateSession(ctx context.Context, opts backend.SessionOpts) (*backend.Session, error) {
    a.mu.Lock()
    defer a.mu.Unlock()

    session := &backend.Session{
        ID:         generateID(),
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

// GetSession retrieves a session by ID.
func (a *Adapter) GetSession(ctx context.Context, id string) (*backend.Session, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    session, ok := a.sessions[id]
    if !ok {
        return nil, backend.ErrSessionNotFound
    }

    return session, nil
}

// ListSessions returns all sessions for a project.
func (a *Adapter) ListSessions(ctx context.Context, projectID string) ([]*backend.Session, error) {
    a.mu.RLock()
    defer a.mu.RUnlock()

    var sessions []*backend.Session
    for _, session := range a.sessions {
        if projectID == "" || session.ProjectID == projectID {
            sessions = append(sessions, session)
        }
    }

    return sessions, nil
}

// DeleteSession removes a session.
func (a *Adapter) DeleteSession(ctx context.Context, id string) error {
    a.mu.Lock()
    defer a.mu.Unlock()

    if _, ok := a.sessions[id]; !ok {
        return backend.ErrSessionNotFound
    }

    delete(a.sessions, id)
    return nil
}
```

### Step 6: Implement Message Methods

```go
// SendMessage sends a message and waits for complete response.
func (a *Adapter) SendMessage(ctx context.Context, req *backend.SendMessageRequest) (*backend.SendMessageResponse, error) {
    // Validate session exists
    session, err := a.GetSession(ctx, req.SessionID)
    if err != nil {
        return nil, err
    }

    // Build command arguments
    args := a.buildArgs(session, req)

    // Execute command
    output, err := a.executeCommand(ctx, args)
    if err != nil {
        return nil, fmt.Errorf("execute command: %w", err)
    }

    // Parse response
    response := &backend.SendMessageResponse{
        ID:       generateID(),
        Content:  output,
        Metadata: make(map[string]interface{}),
    }

    return response, nil
}

// StreamMessage sends a message and streams response chunks.
func (a *Adapter) StreamMessage(ctx context.Context, req *backend.SendMessageRequest) (<-chan backend.StreamChunk, error) {
    // Validate session exists
    _, err := a.GetSession(ctx, req.SessionID)
    if err != nil {
        return nil, err
    }

    ch := make(chan backend.StreamChunk, 100)

    go func() {
        defer close(ch)

        // Build command arguments
        args := a.buildArgs(nil, req)

        // Execute command with streaming
        err := a.executeStreaming(ctx, args, ch)
        if err != nil {
            ch <- backend.StreamChunk{
                Error: fmt.Errorf("streaming failed: %w", err),
                Done:  true,
            }
        }
    }()

    return ch, nil
}

// Capabilities returns what this backend supports.
func (a *Adapter) Capabilities() *backend.Capabilities {
    return &backend.Capabilities{
        SupportsStreaming:   true,
        SupportsFiles:       true,
        SupportsMultiModal:  false,
        SupportsToolCalling: false,
        MaxTokens:           100000,
        SupportedModels:     []string{"default"},
        SupportedAgents:     []string{"build", "plan"},
    }
}
```

### Step 7: Implement Command Execution

```go
// executeCommand runs a command and returns the output.
func (a *Adapter) executeCommand(ctx context.Context, args []string) (string, error) {
    cmd := exec.CommandContext(ctx, a.config.Command, args...)

    // Set working directory
    if a.config.WorkingDir != "" {
        cmd.Dir = a.config.WorkingDir
    }

    // Set environment variables
    env := os.Environ()
    for k, v := range a.config.Environment {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    cmd.Env = env

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    // Run command
    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("command failed: %w\nstderr: %s", err, stderr.String())
    }

    return stdout.String(), nil
}

// executeStreaming runs a command and streams output.
func (a *Adapter) executeStreaming(ctx context.Context, args []string, ch chan<- backend.StreamChunk) error {
    cmd := exec.CommandContext(ctx, a.config.Command, args...)

    // Set working directory
    if a.config.WorkingDir != "" {
        cmd.Dir = a.config.WorkingDir
    }

    // Set environment variables
    env := os.Environ()
    for k, v := range a.config.Environment {
        env = append(env, fmt.Sprintf("%s=%s", k, v))
    }
    cmd.Env = env

    // Get stdout pipe
    stdout, err := cmd.StdoutPipe()
    if err != nil {
        return fmt.Errorf("get stdout pipe: %w", err)
    }

    // Start command
    if err := cmd.Start(); err != nil {
        return fmt.Errorf("start command: %w", err)
    }

    // Stream output
    scanner := bufio.NewScanner(stdout)
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case ch <- backend.StreamChunk{
            Content: scanner.Text() + "\n",
            Done:    false,
        }:
        }
    }

    // Wait for command to finish
    if err := cmd.Wait(); err != nil {
        return fmt.Errorf("wait command: %w", err)
    }

    // Send final chunk
    ch <- backend.StreamChunk{Done: true}

    return nil
}

// buildArgs builds command arguments from session and request.
func (a *Adapter) buildArgs(session *backend.Session, req *backend.SendMessageRequest) []string {
    args := []string{}

    // Add configured args
    args = append(args, a.config.Args...)

    // Add request-specific args
    if req.Content != "" {
        args = append(args, req.Content)
    }

    return args
}
```

---

## Adapter Examples

### OpenCode Adapter

OpenCode uses `opencode run` for non-interactive execution:

```go
// internal/backend/opencode/adapter.go

func (a *Adapter) buildArgs(session *backend.Session, req *backend.SendMessageRequest) []string {
    args := []string{"run"}

    // Add format flag for JSON output
    args = append(args, "--format", "json")

    // Add session flag if continuing
    if session != nil && session.ID != "" {
        args = append(args, "--session", session.ID)
    }

    // Add model if specified
    if req.Model != "" {
        args = append(args, "--model", req.Model)
    }

    // Add agent if specified
    if req.Agent != "" {
        args = append(args, "--agent", req.Agent)
    }

    // Add prompt
    args = append(args, req.Content)

    return args
}
```

### Claude Code Adapter

Claude Code uses `-p` for non-interactive mode:

```go
// internal/backend/claude/adapter.go

func (a *Adapter) buildArgs(session *backend.Session, req *backend.SendMessageRequest) []string {
    args := []string{}

    // Add print mode flag
    args = append(args, "-p")

    // Add model if specified
    if req.Model != "" {
        args = append(args, "--model", req.Model)
    }

    // Add prompt
    args = append(args, req.Content)

    return args
}
```

### Aider Adapter

Aider uses `--yes` for auto-accept mode:

```go
// internal/backend/aider/adapter.go

func (a *Adapter) buildArgs(session *backend.Session, req *backend.SendMessageRequest) []string {
    args := []string{}

    // Add yes flag for auto-accept
    args = append(args, "--yes")

    // Add message flag
    args = append(args, "--message", req.Content)

    // Add model if specified
    if req.Model != "" {
        args = append(args, "--model", req.Model)
    }

    return args
}
```

---

## Registration

Register your adapter in `internal/backend/manager.go`:

```go
package backend

import (
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend/opencode"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend/claude"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend/aider"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend/gemini"
)

// Registry holds all registered backend factories.
var registry = make(map[string]func() Backend)

// Register registers a backend factory.
func Register(name string, factory func() Backend) {
    registry[name] = factory
}

// Get returns a new instance of the named backend.
func Get(name string) (Backend, error) {
    factory, ok := registry[name]
    if !ok {
        return nil, fmt.Errorf("unknown backend: %s", name)
    }
    return factory(), nil
}

// List returns all registered backend names.
func List() []string {
    names := make([]string, 0, len(registry))
    for name := range registry {
        names = append(names, name)
    }
    return names
}

func init() {
    Register("opencode", opencode.New)
    Register("claude", claude.New)
    Register("aider", aider.New)
    Register("gemini", gemini.New)
}
```

---

## Testing

### Unit Tests

```go
package mybackend

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

func TestAdapter_Name(t *testing.T) {
    adapter := New()
    assert.Equal(t, "mybackend", adapter.Name())
}

func TestAdapter_Initialize(t *testing.T) {
    adapter := New()
    config := backend.BackendConfig{
        Enabled: true,
        Command: "mybackend-cli",
    }

    err := adapter.Initialize(context.Background(), config)
    require.NoError(t, err)
    assert.True(t, adapter.initialized)
}

func TestAdapter_Initialize_Disabled(t *testing.T) {
    adapter := New()
    config := backend.BackendConfig{
        Enabled: false,
    }

    err := adapter.Initialize(context.Background(), config)
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "disabled")
}

func TestAdapter_CreateSession(t *testing.T) {
    adapter := setupTestAdapter(t)

    session, err := adapter.CreateSession(context.Background(), backend.SessionOpts{
        ProjectID: "test-project",
        Title:     "Test Session",
        Model:     "default",
    })

    require.NoError(t, err)
    assert.NotEmpty(t, session.ID)
    assert.Equal(t, "Test Session", session.Title)
    assert.Equal(t, backend.SessionStatusActive, session.Status)
}

func TestAdapter_GetSession_NotFound(t *testing.T) {
    adapter := setupTestAdapter(t)

    _, err := adapter.GetSession(context.Background(), "nonexistent")
    assert.ErrorIs(t, err, backend.ErrSessionNotFound)
}

func TestAdapter_Capabilities(t *testing.T) {
    adapter := New()
    caps := adapter.Capabilities()

    assert.True(t, caps.SupportsStreaming)
    assert.True(t, caps.SupportsFiles)
    assert.Contains(t, caps.SupportedModels, "default")
}

func setupTestAdapter(t *testing.T) *Adapter {
    t.Helper()

    adapter := New()
    config := backend.BackendConfig{
        Enabled: true,
        Command: "echo", // Use echo for testing
    }

    err := adapter.Initialize(context.Background(), config)
    require.NoError(t, err)

    return adapter
}
```

### Integration Tests

```go
//go:build integration

package mybackend

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend"
)

func TestAdapter_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    adapter := New()
    config := backend.BackendConfig{
        Enabled: true,
        Command: "mybackend-cli",
        Args:    []string{"--test"},
    }

    require.NoError(t, adapter.Initialize(context.Background(), config))
    require.NoError(t, adapter.Start(context.Background()))

    defer adapter.Stop(context.Background())

    // Test health check
    require.NoError(t, adapter.Health(context.Background()))

    // Test session creation
    session, err := adapter.CreateSession(context.Background(), backend.SessionOpts{
        Title: "Integration Test",
    })
    require.NoError(t, err)

    // Test message sending
    resp, err := adapter.SendMessage(context.Background(), &backend.SendMessageRequest{
        SessionID: session.ID,
        Content:   "Hello, world!",
    })
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Content)
}
```
