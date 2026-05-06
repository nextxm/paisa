package server

import (
	"strconv"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// parseDay parses a YYYY-MM-DD string into a time.Time for seeding test data.
func parseDay(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// expenseCategory unit tests
// ---------------------------------------------------------------------------

func TestExpenseCategory(t *testing.T) {
	cases := []struct {
		account string
		want    string
	}{
		{"Expenses:Groceries", "Groceries"},
		{"Expenses:Groceries:Supermarket", "Groceries"},
		{"Expenses:Tax", "Tax"},
		{"Expenses", "Expenses"},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.account, func(t *testing.T) {
			assert.Equal(t, tc.want, expenseCategory(tc.account))
		})
	}
}

// ---------------------------------------------------------------------------
// ComputeExpenseTrends unit tests
// ---------------------------------------------------------------------------

// TestComputeExpenseTrends_EmptyDB verifies that an empty database returns no
// trends.
func TestComputeExpenseTrends_EmptyDB(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	trends := ComputeExpenseTrends(db)
	assert.Empty(t, trends)
}

// TestComputeExpenseTrends_CurrentOnly verifies the case where there are
// expenses only in the current 30-day window (no previous-window expenses).
// VariancePct must be nil.
func TestComputeExpenseTrends_CurrentOnly(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          parseDay("2024-03-15"), // within last 30 days
		Account:       "Expenses:Groceries",
		Amount:        decimal.NewFromFloat(100),
		Commodity:     "INR",
	}).Error)

	trends := ComputeExpenseTrends(db)
	require.Len(t, trends, 1)

	tr := trends[0]
	assert.Equal(t, "Groceries", tr.Category)
	assert.True(t, tr.CurrentMonth.Equal(decimal.NewFromFloat(100)))
	assert.True(t, tr.PreviousMonth.IsZero())
	assert.True(t, tr.Variance.Equal(decimal.NewFromFloat(100)))
	assert.Nil(t, tr.VariancePct, "VariancePct must be nil when previous is zero")
}

// TestComputeExpenseTrends_PreviousOnly verifies the case where there are
// expenses only in the previous 30-day window.  Current is zero; VariancePct
// is computed as -100%.
func TestComputeExpenseTrends_PreviousOnly(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          parseDay("2024-02-10"), // 39 days ago → previous window
		Account:       "Expenses:Utilities",
		Amount:        decimal.NewFromFloat(80),
		Commodity:     "INR",
	}).Error)

	trends := ComputeExpenseTrends(db)
	require.Len(t, trends, 1)

	tr := trends[0]
	assert.Equal(t, "Utilities", tr.Category)
	assert.True(t, tr.CurrentMonth.IsZero())
	assert.True(t, tr.PreviousMonth.Equal(decimal.NewFromFloat(80)))
	assert.True(t, tr.Variance.Equal(decimal.NewFromFloat(-80)))
	require.NotNil(t, tr.VariancePct)
	assert.True(t, tr.VariancePct.Equal(decimal.NewFromFloat(-100)))
}

// TestComputeExpenseTrends_VariancePct verifies the percentage calculation
// (current - previous) / previous * 100, rounded to 2 decimal places.
func TestComputeExpenseTrends_VariancePct(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	// previous window expense
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          parseDay("2024-02-10"),
		Account:       "Expenses:Groceries",
		Amount:        decimal.NewFromFloat(420),
		Commodity:     "INR",
	}).Error)
	// current window expense
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          parseDay("2024-03-15"),
		Account:       "Expenses:Groceries",
		Amount:        decimal.NewFromFloat(450),
		Commodity:     "INR",
	}).Error)

	trends := ComputeExpenseTrends(db)
	require.Len(t, trends, 1)

	tr := trends[0]
	assert.Equal(t, "Groceries", tr.Category)
	assert.True(t, tr.CurrentMonth.Equal(decimal.NewFromFloat(450)))
	assert.True(t, tr.PreviousMonth.Equal(decimal.NewFromFloat(420)))
	assert.True(t, tr.Variance.Equal(decimal.NewFromFloat(30)))
	require.NotNil(t, tr.VariancePct)
	// (450 - 420) / 420 * 100 = 7.142857... rounded to 2dp = 7.14
	assert.Truef(t, tr.VariancePct.Equal(decimal.NewFromFloat(7.14)), "got %s", tr.VariancePct.String())
}

// TestComputeExpenseTrends_TaxExcluded verifies that Expenses:Tax postings are
// not included in the trend results.
func TestComputeExpenseTrends_TaxExcluded(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          parseDay("2024-03-15"),
		Account:       "Expenses:Tax:IncomeTax",
		Amount:        decimal.NewFromFloat(5000),
		Commodity:     "INR",
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          parseDay("2024-03-16"),
		Account:       "Expenses:Groceries",
		Amount:        decimal.NewFromFloat(300),
		Commodity:     "INR",
	}).Error)

	trends := ComputeExpenseTrends(db)
	// Only Groceries should appear; Tax must be excluded.
	require.Len(t, trends, 1)
	assert.Equal(t, "Groceries", trends[0].Category)
}

// TestComputeExpenseTrends_MultipleCategories verifies that multiple categories
// are returned sorted alphabetically.
func TestComputeExpenseTrends_MultipleCategories(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	accounts := []string{"Expenses:Utilities", "Expenses:Groceries", "Expenses:Dining"}
	for i, acc := range accounts {
		require.NoError(t, db.Create(&posting.Posting{
			TransactionID: "t" + strconv.Itoa(i),
			Date:          parseDay("2024-03-15"),
			Account:       acc,
			Amount:        decimal.NewFromFloat(float64((i + 1) * 100)),
			Commodity:     "INR",
		}).Error)
	}

	trends := ComputeExpenseTrends(db)
	require.Len(t, trends, 3)
	assert.Equal(t, "Dining", trends[0].Category)
	assert.Equal(t, "Groceries", trends[1].Category)
	assert.Equal(t, "Utilities", trends[2].Category)
}

// TestComputeExpenseTrends_OutsideWindowIgnored verifies that postings older
// than 60 days are not included in either window.
func TestComputeExpenseTrends_OutsideWindowIgnored(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	// This posting is 61 days before "now" – outside the 60-day window.
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          parseDay("2024-01-19"),
		Account:       "Expenses:Groceries",
		Amount:        decimal.NewFromFloat(500),
		Commodity:     "INR",
	}).Error)

	trends := ComputeExpenseTrends(db)
	assert.Empty(t, trends, "postings outside the 60-day window must be ignored")
}
