package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestStorage(t *testing.T) *SQLite {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	st := NewSQLite(db)
	require.NoError(t, st.Migrate(context.Background()))
	t.Cleanup(func() {
		require.NoError(t, st.Close())
	})
	return st
}

func TestNewSQLite(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	st := NewSQLite(db)
	require.NotNil(t, st)
	require.NoError(t, st.Close())
}

func TestMigrate(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	st := NewSQLite(db)
	require.NoError(t, st.Migrate(context.Background()))
	require.NoError(t, st.Close())

	db2, err := NewSQLiteDB(dbPath)
	require.NoError(t, err)
	st2 := NewSQLite(db2)
	require.NoError(t, st2.Migrate(context.Background()))
	require.NoError(t, st2.Close())
}

func TestSaveAndGetSession(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	session := &Session{
		ID:        "sess-1",
		Backend:   "opencode",
		ProjectID: "proj-1",
		Title:     "Test Session",
		Status:    SessionStatusActive,
		Model:     "gpt-4",
		Agent:     "coder",
	}
	require.NoError(t, st.SaveSession(ctx, session))

	got, err := st.GetSession(ctx, "sess-1")
	require.NoError(t, err)
	assert.Equal(t, "sess-1", got.ID)
	assert.Equal(t, "opencode", got.Backend)
	assert.Equal(t, "proj-1", got.ProjectID)
	assert.Equal(t, "Test Session", got.Title)
	assert.Equal(t, SessionStatusActive, got.Status)
	assert.Equal(t, "gpt-4", got.Model)
	assert.Equal(t, "coder", got.Agent)
}

func TestSaveSession_NotFound(t *testing.T) {
	st := newTestStorage(t)
	_, err := st.GetSession(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestListSessions(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	sessions := []*Session{
		{ID: "s1", Backend: "opencode", ProjectID: "p1", Status: SessionStatusActive, Title: "A"},
		{ID: "s2", Backend: "claude", ProjectID: "p1", Status: SessionStatusIdle, Title: "B"},
		{ID: "s3", Backend: "opencode", ProjectID: "p2", Status: SessionStatusActive, Title: "C"},
		{ID: "s4", Backend: "opencode", ProjectID: "p1", Status: SessionStatusClosed, Title: "D"},
	}
	for _, s := range sessions {
		s.CreatedAt = time.Now()
		s.UpdatedAt = time.Now()
		require.NoError(t, st.SaveSession(ctx, s))
	}

	tests := []struct {
		name   string
		filter SessionFilter
		want   int
	}{
		{"all", SessionFilter{}, 4},
		{"by project", SessionFilter{ProjectID: "p1"}, 3},
		{"by backend", SessionFilter{Backend: "opencode"}, 3},
		{"by status", SessionFilter{Status: SessionStatusActive}, 2},
		{"by project and backend", SessionFilter{ProjectID: "p1", Backend: "opencode"}, 2},
		{"with limit", SessionFilter{Limit: 2}, 2},
		{"with offset", SessionFilter{Offset: 3}, 1},
		{"combined", SessionFilter{ProjectID: "p1", Backend: "opencode", Limit: 1, Offset: 0}, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := st.ListSessions(ctx, tt.filter)
			require.NoError(t, err)
			assert.Len(t, got, tt.want)
		})
	}
}

func TestUpdateSession(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	session := &Session{
		ID:        "sess-upd",
		Backend:   "opencode",
		ProjectID: "proj-1",
		Title:     "Original",
		Status:    SessionStatusActive,
	}
	require.NoError(t, st.SaveSession(ctx, session))

	session.Title = "Updated"
	session.Status = SessionStatusIdle
	require.NoError(t, st.UpdateSession(ctx, session))

	got, err := st.GetSession(ctx, "sess-upd")
	require.NoError(t, err)
	assert.Equal(t, "Updated", got.Title)
	assert.Equal(t, SessionStatusIdle, got.Status)
}

func TestDeleteSession(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	session := &Session{ID: "sess-del", Backend: "opencode", ProjectID: "proj-1"}
	require.NoError(t, st.SaveSession(ctx, session))

	require.NoError(t, st.DeleteSession(ctx, "sess-del"))

	_, err := st.GetSession(ctx, "sess-del")
	assert.ErrorIs(t, err, ErrSessionNotFound)
}

func TestSaveAndGetMessage(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	msg := &Message{
		ID:        "msg-1",
		SessionID: "sess-1",
		Role:      MessageRoleUser,
		Content:   "Hello, world!",
		Files:     "main.go",
	}
	require.NoError(t, st.SaveMessage(ctx, msg))

	msgs, err := st.GetMessages(ctx, "sess-1", 0, 0)
	require.NoError(t, err)
	require.Len(t, msgs, 1)
	assert.Equal(t, "msg-1", msgs[0].ID)
	assert.Equal(t, MessageRoleUser, msgs[0].Role)
	assert.Equal(t, "Hello, world!", msgs[0].Content)
	assert.Equal(t, "main.go", msgs[0].Files)
}

func TestGetMessages(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		msg := &Message{
			ID:        fmt.Sprintf("msg-%d", i),
			SessionID: "sess-1",
			Role:      MessageRoleAssistant,
			Content:   fmt.Sprintf("response %d", i),
			CreatedAt: time.Now().Add(time.Duration(i) * time.Second),
		}
		require.NoError(t, st.SaveMessage(ctx, msg))
	}

	msgs, err := st.GetMessages(ctx, "sess-1", 3, 0)
	require.NoError(t, err)
	assert.Len(t, msgs, 3)
	assert.Equal(t, "msg-0", msgs[0].ID)

	msgs, err = st.GetMessages(ctx, "sess-1", 2, 2)
	require.NoError(t, err)
	assert.Len(t, msgs, 2)
	assert.Equal(t, "msg-2", msgs[0].ID)
}

func TestSaveAndGetUser(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	user := &User{
		ID:       12345,
		Username: "testuser",
		Role:     UserRoleUser,
	}
	require.NoError(t, st.SaveUser(ctx, user))

	got, err := st.GetUser(ctx, 12345)
	require.NoError(t, err)
	assert.Equal(t, int64(12345), got.ID)
	assert.Equal(t, "testuser", got.Username)
	assert.Equal(t, UserRoleUser, got.Role)
}

func TestGetUser_NotFound(t *testing.T) {
	st := newTestStorage(t)
	_, err := st.GetUser(context.Background(), 99999)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestListUsers(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	users := []*User{
		{ID: 1, Username: "alice", Role: UserRoleAdmin},
		{ID: 2, Username: "bob", Role: UserRoleUser},
		{ID: 3, Username: "charlie", Role: UserRoleViewer},
	}
	for _, u := range users {
		require.NoError(t, st.SaveUser(ctx, u))
	}

	got, err := st.ListUsers(ctx)
	require.NoError(t, err)
	assert.Len(t, got, 3)
}

func TestDeleteUser(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	user := &User{ID: 100, Username: "deleteme", Role: UserRoleUser}
	require.NoError(t, st.SaveUser(ctx, user))

	require.NoError(t, st.DeleteUser(ctx, 100))

	_, err := st.GetUser(ctx, 100)
	assert.ErrorIs(t, err, ErrUserNotFound)
}

func TestSaveAndGetProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	project := &Project{
		ID:             "proj-1",
		Name:           "My Project",
		Path:           "/home/user/project",
		DefaultBackend: "opencode",
		OwnerID:        12345,
	}
	require.NoError(t, st.SaveProject(ctx, project))

	got, err := st.GetProject(ctx, "proj-1")
	require.NoError(t, err)
	assert.Equal(t, "proj-1", got.ID)
	assert.Equal(t, "My Project", got.Name)
	assert.Equal(t, "/home/user/project", got.Path)
	assert.Equal(t, "opencode", got.DefaultBackend)
	assert.Equal(t, int64(12345), got.OwnerID)
}

func TestGetProject_NotFound(t *testing.T) {
	st := newTestStorage(t)
	_, err := st.GetProject(context.Background(), "nonexistent")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}

func TestListProjects(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	projects := []*Project{
		{ID: "p1", Name: "A", OwnerID: 1, DefaultBackend: "opencode"},
		{ID: "p2", Name: "B", OwnerID: 1, DefaultBackend: "claude"},
		{ID: "p3", Name: "C", OwnerID: 2, DefaultBackend: "opencode"},
	}
	for _, p := range projects {
		require.NoError(t, st.SaveProject(ctx, p))
	}

	got, err := st.ListProjects(ctx, 1)
	require.NoError(t, err)
	assert.Len(t, got, 2)

	got, err = st.ListProjects(ctx, 2)
	require.NoError(t, err)
	assert.Len(t, got, 1)

	got, err = st.ListProjects(ctx, 999)
	require.NoError(t, err)
	assert.Empty(t, got)
}

func TestDeleteProject(t *testing.T) {
	st := newTestStorage(t)
	ctx := context.Background()

	project := &Project{ID: "proj-del", Name: "Delete Me", OwnerID: 1, DefaultBackend: "opencode"}
	require.NoError(t, st.SaveProject(ctx, project))

	require.NoError(t, st.DeleteProject(ctx, "proj-del"))

	_, err := st.GetProject(ctx, "proj-del")
	assert.ErrorIs(t, err, ErrProjectNotFound)
}
