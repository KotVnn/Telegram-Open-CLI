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
	Content    string
	Files      []FileAttachment
	Model      string
	Agent      string
	WorkingDir string
	Metadata   map[string]interface{}
}

// FileAttachment represents a file to include with a message.
type FileAttachment struct {
	Name    string
	Content []byte
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
}

// TokenUsage tracks token consumption for a request.
type TokenUsage struct {
	Input  int
	Output int
	Total  int
}

// Capabilities describes what a backend supports.
type Capabilities struct {
	SupportsStreaming    bool
	SupportsFiles        bool
	SupportsMultiModal   bool
	SupportsToolCalling  bool
	MaxTokens            int
	SupportedModels      []string
	SupportedAgents      []string
}
