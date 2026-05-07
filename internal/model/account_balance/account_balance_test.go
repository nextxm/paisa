package account_balance_test

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/account_balance"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
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

// ---------------------------------------------------------------------------
// All / ByAccount
// ---------------------------------------------------------------------------

func TestAll_Empty(t *testing.T) {
	db := openTestDB(t)
	rows, err := account_balance.All(db)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestAll_ReturnsRowsOrderedByAccountCommodity(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&account_balance.AccountBalance{
		Account: "Assets:Savings", Commodity: "INR", Quantity: decimal.NewFromFloat(500), Amount: decimal.NewFromFloat(500),
	}).Error)
	require.NoError(t, db.Create(&account_balance.AccountBalance{
		Account: "Assets:Checking", Commodity: "INR", Quantity: decimal.NewFromFloat(1000), Amount: decimal.NewFromFloat(1000),
	}).Error)

	rows, err := account_balance.All(db)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Assets:Checking", rows[0].Account, "should be ordered by account asc")
	assert.Equal(t, "Assets:Savings", rows[1].Account)
}

func TestByAccount_ReturnsOnlyMatchingAccount(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&account_balance.AccountBalance{
		Account: "Assets:Checking", Commodity: "INR", Quantity: decimal.NewFromFloat(1000), Amount: decimal.NewFromFloat(1000),
	}).Error)
	require.NoError(t, db.Create(&account_balance.AccountBalance{
		Account: "Assets:Savings", Commodity: "INR", Quantity: decimal.NewFromFloat(500), Amount: decimal.NewFromFloat(500),
	}).Error)

	rows, err := account_balance.ByAccount(db, "Assets:Checking")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Assets:Checking", rows[0].Account)
}

func TestByAccount_EmptyWhenAccountNotFound(t *testing.T) {
	db := openTestDB(t)
	rows, err := account_balance.ByAccount(db, "Assets:DoesNotExist")
	require.NoError(t, err)
	assert.Empty(t, rows)
}

// ---------------------------------------------------------------------------
// RefreshFromPostings
// ---------------------------------------------------------------------------

func makePosting(txID, account, commodity string, amount float64) *posting.Posting {
	return &posting.Posting{
		TransactionID: txID,
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       account,
		Commodity:     commodity,
		Quantity:      decimal.NewFromFloat(amount),
		Amount:        decimal.NewFromFloat(amount),
	}
}

func TestRefreshFromPostings_Empty(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, nil)
	}))
	rows, err := account_balance.All(db)
	require.NoError(t, err)
	assert.Empty(t, rows)
}

func TestRefreshFromPostings_SinglePosting(t *testing.T) {
	db := openTestDB(t)
	postings := []*posting.Posting{makePosting("t1", "Assets:Checking", "INR", 1000)}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, postings)
	}))

	rows, err := account_balance.All(db)
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, "Assets:Checking", rows[0].Account)
	assert.Equal(t, "INR", rows[0].Commodity)
	assert.True(t, decimal.NewFromFloat(1000).Equal(rows[0].Amount))
	assert.True(t, decimal.NewFromFloat(1000).Equal(rows[0].Quantity))
}

func TestRefreshFromPostings_AggregatesMultiplePostings(t *testing.T) {
	db := openTestDB(t)
	postings := []*posting.Posting{
		makePosting("t1", "Assets:Checking", "INR", 1000),
		makePosting("t2", "Assets:Checking", "INR", -300),
		makePosting("t3", "Assets:Checking", "INR", 500),
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, postings)
	}))

	rows, err := account_balance.ByAccount(db, "Assets:Checking")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.True(t, decimal.NewFromFloat(1200).Equal(rows[0].Amount),
		"1000 - 300 + 500 = 1200; got %s", rows[0].Amount)
}

func TestRefreshFromPostings_SeparatesByAccountCommodity(t *testing.T) {
	db := openTestDB(t)
	postings := []*posting.Posting{
		makePosting("t1", "Assets:Checking", "INR", 1000),
		makePosting("t2", "Assets:Stock", "AAPL", 1500),
		makePosting("t3", "Assets:Stock", "INR", 200),
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, postings)
	}))

	rows, err := account_balance.All(db)
	require.NoError(t, err)
	require.Len(t, rows, 3, "three distinct (account, commodity) pairs")
}

func TestRefreshFromPostings_ExcludesForecast(t *testing.T) {
	db := openTestDB(t)
	actual := makePosting("t1", "Assets:Checking", "INR", 1000)
	forecast := &posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Quantity:      decimal.NewFromFloat(5000),
		Amount:        decimal.NewFromFloat(5000),
		Forecast:      true,
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, []*posting.Posting{actual, forecast})
	}))

	rows, err := account_balance.ByAccount(db, "Assets:Checking")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.True(t, decimal.NewFromFloat(1000).Equal(rows[0].Amount),
		"forecast posting must be excluded; got %s", rows[0].Amount)
}

func TestRefreshFromPostings_ReplacesExistingRows(t *testing.T) {
	db := openTestDB(t)
	// Seed an initial balance.
	firstSync := []*posting.Posting{makePosting("t1", "Assets:Checking", "INR", 1000)}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, firstSync)
	}))

	// Second sync with different postings – should fully replace.
	secondSync := []*posting.Posting{
		makePosting("t1", "Assets:Checking", "INR", 2000),
		makePosting("t2", "Assets:Savings", "INR", 500),
	}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, secondSync)
	}))

	rows, err := account_balance.All(db)
	require.NoError(t, err)
	require.Len(t, rows, 2, "old rows must be replaced, not accumulated")

	byAcc := make(map[string]account_balance.AccountBalance)
	for _, r := range rows {
		byAcc[r.Account] = r
	}
	assert.True(t, decimal.NewFromFloat(2000).Equal(byAcc["Assets:Checking"].Amount))
	assert.True(t, decimal.NewFromFloat(500).Equal(byAcc["Assets:Savings"].Amount))
}

func TestRefreshFromPostings_EmptySliceClearsTable(t *testing.T) {
	db := openTestDB(t)
	// Seed some data.
	firstSync := []*posting.Posting{makePosting("t1", "Assets:Checking", "INR", 1000)}
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, firstSync)
	}))

	// Refresh with empty slice – table should be empty.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return account_balance.RefreshFromPostings(tx, []*posting.Posting{})
	}))

	rows, err := account_balance.All(db)
	require.NoError(t, err)
	assert.Empty(t, rows, "empty postings must clear the balance table")
}
