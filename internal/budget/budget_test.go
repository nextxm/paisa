package budget

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/glebarez/sqlite"
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

// loadMinimalConfig loads a bare-minimum config so functions that call
// config.GetConfig() do not panic.
func loadMinimalConfig(t *testing.T) {
	t.Helper()
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
}

// ---------------------------------------------------------------------------
// BuildAccountBudget
// ---------------------------------------------------------------------------

func TestBuildAccountBudget_BasicForecastNoRollover(t *testing.T) {
	loadMinimalConfig(t)
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	forecasts := []posting.Posting{
		{Amount: decimal.NewFromFloat(1000), Commodity: "INR"},
	}
	expenses := []posting.Posting{
		{Amount: decimal.NewFromFloat(300), Commodity: "INR"},
	}

	result := BuildAccountBudget(date, "Expenses:Food", decimal.Zero, forecasts, expenses, false)

	assert.Equal(t, "Expenses:Food", result.Account)
	assert.True(t, result.Forecast.Equal(decimal.NewFromFloat(1000)), "forecast should be 1000")
	assert.True(t, result.Actual.Equal(decimal.NewFromFloat(300)), "actual should be 300")
	assert.True(t, result.Rollover.Equal(decimal.Zero), "rollover should be zero when no prior balance")
	assert.True(t, result.Available.Equal(decimal.NewFromFloat(700)), "available should be forecast minus actual")
	assert.Equal(t, date, result.Date)
	assert.Equal(t, expenses, result.Expenses)
}

func TestBuildAccountBudget_PastMonthZeroAvailable(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\nbudget:\n  rollover: no"), ""))
	date := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	forecasts := []posting.Posting{
		{Amount: decimal.NewFromFloat(1000), Commodity: "INR"},
	}

	result := BuildAccountBudget(date, "Expenses:Food", decimal.Zero, forecasts, nil, true)

	assert.True(t, result.Available.Equal(decimal.Zero), "past months must have zero available when rollover is disabled")
	assert.True(t, result.Forecast.Equal(decimal.NewFromFloat(1000)), "forecast is still computed for past months")
	assert.True(t, result.Actual.Equal(decimal.Zero))
}

func TestBuildAccountBudget_EmptyForecastAndExpenses(t *testing.T) {
	loadMinimalConfig(t)
	date := time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)

	result := BuildAccountBudget(date, "Expenses:Food", decimal.Zero, nil, nil, false)

	assert.True(t, result.Forecast.Equal(decimal.Zero))
	assert.True(t, result.Actual.Equal(decimal.Zero))
	assert.True(t, result.Available.Equal(decimal.Zero))
}

func TestBuildAccountBudget_OverspentShowsNegativeAvailable(t *testing.T) {
	loadMinimalConfig(t)
	date := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	forecasts := []posting.Posting{
		{Amount: decimal.NewFromFloat(500), Commodity: "INR"},
	}
	expenses := []posting.Posting{
		{Amount: decimal.NewFromFloat(700), Commodity: "INR"},
	}

	result := BuildAccountBudget(date, "Expenses:Food", decimal.Zero, forecasts, expenses, false)

	assert.True(t, result.Available.Equal(decimal.NewFromFloat(-200)), "overspent budget should show negative available")
}

// ---------------------------------------------------------------------------
// PopExpenses
// ---------------------------------------------------------------------------

func TestPopExpenses_ExactMatch(t *testing.T) {
	p := posting.Posting{Account: "Expenses:Food", Amount: decimal.NewFromFloat(100)}
	byAccount := map[string][]posting.Posting{
		"Expenses:Food": {p},
	}

	result := PopExpenses("Expenses:Food", byAccount)

	assert.Len(t, result, 1)
	_, exists := byAccount["Expenses:Food"]
	assert.False(t, exists, "matched entry must be removed from the map")
}

func TestPopExpenses_SubAccountMatch(t *testing.T) {
	p1 := posting.Posting{Account: "Expenses:Food", Amount: decimal.NewFromFloat(100)}
	p2 := posting.Posting{Account: "Expenses:Food:Restaurant", Amount: decimal.NewFromFloat(50)}
	p3 := posting.Posting{Account: "Expenses:Transport", Amount: decimal.NewFromFloat(200)}

	byAccount := map[string][]posting.Posting{
		"Expenses:Food":            {p1},
		"Expenses:Food:Restaurant": {p2},
		"Expenses:Transport":       {p3},
	}

	result := PopExpenses("Expenses:Food", byAccount)

	assert.Len(t, result, 2, "should return both direct account and sub-account postings")
	_, hasFood := byAccount["Expenses:Food"]
	_, hasRestaurant := byAccount["Expenses:Food:Restaurant"]
	assert.False(t, hasFood, "Expenses:Food must be removed from the map")
	assert.False(t, hasRestaurant, "Expenses:Food:Restaurant must be removed from the map")
	_, hasTransport := byAccount["Expenses:Transport"]
	assert.True(t, hasTransport, "Expenses:Transport must remain in the map")
}

func TestPopExpenses_NoMatch(t *testing.T) {
	byAccount := map[string][]posting.Posting{
		"Expenses:Transport": {{Amount: decimal.NewFromFloat(100)}},
	}

	result := PopExpenses("Expenses:Food", byAccount)

	assert.Empty(t, result)
	assert.Len(t, byAccount, 1, "unmatched map must be unchanged")
}

func TestPopExpenses_EmptyMap(t *testing.T) {
	byAccount := map[string][]posting.Posting{}
	result := PopExpenses("Expenses:Food", byAccount)
	assert.Empty(t, result)
}

// ---------------------------------------------------------------------------
// Compute
// ---------------------------------------------------------------------------

func TestCompute_EmptyPostings(t *testing.T) {
	loadMinimalConfig(t)
	db := openTestDB(t)

	result := Compute(db, nil, nil)

	assert.Empty(t, result.BudgetsByMonth)
	assert.True(t, result.CheckingBalance.Equal(decimal.Zero))
	assert.True(t, result.AvailableForBudgeting.Equal(decimal.Zero))
}
