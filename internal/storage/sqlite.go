package storage

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var allowedOrderBy = map[string]bool{
	"created_at ASC":  true,
	"created_at DESC": true,
	"updated_at ASC":  true,
	"updated_at DESC": true,
	"title ASC":       true,
	"title DESC":      true,
}

// SQLite implements Storage using SQLite via GORM.
type SQLite struct {
	db *gorm.DB
}

// NewSQLite creates a new SQLite storage instance.
func NewSQLite(db *gorm.DB) *SQLite {
	return &SQLite{db: db}
}

func (s *SQLite) Migrate(ctx context.Context) error {
	if err := s.db.AutoMigrate(&Session{}, &Message{}, &User{}, &Project{}); err != nil {
		return fmt.Errorf("migrate database: %w", err)
	}
	return nil
}

func (s *SQLite) Close() error {
	sqlDB, err := s.db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql db: %w", err)
	}
	return sqlDB.Close()
}

func (s *SQLite) SaveSession(ctx context.Context, session *Session) error {
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return fmt.Errorf("save session: %w", err)
	}
	return nil
}

func (s *SQLite) GetSession(ctx context.Context, id string) (*Session, error) {
	var session Session
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrSessionNotFound
		}
		return nil, fmt.Errorf("get session: %w", err)
	}
	return &session, nil
}

func (s *SQLite) ListSessions(ctx context.Context, filter SessionFilter) ([]*Session, error) {
	var sessions []*Session
	query := s.db.WithContext(ctx)

	if filter.ProjectID != "" {
		query = query.Where("project_id = ?", filter.ProjectID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Backend != "" {
		query = query.Where("backend = ?", filter.Backend)
	}

	order := filter.OrderBy
	if order == "" {
		order = "updated_at DESC"
	}
	if !allowedOrderBy[order] {
		order = "updated_at DESC"
	}
	query = query.Order(order)

	if filter.Limit > 0 {
		query = query.Limit(filter.Limit)
	}
	if filter.Offset > 0 {
		query = query.Offset(filter.Offset)
	}

	if err := query.Find(&sessions).Error; err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	return sessions, nil
}

func (s *SQLite) UpdateSession(ctx context.Context, session *Session) error {
	if err := s.db.WithContext(ctx).Save(session).Error; err != nil {
		return fmt.Errorf("update session: %w", err)
	}
	return nil
}

func (s *SQLite) DeleteSession(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&Session{}).Error; err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *SQLite) SaveMessage(ctx context.Context, msg *Message) error {
	if err := s.db.WithContext(ctx).Create(msg).Error; err != nil {
		return fmt.Errorf("save message: %w", err)
	}
	return nil
}

func (s *SQLite) GetMessages(ctx context.Context, sessionID string, limit, offset int) ([]*Message, error) {
	var messages []*Message
	query := s.db.WithContext(ctx).Where("session_id = ?", sessionID).Order("created_at ASC")

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	if err := query.Find(&messages).Error; err != nil {
		return nil, fmt.Errorf("get messages: %w", err)
	}
	return messages, nil
}

func (s *SQLite) SaveUser(ctx context.Context, user *User) error {
	if err := s.db.WithContext(ctx).Create(user).Error; err != nil {
		return fmt.Errorf("save user: %w", err)
	}
	return nil
}

func (s *SQLite) GetUser(ctx context.Context, id int64) (*User, error) {
	var user User
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, fmt.Errorf("get user: %w", err)
	}
	return &user, nil
}

func (s *SQLite) ListUsers(ctx context.Context) ([]*User, error) {
	var users []*User
	if err := s.db.WithContext(ctx).Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	return users, nil
}

func (s *SQLite) DeleteUser(ctx context.Context, id int64) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&User{}).Error; err != nil {
		return fmt.Errorf("delete user: %w", err)
	}
	return nil
}

func (s *SQLite) SaveProject(ctx context.Context, project *Project) error {
	if err := s.db.WithContext(ctx).Create(project).Error; err != nil {
		return fmt.Errorf("save project: %w", err)
	}
	return nil
}

func (s *SQLite) GetProject(ctx context.Context, id string) (*Project, error) {
	var project Project
	if err := s.db.WithContext(ctx).Where("id = ?", id).First(&project).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrProjectNotFound
		}
		return nil, fmt.Errorf("get project: %w", err)
	}
	return &project, nil
}

func (s *SQLite) ListProjects(ctx context.Context, userID int64) ([]*Project, error) {
	var projects []*Project
	if err := s.db.WithContext(ctx).Where("owner_id = ?", userID).Find(&projects).Error; err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s *SQLite) DeleteProject(ctx context.Context, id string) error {
	if err := s.db.WithContext(ctx).Where("id = ?", id).Delete(&Project{}).Error; err != nil {
		return fmt.Errorf("delete project: %w", err)
	}
	return nil
}

var (
	ErrSessionNotFound = errors.New("session not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrProjectNotFound = errors.New("project not found")
)
