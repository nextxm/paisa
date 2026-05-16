package projection_snapshot_test

import (
	"errors"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/projection_snapshot"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
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

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return projection_snapshot.Replace(
			tx,
			decimal.NewFromInt(1250000),
			decimal.NewFromInt(25000),
			decimal.RequireFromString("37.5"),
			decimal.NewFromInt(720000),
		)
	}))

	snapshot, err := projection_snapshot.Get(db)
	require.NoError(t, err)
	assert.Equal(t, projection_snapshot.SnapshotName, snapshot.Name)
	assert.Equal(t, projection_snapshot.SchemaVersion, snapshot.SchemaVersion)
	assert.True(t, snapshot.CurrentNetworth.Equal(decimal.NewFromInt(1250000)))
	assert.True(t, snapshot.MonthlyContribution.Equal(decimal.NewFromInt(25000)))
	assert.True(t, snapshot.SavingsRate.Equal(decimal.RequireFromString("37.5")))
	assert.True(t, snapshot.AnnualExpenses.Equal(decimal.NewFromInt(720000)))
}

func TestReplace_RollbackPreservesPreviousSnapshot(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return projection_snapshot.Replace(
			tx,
			decimal.NewFromInt(100),
			decimal.NewFromInt(10),
			decimal.NewFromInt(20),
			decimal.NewFromInt(30),
		)
	}))

	simulatedErr := errors.New("projection snapshot refresh failed")
	err := db.Transaction(func(tx *gorm.DB) error {
		if err := projection_snapshot.Replace(
			tx,
			decimal.NewFromInt(200),
			decimal.NewFromInt(20),
			decimal.NewFromInt(40),
			decimal.NewFromInt(60),
		); err != nil {
			return err
		}
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	snapshot, err := projection_snapshot.Get(db)
	require.NoError(t, err)
	assert.True(t, snapshot.CurrentNetworth.Equal(decimal.NewFromInt(100)))
	assert.True(t, snapshot.MonthlyContribution.Equal(decimal.NewFromInt(10)))
	assert.True(t, snapshot.SavingsRate.Equal(decimal.NewFromInt(20)))
	assert.True(t, snapshot.AnnualExpenses.Equal(decimal.NewFromInt(30)))
}
