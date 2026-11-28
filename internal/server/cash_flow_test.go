package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/monthly_account_balance"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestGetCashFlow_SinglePassMatchesExpectedBuckets(t *testing.T) {
	loadTestConfig(t, false)
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\ntime_zone: UTC\nchecking_accounts:\n  - Assets:Checking\n"), ""))
	db := openTestDB(t)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "t1", Date: d, Account: "Income:Salary", Commodity: "INR", Amount: decimal.NewFromInt(-10000), Quantity: decimal.NewFromInt(-10000)},
		{TransactionID: "t1", Date: d, Account: "Assets:Checking", Commodity: "INR", Amount: decimal.NewFromInt(10000), Quantity: decimal.NewFromInt(10000)},
		{TransactionID: "t2", Date: d, Account: "Expenses:Food", Commodity: "INR", Amount: decimal.NewFromInt(1500), Quantity: decimal.NewFromInt(1500)},
		{TransactionID: "t2", Date: d, Account: "Expenses:Tax:TDS", Commodity: "INR", Amount: decimal.NewFromInt(500), Quantity: decimal.NewFromInt(500)},
		{TransactionID: "t2", Date: d, Account: "Assets:Equity:NIFTY", Commodity: "INR", Amount: decimal.NewFromInt(2000), Quantity: decimal.NewFromInt(2000)},
		{TransactionID: "t2", Date: d, Account: "Liabilities:Card", Commodity: "INR", Amount: decimal.NewFromInt(-300), Quantity: decimal.NewFromInt(-300)},
		{TransactionID: "t2", Date: d, Account: "Assets:Checking", Commodity: "INR", Amount: decimal.NewFromInt(-3700), Quantity: decimal.NewFromInt(-3700)},
	})

	res := GetCashFlow(db)
	flows, ok := res["cash_flows"].([]CashFlow)
	require.True(t, ok)
	require.NotEmpty(t, flows)

	jan := flows[0]
	assert.Equal(t, "2024-01", jan.Date.Format("2006-01"))
	assert.True(t, jan.Income.Equal(decimal.NewFromInt(10000)))
	assert.True(t, jan.Expenses.Equal(decimal.NewFromInt(1500)))
	assert.True(t, jan.Tax.Equal(decimal.NewFromInt(500)))
	assert.True(t, jan.Investment.Equal(decimal.NewFromInt(2000)))
	assert.True(t, jan.Liabilities.Equal(decimal.NewFromInt(300)))
	assert.True(t, jan.Checking.Equal(decimal.NewFromInt(6300)))
	assert.True(t, jan.Balance.Equal(decimal.NewFromInt(6300)))
}

func TestGetCashFlow_UsesMonthlyAccountBalancesWhenPopulated(t *testing.T) {
	loadTestConfig(t, false)
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\ntime_zone: UTC\nchecking_accounts:\n  - Assets:Checking\n"), ""))
	db := openTestDB(t)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return monthly_account_balance.RefreshFromPostings(tx, []*posting.Posting{
			{Date: d, Account: "Income:Salary", Commodity: "INR", Amount: decimal.NewFromInt(-5000), Quantity: decimal.NewFromInt(-5000)},
			{Date: d, Account: "Assets:Checking", Commodity: "INR", Amount: decimal.NewFromInt(5000), Quantity: decimal.NewFromInt(5000)},
		})
	}))

	res := GetCashFlow(db)
	flows, ok := res["cash_flows"].([]CashFlow)
	require.True(t, ok)
	require.NotEmpty(t, flows)
	assert.True(t, flows[0].Income.Equal(decimal.NewFromInt(5000)))
	assert.True(t, flows[0].Checking.Equal(decimal.NewFromInt(5000)))
}
