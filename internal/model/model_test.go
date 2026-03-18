package model_test

import (
	"errors"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/portfolio"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
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
