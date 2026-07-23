# Data Models

This document defines all data structures used in TOC. These models are used across the application for storage, API communication, and internal state.

## Table of Contents

- [Session Models](#session-models)
- [Message Models](#message-models)
- [User Models](#user-models)
- [Project Models](#project-models)
- [Configuration Models](#configuration-models)
- [Event Models](#event-models)

---

## Session Models

### Session (Storage)

The primary session record stored in the database.

```go
package storage

import "time"

// Session represents a coding session stored in the database.
type Session struct {
    // ID is the unique identifier (UUID format).
    ID string `json:"id" gorm:"primaryKey;type:varchar(36)"`

    // Backend is the backend name (e.g., "opencode").
    Backend string `json:"backend" gorm:"index;type:varchar(50)"`

    // ProjectID links to the parent project.
    ProjectID string `json:"project_id" gorm:"index;type:varchar(36)"`

    // Title is a human-readable name for the session.
    Title string `json:"title" gorm:"type:varchar(255)"`

    // Status indicates the current state.
    Status SessionStatus `json:"status" gorm:"index;type:varchar(20)"`

    // Model is the AI model being used (e.g., "sonnet").
    Model string `json:"model" gorm:"type:varchar(100)"`

    // Agent is the agent mode (e.g., "build", "plan").
    Agent string `json:"agent" gorm:"type:varchar(50)"`

    // WorkingDir is the backend's working directory.
    WorkingDir string `json:"working_dir" gorm:"type:varchar(500)"`

    // CreatedAt is the creation timestamp.
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

    // UpdatedAt is the last update timestamp.
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`

    // ClosedAt is when the session was closed (nil if active).
    ClosedAt *time.Time `json:"closed_at,omitempty"`
}

// SessionStatus represents the state of a session.
type SessionStatus string

const (
    // SessionStatusActive means the session is currently in use.
    SessionStatusActive SessionStatus = "active"

    // SessionStatusIdle means the session exists but is not in use.
    SessionStatusIdle SessionStatus = "idle"

    // SessionStatusError means the session encountered an error.
    SessionStatusError SessionStatus = "error"

    // SessionStatusClosed means the session has been closed.
    SessionStatusClosed SessionStatus = "closed"
)

// String returns the string representation of the status.
func (s SessionStatus) String() string {
    return string(s)
}

// IsActive returns true if the session is in an active state.
func (s SessionStatus) IsActive() bool {
    return s == SessionStatusActive || s == SessionStatusIdle
}
```

### Session (Backend)

The session object used by backends (in-memory representation).

```go
package backend

import "time"

// Session represents a coding session with an AI backend.
type Session struct {
    // ID is the unique identifier.
    ID string

    // Backend is the backend name.
    Backend string

    // ProjectID is the parent project ID.
    ProjectID string

    // Title is a human-readable name.
    Title string

    // Status is the current state.
    Status SessionStatus

    // Model is the AI model in use.
    Model string

    // Agent is the agent mode in use.
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

// SessionStatus represents the state of a session.
type SessionStatus string

const (
    SessionStatusActive SessionStatus = "active"
    SessionStatusIdle   SessionStatus = "idle"
    SessionStatusError  SessionStatus = "error"
    SessionStatusClosed SessionStatus = "closed"
)
```

### SessionFilter

Filters for querying sessions.

```go
package storage

// SessionFilter holds filters for listing sessions.
type SessionFilter struct {
    // ProjectID filters sessions by project.
    ProjectID string

    // Status filters sessions by status.
    Status SessionStatus

    // Backend filters sessions by backend.
    Backend string

    // Limit is the maximum number of results (0 = no limit).
    Limit int

    // Offset is the number of results to skip.
    Offset int

    // OrderBy is the field to sort by (default: "updated_at DESC").
    OrderBy string
}
```

### SessionOpts

Options for creating a new session.

```go
package backend

// SessionOpts holds options for creating a session.
type SessionOpts struct {
    // ProjectID is the parent project ID.
    ProjectID string

    // Title is a human-readable name.
    Title string

    // Model specifies which AI model to use.
    Model string

    // Agent specifies which agent mode to use.
    Agent string

    // WorkingDir is the backend's working directory.
    WorkingDir string

    // Metadata holds additional session configuration.
    Metadata map[string]interface{}
}
```

---

## Message Models

### Message (Storage)

Messages stored in the database.

```go
package storage

import "time"

// Message represents a message in a session.
type Message struct {
    // ID is the unique identifier (UUID format).
    ID string `json:"id" gorm:"primaryKey;type:varchar(36)"`

    // SessionID links to the parent session.
    SessionID string `json:"session_id" gorm:"index;type:varchar(36)"`

    // Role indicates who sent the message.
    Role MessageRole `json:"role" gorm:"type:varchar(20)"`

    // Content is the message text.
    Content string `json:"content" gorm:"type:text"`

    // Files is JSON-serialized file attachments.
    Files string `json:"files,omitempty" gorm:"type:text"`

    // Tokens is JSON-serialized token usage.
    Tokens string `json:"tokens,omitempty" gorm:"type:text"`

    // CreatedAt is the creation timestamp.
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`
}

// MessageRole indicates who sent the message.
type MessageRole string

const (
    // MessageRoleUser is a message from the user.
    MessageRoleUser MessageRole = "user"

    // MessageRoleAssistant is a response from the AI.
    MessageRoleAssistant MessageRole = "assistant"

    // MessageRoleSystem is a system message.
    MessageRoleSystem MessageRole = "system"
)
```

### Message (Backend)

Messages used by backends.

```go
package backend

// Message represents a message to send to a backend.
type Message struct {
    // ID is the unique identifier.
    ID string

    // Content is the message text.
    Content string

    // Files are file attachments.
    Files []FileAttachment

    // Metadata holds additional message data.
    Metadata map[string]interface{}
}

// FileAttachment represents a file to include with a message.
type FileAttachment struct {
    // Name is the filename.
    Name string

    // Content is the file content.
    Content []byte

    // MIMEType is the file's MIME type.
    MIMEType string
}
```

### Response Models

```go
package backend

// SendMessageRequest holds the request for sending a message.
type SendMessageRequest struct {
    // SessionID is the target session.
    SessionID string

    // Content is the message text.
    Content string

    // Files are file attachments.
    Files []FileAttachment

    // Model optionally overrides the session's model.
    Model string

    // Agent optionally overrides the session's agent.
    Agent string

    // Metadata holds additional request data.
    Metadata map[string]interface{}
}

// SendMessageResponse holds the complete response.
type SendMessageResponse struct {
    // ID is the response identifier.
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

// StreamChunk holds a chunk of streaming response data.
type StreamChunk struct {
    // Content is the chunk text.
    Content string

    // Done indicates if this is the final chunk.
    Done bool

    // Error holds any error that occurred.
    Error error
}

// FileChange represents a file modification.
type FileChange struct {
    // Path is the file path.
    Path string

    // Action indicates what was done.
    Action FileAction

    // Content is the new file content.
    Content string

    // Diff is the change diff.
    Diff string
}

// FileAction indicates what action was taken on a file.
type FileAction string

const (
    FileActionCreate FileAction = "create"
    FileActionModify FileAction = "modify"
    FileActionDelete FileAction = "delete"
)

// TokenUsage tracks token consumption.
type TokenUsage struct {
    // Input is the number of input tokens.
    Input int

    // Output is the number of output tokens.
    Output int

    // Total is the total token count.
    Total int
}
```

---

## User Models

### User (Storage)

```go
package storage

import "time"

// User represents a Telegram user.
type User struct {
    // ID is the Telegram user ID.
    ID int64 `json:"id" gorm:"primaryKey"`

    // Username is the Telegram username.
    Username string `json:"username" gorm:"index;type:varchar(100)"`

    // FirstName is the user's first name.
    FirstName string `json:"first_name" gorm:"type:varchar(100)"`

    // LastName is the user's last name.
    LastName string `json:"last_name" gorm:"type:varchar(100)"`

    // Role is the user's permission level.
    Role UserRole `json:"role" gorm:"type:varchar(20);default:user"`

    // AllowedBackends is JSON-serialized list of allowed backends.
    AllowedBackends string `json:"allowed_backends" gorm:"type:text"`

    // IsActive indicates if the user account is active.
    IsActive bool `json:"is_active" gorm:"default:true"`

    // LastActiveAt is when the user last interacted.
    LastActiveAt *time.Time `json:"last_active_at,omitempty"`

    // CreatedAt is the creation timestamp.
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

    // UpdatedAt is the last update timestamp.
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}

// UserRole represents the user's permission level.
type UserRole string

const (
    // UserRoleAdmin has full access.
    UserRoleAdmin UserRole = "admin"

    // UserRoleUser has standard access.
    UserRoleUser UserRole = "user"

    // UserRoleViewer has read-only access.
    UserRoleViewer UserRole = "viewer"
)
```

### User (Application)

```go
package user

// User represents a user in the application layer.
type User struct {
    ID              int64
    Username        string
    FirstName       string
    LastName        string
    Role            UserRole
    AllowedBackends []string
    IsActive        bool
}

// UserRole represents the user's permission level.
type UserRole string

const (
    UserRoleAdmin  UserRole = "admin"
    UserRoleUser   UserRole = "user"
    UserRoleViewer UserRole = "viewer"
)

// HasPermission checks if the user has the given permission.
func (u *User) HasPermission(permission Permission) bool {
    switch u.Role {
    case UserRoleAdmin:
        return true
    case UserRoleUser:
        return permission != PermissionManageUsers && permission != PermissionManageSystem
    case UserRoleViewer:
        return permission == PermissionViewSessions || permission == PermissionViewProjects
    default:
        return false
    }
}

// Permission represents a specific permission.
type Permission string

const (
    PermissionCreateSession  Permission = "session:create"
    PermissionDeleteSession  Permission = "session:delete"
    PermissionViewSessions   Permission = "session:view"
    PermissionCreateProject  Permission = "project:create"
    PermissionDeleteProject  Permission = "project:delete"
    PermissionViewProjects   Permission = "project:view"
    PermissionManageUsers    Permission = "users:manage"
    PermissionManageSystem   Permission = "system:manage"
)
```

---

## Project Models

### Project (Storage)

```go
package storage

import "time"

// Project represents a coding project.
type Project struct {
    // ID is the unique identifier (UUID format).
    ID string `json:"id" gorm:"primaryKey;type:varchar(36)"`

    // Name is the project name.
    Name string `json:"name" gorm:"type:varchar(255)"`

    // Path is the project directory path.
    Path string `json:"path" gorm:"type:varchar(500)"`

    // Description is an optional project description.
    Description string `json:"description,omitempty" gorm:"type:text"`

    // DefaultBackend is the preferred backend for this project.
    DefaultBackend string `json:"default_backend" gorm:"type:varchar(50)"`

    // DefaultModel is the preferred model for this project.
    DefaultModel string `json:"default_model,omitempty" gorm:"type:varchar(100)"`

    // OwnerID is the Telegram user ID of the project owner.
    OwnerID int64 `json:"owner_id" gorm:"index"`

    // IsActive indicates if the project is active.
    IsActive bool `json:"is_active" gorm:"default:true"`

    // CreatedAt is the creation timestamp.
    CreatedAt time.Time `json:"created_at" gorm:"autoCreateTime"`

    // UpdatedAt is the last update timestamp.
    UpdatedAt time.Time `json:"updated_at" gorm:"autoUpdateTime"`
}
```

### Workspace (Future)

```go
package project

// Workspace represents a workspace within a project.
type Workspace struct {
    ID        string
    ProjectID string
    Name      string
    Path      string
    IsActive  bool
    CreatedAt time.Time
    UpdatedAt time.Time
}
```

---

## Configuration Models

### Config (Main)

```go
package config

import "time"

// Config is the root configuration structure.
type Config struct {
    // Version is the config file version.
    Version string `toml:"version"`

    // DefaultBackend is the default backend name.
    DefaultBackend string `toml:"default_backend"`

    // Telegram holds Telegram bot configuration.
    Telegram TelegramConfig `toml:"telegram"`

    // Storage holds storage configuration.
    Storage StorageConfig `toml:"storage"`

    // Backends holds backend-specific configurations.
    Backends map[string]BackendConfig `toml:"backends"`

    // Security holds security configuration.
    Security SecurityConfig `toml:"security"`

    // Logging holds logging configuration.
    Logging LoggingConfig `toml:"logging"`
}

// TelegramConfig holds Telegram bot settings.
type TelegramConfig struct {
    // Token is the bot token from @BotFather.
    Token string `toml:"token"`

    // AllowedUsers is a list of user IDs allowed to use the bot.
    // Empty means all users are allowed.
    AllowedUsers []int64 `toml:"allowed_users"`

    // AllowedGroups is a list of group chat IDs allowed.
    // Empty means all groups are allowed.
    AllowedGroups []int64 `toml:"allowed_groups"`

    // WebhookURL is the webhook URL (empty = use polling).
    WebhookURL string `toml:"webhook_url,omitempty"`

    // WebhookSecret is the webhook secret token.
    WebhookSecret string `toml:"webhook_secret,omitempty"`
}

// StorageConfig holds storage settings.
type StorageConfig struct {
    // Path is the database file path.
    // Empty defaults to ~/.toc/toc.db
    Path string `toml:"path"`
}

// BackendConfig holds backend-specific settings.
type BackendConfig struct {
    // Enabled indicates if this backend is available.
    Enabled bool `toml:"enabled"`

    // Command is the executable name or path.
    Command string `toml:"command"`

    // Args are additional command-line arguments.
    Args []string `toml:"args"`

    // Environment holds environment variables.
    Environment map[string]string `toml:"environment"`

    // Timeout is the maximum time for operations (e.g., "30m").
    Timeout string `toml:"timeout"`
}

// SecurityConfig holds security settings.
type SecurityConfig struct {
    // RequireAuth indicates if authentication is required.
    RequireAuth bool `toml:"require_auth"`

    // AdminUsers is a list of admin user IDs.
    AdminUsers []int64 `toml:"admin_users"`

    // MaxSessionsPerUser limits concurrent sessions per user.
    MaxSessionsPerUser int `toml:"max_sessions_per_user"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
    // Level is the log level (debug, info, warn, error).
    Level string `toml:"level"`

    // Format is the log format (json, console).
    Format string `toml:"format"`

    // Output is the log destination (stdout, stderr, or file path).
    Output string `toml:"output"`
}

// Timeout returns the parsed timeout duration.
func (b BackendConfig) TimeoutDuration() time.Duration {
    if b.Timeout == "" {
        return 30 * time.Minute
    }
    d, err := time.ParseDuration(b.Timeout)
    if err != nil {
        return 30 * time.Minute
    }
    return d
}
```

---

## Event Models

### Event

```go
package app

import "time"

// Event represents an application event.
type Event struct {
    // Type is the event type (e.g., "message.received").
    Type EventType

    // Source identifies where the event originated.
    Source string

    // Payload is the event data.
    Payload interface{}

    // Timestamp is when the event occurred.
    Timestamp time.Time
}

// EventType represents the type of event.
type EventType string

const (
    // Message events
    EventMessageReceived  EventType = "message.received"
    EventMessageSent      EventType = "message.sent"
    EventMessageEdited    EventType = "message.edited"

    // Session events
    EventSessionCreated   EventType = "session.created"
    EventSessionClosed    EventType = "session.closed"
    EventSessionError     EventType = "session.error"

    // Response events
    EventResponseChunk    EventType = "response.chunk"
    EventResponseComplete EventType = "response.complete"

    // User events
    EventUserJoined       EventType = "user.joined"
    EventUserLeft         EventType = "user.left"
    EventUserUpdated      EventType = "user.updated"

    // System events
    EventError            EventType = "error.occurred"
    EventStartup          EventType = "system.startup"
    EventShutdown         EventType = "system.shutdown"
)

// EventPayloads

// MessageReceivedEvent is emitted when a message is received.
type MessageReceivedEvent struct {
    ChatID    int64
    UserID    int64
    Username  string
    MessageID int
    Content   string
    Command   string
}

// SessionCreatedEvent is emitted when a session is created.
type SessionCreatedEvent struct {
    SessionID string
    ProjectID string
    Backend   string
    UserID    int64
}

// ResponseChunkEvent is emitted during streaming responses.
type ResponseChunkEvent struct {
    SessionID string
    Chunk     string
    Done      bool
}

// ErrorEvent is emitted when an error occurs.
type ErrorEvent struct {
    Err       error
    Component string
    Operation string
    Severity  string
}
```
