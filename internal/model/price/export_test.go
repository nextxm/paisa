package price

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mustDate parses a YYYY-MM-DD string as a UTC midnight time.Time.
func mustDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func TestFormatPrices_Ledger(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-01-15"),
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(83.0),
		},
	}

	out, err := FormatPrices(prices, FormatLedger)
	require.NoError(t, err)
	assert.Equal(t, "P 2024/01/15 00:00:00 USD 83 INR\n", out)
}

func TestFormatPrices_HLedger(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-01-15"),
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(83.5),
		},
	}

	out, err := FormatPrices(prices, FormatHLedger)
	require.NoError(t, err)
	assert.Equal(t, "P 2024-01-15 USD 83.5 INR\n", out)
}

func TestFormatPrices_Beancount(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-01-15"),
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(83.0),
		},
	}

	out, err := FormatPrices(prices, FormatBeancount)
	require.NoError(t, err)
	assert.Equal(t, "2024-01-15 price USD 83 INR\n", out)
}

func TestFormatPrices_MultipleEntries_DeterministicOrder(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-01-01"),
			CommodityName:  "EUR",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(90.0),
		},
		{
			Date:           mustDate("2024-01-02"),
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(83.0),
		},
	}

	out, err := FormatPrices(prices, FormatHLedger)
	require.NoError(t, err)
	assert.Equal(t, "P 2024-01-01 EUR 90 INR\nP 2024-01-02 USD 83 INR\n", out)
}

func TestFormatPrices_EmptySlice(t *testing.T) {
	out, err := FormatPrices(nil, FormatLedger)
	require.NoError(t, err)
	assert.Equal(t, "", out)
}

func TestFormatPrices_UnknownFormat_ReturnsError(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-01-01"),
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(83.0),
		},
	}
	_, err := FormatPrices(prices, ExportFormat("unsupported"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported")
}

func TestFormatPrices_SpacesInCommodityName_Quoted_LedgerHLedger(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-03-01"),
			CommodityName:  "My Fund",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(500.25),
		},
	}

	ledgerOut, err := FormatPrices(prices, FormatLedger)
	require.NoError(t, err)
	assert.Contains(t, ledgerOut, `"My Fund"`,
		"commodity with spaces must be quoted in ledger output")

	hledgerOut, err := FormatPrices(prices, FormatHLedger)
	require.NoError(t, err)
	assert.Contains(t, hledgerOut, `"My Fund"`,
		"commodity with spaces must be quoted in hledger output")
}

func TestFormatPrices_DecimalPrecision(t *testing.T) {
	prices := []Price{
		{
			Date:           mustDate("2024-06-01"),
			CommodityName:  "BTC",
			QuoteCommodity: "USD",
			Value:          decimal.RequireFromString("60000.123456"),
		},
	}

	out, err := FormatPrices(prices, FormatHLedger)
	require.NoError(t, err)
	assert.Equal(t, "P 2024-06-01 BTC 60000.123456 USD\n", out)
}

func TestIsValidExportFormat(t *testing.T) {
	assert.True(t, IsValidExportFormat(FormatLedger))
	assert.True(t, IsValidExportFormat(FormatHLedger))
	assert.True(t, IsValidExportFormat(FormatBeancount))
	assert.False(t, IsValidExportFormat(ExportFormat("xml")))
	assert.False(t, IsValidExportFormat(ExportFormat("")))
}
