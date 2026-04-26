package metadata_test

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&metadata.Metadata{}))
	return db
}

func TestSet_Insert(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "last_hash", "abc123"))

	val, err := metadata.Get(db, "last_hash")
	require.NoError(t, err)
	assert.Equal(t, "abc123", val)
}

func TestSet_Update(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "last_hash", "abc123"))
	require.NoError(t, metadata.Set(db, "last_hash", "def456"))

	val, err := metadata.Get(db, "last_hash")
	require.NoError(t, err)
	assert.Equal(t, "def456", val)
}

func TestGet_NotFound(t *testing.T) {
	db := openTestDB(t)

	_, err := metadata.Get(db, "nonexistent")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

func TestGetOrDefault_Missing(t *testing.T) {
	db := openTestDB(t)

	val, err := metadata.GetOrDefault(db, "missing_key", "fallback")
	require.NoError(t, err)
	assert.Equal(t, "fallback", val)
}

func TestGetOrDefault_Present(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "my_key", "stored_value"))

	val, err := metadata.GetOrDefault(db, "my_key", "fallback")
	require.NoError(t, err)
	assert.Equal(t, "stored_value", val)
}

func TestKeyUniqueness(t *testing.T) {
	db := openTestDB(t)

	// Inserting the same key twice should not create duplicate rows.
	require.NoError(t, metadata.Set(db, "dup_key", "v1"))
	require.NoError(t, metadata.Set(db, "dup_key", "v2"))

	var count int64
	require.NoError(t, db.Model(&metadata.Metadata{}).Where("key = ?", "dup_key").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}
