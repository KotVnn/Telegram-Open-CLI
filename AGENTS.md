# AGENTS.md

## Project Overview

TOC (Telegram OpenCode CLI) is a Go CLI tool that allows controlling AI Coding Agents (OpenCode, Claude Code, Aider, Gemini CLI) through Telegram.

**Goal:** Users install once, run `toc init` then `toc start`, and can control AI coding agents remotely via Telegram.

**Key Design Principle:** Abstract layer between Telegram Bot and AI Backend. No hard dependency on any specific backend.

## Tech Stack

| Component | Technology | Version |
|-----------|-----------|---------|
| Language | Go | 1.22+ |
| CLI Framework | github.com/spf13/cobra | v1.8+ |
| Configuration | github.com/spf13/viper + TOML | v1.18+ |
| Telegram | github.com/go-telegram/bot | v1.19+ |
| Storage | SQLite via gorm.io/gorm + glebarez/sqlite | v1.5+ |
| Logging | github.com/rs/zerolog | v1.32+ |
| Testing | github.com/stretchr/testify | v1.9+ |
| Errors | github.com/cockroachdb/errors | v1.11+ |

## Project Structure

```
toc/
├── cmd/toc/main.go                  # Entry point - minimal, just calls internal/app
├── internal/
│   ├── app/
│   │   ├── app.go                   # Application struct, dependency injection
│   │   ├── lifecycle.go             # Start/Stop orchestration
│   │   └── eventbus.go              # Internal event system
│   ├── config/
│   │   ├── config.go                # Config struct definition
│   │   ├── loader.go                # Load from file, env, flags
│   │   └── schema.go                # Validation rules
│   ├── backend/
│   │   ├── backend.go               # Backend interface definition
│   │   ├── manager.go               # Backend registry & selection
│   │   ├── session.go               # Session management logic
│   │   ├── opencode/
│   │   │   ├── adapter.go           # OpenCode adapter implementation
│   │   │   └── adapter_test.go
│   │   ├── claude/
│   │   │   ├── adapter.go           # Claude Code adapter
│   │   │   └── adapter_test.go
│   │   ├── aider/
│   │   │   ├── adapter.go           # Aider adapter
│   │   │   └── adapter_test.go
│   │   └── gemini/
│   │       ├── adapter.go           # Gemini CLI adapter
│   │       └── adapter_test.go
│   ├── telegram/
│   │   ├── bot.go                   # Bot initialization & lifecycle
│   │   ├── handlers.go              # Command & message handlers
│   │   ├── middleware.go             # Telegram-specific middleware
│   │   └── keyboards.go             # Keyboard builders
│   ├── storage/
│   │   ├── storage.go               # Storage interface
│   │   ├── sqlite.go                # SQLite implementation
│   │   └── migrations.go            # Schema migrations
│   ├── user/
│   │   ├── user.go                  # User management
│   │   └── permission.go            # RBAC permission system
│   ├── project/
│   │   ├── project.go               # Project management
│   │   └── workspace.go             # Workspace management
│   └── plugin/
│       ├── plugin.go                # Plugin interface
│       └── registry.go              # Plugin discovery & loading
├── pkg/
│   ├── version/
│   │   └── version.go               # Build version info (ldflags)
│   └── errors/
│       └── errors.go                # Custom error types
├── docs/                            # Documentation
│   ├── interfaces.md
│   ├── data-models.md
│   ├── backend-adapter.md
│   ├── telegram-adapter.md
│   ├── middleware.md
│   ├── setup.md
│   ├── testing.md
│   └── adr/                         # Architecture Decision Records
├── configs/
│   └── example.toml                 # Example configuration
├── examples/
│   └── basic/                       # Usage examples
├── scripts/
│   ├── build.sh                     # Build script
│   └── install.sh                   # Install script
├── .github/
│   └── workflows/
│       ├── ci.yml                   # CI pipeline
│       └── release.yml              # Release automation
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── CONTRIBUTING.md
├── CHANGELOG.md
├── ROADMAP.md
├── CODE_OF_CONDUCT.md
├── SECURITY.md
└── LICENSE
```

## Build & Test Commands

```bash
# Build
go build -o toc.exe ./cmd/toc          # Windows
go build -o toc ./cmd/toc              # Linux/macOS

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out       # View coverage report

# Run specific package tests
go test ./internal/backend/...
go test -run TestSessionManager ./internal/backend/

# Lint (install golangci-lint first)
golangci-lint run

# Format
gofmt -s -w .
goimports -w .

# Vet
go vet ./...

# Clean
go clean
```

## Code Style Rules

1. **Formatting:** Always run `gofmt -s -w .` before committing
2. **Imports:** Group: stdlib, external, internal. Use `goimports`.
3. **Errors:** Always wrap with context: `fmt.Errorf("create session: %w", err)`
4. **Context:** Pass `context.Context` as first parameter in all functions
5. **Interfaces:** Define in consumer package, not provider package
6. **Naming:**
   - Unexported: `camelCase`
   - Exported: `PascalCase`
   - Interfaces: `-er` suffix when single method (e.g., `Reader`), or descriptive name
   - Avoid `Base`, `Common`, `Util`, `Helper` in names
7. **Tests:** Table-driven tests preferred. Use `testify/assert` and `testify/require`.
8. **Comments:** Only on exported types/functions. Use `// FunctionName does X` format.

## Key Interfaces to Implement

**Import path:** `github.com/KotVnn/Telegram-Open-CLI/internal/...`

### Backend (internal/backend/backend.go)

This is the CORE interface. Every AI backend must implement it.

```go
type Backend interface {
    // Metadata
    Name() string
    Description() string
    Version() string

    // Lifecycle
    Initialize(ctx context.Context, config BackendConfig) error
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health(ctx context.Context) error

    // Session management
    CreateSession(ctx context.Context, opts SessionOpts) (*Session, error)
    GetSession(ctx context.Context, id string) (*Session, error)
    ListSessions(ctx context.Context, projectID string) ([]*Session, error)
    DeleteSession(ctx context.Context, id string) error

    // Message handling
    SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)
    StreamMessage(ctx context.Context, req *SendMessageRequest) (<-chan StreamChunk, error)

    // Capabilities
    Capabilities() *Capabilities
}
```

### Storage (internal/storage/storage.go)

```go
type Storage interface {
    // Lifecycle
    Migrate(ctx context.Context) error
    Close() error

    // Sessions
    SaveSession(ctx context.Context, session *Session) error
    GetSession(ctx context.Context, id string) (*Session, error)
    ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error)
    UpdateSession(ctx context.Context, session *Session) error
    DeleteSession(ctx context.Context, id string) error

    // Messages
    SaveMessage(ctx context.Context, msg *Message) error
    GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*Message, error)

    // Users
    SaveUser(ctx context.Context, user *User) error
    GetUser(ctx context.Context, id int64) (*User, error)
    ListUsers(ctx context.Context) ([]*User, error)
    DeleteUser(ctx context.Context, id int64) error

    // Projects
    SaveProject(ctx context.Context, project *Project) error
    GetProject(ctx context.Context, id string) (*Project, error)
    ListProjects(ctx context.Context, userID int64) ([]*Project, error)
    DeleteProject(ctx context.Context, id string) error
}
```

### Telegram Adapter (internal/telegram/bot.go)

```go
type Adapter interface {
    // Lifecycle
    Start(ctx context.Context) error
    Stop(ctx context.Context) error

    // Message operations
    SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error
    EditMessage(ctx context.Context, chatID int64, messageID int, text string) error
    SendDocument(ctx context.Context, chatID int64, doc Document) error
    AnswerCallback(ctx context.Context, callbackID string, text string) error

    // Handler registration
    HandleCommand(cmd string, handler HandlerFunc)
    HandleMessage(pattern string, handler HandlerFunc)
    HandleCallback(pattern string, handler CallbackHandlerFunc)
    HandleDefault(handler HandlerFunc)

    // Middleware
    Use(middlewares ...Middleware)
}
```

## Configuration

Config file: `~/.toc/config.toml`

```toml
version = "1"

default_backend = "opencode"

[telegram]
token = ""                          # Required: Your bot token from @BotFather

[storage]
path = ""                           # Default: ~/.toc/toc.db

[backends.opencode]
enabled = true
command = "opencode"
args = ["run", "--format", "json"]
timeout = "30m"

[backends.claude]
enabled = false
command = "claude"
args = ["-p"]
timeout = "30m"

[backends.aider]
enabled = false
command = "aider"
args = ["--yes"]
timeout = "30m"

[backends.gemini]
enabled = false
command = "gemini"
timeout = "30m"

[security]
require_auth = true
max_sessions_per_user = 5

[logging]
level = "info"
format = "json"
output = "stdout"
```

## Implementation Order

Follow this order when implementing from scratch:

1. `pkg/version/version.go` - Version info
2. `pkg/errors/errors.go` - Custom error types
3. `internal/config/` - Config loading and validation
4. `internal/storage/storage.go` - Storage interface
5. `internal/storage/sqlite.go` - SQLite implementation
6. `internal/storage/migrations.go` - Database schema
7. `internal/backend/backend.go` - Backend interface
8. `internal/backend/manager.go` - Backend registry
9. `internal/backend/opencode/` - First backend adapter
10. `internal/telegram/bot.go` - Telegram bot core
11. `internal/telegram/handlers.go` - Message handlers
12. `internal/telegram/keyboards.go` - Keyboard builders
13. `internal/telegram/middleware.go` - Middleware
14. `internal/user/` - User management
15. `internal/project/` - Project management
16. `internal/app/app.go` - Application wiring
17. `internal/app/lifecycle.go` - Startup/shutdown
18. `cmd/toc/main.go` - Entry point
19. Remaining backends (claude, aider, gemini)
20. `internal/plugin/` - Plugin system

## Testing Guidelines

- **Unit tests:** Every package must have tests
- **Coverage goal:** > 80% for business logic
- **Mocking:** Mock interfaces, not concrete types
- **Table tests:** Use for multiple input/output scenarios
- **Integration tests:** Use `//go:build integration` tag
- **E2E tests:** Use `//go:build e2e` tag

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Run integration tests
go test -tags=integration ./...

# Generate coverage
go test -coverprofile=coverage.out ./...
```

## Common Patterns

### Error Handling
```go
// Always wrap errors with context
func (s *Storage) GetSession(ctx context.Context, id string) (*Session, error) {
    var session Session
    if err := s.db.Where("id = ?", id).First(&session).Error; err != nil {
        return nil, fmt.Errorf("get session %s: %w", id, err)
    }
    return &session, nil
}
```

### Context Values
```go
// Store values in context
ctx = context.WithValue(ctx, sessionKey, session)
ctx = context.WithValue(ctx, userKey, user)

// Retrieve values from context
session := ctx.Value(sessionKey).(*Session)
```

### Functional Options
```go
type Option func(*Config)

func WithBackend(name string) Option {
    return func(c *Config) {
        c.DefaultBackend = name
    }
}

func NewConfig(opts ...Option) *Config {
    cfg := &Config{ /* defaults */ }
    for _, opt := range opts {
        opt(cfg)
    }
    return cfg
}
```
