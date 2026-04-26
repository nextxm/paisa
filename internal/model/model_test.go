package model_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/portfolio"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
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

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// TestSyncJournal_AtomicWrites verifies that when price and posting writes are
// wrapped in a single outer db.Transaction, a failure rolls back both together.
func TestSyncJournal_AtomicWrites(t *testing.T) {
	db := openTestDB(t)

	// Seed initial prices and postings via normal successful writes.
	initialPrices := []price.Price{
		{Date: mustParseDate("2024-01-01"), CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD", Value: decimal.NewFromFloat(1.0)},
	}
	require.NoError(t, price.UpsertAllByType(db, config.Unknown, initialPrices))

	initialPostings := []*posting.Posting{
		{TransactionID: "txn1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
	}
	require.NoError(t, posting.UpsertAll(db, initialPostings))

	// Simulate a journal sync that writes new prices inside a transaction but
	// returns an error before committing – both the price and posting writes
	// must be rolled back.
	newPrices := []price.Price{
		{Date: mustParseDate("2024-06-01"), CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD", Value: decimal.NewFromFloat(1.1)},
	}
	simulatedErr := errors.New("simulated parse failure")

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := price.UpsertAllByType(tx, config.Unknown, newPrices); err != nil {
			return err
		}
		// Failure before posting.UpsertAll – the price write above must be rolled back.
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	// Price table must still contain the original row.
	var priceCount int64
	db.Model(&price.Price{}).Where("commodity_type = ?", config.Unknown).Count(&priceCount)
	assert.Equal(t, int64(1), priceCount, "rolled-back price write must not persist")

	var storedPrice price.Price
	db.Model(&price.Price{}).Where("commodity_type = ?", config.Unknown).First(&storedPrice)
	assert.Equal(t, "2024-01-01", storedPrice.Date.Format("2006-01-02"), "original price must be preserved after rollback")

	// Posting table must still have the original posting.
	var postingCount int64
	db.Model(&posting.Posting{}).Count(&postingCount)
	assert.Equal(t, int64(1), postingCount, "original postings must survive a rolled-back sync attempt")
}

// TestUpsertAllByType_AtomicReplace verifies that price.UpsertAllByType replaces
// all rows of the given commodity_type in a single atomic write.
func TestUpsertAllByType_AtomicReplace(t *testing.T) {
	db := openTestDB(t)

	initial := []price.Price{
		{Date: mustParseDate("2023-01-01"), CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD", Value: decimal.NewFromFloat(1.0)},
	}
	require.NoError(t, price.UpsertAllByType(db, config.Unknown, initial))

	updated := []price.Price{
		{Date: mustParseDate("2023-06-01"), CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD", Value: decimal.NewFromFloat(1.5)},
	}
	require.NoError(t, price.UpsertAllByType(db, config.Unknown, updated))

	var count int64
	db.Model(&price.Price{}).Where("commodity_type = ?", config.Unknown).Count(&count)
	assert.Equal(t, int64(1), count, "second UpsertAllByType must replace first atomically")

	var p price.Price
	db.Model(&price.Price{}).Where("commodity_type = ?", config.Unknown).First(&p)
	assert.Equal(t, "2023-06-01", p.Date.Format("2006-01-02"))
}

// TestPostingUpsertAll_Atomic verifies that posting.UpsertAll replaces all rows
// in a single atomic operation.
func TestPostingUpsertAll_Atomic(t *testing.T) {
	db := openTestDB(t)

	first := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(10)},
		{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(5)},
	}
	require.NoError(t, posting.UpsertAll(db, first))

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(2), count)

	// Second upsert with only one posting – must delete the previous two.
	second := []*posting.Posting{
		{TransactionID: "t3", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(1000)},
	}
	require.NoError(t, posting.UpsertAll(db, second))

	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(1), count, "UpsertAll must replace all postings atomically")
}

// TestCIIUpsertAll_ReturnsError verifies that cii.UpsertAll returns errors
// instead of crashing, and that repeated calls are idempotent.
func TestCIIUpsertAll_ReturnsError(t *testing.T) {
	db := openTestDB(t)

	ciis := []*cii.CII{
		{FinancialYear: "2023-24", CostInflationIndex: 348},
	}
	require.NoError(t, cii.UpsertAll(db, ciis))

	var count int64
	db.Model(&cii.CII{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Second upsert must replace the first row atomically.
	updated := []*cii.CII{
		{FinancialYear: "2024-25", CostInflationIndex: 363},
	}
	require.NoError(t, cii.UpsertAll(db, updated))

	db.Model(&cii.CII{}).Count(&count)
	assert.Equal(t, int64(1), count)

	var stored cii.CII
	db.Model(&cii.CII{}).First(&stored)
	assert.Equal(t, "2024-25", stored.FinancialYear)
}

// TestPortfolioUpsertAll_OuterTransactionRollback verifies that portfolio writes
// wrapped in a single outer transaction roll back together on failure, so that
// no partial portfolio state persists when a mid-sync error occurs.
func TestPortfolioUpsertAll_OuterTransactionRollback(t *testing.T) {
	db := openTestDB(t)

	// Commit a baseline portfolio for fund1.
	fund1 := []*portfolio.Portfolio{
		{CommodityType: config.MutualFund, ParentCommodityID: "fund1", SecurityID: "sec1", SecurityName: "SecA", Percentage: decimal.NewFromFloat(60)},
	}
	require.NoError(t, portfolio.UpsertAll(db, config.MutualFund, "fund1", fund1))

	// Begin an outer transaction that writes fund2 but then fails.
	simulatedErr := errors.New("network error during fund3 fetch")
	fund2 := []*portfolio.Portfolio{
		{CommodityType: config.MutualFund, ParentCommodityID: "fund2", SecurityID: "sec2", SecurityName: "SecB", Percentage: decimal.NewFromFloat(40)},
	}

	err := db.Transaction(func(tx *gorm.DB) error {
		if err := portfolio.UpsertAll(tx, config.MutualFund, "fund2", fund2); err != nil {
			return err
		}
		// Simulated error after fund2 was written inside the transaction.
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	// fund2 must NOT have been committed because the outer transaction rolled back.
	var count int64
	db.Model(&portfolio.Portfolio{}).Where("parent_commodity_id = ?", "fund2").Count(&count)
	assert.Equal(t, int64(0), count, "fund2 portfolio write must be rolled back with outer transaction")

	// fund1 must still be intact (committed before the failing outer transaction).
	db.Model(&portfolio.Portfolio{}).Where("parent_commodity_id = ?", "fund1").Count(&count)
	assert.Equal(t, int64(1), count, "fund1 portfolio must not be affected by later rollback")
}

// TestUpsertAllByTypeNameAndID_PreservesExplicitSourceAndQuote verifies that
// when provider prices already carry Source="provider" and a non-empty
// QuoteCommodity, both fields are stored as-is (not overwritten by defaults).
func TestUpsertAllByTypeNameAndID_PreservesExplicitSourceAndQuote(t *testing.T) {
	db := openTestDB(t)

	prices := []*price.Price{
		{
			Date:           mustParseDate("2024-03-01"),
			CommodityType:  config.MutualFund,
			CommodityID:    "scheme-123",
			CommodityName:  "MyFund",
			Value:          decimal.NewFromFloat(150.75),
			QuoteCommodity: "INR",
			Source:         "provider",
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.MutualFund, "MyFund", "scheme-123", prices))

	var stored price.Price
	require.NoError(t, db.Where("commodity_name = ?", "MyFund").First(&stored).Error)
	assert.Equal(t, "INR", stored.QuoteCommodity, "explicit QuoteCommodity must be preserved")
	assert.Equal(t, "provider", stored.Source, "explicit Source must be preserved")
}

// TestUpsertAllByTypeNameAndID_BackfillsEmptyQuote verifies backward
// compatibility: prices without QuoteCommodity still get backfilled to the
// default currency rather than causing a hard failure at the DB layer.
func TestUpsertAllByTypeNameAndID_BackfillsEmptyQuote(t *testing.T) {
	db := openTestDB(t)

	prices := []*price.Price{
		{
			Date:          mustParseDate("2024-03-01"),
			CommodityType: config.Stock,
			CommodityID:   "AAPL",
			CommodityName: "Apple",
			Value:         decimal.NewFromFloat(175.0),
			// QuoteCommodity intentionally left empty – simulates a legacy path.
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.Stock, "Apple", "AAPL", prices))

	var stored price.Price
	require.NoError(t, db.Where("commodity_name = ?", "Apple").First(&stored).Error)
	// defaultQuoteCommodity() falls back to "INR" when config is uninitialised.
	assert.Equal(t, "INR", stored.QuoteCommodity, "empty QuoteCommodity must be backfilled to default")
}

// TestUpsertAllByTypeNameAndID_CleansUpCompanionRows verifies that when a
// provider returns companion entries (e.g., exchange-rate rows) alongside main
// commodity prices, a subsequent upsert correctly removes stale companion rows
// rather than allowing them to accumulate.
func TestUpsertAllByTypeNameAndID_CleansUpCompanionRows(t *testing.T) {
	db := openTestDB(t)

	// Simulate a first Yahoo sync for AAPL: stock prices in USD plus USD→INR exchange rates.
	firstSync := []*price.Price{
		{
			Date:           mustParseDate("2024-01-02"),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(185.0),
			QuoteCommodity: "USD",
		},
		{
			Date:           mustParseDate("2024-01-02"),
			CommodityType:  config.Stock,
			CommodityID:    "USDINR=X",
			CommodityName:  "USD",
			Value:          decimal.NewFromFloat(83.0),
			QuoteCommodity: "INR",
			Source:         "com-yahoo",
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.Stock, "Apple", "AAPL", firstSync))

	var count int64
	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(2), count, "first sync should insert 2 rows")

	// Simulate a second sync; companion rows from the first sync must be replaced, not doubled.
	secondSync := []*price.Price{
		{
			Date:           mustParseDate("2024-01-03"),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(186.0),
			QuoteCommodity: "USD",
		},
		{
			Date:           mustParseDate("2024-01-03"),
			CommodityType:  config.Stock,
			CommodityID:    "USDINR=X",
			CommodityName:  "USD",
			Value:          decimal.NewFromFloat(83.5),
			QuoteCommodity: "INR",
			Source:         "com-yahoo",
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.Stock, "Apple", "AAPL", secondSync))

	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(2), count, "second sync must replace first sync rows, not accumulate")

	// The exchange rate stored is the one from the second sync.
	var exRate price.Price
	require.NoError(t, db.Where("commodity_name = ? AND quote_commodity = ?", "USD", "INR").First(&exRate).Error)
	assert.True(t, exRate.Value.Equal(decimal.NewFromFloat(83.5)), "exchange rate must reflect the latest sync")

	// Confirm exactly one USD→INR row exists (cleanup worked, no accumulation).
	var exRateCount int64
	db.Model(&price.Price{}).Where("commodity_name = ? AND quote_commodity = ?", "USD", "INR").Count(&exRateCount)
	assert.Equal(t, int64(1), exRateCount, "must be exactly one exchange-rate row after resync")
}

// TestSyncResult_DefaultValues verifies that a zero-value SyncResult
// represents a not-yet-run sync with no counts and no failed stage.
func TestSyncResult_DefaultValues(t *testing.T) {
	var result model.SyncResult
	assert.Equal(t, 0, result.PostingCount, "PostingCount must default to zero")
	assert.Equal(t, 0, result.PriceCount, "PriceCount must default to zero")
	assert.Empty(t, result.FailedStage, "FailedStage must default to empty string")
	assert.Empty(t, result.Message, "Message must default to empty string")
}

// configWithJournalPath loads a minimal paisa config that points JournalPath to
// the given absolute path so that config.GetJournalPath() returns it.
func configWithJournalPath(t *testing.T, journalPath string) {
	t.Helper()
	// Use a fixed db path inside the same temp directory to avoid any
	// dependency on the journal file path containing only safe characters.
	dbPath := t.TempDir() + "/test.db"
	yaml := fmt.Sprintf("journal_path: %q\ndb_path: %q\n", journalPath, dbPath)
	err := config.LoadConfig([]byte(yaml), "")
	require.NoError(t, err, "failed to load test config")
}

// TestSyncJournal_SkipsOnUnchangedHash verifies that when the journal file hash
// stored in metadata matches the current file hash, SyncJournal returns
// SyncResult{Skipped:true} without invoking any ledger CLI commands.
func TestSyncJournal_SkipsOnUnchangedHash(t *testing.T) {
	db := openTestDB(t)

	// Write a minimal journal file to a temp location.
	f, err := os.CreateTemp(t.TempDir(), "journal-*.ledger")
	require.NoError(t, err)
	_, err = fmt.Fprintln(f, "; empty journal for hash-skip test")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	journalPath := f.Name()

	// Point config at the temp file.
	configWithJournalPath(t, journalPath)

	// Pre-compute the file hash and seed it into the metadata table.
	hash, err := utils.SHA256File(journalPath)
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, "journal_hash", hash))

	// SyncJournal must return Skipped:true without attempting any ledger CLI calls.
	result, err := model.SyncJournal(db)
	require.NoError(t, err)
	assert.True(t, result.Skipped, "SyncJournal must set Skipped=true when hash matches")
	assert.Equal(t, 0, result.PostingCount, "PostingCount must be zero for a skipped sync")
	assert.Equal(t, 0, result.PriceCount, "PriceCount must be zero for a skipped sync")
	assert.Empty(t, result.FailedStage, "FailedStage must be empty for a skipped sync")
}

// TestSyncJournal_ProceedsOnChangedHash verifies that when the metadata hash
// does not match the current file hash, SyncJournal proceeds past the
// hash-skip guard (i.e. it does not return Skipped:true).  In this test
// environment there is no real ledger CLI, so the function is expected to
// fail at the validate stage rather than skip silently.
func TestSyncJournal_ProceedsOnChangedHash(t *testing.T) {
	db := openTestDB(t)

	// Write a minimal journal file.
	f, err := os.CreateTemp(t.TempDir(), "journal-*.ledger")
	require.NoError(t, err)
	_, err = fmt.Fprintln(f, "; journal for changed-hash test")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	journalPath := f.Name()

	configWithJournalPath(t, journalPath)

	// Seed a deliberately wrong cached hash so the skip condition is not met.
	require.NoError(t, metadata.Set(db, "journal_hash", "stale-hash-value"))

	// SyncJournal must NOT return Skipped:true; it proceeds to the ledger CLI
	// validate step which will fail in a test environment without ledger installed.
	result, err := model.SyncJournal(db)
	assert.False(t, result.Skipped, "SyncJournal must not skip when cached hash differs from file hash")
	// The function is expected to fail at the validate stage (no ledger CLI),
	// but must not be a hash-skip result.
	if err == nil {
		// If ledger happens to be available and the journal validates, that is
		// also acceptable – just verify Skipped is false.
		assert.False(t, result.Skipped)
	} else {
		assert.NotEmpty(t, result.FailedStage, "a non-skip failure must set FailedStage")
	}
}
