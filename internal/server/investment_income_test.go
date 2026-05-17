package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	response := GetInvestmentIncome(db, utils.ToDate(utils.Now()))
	holdings := response["holdings"].([]InvestmentIncomeHolding)
	require.Len(t, holdings, 2)

	assert.True(t, response["ttm_dividend"].(decimal.Decimal).Equal(decimal.NewFromInt(120)))
	assert.True(t, response["ttm_interest"].(decimal.Decimal).Equal(decimal.NewFromInt(60)))
	assert.True(t, response["ttm_distribution"].(decimal.Decimal).Equal(decimal.Zero))

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

func TestGetInvestmentIncome_AcceptsAsOfDateAndYearQueryParams(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	router := Build(db, false)
	postings := []posting.Posting{
		{
			TransactionID: "buy-abc",
			Date:          parseDay("2023-01-10"),
			Account:       "Assets:Equity:ABC",
			Amount:        decimal.NewFromInt(1000),
			Commodity:     "INR",
		},
		{
			TransactionID: "dividend-older",
			Date:          parseDay("2024-02-10"),
			Account:       "Income:Dividend:Equity:ABC",
			Amount:        decimal.NewFromInt(-100),
			Commodity:     "INR",
		},
		{
			TransactionID: "dividend-latest",
			Date:          parseDay("2025-02-10"),
			Account:       "Income:Dividend:Equity:ABC",
			Amount:        decimal.NewFromInt(-200),
			Commodity:     "INR",
		},
	}
	require.NoError(t, db.Create(&postings).Error)

	parseTTMTotal := func(target string) decimal.Decimal {
		req := httptest.NewRequest(http.MethodGet, target, nil)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)

		var response struct {
			TTMTotal decimal.Decimal `json:"ttm_total"`
		}
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		return response.TTMTotal
	}

	assert.True(t, parseTTMTotal("/api/income/investment?year=2023%20-%2024").Equal(decimal.NewFromInt(100)))
	assert.True(t, parseTTMTotal("/api/income/investment?as_of_date=2024-03-31").Equal(decimal.NewFromInt(100)))
	assert.True(t, parseTTMTotal("/api/income/investment").Equal(decimal.NewFromInt(200)))
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

func TestGetGain_IncludesInvestmentIncomeInTotalReturn_MultipleHoldings(t *testing.T) {
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
		{
			TransactionID: "buy-xyz",
			Date:          parseDay("2024-01-10"),
			Account:       "Assets:Equity:XYZ",
			Amount:        decimal.NewFromInt(2000),
			Commodity:     "INR",
		},
		{
			TransactionID: "dividend-xyz",
			Date:          parseDay("2025-02-10"),
			Account:       "Income:Dividend:Equity:XYZ",
			Amount:        decimal.NewFromInt(-300),
			Commodity:     "INR",
		},
	}
	require.NoError(t, db.Create(&postings).Error)

	response := GetGain(db)
	gains := response["gain_breakdown"].([]Gain)
	require.Len(t, gains, 2)

	var gainABC, gainXYZ Gain
	for _, g := range gains {
		if g.Account == "Assets:Equity:ABC" {
			gainABC = g
		} else if g.Account == "Assets:Equity:XYZ" {
			gainXYZ = g
		}
	}

	assert.Equal(t, "Assets:Equity:ABC", gainABC.Account)
	assert.True(t, gainABC.IncomeReceived.Equal(decimal.NewFromInt(100)))
	assert.True(t, gainABC.TotalReturn.Equal(decimal.NewFromInt(100)))

	assert.Equal(t, "Assets:Equity:XYZ", gainXYZ.Account)
	assert.True(t, gainXYZ.IncomeReceived.Equal(decimal.NewFromInt(300)))
	assert.True(t, gainXYZ.TotalReturn.Equal(decimal.NewFromInt(300)))
}

func TestGetInvestmentIncome_PrefersSnapshotWhenPresent(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	router := Build(db, false)

	// Create test postings
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

	// Build the snapshot and refresh it
	require.NoError(t, RefreshInvestmentIncomeSnapshot(db))

	// Delete postings so that live computation would return empty
	require.NoError(t, db.Exec("DELETE FROM postings").Error)

	// Call the endpoint without parameters and assert it returns the snapshot payload correctly
	req := httptest.NewRequest(http.MethodGet, "/api/income/investment", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var response struct {
		TTMTotal decimal.Decimal `json:"ttm_total"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
	assert.True(t, response.TTMTotal.Equal(decimal.NewFromInt(100)))
}
