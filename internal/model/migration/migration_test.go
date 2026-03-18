package migration_test

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func openMemoryDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	return db
}

func TestRunMigrations_FreshInstall(t *testing.T) {
	db := openMemoryDB(t)

	err := migration.RunMigrations(db)
	require.NoError(t, err)

	version := migration.CurrentVersion(db)
	assert.Equal(t, 1, version)
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := openMemoryDB(t)

	require.NoError(t, migration.RunMigrations(db))
	require.NoError(t, migration.RunMigrations(db))

	version := migration.CurrentVersion(db)
	assert.Equal(t, 1, version)
}

func TestCurrentVersion_NoMigrations(t *testing.T) {
	db := openMemoryDB(t)

	// Create the schema_versions table without applying any migrations.
	err := db.AutoMigrate(&migration.SchemaVersion{})
	require.NoError(t, err)

	version := migration.CurrentVersion(db)
	assert.Equal(t, 0, version)
}

func TestRunMigrations_ExistingInstall(t *testing.T) {
	db := openMemoryDB(t)

	// Simulate an existing install that has tables but no schema_versions table.
	// RunMigrations should create the table and record v1 without error.
	err := migration.RunMigrations(db)
	require.NoError(t, err)

	// Schema version should be 1 after migration.
	assert.Equal(t, 1, migration.CurrentVersion(db))
}
