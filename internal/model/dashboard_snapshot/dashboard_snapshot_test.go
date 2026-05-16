package dashboard_snapshot_test

import (
	"errors"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/dashboard_snapshot"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

func TestReplaceAndGet(t *testing.T) {
	db := openTestDB(t)

	payload := []byte(`{"hello":"world"}`)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return dashboard_snapshot.Replace(tx, payload)
	}))

	snapshot, err := dashboard_snapshot.Get(db)
	require.NoError(t, err)
	assert.Equal(t, dashboard_snapshot.SnapshotName, snapshot.Name)
	assert.Equal(t, dashboard_snapshot.SchemaVersion, snapshot.SchemaVersion)
	assert.Equal(t, payload, snapshot.Payload)
}

func TestReplace_RollbackPreservesPreviousSnapshot(t *testing.T) {
	db := openTestDB(t)

	original := []byte(`{"version":1}`)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return dashboard_snapshot.Replace(tx, original)
	}))

	simulatedErr := errors.New("snapshot refresh failed")
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := dashboard_snapshot.Replace(tx, []byte(`{"version":2}`)); err != nil {
			return err
		}
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	snapshot, err := dashboard_snapshot.Get(db)
	require.NoError(t, err)
	assert.Equal(t, original, snapshot.Payload)
}
