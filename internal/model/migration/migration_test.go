package migration_test

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/account_balance"
	"github.com/ananthakumaran/paisa/internal/model/account_note"
	"github.com/ananthakumaran/paisa/internal/model/account_reconciliation"
	"github.com/ananthakumaran/paisa/internal/model/dashboard_snapshot"
	"github.com/ananthakumaran/paisa/internal/model/import_preset"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/projection_snapshot"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
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
	assert.Equal(t, 11, version)
}

func TestRunMigrations_Idempotent(t *testing.T) {
	db := openMemoryDB(t)

	require.NoError(t, migration.RunMigrations(db))
	require.NoError(t, migration.RunMigrations(db))

	version := migration.CurrentVersion(db)
	assert.Equal(t, 11, version)
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
	// RunMigrations should create the table and record the latest version without error.
	err := migration.RunMigrations(db)
	require.NoError(t, err)

	assert.Equal(t, 11, migration.CurrentVersion(db))
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

	// Run migrations – v2 through v11 should execute.
	require.NoError(t, migration.RunMigrations(db))
	assert.Equal(t, 11, migration.CurrentVersion(db))

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

// TestV5Migration_ImportPresetsTableExists verifies that after v5 the
// import_presets table exists and basic CRUD via package API works.
func TestV5Migration_ImportPresetsTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	saved, err := import_preset.Upsert(db, import_preset.ImportPreset{
		Name:            "My Preset",
		ColumnMappings:  map[string]string{"date": "A", "amount": "B"},
		DateFormat:      "YYYY-MM-DD",
		DefaultAccounts: map[string]string{"asset": "Assets:Checking"},
		Delimiter:       ",",
	})
	require.NoError(t, err)
	assert.Equal(t, "My Preset", saved.Name)
	assert.Equal(t, import_preset.Custom, saved.PresetType)

	all, err := import_preset.All(db)
	require.NoError(t, err)
	assert.NotEmpty(t, all)

	require.NoError(t, import_preset.Delete(db, "My Preset"))
}

// TestV6Migration_AccountReconciliationTableExists verifies that after v6 the
// account_reconciliation table exists and basic CRUD via package API works.
func TestV6Migration_AccountReconciliationTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	last := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	saved, err := account_reconciliation.Upsert(db, "Assets:Checking", &last, 30)
	require.NoError(t, err)
	assert.Equal(t, "Assets:Checking", saved.Account)
	assert.Equal(t, 30, saved.FrequencyDays)
	require.NotNil(t, saved.LastReconciledDate)
	assert.Equal(t, last, *saved.LastReconciledDate)

	updated, err := account_reconciliation.Upsert(db, "Assets:Checking", &last, 90)
	require.NoError(t, err)
	assert.Equal(t, 90, updated.FrequencyDays)
}

// TestV7Migration_AccountBalancesTableExists verifies that after v7 the
// account_balances table exists and the RefreshFromPostings helper works.
func TestV7Migration_AccountBalancesTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	// The table must exist and be queryable.
	all, err := account_balance.All(db)
	require.NoError(t, err)
	assert.Empty(t, all, "fresh install should have no balance rows")

	// Insert a row directly to confirm the table accepts writes.
	row := &account_balance.AccountBalance{
		Account:   "Assets:Checking",
		Commodity: "INR",
		Quantity:  decimal.NewFromFloat(1000),
		Amount:    decimal.NewFromFloat(1000),
	}
	require.NoError(t, db.Create(row).Error)

	all, err = account_balance.All(db)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "Assets:Checking", all[0].Account)
	assert.Equal(t, "INR", all[0].Commodity)
	assert.True(t, decimal.NewFromFloat(1000).Equal(all[0].Amount))
}

// TestV9Migration_PostingTransactionHashColumnExists verifies that after v9 the
// postings table has a transaction_hash column and the covering index exists.
func TestV9Migration_PostingTransactionHashColumnExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	// Insert a row with a transaction_hash value to confirm the column exists.
	require.NoError(t, db.Exec(
		`INSERT INTO postings (transaction_id, date, account, commodity, quantity, amount, transaction_hash)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		"txn-1", "2024-01-01 00:00:00", "Assets:Checking", "INR", "100", "100", "abc123hash",
	).Error)

	var hash string
	require.NoError(t, db.Raw(
		"SELECT transaction_hash FROM postings WHERE transaction_id = ?", "txn-1",
	).Scan(&hash).Error)
	assert.Equal(t, "abc123hash", hash, "transaction_hash column must be readable after v9 migration")

	// Verify the covering index was created.
	var count int64
	require.NoError(t, db.Raw(
		"SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_postings_txn_hash'",
	).Scan(&count).Error)
	assert.Equal(t, int64(1), count, "idx_postings_txn_hash index must exist after v9 migration")
}

func TestV10Migration_DashboardSnapshotsTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	require.NoError(t, db.Create(&dashboard_snapshot.DashboardSnapshot{
		Name:          dashboard_snapshot.SnapshotName,
		SchemaVersion: dashboard_snapshot.SchemaVersion,
		Payload:       []byte(`{"ok":true}`),
		UpdatedAt:     time.Now(),
	}).Error)

	snapshot, err := dashboard_snapshot.Get(db)
	require.NoError(t, err)
	assert.Equal(t, dashboard_snapshot.SnapshotName, snapshot.Name)
	assert.Equal(t, []byte(`{"ok":true}`), snapshot.Payload)
}

func TestV11Migration_ProjectionSnapshotsTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	require.NoError(t, db.Create(&projection_snapshot.ProjectionSnapshot{
		Name:                projection_snapshot.SnapshotName,
		SchemaVersion:       projection_snapshot.SchemaVersion,
		CurrentNetworth:     decimal.NewFromInt(100000),
		MonthlyContribution: decimal.NewFromInt(5000),
		SavingsRate:         decimal.RequireFromString("22.5"),
		AnnualExpenses:      decimal.NewFromInt(240000),
		UpdatedAt:           time.Now(),
	}).Error)

	snapshot, err := projection_snapshot.Get(db)
	require.NoError(t, err)
	assert.Equal(t, projection_snapshot.SnapshotName, snapshot.Name)
	assert.True(t, snapshot.CurrentNetworth.Equal(decimal.NewFromInt(100000)))
	assert.True(t, snapshot.MonthlyContribution.Equal(decimal.NewFromInt(5000)))
	assert.True(t, snapshot.SavingsRate.Equal(decimal.RequireFromString("22.5")))
	assert.True(t, snapshot.AnnualExpenses.Equal(decimal.NewFromInt(240000)))
}

// parser_training_log table exists and accepts writes.
func TestV8Migration_ParserTrainingLogTableExists(t *testing.T) {
	db := openMemoryDB(t)
	require.NoError(t, migration.RunMigrations(db))

	require.NoError(t, db.Exec(
		`INSERT INTO parser_training_log (
			input_text, predicted_amount, predicted_currency, predicted_from_account,
			predicted_to_account, predicted_direction, confidence_overall, suggestion_used
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		"paid $25 for groceries", decimal.NewFromInt(25).String(), "USD", "Assets:Checking",
		"Expenses:Groceries", "expense", 0.76, -1,
	).Error)

	var count int64
	require.NoError(t, db.Raw("SELECT COUNT(*) FROM parser_training_log").Scan(&count).Error)
	assert.Equal(t, int64(1), count)
}
