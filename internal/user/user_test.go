package user

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
	require.NoError(t, db.AutoMigrate(&storage.User{}))
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

func TestGetOrCreate_NewUser(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	user, err := m.GetOrCreate(ctx, 1001, "alice", "Alice", "Smith")
	require.NoError(t, err)
	require.NotNil(t, user)
	assert.Equal(t, int64(1001), user.ID)
	assert.Equal(t, "alice", user.Username)
	assert.Equal(t, "Alice", user.FirstName)
	assert.Equal(t, "Smith", user.LastName)
	assert.Equal(t, storage.UserRoleUser, user.Role)
	assert.True(t, user.IsActive)
}

func TestGetOrCreate_ExistingUser(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	user1, err := m.GetOrCreate(ctx, 2001, "bob", "Bob", "Jones")
	require.NoError(t, err)
	require.NotNil(t, user1)
	assert.Equal(t, "bob", user1.Username)

	user2, err := m.GetOrCreate(ctx, 2001, "bob_updated", "Robert", "Jones")
	require.NoError(t, err)
	require.NotNil(t, user2)
	assert.Equal(t, int64(2001), user2.ID)
	assert.Equal(t, "bob_updated", user2.Username)
	assert.Equal(t, "Robert", user2.FirstName)
}

func TestIsAuthorized_ActiveUser(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	_, err := m.GetOrCreate(ctx, 3001, "active", "Active", "User")
	require.NoError(t, err)

	assert.True(t, m.IsAuthorized(ctx, 3001))
}

func TestIsAuthorized_InactiveUser(t *testing.T) {
	st, db := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	_, err := m.GetOrCreate(ctx, 3002, "inactive", "Inactive", "User")
	require.NoError(t, err)

	require.NoError(t, db.Model(&storage.User{}).Where("id = ?", int64(3002)).Update("is_active", false).Error)

	assert.False(t, m.IsAuthorized(ctx, 3002))
}

func TestIsAuthorized_UnknownUser(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	assert.False(t, m.IsAuthorized(ctx, 9999))
}

func TestIsAdmin_AdminUser(t *testing.T) {
	st, db := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	_, err := m.GetOrCreate(ctx, 4001, "admin", "Admin", "User")
	require.NoError(t, err)

	require.NoError(t, db.Model(&storage.User{}).Where("id = ?", int64(4001)).Update("role", storage.UserRoleAdmin).Error)

	assert.True(t, m.IsAdmin(ctx, 4001))
}

func TestIsAdmin_RegularUser(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	_, err := m.GetOrCreate(ctx, 4002, "regular", "Regular", "User")
	require.NoError(t, err)

	assert.False(t, m.IsAdmin(ctx, 4002))
}

func TestList(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	for i := int64(5001); i <= 5003; i++ {
		_, err := m.GetOrCreate(ctx, i, "user", "First", "Last")
		require.NoError(t, err)
	}

	users, err := m.List(ctx)
	require.NoError(t, err)
	assert.Len(t, users, 3)
}

func TestDelete(t *testing.T) {
	st, _ := setupTestStorage(t)
	m := NewManager(st)
	ctx := context.Background()

	_, err := m.GetOrCreate(ctx, 6001, "deleteme", "Delete", "Me")
	require.NoError(t, err)

	err = m.Delete(ctx, 6001)
	require.NoError(t, err)

	assert.False(t, m.IsAuthorized(ctx, 6001))
}
