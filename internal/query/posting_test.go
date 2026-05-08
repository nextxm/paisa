package query

import (
	"fmt"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and runs migrations.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

// ---------------------------------------------------------------------------
// GroupSum tests
// ---------------------------------------------------------------------------

// TestGroupSum_Empty verifies that an empty database returns an empty slice.
func TestGroupSum_Empty(t *testing.T) {
	db := openTestDB(t)
	sums := Init(db).GroupSum()
	assert.Empty(t, sums)
}

// TestGroupSum_SinglePosting verifies that a single posting is returned as
// one AccountCommoditySum entry.
func TestGroupSum_SinglePosting(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(1000),
		Quantity:      decimal.NewFromFloat(1000),
	}).Error)

	sums := Init(db).GroupSum()
	require.Len(t, sums, 1)
	assert.Equal(t, "Assets:Checking", sums[0].Account)
	assert.Equal(t, "INR", sums[0].Commodity)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Amount),
		"amount mismatch: got %s", sums[0].Amount)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Quantity),
		"quantity mismatch: got %s", sums[0].Quantity)
}

// TestGroupSum_MultiplePostingsSameAccountCommodity verifies that multiple
// postings for the same (account, commodity) pair are aggregated into one row.
func TestGroupSum_MultiplePostingsSameAccountCommodity(t *testing.T) {
	db := openTestDB(t)
	for i, amount := range []float64{500, 300, 200} {
		require.NoError(t, db.Create(&posting.Posting{
			TransactionID: fmt.Sprintf("t%d", i+1),
			Date:          time.Date(2024, 1, 10+i, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Checking",
			Commodity:     "INR",
			Amount:        decimal.NewFromFloat(amount),
			Quantity:      decimal.NewFromFloat(amount),
		}).Error)
	}

	sums := Init(db).GroupSum()
	require.Len(t, sums, 1)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Amount),
		"expected SUM(amount)=1000, got %s", sums[0].Amount)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Quantity),
		"expected SUM(quantity)=1000, got %s", sums[0].Quantity)
}

// TestGroupSum_DifferentCommodities verifies that different commodities in the
// same account produce separate GroupSum rows.
func TestGroupSum_DifferentCommodities(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Stock",
		Commodity:     "AAPL",
		Amount:        decimal.NewFromFloat(1500),
		Quantity:      decimal.NewFromFloat(10),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Stock",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(200),
		Quantity:      decimal.NewFromFloat(200),
	}).Error)

	sums := Init(db).GroupSum()
	require.Len(t, sums, 2)

	// Find each entry by commodity.
	sumByComm := make(map[string]AccountCommoditySum)
	for _, s := range sums {
		sumByComm[s.Commodity] = s
	}

	aaplSum, ok := sumByComm["AAPL"]
	require.True(t, ok, "expected AAPL row")
	assert.True(t, decimal.NewFromFloat(1500).Equal(aaplSum.Amount))
	assert.True(t, decimal.NewFromFloat(10).Equal(aaplSum.Quantity))

	inrSum, ok := sumByComm["INR"]
	require.True(t, ok, "expected INR row")
	assert.True(t, decimal.NewFromFloat(200).Equal(inrSum.Amount))
}

// TestGroupSum_DifferentAccounts verifies that different accounts produce
// separate rows even for the same commodity.
func TestGroupSum_DifferentAccounts(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking:HDFC",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(2000),
		Quantity:      decimal.NewFromFloat(2000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking:SBI",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(3000),
		Quantity:      decimal.NewFromFloat(3000),
	}).Error)

	sums := Init(db).GroupSum()
	require.Len(t, sums, 2)

	sumByAcc := make(map[string]AccountCommoditySum)
	for _, s := range sums {
		sumByAcc[s.Account] = s
	}

	assert.True(t, decimal.NewFromFloat(2000).Equal(sumByAcc["Assets:Checking:HDFC"].Amount))
	assert.True(t, decimal.NewFromFloat(3000).Equal(sumByAcc["Assets:Checking:SBI"].Amount))
}

// TestGroupSum_ForecastExcluded verifies that forecast=true postings are NOT
// included in GroupSum results (matching the behaviour of All()).
func TestGroupSum_ForecastExcluded(t *testing.T) {
	db := openTestDB(t)
	// Regular posting
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(1000),
		Quantity:      decimal.NewFromFloat(1000),
		Forecast:      false,
	}).Error)
	// Forecast posting (should be excluded)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(5000),
		Quantity:      decimal.NewFromFloat(5000),
		Forecast:      true,
	}).Error)

	sums := Init(db).GroupSum()
	require.Len(t, sums, 1)
	// Only the non-forecast posting's amount should be summed.
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Amount),
		"forecast posting must be excluded; got %s", sums[0].Amount)
}

// TestGroupSum_ForecastIncluded verifies that Forecast() enables the inclusion
// of forecast postings.
func TestGroupSum_ForecastIncluded(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(1000),
		Quantity:      decimal.NewFromFloat(1000),
		Forecast:      true,
	}).Error)

	// Without Forecast() – should return empty.
	sums := Init(db).GroupSum()
	assert.Empty(t, sums)

	// With Forecast() – should return the row.
	sums = Init(db).Forecast().GroupSum()
	require.Len(t, sums, 1)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Amount))
}

// TestGroupSum_AccountPrefixFilter verifies that AccountPrefix restricts the
// GroupSum to matching accounts only.
func TestGroupSum_AccountPrefixFilter(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(1000),
		Quantity:      decimal.NewFromFloat(1000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
		Account:       "Expenses:Groceries",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(200),
		Quantity:      decimal.NewFromFloat(200),
	}).Error)

	sums := Init(db).AccountPrefix("Assets:Checking").GroupSum()
	require.Len(t, sums, 1)
	assert.Equal(t, "Assets:Checking", sums[0].Account)
	assert.True(t, decimal.NewFromFloat(1000).Equal(sums[0].Amount))
}

// TestGroupSum_NegativeAmounts verifies that withdrawals (negative amounts)
// are correctly summed, including cases where the net is negative.
func TestGroupSum_NegativeAmounts(t *testing.T) {
	db := openTestDB(t)
	// Deposit 1000
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(1000),
		Quantity:      decimal.NewFromFloat(1000),
	}).Error)
	// Withdraw 300
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(-300),
		Quantity:      decimal.NewFromFloat(-300),
	}).Error)

	sums := Init(db).AccountPrefix("Assets:Checking").GroupSum()
	require.Len(t, sums, 1)
	assert.True(t, decimal.NewFromFloat(700).Equal(sums[0].Amount),
		"net balance should be 700 after deposit and withdrawal; got %s", sums[0].Amount)
}

// TestNotAccountPrefix verifies that NotAccountPrefix excludes matching accounts.
func TestNotAccountPrefix(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Account:       "Assets:Checking:HDFC",
		Amount:        decimal.NewFromInt(1000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Account:       "Assets:Savings:SBI",
		Amount:        decimal.NewFromInt(2000),
	}).Error)

	postings := Init(db).NotAccountPrefix("Assets:Checking").All()
	require.Len(t, postings, 1)
	assert.Equal(t, "Assets:Savings:SBI", postings[0].Account)

	postings = Init(db).NotAccountPrefix("Assets:Checking", "Assets:Savings").All()
	assert.Empty(t, postings)
}

// TestNotInactive verifies that NotInactive filters out inactive accounts.
func TestNotInactive(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Account:       "Assets:Active",
		Amount:        decimal.NewFromInt(1000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Account:       "Assets:InactivePattern",
		Amount:        decimal.NewFromInt(2000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t3",
		Account:       "Assets:InactiveFlag",
		Amount:        decimal.NewFromInt(3000),
	}).Error)

	// Setup config
	orig := config.GetConfig()
	t.Cleanup(func() {
		config.SaveConfigObject(orig)
	})
	yaml := `
journal_path: test.ledger
db_path: test.db
inactive_accounts:
  - Assets:InactivePattern
accounts:
  - name: Assets:InactiveFlag
    inactive: true
`
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))

	postings := Init(db).NotInactive().All()
	require.Len(t, postings, 1)
	assert.Equal(t, "Assets:Active", postings[0].Account)
}
