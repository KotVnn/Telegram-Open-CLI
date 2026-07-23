package storage

import (
	"context"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Storage defines the interface for data persistence.
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

// NewSQLiteDB creates a new SQLite GORM DB instance.
func NewSQLiteDB(path string) (*gorm.DB, error) {
	return gorm.Open(sqlite.Open(path), &gorm.Config{})
}

// SessionFilter holds filters for listing sessions.
type SessionFilter struct {
	ProjectID string
	Status    SessionStatus
	Backend   string
	Limit     int
	Offset    int
	OrderBy   string
}

// Session represents a stored session record.
type Session struct {
	ID         string        `json:"id" gorm:"primaryKey"`
	Backend    string        `json:"backend" gorm:"index"`
	ProjectID  string        `json:"project_id" gorm:"index"`
	Title      string        `json:"title"`
	Status     SessionStatus `json:"status" gorm:"index"`
	Model      string        `json:"model"`
	Agent      string        `json:"agent"`
	WorkingDir string        `json:"working_dir"`
	CreatedAt  time.Time     `json:"created_at"`
	UpdatedAt  time.Time     `json:"updated_at"`
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
	Files     string      `json:"files,omitempty" gorm:"type:text"`
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
	AllowedBackends string    `json:"allowed_backends" gorm:"type:text"`
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
