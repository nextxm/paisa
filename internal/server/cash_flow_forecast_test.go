package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// recurringOccurrencesInMonth
// ---------------------------------------------------------------------------

func TestRecurringOccurrencesInMonth_MonthlyInterval(t *testing.T) {
	// A transaction that last occurred on 2024-01-15 with a 30-day interval.
	// The next occurrence is 2024-02-14 which is in February 2024.
	lastDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 1, recurringOccurrencesInMonth(lastDate, 30, monthStart))
}

func TestRecurringOccurrencesInMonth_NoOccurrence(t *testing.T) {
	// lastDate 2024-01-28 + 30 days = 2024-02-27, then +30 = 2024-03-28.
	// Neither falls in March's first week – but 2024-03-28 IS in March.
	lastDate := time.Date(2024, 1, 28, 0, 0, 0, 0, time.UTC)
	// Check April: 2024-04-27 is in April? 28+30=27 feb, 27+30=28 mar, 28+30=27 apr
	monthStart := time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC)
	count := recurringOccurrencesInMonth(lastDate, 30, monthStart)
	// 28 jan -> 27 feb -> 28 mar -> 27 apr -> 27 may: 1 occurrence in May
	assert.Equal(t, 1, count)
}

func TestRecurringOccurrencesInMonth_WeeklyInterval(t *testing.T) {
	// Weekly (7 days) starting 2024-01-07.
	// January 2024: 7, 14, 21, 28 = 4 occurrences.
	lastDate := time.Date(2024, 1, 7, 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	// The first occurrence at or after Jan 1 is Jan 7 (which IS in Jan).
	count := recurringOccurrencesInMonth(lastDate, 7, monthStart)
	assert.Equal(t, 4, count)
}

func TestRecurringOccurrencesInMonth_ZeroInterval(t *testing.T) {
	lastDate := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 0, recurringOccurrencesInMonth(lastDate, 0, monthStart))
}

func TestRecurringOccurrencesInMonth_LastDayOfMonth(t *testing.T) {
	// Last occurrence on Jan 31, interval 28 days → next 2024-02-28 (in Feb).
	lastDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)
	monthStart := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	assert.Equal(t, 1, recurringOccurrencesInMonth(lastDate, 28, monthStart))
}

// ---------------------------------------------------------------------------
// computeCashFlowForecast – unit tests against an in-memory DB
// ---------------------------------------------------------------------------

func TestComputeCashFlowForecast_EmptyDB(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
	db := openTestDB(t)
	forecasts := computeCashFlowForecast(db)
	// With no postings there should still be 12 months of (zero) projections.
	assert.Len(t, forecasts, cashFlowForecastMonths)
	for _, f := range forecasts {
		assert.True(t, f.Income.IsZero(), "income should be zero with no postings")
		assert.True(t, f.Expense.IsZero(), "expense should be zero with no postings")
		assert.True(t, f.Balance.IsZero(), "balance should be zero with no postings")
	}
}

func TestComputeCashFlowForecast_DatesAreConsecutiveMonths(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
	db := openTestDB(t)
	forecasts := computeCashFlowForecast(db)
	require.Len(t, forecasts, cashFlowForecastMonths)

	now := utils.Now()
	expectedStart := utils.BeginningOfMonth(now).AddDate(0, 1, 0)
	for i, f := range forecasts {
		expected := expectedStart.AddDate(0, i, 0)
		assert.Equal(t, expected.Year(), f.Date.Year(), "year mismatch at index %d", i)
		assert.Equal(t, expected.Month(), f.Date.Month(), "month mismatch at index %d", i)
		assert.Equal(t, 1, f.Date.Day(), "forecast date must be the first of the month, index %d", i)
	}
}

func TestComputeCashFlowForecast_BalanceRunningTotal(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
	db := openTestDB(t)
	forecasts := computeCashFlowForecast(db)
	require.Len(t, forecasts, cashFlowForecastMonths)

	// Verify that each balance = previous balance + income – expense.
	prev := decimal.Zero // starting balance is zero for an empty DB
	for i, f := range forecasts {
		expected := prev.Add(f.Income).Sub(f.Expense)
		assert.True(t, expected.Equal(f.Balance),
			"balance at month %d: expected %s got %s", i, expected, f.Balance)
		prev = f.Balance
	}
}

// ---------------------------------------------------------------------------
// GET /api/cash_flow/forecast – HTTP integration test
// ---------------------------------------------------------------------------

func TestHTTP_GetCashFlowForecast_Returns200(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodGet, "/api/cash_flow/forecast", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestHTTP_GetCashFlowForecast_ResponseShape(t *testing.T) {
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db"), ""))
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodGet, "/api/cash_flow/forecast", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var top map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&top))
	_, ok := top["forecasts"]
	assert.True(t, ok, "response must contain top-level \"forecasts\" key")

	var forecasts []CashFlowForecast
	require.NoError(t, json.Unmarshal(top["forecasts"], &forecasts))
	assert.Len(t, forecasts, cashFlowForecastMonths)
}
