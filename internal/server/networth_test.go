package server

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeNetworthTimeline_FXDecomposition(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	seedPricesIntoDB(t, db, []price.Price{
		{
			CommodityType:  config.Unknown,
			CommodityID:    "USD",
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Date:           mustParseDate("2024-01-01"),
			Value:          decimal.NewFromInt(80),
			Source:         "journal",
		},
		{
			CommodityType:  config.Unknown,
			CommodityID:    "USD",
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Date:           mustParseDate("2024-01-02"),
			Value:          decimal.NewFromInt(90),
			Source:         "journal",
		},
		{
			CommodityType:  config.Unknown,
			CommodityID:    "AAPL",
			CommodityName:  "AAPL",
			QuoteCommodity: "USD",
			Date:           mustParseDate("2024-01-01"),
			Value:          decimal.NewFromInt(100),
			Source:         "journal",
		},
		{
			CommodityType:  config.Unknown,
			CommodityID:    "AAPL",
			CommodityName:  "AAPL",
			QuoteCommodity: "USD",
			Date:           mustParseDate("2024-01-02"),
			Value:          decimal.NewFromInt(110),
			Source:         "journal",
		},
	})
	service.ClearPriceCache()
	service.ClearRateCache()

	timeline := computeNetworthTimeline(db, []posting.Posting{
		{
			Date:      mustParseDate("2024-01-01"),
			Account:   "Assets:Investments:US",
			Commodity: "AAPL",
			Quantity:  decimal.NewFromInt(1),
			Amount:    decimal.NewFromInt(8000),
		},
	}, false, mustParseDate("2024-01-02"))

	require.Len(t, timeline, 2)
	current := timeline[1]

	assert.True(t, current.Contribution.Equal(decimal.NewFromInt(8000)))
	assert.True(t, current.GainAmount.Equal(decimal.NewFromInt(1900)))
	assert.True(t, current.FXImpact.Equal(decimal.NewFromInt(1000)))
	assert.True(t, current.InvestmentReturn.Equal(decimal.NewFromInt(900)))
	assert.True(
		t,
		current.Contribution.Add(current.InvestmentReturn).Add(current.FXImpact).Equal(current.BalanceAmount),
	)
}

func TestComputeCurrencyExposure_GroupsByDenomination(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	seedPricesIntoDB(t, db, []price.Price{
		{
			CommodityType:  config.Unknown,
			CommodityID:    "USD",
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Date:           mustParseDate("2024-01-02"),
			Value:          decimal.NewFromInt(90),
			Source:         "journal",
		},
		{
			CommodityType:  config.Unknown,
			CommodityID:    "AAPL",
			CommodityName:  "AAPL",
			QuoteCommodity: "USD",
			Date:           mustParseDate("2024-01-02"),
			Value:          decimal.NewFromInt(110),
			Source:         "journal",
		},
	})
	service.ClearPriceCache()
	service.ClearRateCache()

	postings := []posting.Posting{
		{
			Date:      mustParseDate("2024-01-01"),
			Account:   "Assets:Checking:INR",
			Commodity: "INR",
			Quantity:  decimal.NewFromInt(5000),
			Amount:    decimal.NewFromInt(5000),
		},
		{
			Date:      mustParseDate("2024-01-01"),
			Account:   "Assets:Checking:USD",
			Commodity: "USD",
			Quantity:  decimal.NewFromInt(100),
			Amount:    decimal.NewFromInt(8000),
		},
		{
			Date:      mustParseDate("2024-01-01"),
			Account:   "Assets:Investments:US",
			Commodity: "AAPL",
			Quantity:  decimal.NewFromInt(1),
			Amount:    decimal.NewFromInt(8000),
		},
	}

	exposures := computeCurrencyExposure(db, postings, mustParseDate("2024-01-02"))
	require.Len(t, exposures, 2)

	byCurrency := map[string]decimal.Decimal{}
	total := decimal.Zero
	for _, exposure := range exposures {
		byCurrency[exposure.Currency] = exposure.Amount
		total = total.Add(exposure.Amount)
	}

	assert.True(t, byCurrency["USD"].GreaterThan(decimal.Zero))
	assert.True(t, byCurrency["INR"].GreaterThan(decimal.Zero))
	assert.True(t, total.Equal(decimal.NewFromInt(23900)))

	totalPct := exposures[0].Percentage.Add(exposures[1].Percentage)
	assert.InDelta(t, 100, mustFloat(totalPct), 0.00001)
}

func mustFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}
