# Testing Strategy

This guide explains the testing approach and best practices for TOC.

## Table of Contents

- [Overview](#overview)
- [Test Types](#test-types)
- [Test Organization](#test-organization)
- [Writing Tests](#writing-tests)
- [Mocking](#mocking)
- [Test Utilities](#test-utilities)
- [Running Tests](#running-tests)
- [Coverage](#coverage)

---

## Overview

TOC uses a multi-level testing approach:

| Level | Purpose | Scope | Speed |
|-------|---------|-------|-------|
| Unit | Test individual functions | Single package | Fast |
| Integration | Test component interaction | Multiple packages | Medium |
| E2E | Test complete workflows | Full application | Slow |

**Goals:**
- > 80% coverage for business logic
- All public APIs tested
- Critical paths covered by integration tests

---

## Test Types

### Unit Tests

Test individual functions and methods in isolation.

```go
package backend

import (
    "context"
    "testing"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestAdapter_Name(t *testing.T) {
    adapter := New()
    assert.Equal(t, "opencode", adapter.Name())
}

func TestAdapter_CreateSession(t *testing.T) {
    adapter := setupTestAdapter(t)

    session, err := adapter.CreateSession(context.Background(), SessionOpts{
        Title: "Test Session",
    })

    require.NoError(t, err)
    assert.NotEmpty(t, session.ID)
    assert.Equal(t, "Test Session", session.Title)
}
```

### Integration Tests

Test interaction between components.

```go
//go:build integration

package integration

import (
    "context"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/KotVnn/Telegram-Open-CLI/internal/backend"
    "github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

func TestSessionWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup
    db := setupTestDB(t)
    adapter := backend.New()

    // Create session
    session, err := adapter.CreateSession(context.Background(), backend.SessionOpts{
        Title: "Integration Test",
    })
    require.NoError(t, err)

    // Save to storage
    err = db.SaveSession(context.Background(), &storage.Session{
        ID:    session.ID,
        Title: session.Title,
    })
    require.NoError(t, err)

    // Retrieve from storage
    stored, err := db.GetSession(context.Background(), session.ID)
    require.NoError(t, err)
    assert.Equal(t, session.Title, stored.Title)
}
```

### E2E Tests

Test complete user workflows.

```go
//go:build e2e

package e2e

import (
    "testing"
    "time"

    "github.com/stretchr/testify/require"
)

func TestNewSessionWorkflow(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping e2e test")
    }

    // Start bot
    bot := startTestBot(t)
    defer bot.Stop()

    // Send /new command
    response, err := bot.SendMessage("/new test-project")
    require.NoError(t, err)

    // Verify response
    assert.Contains(t, response, "Created session")

    // Send a message
    response, err = bot.SendMessage("Hello, AI!")
    require.NoError(t, err)

    // Wait for response
    time.Sleep(5 * time.Second)

    // Verify session exists
    sessions, err := bot.ListSessions()
    require.NoError(t, err)
    assert.Len(t, sessions, 1)
}
```

---

## Test Organization

### File Structure

```
toc/
├── internal/
│   ├── backend/
│   │   ├── backend.go
│   │   ├── backend_test.go          # Unit tests
│   │   ├── manager.go
│   │   ├── manager_test.go          # Unit tests
│   │   ├── opencode/
│   │   │   ├── adapter.go
│   │   │   ├── adapter_test.go      # Unit tests
│   │   │   └── adapter_e2e_test.go  # E2E tests
│   │   └── integration_test.go      # Integration tests
│   └── storage/
│       ├── storage.go
│       ├── sqlite.go
│       ├── sqlite_test.go           # Unit tests
│       └── integration_test.go      # Integration tests
├── test/
│   ├── fixtures/                    # Test data
│   ├── helpers/                     # Test utilities
│   └── e2e/                         # E2E test suite
```

### Naming Conventions

```go
// Unit test: TestFunctionName
func TestAdapter_CreateSession(t *testing.T) { }

// Table test: TestFunctionName_Scenario
func TestAdapter_CreateSession_WithModel(t *testing.T) { }

// Error test: TestFunctionName_Error
func TestAdapter_CreateSession_InvalidConfig(t *testing.T) { }

// Integration test: TestWorkflow
func TestSessionWorkflow(t *testing.T) { }
```

---

## Writing Tests

### Table-Driven Tests

```go
func TestSessionStatus_String(t *testing.T) {
    tests := []struct {
        name     string
        status   SessionStatus
        expected string
    }{
        {
            name:     "active",
            status:   SessionStatusActive,
            expected: "active",
        },
        {
            name:     "idle",
            status:   SessionStatusIdle,
            expected: "idle",
        },
        {
            name:     "error",
            status:   SessionStatusError,
            expected: "error",
        },
        {
            name:     "closed",
            status:   SessionStatusClosed,
            expected: "closed",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := tt.status.String()
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Error Testing

```go
func TestAdapter_GetSession_NotFound(t *testing.T) {
    adapter := setupTestAdapter(t)

    _, err := adapter.GetSession(context.Background(), "nonexistent")

    require.Error(t, err)
    assert.ErrorIs(t, err, ErrSessionNotFound)
    assert.Contains(t, err.Error(), "nonexistent")
}

func TestAdapter_Initialize_InvalidConfig(t *testing.T) {
    adapter := New()

    err := adapter.Initialize(context.Background(), BackendConfig{
        Enabled: true,
        Command: "", // Missing command
    })

    require.Error(t, err)
    assert.Contains(t, err.Error(), "requires a command")
}
```

### Context Testing

```go
func TestAdapter_CreateSession_WithCancel(t *testing.T) {
    adapter := setupTestAdapter(t)

    ctx, cancel := context.WithCancel(context.Background())
    cancel() // Cancel immediately

    _, err := adapter.CreateSession(ctx, SessionOpts{
        Title: "Cancelled Session",
    })

    // Should handle context cancellation gracefully
    assert.Error(t, err)
    assert.Equal(t, context.Canceled, err)
}
```

---

## Mocking

### Mock Interface

```go
// MockBackend is a mock implementation of the Backend interface.
type MockBackend struct {
    // Function fields for custom behavior
    NameFn            func() string
    InitializeFn      func(ctx context.Context, config BackendConfig) error
    CreateSessionFn   func(ctx context.Context, opts SessionOpts) (*Session, error)
    SendMessageFn     func(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
    StreamMessageFn   func(ctx context.Context, req *SendMessageRequest) (<-chan StreamChunk, error)
    CapabilitiesFn    func() *Capabilities

    // Call tracking
    CreateSessionCalls []SessionOpts
    SendMessageCalls   []*SendMessageRequest
}

func (m *MockBackend) Name() string {
    if m.NameFn != nil {
        return m.NameFn()
    }
    return "mock"
}

func (m *MockBackend) Initialize(ctx context.Context, config BackendConfig) error {
    if m.InitializeFn != nil {
        return m.InitializeFn(ctx, config)
    }
    return nil
}

func (m *MockBackend) CreateSession(ctx context.Context, opts SessionOpts) (*Session, error) {
    m.CreateSessionCalls = append(m.CreateSessionCalls, opts)
    if m.CreateSessionFn != nil {
        return m.CreateSessionFn(ctx, opts)
    }
    return &Session{
        ID:    "mock-session",
        Title: opts.Title,
    }, nil
}

func (m *MockBackend) SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error) {
    m.SendMessageCalls = append(m.SendMessageCalls, req)
    if m.SendMessageFn != nil {
        return m.SendMessageFn(ctx, req)
    }
    return &SendMessageResponse{
        Content: "mock response",
    }, nil
}

func (m *MockBackend) Capabilities() *Capabilities {
    if m.CapabilitiesFn != nil {
        return m.CapabilitiesFn()
    }
    return &Capabilities{
        SupportsStreaming: true,
        SupportsFiles:     true,
    }
}
```

### Using Mocks

```go
func TestSessionManager_Create(t *testing.T) {
    backend := &MockBackend{
        CreateSessionFn: func(ctx context.Context, opts SessionOpts) (*Session, error) {
            return &Session{
                ID:    "test-123",
                Title: opts.Title,
            }, nil
        },
    }

    mgr := NewSessionManager(backend)
    session, err := mgr.Create(context.Background(), SessionOpts{
        Title: "Test",
    })

    require.NoError(t, err)
    assert.Equal(t, "test-123", session.ID)

    // Verify mock was called
    assert.Len(t, backend.CreateSessionCalls, 1)
    assert.Equal(t, "Test", backend.CreateSessionCalls[0].Title)
}
```

---

## Test Utilities

### Helper Functions

```go
package testutil

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "github.com/stretchr/testify/require"
    "github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

// SetupTestDB creates an in-memory SQLite database for testing.
func SetupTestDB(t *testing.T) storage.Storage {
    t.Helper()

    db, err := storage.NewSQLite(":memory:")
    require.NoError(t, err)

    require.NoError(t, db.Migrate(context.Background()))

    t.Cleanup(func() {
        db.Close()
    })

    return db
}

// SetupTestDir creates a temporary directory for testing.
func SetupTestDir(t *testing.T) string {
    t.Helper()

    dir, err := os.MkdirTemp("", "toc-test-*")
    require.NoError(t, err)

    t.Cleanup(func() {
        os.RemoveAll(dir)
    })

    return dir
}

// CreateTestUser creates a test user in the database.
func CreateTestUser(t *testing.T, db storage.Storage) *storage.User {
    t.Helper()

    user := &storage.User{
        ID:       12345,
        Username: "testuser",
        Role:     storage.UserRoleAdmin,
    }

    require.NoError(t, db.SaveUser(context.Background(), user))
    return user
}

// CreateTestSession creates a test session in the backend.
func CreateTestSession(t *testing.T, backend backend.Backend) *backend.Session {
    t.Helper()

    session, err := backend.CreateSession(context.Background(), backend.SessionOpts{
        Title: "Test Session",
    })
    require.NoError(t, err)

    return session
}
```

### Test Fixtures

```go
package fixtures

// LoadTestConfig returns a test configuration.
func LoadTestConfig() *config.Config {
    return &config.Config{
        Version:        "1",
        DefaultBackend: "opencode",
        Telegram: config.TelegramConfig{
            Token: "test-token",
        },
        Storage: config.StorageConfig{
            Path: ":memory:",
        },
        Backends: map[string]config.BackendConfig{
            "opencode": {
                Enabled: true,
                Command: "echo",
            },
        },
        Logging: config.LoggingConfig{
            Level:  "debug",
            Format: "console",
        },
    }
}
```

---

## Running Tests

### Basic Commands

```bash
# All tests
go test ./...

# Specific package
go test ./internal/backend/...

# With verbose output
go test -v ./...

# With race detection
go test -race ./...

# With timeout
go test -timeout 30s ./...

# Run specific test
go test -run TestAdapter_CreateSession ./internal/backend/...

# Run tests matching pattern
go test -run "Test.*Session" ./...
```

### Integration Tests

```bash
# Run integration tests
go test -tags=integration ./...

# Run specific integration test
go test -tags=integration -run TestSessionWorkflow ./internal/...
```

### E2E Tests

```bash
# Run E2E tests
go test -tags=e2e ./test/e2e/...

# With longer timeout
go test -tags=e2e -timeout 5m ./test/e2e/...
```

### Using Makefile

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run E2E tests
make test-e2e

# Run with coverage
make test-cover
```

---

## Coverage

### Generate Coverage

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...

# View coverage in browser
go tool cover -html=coverage.out

# View coverage summary
go tool cover -func=coverage.out

# Generate coverage for specific package
go test -coverprofile=coverage.out ./internal/backend/...
```

### Coverage Goals

| Package Type | Target |
|-------------|--------|
| Business Logic | > 80% |
| Adapters | > 70% |
| Utilities | > 90% |
| CLI Commands | > 60% |

### Coverage Report

```bash
# Generate and view coverage
go test -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -1

# Output example:
# github.com/KotVnn/Telegram-Open-CLI/internal/backend: 85.2%
```

### Coverage in CI

```yaml
# .github/workflows/ci.yml
- name: Test with coverage
  run: go test -coverprofile=coverage.out -covermode=atomic ./...

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```
