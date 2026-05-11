package model_test

import (
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model"
	accountbalance "github.com/ananthakumaran/paisa/internal/model/account_balance"
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

// TestPostingUpsertAll_BatchInsert verifies that UpsertAll inserts all rows
// when the input size exceeds a single batch.
func TestPostingUpsertAll_BatchInsert(t *testing.T) {
	db := openTestDB(t)

	postings := make([]*posting.Posting, 0, 1100)
	for i := 0; i < 1100; i++ {
		postings = append(postings, &posting.Posting{
			TransactionID: fmt.Sprintf("txn-%d", i),
			Account:       "Assets:Bank",
			Commodity:     "USD",
			Amount:        decimal.NewFromInt(1),
		})
	}

	require.NoError(t, posting.UpsertAll(db, postings))

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(len(postings)), count, "all postings must be inserted across batches")
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

// TestUpsertAllByTypeNameAndID_PreservesHistoryAcrossSyncs verifies that
// UpsertAllByTypeNameAndID uses a true UPSERT (no DELETE), so that prices from
// a previous sync are preserved when a subsequent incremental sync adds new rows
// for different dates.  This is the expected behavior for incremental price sync.
func TestUpsertAllByTypeNameAndID_PreservesHistoryAcrossSyncs(t *testing.T) {
	db := openTestDB(t)

	// Simulate a first sync for AAPL: one stock price and one exchange-rate entry.
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

	// Simulate a second incremental sync returning prices for a newer date.
	// History from the first sync (2024-01-02) must be preserved.
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

	// With UPSERT-only semantics, both sync rows are preserved (4 total).
	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(4), count, "incremental sync must preserve history – 4 rows across 2 dates")

	// Both exchange-rate rows must be present (one per date).
	var exRateCount int64
	db.Model(&price.Price{}).Where("commodity_name = ? AND quote_commodity = ?", "USD", "INR").Count(&exRateCount)
	assert.Equal(t, int64(2), exRateCount, "one exchange-rate row per sync date must be preserved")

	// Verify the newer exchange rate is present.
	var exRate price.Price
	require.NoError(t, db.Where("commodity_name = ? AND quote_commodity = ? AND date = ?", "USD", "INR", mustParseDate("2024-01-03")).First(&exRate).Error)
	assert.True(t, exRate.Value.Equal(decimal.NewFromFloat(83.5)), "exchange rate for 2024-01-03 must match the second sync value")
}

// TestUpsertAllByTypeNameAndID_UpdatesSameDateRow verifies that when a sync
// provides a price for a date that already exists in the DB, the existing row
// is updated in place (UPSERT ON CONFLICT DO UPDATE) rather than duplicated.
func TestUpsertAllByTypeNameAndID_UpdatesSameDateRow(t *testing.T) {
	db := openTestDB(t)

	initial := []*price.Price{
		{
			Date:           mustParseDate("2024-01-02"),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(185.0),
			QuoteCommodity: "USD",
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.Stock, "Apple", "AAPL", initial))

	var count int64
	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(1), count)

	// Re-sync the same date with a corrected value.
	corrected := []*price.Price{
		{
			Date:           mustParseDate("2024-01-02"),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(185.5),
			QuoteCommodity: "USD",
		},
	}
	require.NoError(t, price.UpsertAllByTypeNameAndID(db, config.Stock, "Apple", "AAPL", corrected))

	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(1), count, "re-syncing the same date must update in place, not duplicate")

	var stored price.Price
	require.NoError(t, db.Where("commodity_name = ?", "Apple").First(&stored).Error)
	assert.True(t, stored.Value.Equal(decimal.NewFromFloat(185.5)), "value must reflect the corrected price")
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

// TestSyncJournal_ForceJournal_BypassesHashCheck verifies that when
// forceJournal=true the hash check is skipped even when the cached hash
// matches the current file hash.  It also checks the final state of the
// stored hash for both the success and failure paths:
//   - Success: the stored hash is updated to the current file hash (so the
//     next ordinary sync can use it for file-level skipping).
//   - Failure: the stored hash is left as "" (cleared during the force path
//     and not re-written because the sync never completed), ensuring the
//     next ordinary sync does a full re-parse rather than silently skipping.
func TestSyncJournal_ForceJournal_BypassesHashCheck(t *testing.T) {
	db := openTestDB(t)

	// Write a minimal journal file.
	f, err := os.CreateTemp(t.TempDir(), "journal-*.ledger")
	require.NoError(t, err)
	_, err = fmt.Fprintln(f, "; journal for force-journal test")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	journalPath := f.Name()

	configWithJournalPath(t, journalPath)

	// Pre-seed the correct hash so a normal sync would skip.
	hash, err := utils.SHA256File(journalPath)
	require.NoError(t, err)
	require.NoError(t, metadata.Set(db, "journal_hash", hash))

	// With forceJournal=true the sync must NOT return Skipped:true.
	result, syncErr := model.SyncJournal(db, true)
	assert.False(t, result.Skipped, "SyncJournal with forceJournal=true must never skip, even when hash matches")
	if syncErr != nil {
		assert.NotEmpty(t, result.FailedStage, "force-journal failure must set FailedStage")
	}

	storedHash, hashErr := metadata.GetOrDefault(db, "journal_hash", "sentinel")
	require.NoError(t, hashErr)

	if syncErr == nil {
		// Successful sync: the hash must be updated to the current file hash so
		// that the next ordinary sync can use it for file-level skipping.
		assert.Equal(t, hash, storedHash,
			"after a successful force sync, the journal hash must be the current file hash")
	} else {
		// Failed sync: the hash must remain cleared ("") so the next ordinary
		// sync does not silently skip.
		assert.Equal(t, "", storedHash,
			"after a failed force sync, the journal hash must be cleared so the next sync proceeds")
	}
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
	result, err := model.SyncJournal(db, false)
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
	result, err := model.SyncJournal(db, false)
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

// TestSyncJournal_AccountBalancesRefreshed verifies that the account_balances
// table is populated atomically alongside postings in a sync transaction, and
// that a rollback clears both postings and balance rows together.
func TestSyncJournal_AccountBalancesRefreshed(t *testing.T) {
	db := openTestDB(t)

	postings := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Checking", Commodity: "INR", Quantity: decimal.NewFromFloat(1000), Amount: decimal.NewFromFloat(1000)},
		{TransactionID: "t2", Account: "Expenses:Groceries", Commodity: "INR", Quantity: decimal.NewFromFloat(-500), Amount: decimal.NewFromFloat(-500)},
	}

	// Simulate the write portion of SyncJournal: postings + balance refresh.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		if err := posting.UpsertAll(tx, postings); err != nil {
			return err
		}
		return accountbalance.RefreshFromPostings(tx, postings)
	}))

	// Verify account balances are now materialized.
	balances, err := accountbalance.All(db)
	require.NoError(t, err)
	require.Len(t, balances, 2, "one row per distinct (account, commodity) pair")

	byAcc := make(map[string]accountbalance.AccountBalance)
	for _, b := range balances {
		byAcc[b.Account] = b
	}
	assert.True(t, decimal.NewFromFloat(1000).Equal(byAcc["Assets:Checking"].Amount),
		"checking balance must be 1000; got %s", byAcc["Assets:Checking"].Amount)
	assert.True(t, decimal.NewFromFloat(-500).Equal(byAcc["Expenses:Groceries"].Amount),
		"expenses balance must be -500; got %s", byAcc["Expenses:Groceries"].Amount)

	// Verify rollback clears both postings and balances atomically.
	simulatedErr := errors.New("sync failure")
	newPostings := []*posting.Posting{
		{TransactionID: "t3", Account: "Income:Salary", Commodity: "INR", Quantity: decimal.NewFromFloat(5000), Amount: decimal.NewFromFloat(5000)},
	}
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := posting.UpsertAll(tx, newPostings); err != nil {
			return err
		}
		if err := accountbalance.RefreshFromPostings(tx, newPostings); err != nil {
			return err
		}
		return simulatedErr
	})
	require.ErrorIs(t, err, simulatedErr)

	// Original data must be intact after rollback.
	var postingCount int64
	db.Model(&posting.Posting{}).Count(&postingCount)
	assert.Equal(t, int64(2), postingCount, "postings must be unchanged after rollback")

	balancesAfterRollback, err := accountbalance.All(db)
	require.NoError(t, err)
	assert.Len(t, balancesAfterRollback, 2, "account_balances must be unchanged after rollback")
}

// TestFilterSince verifies that price.FilterSince returns only prices on or
// after the start-of-day of since, and returns all prices when since is zero.
func TestFilterSince(t *testing.T) {
	prices := []*price.Price{
		{Date: mustParseDate("2024-01-01"), CommodityName: "A"},
		{Date: mustParseDate("2024-01-10"), CommodityName: "B"},
		{Date: mustParseDate("2024-01-20"), CommodityName: "C"},
	}

	// Zero since → all prices returned.
	result := price.FilterSince(prices, time.Time{})
	assert.Len(t, result, 3, "zero since must return all prices")

	// since = 2024-01-10 at mid-day → prices on or after 2024-01-10.
	since := time.Date(2024, 1, 10, 14, 30, 0, 0, time.UTC)
	result = price.FilterSince(prices, since)
	assert.Len(t, result, 2, "since 2024-01-10 must include prices on and after that date")
	assert.Equal(t, "B", result[0].CommodityName)
	assert.Equal(t, "C", result[1].CommodityName)

	// since after all prices → empty slice.
	result = price.FilterSince(prices, time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	assert.Empty(t, result, "since after all prices must return empty slice")
}

// --- DeltaUpsert tests ---

// TestDeltaUpsert_FirstSync verifies that on an empty postings table
// DeltaUpsert inserts all supplied postings and reports every transaction as added.
func TestDeltaUpsert_FirstSync(t *testing.T) {
	db := openTestDB(t)

	postings := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
		{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(30)},
		{TransactionID: "t2", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(-30)},
	}

	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, postings)
	require.NoError(t, err)

	assert.Equal(t, 2, added, "both transactions must be reported as added on first sync")
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 0, unchanged)

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(4), count, "all 4 posting rows must be present after first sync")
}

// TestDeltaUpsert_NoChanges verifies that a second sync with identical postings
// performs no DB writes (all transactions reported as unchanged).
func TestDeltaUpsert_NoChanges(t *testing.T) {
	db := openTestDB(t)

	postings := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
	}

	// First sync.
	_, _, _, _, err := posting.DeltaUpsert(db, postings)
	require.NoError(t, err)

	// Second sync with the same data.
	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, postings)
	require.NoError(t, err)

	assert.Equal(t, 0, added)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, unchanged, "unchanged transaction must be skipped on second sync")

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(2), count, "row count must not change when nothing is modified")
}

// TestDeltaUpsert_AddNewTransaction verifies that adding one new transaction
// to an already-synced journal only inserts the new rows without touching
// the existing ones.
func TestDeltaUpsert_AddNewTransaction(t *testing.T) {
	db := openTestDB(t)

	initial := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
	}
	_, _, _, _, err := posting.DeltaUpsert(db, initial)
	require.NoError(t, err)

	// Second sync adds t2 while t1 is unchanged.
	withNew := append(initial,
		&posting.Posting{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(30)},
		&posting.Posting{TransactionID: "t2", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(-30)},
	)
	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, withNew)
	require.NoError(t, err)

	assert.Equal(t, 1, added, "only the new transaction must be reported as added")
	assert.Equal(t, 0, updated)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, unchanged, "t1 must be reported as unchanged")

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(4), count, "total posting count must grow by 2 new rows")
}

// TestDeltaUpsert_ModifyTransaction verifies that changing a posting inside an
// existing transaction causes only that transaction's rows to be replaced.
func TestDeltaUpsert_ModifyTransaction(t *testing.T) {
	db := openTestDB(t)

	initial := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
		{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(30)},
		{TransactionID: "t2", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(-30)},
	}
	_, _, _, _, err := posting.DeltaUpsert(db, initial)
	require.NoError(t, err)

	// Modify t2's amount – simulates editing a transaction in the journal.
	modified := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
		{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(35)}, // changed
		{TransactionID: "t2", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(-35)},  // changed
	}
	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, modified)
	require.NoError(t, err)

	assert.Equal(t, 0, added)
	assert.Equal(t, 1, updated, "t2 must be reported as updated")
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, unchanged, "t1 must be unchanged")

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(4), count, "total posting count must stay the same after an in-place edit")

	// Verify the updated amount is present.
	var p posting.Posting
	db.Where("transaction_id = ? AND account = ?", "t2", "Expenses:Food").First(&p)
	assert.True(t, decimal.NewFromFloat(35).Equal(p.Amount), "modified amount must be reflected in the DB")
}

// TestDeltaUpsert_RemoveTransaction verifies that removing a transaction from
// the journal deletes only its rows from the postings table.
func TestDeltaUpsert_RemoveTransaction(t *testing.T) {
	db := openTestDB(t)

	initial := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
		{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)},
		{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(30)},
		{TransactionID: "t2", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(-30)},
	}
	_, _, _, _, err := posting.DeltaUpsert(db, initial)
	require.NoError(t, err)

	// Second sync without t2.
	withoutT2 := initial[:2]
	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, withoutT2)
	require.NoError(t, err)

	assert.Equal(t, 0, added)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, removed, "t2 must be reported as removed")
	assert.Equal(t, 1, unchanged, "t1 must be reported as unchanged")

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(2), count, "only t1 rows must remain after t2 is removed")
}

// TestDeltaUpsert_EmptyInput verifies that syncing an empty posting slice
// deletes all existing rows and reports them as removed.
func TestDeltaUpsert_EmptyInput(t *testing.T) {
	db := openTestDB(t)

	initial := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
	}
	_, _, _, _, err := posting.DeltaUpsert(db, initial)
	require.NoError(t, err)

	added, updated, removed, unchanged, err := posting.DeltaUpsert(db, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 0, updated)
	assert.Equal(t, 1, removed, "t1 must be removed when syncing an empty set")
	assert.Equal(t, 0, unchanged)

	var count int64
	db.Model(&posting.Posting{}).Count(&count)
	assert.Equal(t, int64(0), count, "postings table must be empty after syncing nil")
}

// TestComputeTransactionHash_Deterministic verifies that the same set of
// postings always produces the same hash regardless of slice order.
func TestComputeTransactionHash_Deterministic(t *testing.T) {
	p1 := &posting.Posting{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)}
	p2 := &posting.Posting{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)}

	hash1 := posting.ComputeTransactionHash([]*posting.Posting{p1, p2})
	hash2 := posting.ComputeTransactionHash([]*posting.Posting{p2, p1})
	assert.Equal(t, hash1, hash2, "ComputeTransactionHash must be order-independent")
}

// TestComputeTransactionHash_DifferentContent verifies that changing any field
// in a posting produces a different transaction hash.
func TestComputeTransactionHash_DifferentContent(t *testing.T) {
	base := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)},
	}
	modified := []*posting.Posting{
		{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(101)},
	}
	assert.NotEqual(t, posting.ComputeTransactionHash(base), posting.ComputeTransactionHash(modified),
		"differing amounts must yield different hashes")
}

// TestStampTransactionHash verifies that StampTransactionHash sets the same
// TransactionHash on every posting that belongs to the same transaction.
func TestStampTransactionHash(t *testing.T) {
	p1 := &posting.Posting{TransactionID: "t1", Account: "Assets:Bank", Commodity: "USD", Amount: decimal.NewFromFloat(100)}
	p2 := &posting.Posting{TransactionID: "t1", Account: "Income:Salary", Commodity: "USD", Amount: decimal.NewFromFloat(-100)}
	p3 := &posting.Posting{TransactionID: "t2", Account: "Expenses:Food", Commodity: "USD", Amount: decimal.NewFromFloat(50)}

	posting.StampTransactionHash([]*posting.Posting{p1, p2, p3})

	assert.NotEmpty(t, p1.TransactionHash)
	assert.Equal(t, p1.TransactionHash, p2.TransactionHash, "all postings in t1 must share the same hash")
	assert.NotEqual(t, p1.TransactionHash, p3.TransactionHash, "different transactions must have different hashes")
}
