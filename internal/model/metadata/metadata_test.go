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

func TestGet_MissingKey(t *testing.T) {
	db := openTestDB(t)
	val, ok := metadata.Get(db, "missing")
	assert.False(t, ok)
	assert.Equal(t, "", val)
}

func TestSetAndGet(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "last_hash", "abc123"))

	val, ok := metadata.Get(db, "last_hash")
	assert.True(t, ok)
	assert.Equal(t, "abc123", val)
}

func TestSet_UpdatesExistingKey(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "last_hash", "first"))
	require.NoError(t, metadata.Set(db, "last_hash", "second"))

	val, ok := metadata.Get(db, "last_hash")
	assert.True(t, ok)
	assert.Equal(t, "second", val)

	// Confirm only one row exists for the key.
	var count int64
	require.NoError(t, db.Model(&metadata.Metadata{}).Where("key = ?", "last_hash").Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestDelete_ExistingKey(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "key1", "value1"))
	require.NoError(t, metadata.Delete(db, "key1"))

	_, ok := metadata.Get(db, "key1")
	assert.False(t, ok)
}

func TestDelete_MissingKey(t *testing.T) {
	db := openTestDB(t)
	// Deleting a non-existent key must not return an error.
	assert.NoError(t, metadata.Delete(db, "nonexistent"))
}

func TestKeyUniqueness_DirectInsert(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Create(&metadata.Metadata{Key: "dup", Value: "v1"}).Error)
	err := db.Create(&metadata.Metadata{Key: "dup", Value: "v2"}).Error
	assert.Error(t, err, "duplicate key must be rejected")
}

func TestMultipleKeys(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, metadata.Set(db, "a", "1"))
	require.NoError(t, metadata.Set(db, "b", "2"))

	v1, ok1 := metadata.Get(db, "a")
	v2, ok2 := metadata.Get(db, "b")

	assert.True(t, ok1)
	assert.Equal(t, "1", v1)
	assert.True(t, ok2)
	assert.Equal(t, "2", v2)
}
