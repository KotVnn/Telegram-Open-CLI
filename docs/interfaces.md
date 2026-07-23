# Interface Definitions

This document defines all core interfaces for TOC. These interfaces establish the contracts between components and enable testability through mocking.

## Table of Contents

- [Backend Interface](#backend-interface)
- [Storage Interface](#storage-interface)
- [Telegram Adapter Interface](#telegram-adapter-interface)
- [Plugin Interface](#plugin-interface)
- [Middleware Types](#middleware-types)

---

## Backend Interface

The Backend interface is the core abstraction for AI coding agent integration. Every backend must implement this interface.

**Location:** `internal/backend/backend.go`

```go
package backend

import (
    "context"
    "time"
)

// Backend defines the interface for AI coding agent backends.
// Each AI coding agent (OpenCode, Claude Code, Aider, Gemini CLI) must implement this interface.
type Backend interface {
    // Metadata

    // Name returns the backend identifier (e.g., "opencode", "claude").
    // This is used for configuration and user-facing display.
    Name() string

    // Description returns a human-readable description of the backend.
    Description() string

    // Version returns the backend version string.
    Version() string

    // Lifecycle

    // Initialize sets up the backend with the provided configuration.
    // This is called once during application startup.
    Initialize(ctx context.Context, config BackendConfig) error

    // Start activates the backend.
    // For backends that run persistent processes, this starts them.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the backend.
    // This should clean up all resources.
    Stop(ctx context.Context) error

    // Health checks if the backend is available and healthy.
    // Returns nil if healthy, error otherwise.
    Health(ctx context.Context) error

    // Session Management

    // CreateSession creates a new coding session with the backend.
    // Each session represents an isolated conversation context.
    CreateSession(ctx context.Context, opts SessionOpts) (*Session, error)

    // GetSession retrieves a session by ID.
    // Returns ErrSessionNotFound if the session doesn't exist.
    GetSession(ctx context.Context, id string) (*Session, error)

    // ListSessions returns all sessions for the given project.
    ListSessions(ctx context.Context, projectID string) ([]*Session, error)

    // DeleteSession removes a session and its associated data.
    DeleteSession(ctx context.Context, id string) error

    // Message Handling

    // SendMessage sends a message and waits for the complete response.
    // Use StreamMessage for streaming responses.
    SendMessage(ctx context.Context, req *SendMessageRequest) (*SendMessageResponse, error)

    // StreamMessage sends a message and returns a channel of response chunks.
    // The channel is closed when the response is complete or an error occurs.
    StreamMessage(ctx context.Context, req *SendMessageRequest) (<-chan StreamChunk, error)

    // Capabilities

    // Capabilities returns the backend's capabilities and limitations.
    Capabilities() *Capabilities
}

// BackendConfig holds backend-specific configuration.
// This is loaded from the config file and passed to Initialize.
type BackendConfig struct {
    // Enabled indicates if this backend is available for use.
    Enabled bool `toml:"enabled"`

    // Command is the executable name or path (e.g., "opencode", "claude").
    Command string `toml:"command"`

    // Args are additional command-line arguments.
    Args []string `toml:"args"`

    // WorkingDir is the default working directory for the backend.
    WorkingDir string `toml:"working_dir"`

    // Environment holds environment variables to set when running the backend.
    Environment map[string]string `toml:"environment"`

    // Timeout is the maximum time to wait for backend operations.
    Timeout time.Duration `toml:"timeout"`
}

// SessionOpts holds options for creating a new session.
type SessionOpts struct {
    // ProjectID is the ID of the project this session belongs to.
    ProjectID string

    // Title is a human-readable title for the session.
    Title string

    // Model specifies which AI model to use (e.g., "sonnet", "gpt-4").
    Model string

    // Agent specifies which agent mode to use (e.g., "build", "plan").
    Agent string

    // WorkingDir is the directory where the backend operates.
    WorkingDir string

    // Metadata holds additional session-specific data.
    Metadata map[string]interface{}
}

// Session represents a coding session with an AI backend.
type Session struct {
    // ID is the unique identifier for this session.
    ID string

    // Backend is the name of the backend (e.g., "opencode").
    Backend string

    // ProjectID is the ID of the project this session belongs to.
    ProjectID string

    // Title is a human-readable title.
    Title string

    // Status indicates the current state of the session.
    Status SessionStatus

    // Model is the AI model being used.
    Model string

    // Agent is the agent mode being used.
    Agent string

    // WorkingDir is the backend's working directory.
    WorkingDir string

    // CreatedAt is when the session was created.
    CreatedAt time.Time

    // UpdatedAt is when the session was last updated.
    UpdatedAt time.Time

    // Metadata holds additional session data.
    Metadata map[string]interface{}
}

// SessionStatus represents the current state of a session.
type SessionStatus string

const (
    SessionStatusActive SessionStatus = "active"
    SessionStatusIdle   SessionStatus = "idle"
    SessionStatusError  SessionStatus = "error"
    SessionStatusClosed SessionStatus = "closed"
)

// SendMessageRequest holds the request data for sending a message.
type SendMessageRequest struct {
    // SessionID is the target session.
    SessionID string

    // Content is the message text.
    Content string

    // Files are file attachments to include.
    Files []FileAttachment

    // Model optionally overrides the session's model for this message.
    Model string

    // Agent optionally overrides the session's agent for this message.
    Agent string

    // Metadata holds additional request data.
    Metadata map[string]interface{}
}

// FileAttachment represents a file to include with a message.
type FileAttachment struct {
    // Name is the filename.
    Name string

    // Content is the file content as bytes.
    Content []byte

    // MIMEType is the file's MIME type (e.g., "text/plain").
    MIMEType string
}

// SendMessageResponse holds the complete response from a backend.
type SendMessageResponse struct {
    // ID is the unique identifier for this response.
    ID string

    // Content is the response text.
    Content string

    // Files are file changes made by the backend.
    Files []FileChange

    // Tokens tracks token usage.
    Tokens TokenUsage

    // Metadata holds additional response data.
    Metadata map[string]interface{}
}

// FileChange represents a file modification made by the backend.
type FileChange struct {
    // Path is the relative file path.
    Path string

    // Action indicates what was done to the file.
    Action FileAction

    // Content is the new file content (for create/modify).
    Content string

    // Diff is the change diff (for modify).
    Diff string
}

// FileAction indicates what action was taken on a file.
type FileAction string

const (
    FileActionCreate FileAction = "create"
    FileActionModify FileAction = "modify"
    FileActionDelete FileAction = "delete"
)

// StreamChunk holds a chunk of streaming response data.
type StreamChunk struct {
    // Content is the chunk text.
    Content string

    // Done indicates if this is the final chunk.
    Done bool

    // Error holds any error that occurred during streaming.
    Error error
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
    // Input is the number of input tokens.
    Input int

    // Output is the number of output tokens.
    Output int

    // Total is the total token count.
    Total int
}

// Capabilities describes what a backend supports.
// Use this to determine if a feature is available.
type Capabilities struct {
    // SupportsStreaming indicates if the backend supports streaming responses.
    SupportsStreaming bool

    // SupportsFiles indicates if the backend can handle file attachments.
    SupportsFiles bool

    // SupportsMultiModal indicates if the backend can process images/audio.
    SupportsMultiModal bool

    // SupportsToolCalling indicates if the backend supports tool calling.
    SupportsToolCalling bool

    // MaxTokens is the maximum context window size.
    MaxTokens int

    // SupportedModels lists available model identifiers.
    SupportedModels []string

    // SupportedAgents lists available agent modes.
    SupportedAgents []string
}
```

---

## Storage Interface

The Storage interface abstracts data persistence. The default implementation uses SQLite via GORM.

**Location:** `internal/storage/storage.go`

```go
package storage

import (
    "context"
    "time"
)

// Storage defines the interface for data persistence.
// All storage operations must be context-aware for cancellation support.
type Storage interface {
    // Lifecycle

    // Migrate runs database migrations to set up or update the schema.
    // This should be called once during application startup.
    Migrate(ctx context.Context) error

    // Close closes the storage connection and releases resources.
    Close() error

    // Session Operations

    // SaveSession creates or updates a session.
    SaveSession(ctx context.Context, session *Session) error

    // GetSession retrieves a session by ID.
    // Returns ErrSessionNotFound if not found.
    GetSession(ctx context.Context, id string) (*Session, error)

    // ListSessions returns sessions matching the filter.
    ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error)

    // UpdateSession updates an existing session.
    UpdateSession(ctx context.Context, session *Session) error

    // DeleteSession removes a session by ID.
    DeleteSession(ctx context.Context, id string) error

    // Message Operations

    // SaveMessage stores a message.
    SaveMessage(ctx context.Context, msg *Message) error

    // GetMessages returns messages for a session, ordered by creation time.
    // limit controls the maximum number of messages returned.
    // offset is the number of messages to skip (for pagination).
    GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*Message, error)

    // User Operations

    // SaveUser creates or updates a user.
    SaveUser(ctx context.Context, user *User) error

    // GetUser retrieves a user by Telegram ID.
    GetUser(ctx context.Context, id int64) (*User, error)

    // ListUsers returns all users.
    ListUsers(ctx context.Context) ([]*User, error)

    // DeleteUser removes a user by ID.
    DeleteUser(ctx context.Context, id int64) error

    // Project Operations

    // SaveProject creates or updates a project.
    SaveProject(ctx context.Context, project *Project) error

    // GetProject retrieves a project by ID.
    GetProject(ctx context.Context, id string) (*Project, error)

    // ListProjects returns projects for a user.
    ListProjects(ctx context.Context, userID int64) ([]*Project, error)

    // DeleteProject removes a project by ID.
    DeleteProject(ctx context.Context, id string) error
}

// SessionFilter holds filters for listing sessions.
type SessionFilter struct {
    // ProjectID filters by project.
    ProjectID string

    // Status filters by session status.
    Status SessionStatus

    // Backend filters by backend.
    Backend string

    // Limit is the maximum number of results.
    Limit int

    // Offset is the number of results to skip.
    Offset int

    // OrderBy is the field to sort by (default: "updated_at DESC").
    OrderBy string
}

// Session represents a stored session record.
type Session struct {
    ID          string        `json:"id" gorm:"primaryKey"`
    Backend     string        `json:"backend" gorm:"index"`
    ProjectID   string        `json:"project_id" gorm:"index"`
    Title       string        `json:"title"`
    Status      SessionStatus `json:"status" gorm:"index"`
    Model       string        `json:"model"`
    Agent       string        `json:"agent"`
    WorkingDir  string        `json:"working_dir"`
    CreatedAt   time.Time     `json:"created_at"`
    UpdatedAt   time.Time     `json:"updated_at"`
}

// SessionStatus represents the state of a session.
type SessionStatus string

const (
    SessionStatusActive SessionStatus = "active"
    SessionStatusIdle   SessionStatus = "idle"
    SessionStatusError  SessionStatus = "error"
    SessionStatusClosed SessionStatus = "closed"
)

// Message represents a stored message record.
type Message struct {
    ID        string      `json:"id" gorm:"primaryKey"`
    SessionID string      `json:"session_id" gorm:"index"`
    Role      MessageRole `json:"role"`
    Content   string      `json:"content" gorm:"type:text"`
    Files     string      `json:"files,omitempty" gorm:"type:text"` // JSON serialized
    CreatedAt time.Time   `json:"created_at"`
}

// MessageRole indicates who sent the message.
type MessageRole string

const (
    MessageRoleUser      MessageRole = "user"
    MessageRoleAssistant MessageRole = "assistant"
    MessageRoleSystem    MessageRole = "system"
)

// User represents a Telegram user.
type User struct {
    ID              int64     `json:"id" gorm:"primaryKey"`
    Username        string    `json:"username" gorm:"index"`
    FirstName       string    `json:"first_name"`
    LastName        string    `json:"last_name"`
    Role            UserRole  `json:"role" gorm:"default:user"`
    AllowedBackends string    `json:"allowed_backends" gorm:"type:text"` // JSON serialized
    IsActive        bool      `json:"is_active" gorm:"default:true"`
    CreatedAt       time.Time `json:"created_at"`
    UpdatedAt       time.Time `json:"updated_at"`
}

// UserRole represents the user's permission level.
type UserRole string

const (
    UserRoleAdmin  UserRole = "admin"
    UserRoleUser   UserRole = "user"
    UserRoleViewer UserRole = "viewer"
)

// Project represents a coding project.
type Project struct {
    ID             string    `json:"id" gorm:"primaryKey"`
    Name           string    `json:"name"`
    Path           string    `json:"path"`
    Description    string    `json:"description,omitempty"`
    DefaultBackend string    `json:"default_backend"`
    DefaultModel   string    `json:"default_model,omitempty"`
    OwnerID        int64     `json:"owner_id" gorm:"index"`
    IsActive       bool      `json:"is_active" gorm:"default:true"`
    CreatedAt      time.Time `json:"created_at"`
    UpdatedAt      time.Time `json:"updated_at"`
}
```

---

## Telegram Adapter Interface

The Telegram Adapter interface wraps the Telegram Bot API library.

**Location:** `internal/telegram/bot.go`

```go
package telegram

import (
    "context"
)

// Adapter defines the interface for Telegram bot operations.
type Adapter interface {
    // Lifecycle

    // Start begins receiving updates from Telegram.
    // This blocks until ctx is canceled or an error occurs.
    Start(ctx context.Context) error

    // Stop gracefully shuts down the bot.
    Stop(ctx context.Context) error

    // Message Operations

    // SendMessage sends a text message to a chat.
    SendMessage(ctx context.Context, chatID int64, msg OutgoingMessage) error

    // EditMessage edits an existing message.
    EditMessage(ctx context.Context, chatID int64, messageID int, text string) error

    // SendDocument sends a file to a chat.
    SendDocument(ctx context.Context, chatID int64, doc Document) error

    // AnswerCallback answers a callback query from an inline button.
    AnswerCallback(ctx context.Context, callbackID string, text string) error

    // Handler Registration

    // HandleCommand registers a handler for a specific command.
    // The command should not include the leading slash.
    HandleCommand(cmd string, handler HandlerFunc)

    // HandleMessage registers a handler for messages matching a pattern.
    HandleMessage(pattern string, handler HandlerFunc)

    // HandleCallback registers a handler for callback queries.
    HandleCallback(pattern string, handler CallbackHandlerFunc)

    // HandleDefault registers a handler for unmatched messages.
    HandleDefault(handler HandlerFunc)

    // Middleware

    // Use adds middleware to the handler chain.
    Use(middlewares ...Middleware)
}

// OutgoingMessage holds data for messages sent by the bot.
type OutgoingMessage struct {
    // Text is the message content.
    Text string

    // ParseMode is the formatting mode ("HTML", "Markdown", "MarkdownV2", or "").
    ParseMode string

    // ReplyMarkup is optional keyboard markup.
    ReplyMarkup interface{}

    // DisableWebPreview disables link previews.
    DisableWebPreview bool

    // DisableNotification sends silently.
    DisableNotification bool
}

// Document holds file data for sending.
type Document struct {
    // FileID is the Telegram file ID (for existing files).
    FileID string

    // FileName is the display name of the file.
    FileName string

    // Content is the file content (for new uploads).
    Content []byte

    // Caption is optional text to include with the file.
    Caption string
}

// IncomingMessage holds parsed incoming message data.
type IncomingMessage struct {
    // MessageID is the Telegram message ID.
    MessageID int

    // ChatID is the chat where the message was sent.
    ChatID int64

    // FromID is the sender's user ID.
    FromID int64

    // FromUsername is the sender's username.
    FromUsername string

    // FromFirstName is the sender's first name.
    FromFirstName string

    // FromLastName is the sender's last name.
    FromLastName string

    // Text is the message content.
    Text string

    // Command is the parsed command (e.g., "new" for /new).
    Command string

    // Args are the command arguments.
    Args []string

    // ReplyToMessageID is the ID of the message being replied to.
    ReplyToMessageID int

    // Document is attached document (if any).
    Document *DocumentInfo
}

// DocumentInfo holds metadata about an attached document.
type DocumentInfo struct {
    FileID   string
    FileName string
    MimeType string
    FileSize int64
}

// CallbackQuery holds callback query data from inline buttons.
type CallbackQuery struct {
    // ID is the callback query ID.
    ID string

    // ChatID is the chat context.
    ChatID int64

    // FromID is the user who clicked the button.
    FromID int64

    // FromUsername is the user's username.
    FromUsername string

    // Data is the callback data string.
    Data string

    // MessageID is the message containing the inline keyboard.
    MessageID int
}

// HandlerFunc handles incoming messages.
type HandlerFunc func(ctx context.Context, msg *IncomingMessage) error

// CallbackHandlerFunc handles callback queries.
type CallbackHandlerFunc func(ctx context.Context, cb *CallbackQuery) error

// Middleware processes messages before/after handlers.
type Middleware func(next HandlerFunc) HandlerFunc
```

---

## Plugin Interface

The Plugin interface allows extending TOC with additional functionality.

**Location:** `internal/plugin/plugin.go`

```go
package plugin

import (
    "context"
)

// Plugin defines the interface for TOC plugins.
// Plugins can add commands, middleware, and event handlers.
type Plugin interface {
    // Metadata

    // Name returns the unique plugin identifier.
    Name() string

    // Version returns the plugin version (semver recommended).
    Version() string

    // Description returns a human-readable description.
    Description() string

    // Author returns the plugin author's name.
    Author() string

    // Lifecycle

    // Initialize sets up the plugin with required dependencies.
    // Called once during application startup.
    Initialize(ctx context.Context, deps *Dependencies) error

    // Start activates the plugin.
    // Called after all plugins are initialized.
    Start(ctx context.Context) error

    // Stop deactivates the plugin and cleans up resources.
    Stop(ctx context.Context) error

    // Extensions

    // Commands returns additional CLI commands provided by the plugin.
    Commands() []*Command

    // Middleware returns middleware for message processing.
    Middleware() []*Middleware

    // EventHandlers returns handlers for application events.
    EventHandlers() map[string]EventHandler
}

// Dependencies holds dependencies injected into plugins.
type Dependencies struct {
    // Config provides access to application configuration.
    Config interface{}

    // Storage provides access to data persistence.
    Storage interface{}

    // EventBus allows subscribing to and publishing events.
    EventBus interface{}

    // Logger provides structured logging.
    Logger interface{}

    // BackendManager provides access to AI backends.
    BackendManager interface{}

    // TelegramAdapter provides access to the Telegram bot.
    TelegramAdapter interface{}
}

// Command represents a CLI command provided by a plugin.
type Command struct {
    // Use is the command name (e.g., "my-plugin").
    Use string

    // Short is a short description.
    Short string

    // Long is a detailed description.
    Long string

    // Aliases are alternative command names.
    Aliases []string

    // Run is the command handler.
    Run func(ctx context.Context, args []string) error
}

// Middleware represents message processing middleware.
type Middleware struct {
    // Name identifies the middleware.
    Name string

    // Priority determines execution order (lower = earlier).
    Priority int

    // Handler is the middleware function.
    Handler func(next interface{}) interface{}
}

// EventHandler processes application events.
type EventHandler func(ctx context.Context, event *Event) error

// Event represents an application event.
type Event struct {
    // Type is the event type (e.g., "message.received").
    Type string

    // Source identifies where the event originated.
    Source string

    // Payload is the event data.
    Payload interface{}
}
```

---

## Middleware Types

Middleware provides a way to process messages before and after handlers.

**Location:** `internal/telegram/middleware.go`

```go
package telegram

import "context"

// Middleware is a function that wraps a handler.
type Middleware func(next HandlerFunc) HandlerFunc

// Common middleware types:

// AuthMiddleware checks user authorization.
type AuthMiddleware struct {
    // AllowedUsers is a list of allowed user IDs (empty = allow all).
    AllowedUsers []int64

    // AllowedChats is a list of allowed chat IDs (empty = allow all).
    AllowedChats []int64
}

// RateLimitMiddleware limits request frequency.
type RateLimitMiddleware struct {
    // RequestsPerSecond is the maximum requests per second.
    RequestsPerSecond float64

    // Burst is the maximum burst size.
    Burst int
}

// LoggingMiddleware logs all messages.
type LoggingMiddleware struct {
    // Logger is the zerolog logger instance.
    Logger interface{}

    // LogContent indicates if message content should be logged.
    LogContent bool
}

// RecoveryMiddleware catches panics.
type RecoveryMiddleware struct {
    // Logger is the zerolog logger instance.
    Logger interface{}

    // Handler is called when a panic is recovered.
    Handler func(ctx context.Context, r interface{}) error
}
```

---

## Error Types

Shared error types used across interfaces.

**Location:** `pkg/errors/errors.go`

```go
package errors

import "errors"

var (
    // ErrSessionNotFound is returned when a session doesn't exist.
    ErrSessionNotFound = errors.New("session not found")

    // ErrProjectNotFound is returned when a project doesn't exist.
    ErrProjectNotFound = errors.New("project not found")

    // ErrUserNotFound is returned when a user doesn't exist.
    ErrUserNotFound = errors.New("user not found")

    // ErrUnauthorized is returned when a user is not authorized.
    ErrUnauthorized = errors.New("unauthorized")

    // ErrForbidden is returned when a user doesn't have permission.
    ErrForbidden = errors.New("forbidden")

    // ErrBackendUnavailable is returned when a backend is not available.
    ErrBackendUnavailable = errors.New("backend unavailable")

    // ErrBackendTimeout is returned when a backend operation times out.
    ErrBackendTimeout = errors.New("backend timeout")

    // ErrRateLimited is returned when rate limit is exceeded.
    ErrRateLimited = errors.New("rate limited")

    // ErrInvalidConfig is returned when configuration is invalid.
    ErrInvalidConfig = errors.New("invalid configuration")

    // ErrStorageUnavailable is returned when storage is not available.
    ErrStorageUnavailable = errors.New("storage unavailable")

    // ErrPluginLoadFailed is returned when a plugin fails to load.
    ErrPluginLoadFailed = errors.New("plugin load failed")
)
```
