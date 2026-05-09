package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadNetworthTestConfig(t *testing.T) {
	t.Helper()
	orig := config.GetConfig()
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\n"), ""))
}

func TestComputeNetworthTimeline_WindowFunctionRunningTotals(t *testing.T) {
	loadNetworthTestConfig(t)
	utils.SetNow("2024-01-06")
	t.Cleanup(utils.UnsetNow)

	db := openTestDB(t)
	require.NoError(t, db.Create([]posting.Posting{
		{
			TransactionID: "tx-1",
			Date:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Payee:         "Salary",
			Account:       "Assets:Checking",
			Commodity:     "INR",
			Amount:        decimal.NewFromInt(1000),
			Quantity:      decimal.NewFromInt(1000),
		},
		{
			TransactionID: "tx-2",
			Date:          time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Payee:         "Broker",
			Account:       "Income:CapitalGains:Checking",
			Commodity:     "INR",
			Amount:        decimal.RequireFromString("-200.10"),
			Quantity:      decimal.RequireFromString("-200.10"),
		},
		{
			TransactionID: "tx-3",
			Date:          time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			Payee:         "Withdrawal",
			Account:       "Assets:Checking",
			Commodity:     "INR",
			Amount:        decimal.RequireFromString("-100.10"),
			Quantity:      decimal.RequireFromString("-100.10"),
		},
		{
			TransactionID: "tx-4",
			Date:          time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			Payee:         "Bank",
			Account:       "Assets:Checking",
			Commodity:     "INR",
			Amount:        decimal.NewFromInt(50),
			Quantity:      decimal.NewFromInt(50),
		},
		// Contra posting used only to classify tx-4 as interest.
		{
			TransactionID: "tx-4",
			Date:          time.Date(2024, 1, 5, 0, 0, 0, 0, time.UTC),
			Payee:         "Bank",
			Account:       "Income:Interest:Bank",
			Commodity:     "INR",
			Amount:        decimal.NewFromInt(-50),
			Quantity:      decimal.NewFromInt(-50),
		},
	}).Error)

	posts := query.Init(db).
		Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").
		UntilToday().
		All()

	timeline := computeNetworthTimeline(db, posts, false)
	require.Len(t, timeline, 6)

	// Day 5 should include interest as balance only (no investment increase).
	day5 := timeline[4]
	assert.True(t, decimal.NewFromInt(1000).Equal(day5.InvestmentAmount))
	assert.True(t, decimal.RequireFromString("300.20").Equal(day5.WithdrawalAmount))
	assert.True(t, decimal.RequireFromString("949.90").Equal(day5.BalanceAmount))
	assert.True(t, decimal.RequireFromString("250.10").Equal(day5.GainAmount))
	assert.True(t, decimal.RequireFromString("699.80").Equal(day5.NetInvestmentAmount))

	// Day 6 has no postings; running totals should carry forward.
	day6 := timeline[5]
	assert.True(t, day5.InvestmentAmount.Equal(day6.InvestmentAmount))
	assert.True(t, day5.WithdrawalAmount.Equal(day6.WithdrawalAmount))
	assert.True(t, day5.BalanceAmount.Equal(day6.BalanceAmount))
}
