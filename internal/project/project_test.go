package project

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"

	"github.com/KotVnn/Telegram-Open-CLI/internal/storage"
)

func setupTestStorage(t *testing.T) (storage.Storage, *gorm.DB) {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := storage.NewSQLiteDB(dbPath)
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&storage.Project{}))
	t.Cleanup(func() {
		sqlDB, err := db.DB()
		if err == nil {
			sqlDB.Close()
		}
	})
	return storage.NewSQLite(db), db
}

func TestNewManager(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	require.NotNil(t, m)
}

func TestCreate(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	project, err := m.Create(ctx, "my-project", "/home/user/my-project", "A test project", "opencode", 1001)
	require.NoError(t, err)
	require.NotNil(t, project)
	assert.Equal(t, "my-project", project.Name)
	assert.Equal(t, "/home/user/my-project", project.Path)
	assert.Equal(t, "A test project", project.Description)
	assert.Equal(t, "opencode", project.DefaultBackend)
	assert.Equal(t, int64(1001), project.OwnerID)
	assert.True(t, project.IsActive)
	assert.NotEmpty(t, project.ID)
}

func TestGet(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	created, err := m.Create(ctx, "get-project", "/path", "desc", "claude", 2001)
	require.NoError(t, err)

	got, err := m.Get(ctx, created.ID)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, created.ID, got.ID)
	assert.Equal(t, "get-project", got.Name)
	assert.Equal(t, int64(2001), got.OwnerID)
}

func TestGet_NotFound(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	got, err := m.Get(ctx, "nonexistent-id")
	assert.ErrorIs(t, err, storage.ErrProjectNotFound)
	assert.Nil(t, got)
}

func TestList(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	ownerID := int64(3001)
	_, err := m.Create(ctx, "proj-a", "/a", "", "opencode", ownerID)
	require.NoError(t, err)
	_, err = m.Create(ctx, "proj-b", "/b", "", "opencode", ownerID)
	require.NoError(t, err)

	_, err = m.Create(ctx, "proj-other", "/other", "", "opencode", 9999)
	require.NoError(t, err)

	projects, err := m.List(ctx, ownerID)
	require.NoError(t, err)
	assert.Len(t, projects, 2)

	for _, p := range projects {
		assert.Equal(t, ownerID, p.OwnerID)
	}
}

func TestDelete(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	created, err := m.Create(ctx, "del-project", "/del", "to be deleted", "aider", 4001)
	require.NoError(t, err)

	err = m.Delete(ctx, created.ID)
	require.NoError(t, err)

	got, err := m.Get(ctx, created.ID)
	assert.ErrorIs(t, err, storage.ErrProjectNotFound)
	assert.Nil(t, got)
}
