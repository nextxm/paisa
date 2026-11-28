package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeStatement_ForeignCurrencyAssetPnlReconcilesNetWorth(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// Year 1 (FY 2021-22): Deposit 100 USD at 80 INR/USD (8000 INR).
	// End of FY 2021-22: 1 USD = 80 INR -> EndingBalance = 8000.
	// Year 2 (FY 2022-23): No new transactions, 1 USD = 85 INR -> EndingBalance = 8500.
	p1 := posting.Posting{
		ID:        1,
		Date:      time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Assets:Bank:USD",
		Commodity: "USD",
		Quantity:  decimal.NewFromInt(100),
		Amount:    decimal.NewFromInt(8000),
	}
	p2 := posting.Posting{
		ID:        2,
		Date:      time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Income:Salary",
		Commodity: "INR",
		Quantity:  decimal.NewFromInt(-8000),
		Amount:    decimal.NewFromInt(-8000),
	}
	p3 := posting.Posting{
		ID:        3,
		Date:      time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Assets:Cash",
		Commodity: "INR",
		Quantity:  decimal.NewFromInt(1000),
		Amount:    decimal.NewFromInt(1000),
	}
	p4 := posting.Posting{
		ID:        4,
		Date:      time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Income:Salary",
		Commodity: "INR",
		Quantity:  decimal.NewFromInt(-1000),
		Amount:    decimal.NewFromInt(-1000),
	}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)
	require.NoError(t, db.Create(&p3).Error)
	require.NoError(t, db.Create(&p4).Error)

	// Prices for USD -> INR
	require.NoError(t, db.Create(&price.Price{
		Date:          time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		CommodityName: "USD",
		Value:         decimal.NewFromInt(80),
	}).Error)
	require.NoError(t, db.Create(&price.Price{
		Date:          time.Date(2022, 6, 1, 0, 0, 0, 0, time.UTC),
		CommodityName: "USD",
		Value:         decimal.NewFromInt(85),
	}).Error)

	stmts := computeStatement(db, []posting.Posting{p1, p2, p3, p4})
	fy22, ok := stmts["2022 - 23"]
	require.True(t, ok)

	assert.Equal(t, 8000.0, fy22.StartingBalance.InexactFloat64())
	assert.Equal(t, 9500.0, fy22.EndingBalance.InexactFloat64())
	// Pnl should capture the +500 INR FX gain on the 100 USD held across the year
	assert.Equal(t, 500.0, fy22.Pnl["Assets:Bank:USD"].InexactFloat64())
}

func TestComputeStatement_ReconciliationAttributesForeignCurrencyIncomeFXTiming(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// Receive 100 USD salary on May 1 at 80 INR/USD (8000 INR), held until FY end when 1 USD = 85 INR.
	// EndingBalance = 8500 INR, Pnl = +500 INR, Income (valued at FY-end 85) = +8500 INR.
	// Residual = 8500 - (8500 + 500) = -500 INR, attributed to FX Timing: Income:Salary.
	p1 := posting.Posting{
		ID:        1,
		Date:      time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Assets:Bank:USD",
		Commodity: "USD",
		Quantity:  decimal.NewFromInt(100),
		Amount:    decimal.NewFromInt(8000),
	}
	p2 := posting.Posting{
		ID:        2,
		Date:      time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		Account:   "Income:Salary",
		Commodity: "USD",
		Quantity:  decimal.NewFromInt(-100),
		Amount:    decimal.NewFromInt(-8000),
	}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)
	require.NoError(t, db.Create(&price.Price{
		Date:          time.Date(2021, 5, 1, 0, 0, 0, 0, time.UTC),
		CommodityName: "USD",
		Value:         decimal.NewFromInt(80),
	}).Error)
	require.NoError(t, db.Create(&price.Price{
		Date:          time.Date(2022, 3, 31, 0, 0, 0, 0, time.UTC),
		CommodityName: "USD",
		Value:         decimal.NewFromInt(85),
	}).Error)

	stmts := computeStatement(db, []posting.Posting{p1, p2})
	fy21, ok := stmts["2021 - 22"]
	require.True(t, ok)

	assert.Equal(t, 8500.0, fy21.EndingBalance.InexactFloat64())
	assert.Equal(t, -500.0, fy21.Reconciliation["FX Timing: Income:Salary"].InexactFloat64())
}
