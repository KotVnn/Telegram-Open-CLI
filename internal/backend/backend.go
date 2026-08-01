package backend

import (
	"context"
	"time"
)

// Backend defines the interface for AI coding agent backends.
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

// BackendConfig holds backend-specific configuration.
type BackendConfig struct {
	Enabled     bool              `toml:"enabled"`
	Command     string            `toml:"command"`
	Args        []string          `toml:"args"`
	WorkingDir  string            `toml:"working_dir"`
	Environment map[string]string `toml:"environment"`
	Timeout     time.Duration     `toml:"timeout"`

	// Serve-specific settings (used by backends that talk to a headless server).
	Hostname    string `toml:"hostname"`
	Port        int    `toml:"port"`
	Password    string `toml:"password"`
	AutoRestart bool   `toml:"auto_restart"`
	Permission  string `toml:"permission"` // "ask", "auto", "deny"
}

// SessionOpts holds options for creating a new session.
type SessionOpts struct {
	ProjectID  string
	Title      string
	Model      string
	Agent      string
	WorkingDir string
	Metadata   map[string]interface{}
}

// Session represents a coding session with an AI backend.
type Session struct {
	ID         string
	Backend    string
	ProjectID  string
	Title      string
	Status     SessionStatus
	Model      string
	Agent      string
	WorkingDir string
	ExternalID string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Metadata   map[string]interface{}
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
	SessionID  string
	ExternalID string
	Content    string
	Files      []FileAttachment
	Model      string
	Agent      string
	WorkingDir string
	Metadata   map[string]interface{}

	// PermissionHandler, when set, is invoked whenever the backend requests
	// permission to perform an action during this message. It may block while
	// waiting for a human decision. When nil the backend applies its own
	// configured policy.
	PermissionHandler PermissionHandler
}

// FileAttachment represents a file to include with a message.
type FileAttachment struct {
	Name     string
	Content  []byte
	MIMEType string
}

// SendMessageResponse holds the complete response from a backend.
type SendMessageResponse struct {
	ID       string
	Content  string
	Files    []FileChange
	Tokens   TokenUsage
	Metadata map[string]interface{}
}

// FileChange represents a file modification made by the backend.
type FileChange struct {
	Path    string
	Action  FileAction
	Content string
	Diff    string
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
	Content string
	Done    bool
	Error   error
	Status  string // optional live status line; not part of the final answer
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	Input  int
	Output int
	Total  int
}

// Capabilities describes what a backend supports.
type Capabilities struct {
	SupportsStreaming   bool
	SupportsFiles       bool
	SupportsMultiModal  bool
	SupportsToolCalling bool
	MaxTokens           int
	SupportedModels     []string
	SupportedAgents     []string
}

// The interfaces below describe optional capabilities. Backends that support
// them are discovered via type assertion so that simple CLI backends are not
// forced to implement every feature.

// Aborter can abort a running message/task in a session.
type Aborter interface {
	Abort(ctx context.Context, sessionID string) error
}

// MessageLister can list the messages that make up a session.
type MessageLister interface {
	ListMessages(ctx context.Context, sessionID string, limit int) ([]*MessageInfo, error)
}

// MessageReverter can revert a message inside a session.
type MessageReverter interface {
	RevertMessage(ctx context.Context, sessionID, messageID string) error
}

// FileBrowser lists files and directories inside the backend's workspace.
type FileBrowser interface {
	ListFiles(ctx context.Context, dir string) ([]*FileEntry, error)
}

// ModelLister enumerates the models and agents available on the backend.
type ModelLister interface {
	ListModels(ctx context.Context) (models []string, agents []string, err error)
}

// SessionLinker resolves the backend's external session identifier for a
// TOC session so callers can persist it.
type SessionLinker interface {
	ExternalSessionID(ctx context.Context, sessionID string) (string, error)
}

// PermissionResponder responds to a pending permission request.
type PermissionResponder interface {
	RespondPermission(ctx context.Context, sessionID, permissionID string, decision PermissionDecision) error
}

// MessageInfo describes a single message in a session.
type MessageInfo struct {
	ID      string
	Role    string
	Content string
	Time    time.Time
}

// FileEntry describes a file or directory in the workspace.
type FileEntry struct {
	Name  string
	Path  string
	IsDir bool
	Size  int64
}

// PermissionRequest describes a permission request raised by the backend.
type PermissionRequest struct {
	SessionID    string
	PermissionID string
	Permission   string
	Patterns     []string
}

// PermissionDecision is how a caller answers a PermissionRequest.
type PermissionDecision struct {
	// Response is one of "allow" or "deny".
	Response string
	// Remember persists the decision for the matching pattern.
	Remember bool
}

// PermissionHandler decides how to answer a permission request. It may block
// while waiting for a human decision.
type PermissionHandler func(ctx context.Context, req PermissionRequest) PermissionDecision
