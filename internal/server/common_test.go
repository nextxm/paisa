package server

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and runs migrations for testing.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	model.AutoMigrate(db)
	return db
}
