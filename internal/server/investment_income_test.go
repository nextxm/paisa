package server

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInvestmentIncome_GroupsByTypeHoldingAndYield(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	postings := []posting.Posting{
		{
			TransactionID: "buy-abc",
			Date:          parseDay("2024-01-10"),
			Account:       "Assets:Equity:ABC",
			Amount:        decimal.NewFromInt(1000),
			Commodity:     "INR",
		},
		{
			TransactionID: "buy-bond",
			Date:          parseDay("2024-01-10"),
			Account:       "Assets:Debt:Bond",
			Amount:        decimal.NewFromInt(2000),
			Commodity:     "INR",
		},
		{
			TransactionID: "dividend-abc",
			Date:          parseDay("2025-02-10"),
			Account:       "Income:Dividend:Equity:ABC",
			Amount:        decimal.NewFromInt(-120),
			Commodity:     "INR",
		},
		{
			TransactionID: "interest-bond",
			Date:          parseDay("2025-01-15"),
			Account:       "Income:Interest:Assets:Debt:Bond",
			Amount:        decimal.NewFromInt(-60),
			Commodity:     "INR",
		},
	}
	require.NoError(t, db.Create(&postings).Error)

	response := GetInvestmentIncome(db)
	holdings := response["holdings"].([]InvestmentIncomeHolding)
	require.Len(t, holdings, 2)

	byKey := map[string]InvestmentIncomeHolding{}
	for _, h := range holdings {
		byKey[h.Type+"|"+h.Holding] = h
	}

	dividend := byKey["Dividend|Assets:Equity:ABC"]
	assert.True(t, dividend.TotalIncome.Equal(decimal.NewFromInt(120)))
	assert.True(t, dividend.TTMIncome.Equal(decimal.NewFromInt(120)))
	assert.True(t, dividend.TTMYield.Equal(decimal.NewFromInt(12)))

	interest := byKey["Interest|Assets:Debt:Bond"]
	assert.True(t, interest.TotalIncome.Equal(decimal.NewFromInt(60)))
	assert.True(t, interest.TTMIncome.Equal(decimal.NewFromInt(60)))
	assert.True(t, interest.TTMYield.Equal(decimal.NewFromInt(3)))
}

func TestGetGain_IncludesInvestmentIncomeInTotalReturn(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	postings := []posting.Posting{
		{
			TransactionID: "buy-abc",
			Date:          parseDay("2024-01-10"),
			Account:       "Assets:Equity:ABC",
			Amount:        decimal.NewFromInt(1000),
			Commodity:     "INR",
		},
		{
			TransactionID: "dividend-abc",
			Date:          parseDay("2025-02-10"),
			Account:       "Income:Dividend:Equity:ABC",
			Amount:        decimal.NewFromInt(-100),
			Commodity:     "INR",
		},
	}
	require.NoError(t, db.Create(&postings).Error)

	response := GetGain(db)
	gains := response["gain_breakdown"].([]Gain)
	require.Len(t, gains, 1)

	gain := gains[0]
	assert.Equal(t, "Assets:Equity:ABC", gain.Account)
	assert.True(t, gain.IncomeReceived.Equal(decimal.NewFromInt(100)))
	assert.True(t, gain.TotalReturn.Equal(decimal.NewFromInt(100)))
	assert.True(t, gain.PriceAppreciation.Equal(decimal.Zero))
	assert.True(t, gain.TTMYield.Equal(decimal.NewFromInt(10)))
}
