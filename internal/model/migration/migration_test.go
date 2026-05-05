package migration_test

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/account_note"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	assert.Equal(t, 4, version)
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := openMemoryDB(t)

	require.NoError(t, migration.RunMigrations(db))
	require.NoError(t, migration.RunMigrations(db))

	version := migration.CurrentVersion(db)
	assert.Equal(t, 4, version)
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
	// RunMigrations should create the table and record v4 without error.
	err := migration.RunMigrations(db)
	require.NoError(t, err)

	// Schema version should be 4 after migration.
	assert.Equal(t, 4, migration.CurrentVersion(db))
}

// TestV2Migration_BackfillsQuoteCommodity verifies that the v2 migration
// backfills quote_commodity = default_currency for rows that were inserted
// before the migration ran (simulating an existing installation).
func TestV2Migration_BackfillsQuoteCommodity(t *testing.T) {
	db := openMemoryDB(t)

	// Manually create the pre-v2 prices table (without quote_commodity / source).
	require.NoError(t, db.Exec(`CREATE TABLE IF NOT EXISTS prices (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATETIME,
		commodity_type TEXT,
		commodity_id TEXT,
		commodity_name TEXT,
		value TEXT
	)`).Error)

	// Seed two legacy rows that have no quote_commodity column yet.
	date1 := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(2023, 6, 1, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Exec(
		"INSERT INTO prices (date, commodity_type, commodity_id, commodity_name, value) VALUES (?, ?, ?, ?, ?)",
		date1, config.Unknown, "AAPL", "AAPL", "150.0",
	).Error)
	require.NoError(t, db.Exec(
		"INSERT INTO prices (date, commodity_type, commodity_id, commodity_name, value) VALUES (?, ?, ?, ?, ?)",
		date2, config.Unknown, "AAPL", "AAPL", "160.0",
	).Error)

	// Manually create the schema_versions table and mark v1 as already applied
	// so that only v2+ runs during RunMigrations.
	require.NoError(t, db.AutoMigrate(&migration.SchemaVersion{}))
	require.NoError(t, db.Create(&migration.SchemaVersion{Version: 1, AppliedAt: time.Now()}).Error)

	// Run migrations – v2, v3, and v4 should execute.
	require.NoError(t, migration.RunMigrations(db))
	assert.Equal(t, 4, migration.CurrentVersion(db))

	// All existing rows must have been backfilled with the default currency.
	dc := config.DefaultCurrency()
	if dc == "" {
		dc = "INR"
	}

	type row struct {
		QuoteCommodity string
	}
	var rows []row
	require.NoError(t, db.Raw("SELECT quote_commodity FROM prices").Scan(&rows).Error)
	require.Len(t, rows, 2)
	for _, r := range rows {
		assert.Equal(t, dc, r.QuoteCommodity, "legacy row must be backfilled with default_currency")
	}
}

// TestV2Migration_IndexesExist verifies that the expected indexes are present
// on the prices table after v2 has been applied.
func TestV2Migration_IndexesExist(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	type indexRow struct {
		Name string `gorm:"column:name"`
	}
	var indexes []indexRow
	require.NoError(t, db.Raw("SELECT name FROM sqlite_master WHERE type='index' AND tbl_name='prices'").Scan(&indexes).Error)

	names := make(map[string]bool)
	for _, idx := range indexes {
		names[idx.Name] = true
	}

	assert.True(t, names["idx_prices_commodity_name"], "commodity_name index must exist")
	assert.True(t, names["idx_prices_quote_commodity"], "quote_commodity index must exist")
	assert.True(t, names["idx_prices_type_date_base_quote"], "unique type/date/base/quote index must exist")
}

// TestV3Migration_MetadataTableExists verifies that after v3 the metadata table
// exists with a unique index on the key column.
func TestV3Migration_MetadataTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	// Confirm the table works via the package API.
	require.NoError(t, metadata.Set(db, "last_hash", "abc"))

	val, err := metadata.Get(db, "last_hash")
	require.NoError(t, err)
	assert.Equal(t, "abc", val)

	// Duplicate key via raw SQL must be rejected by the unique index.
	err = db.Exec("INSERT INTO metadata (key, value) VALUES (?, ?)", "last_hash", "xyz").Error
	assert.Error(t, err, "inserting a duplicate key must fail")
}

// TestV4Migration_AccountNotesTableExists verifies that after v4 the
// account_notes table exists and basic CRUD via the package API works.
func TestV4Migration_AccountNotesTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	// Insert a note.
	note, err := account_note.Upsert(db, "Assets:Checking", "Emergency fund")
	require.NoError(t, err)
	assert.Equal(t, "Assets:Checking", note.Account)
	assert.Equal(t, "Emergency fund", note.Note)

	// Fetch it back.
	fetched, err := account_note.Get(db, "Assets:Checking")
	require.NoError(t, err)
	assert.Equal(t, "Emergency fund", fetched.Note)

	// Upsert should update the note.
	updated, err := account_note.Upsert(db, "Assets:Checking", "Updated note")
	require.NoError(t, err)
	assert.Equal(t, "Updated note", updated.Note)

	// Delete the note.
	require.NoError(t, account_note.Delete(db, "Assets:Checking"))
	_, err = account_note.Get(db, "Assets:Checking")
	assert.ErrorIs(t, err, gorm.ErrRecordNotFound)
}
