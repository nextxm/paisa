package local

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseLocalPrices_HappyPath verifies that a well-formed JSON file is
// parsed into the expected price slice.
func TestParseLocalPrices_HappyPath(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"commodity": "MYFUND",
		"currency": "INR",
		"entries": [
			{"date": "2024-01-01", "value": "123.45"},
			{"date": "2024-02-01", "value": "124.00"}
		]
	}`)

	prices, err := parseLocalPrices(data, "MYFUND")
	require.NoError(t, err)
	require.Len(t, prices, 2)

	assert.Equal(t, "2024-01-01", prices[0].Date.UTC().Format("2006-01-02"))
	assert.True(t, prices[0].Value.Equal(decimal.NewFromFloat(123.45)))
	assert.Equal(t, "MYFUND", prices[0].CommodityName)
	assert.Equal(t, "INR", prices[0].QuoteCommodity)

	assert.Equal(t, "2024-02-01", prices[1].Date.UTC().Format("2006-01-02"))
	assert.True(t, prices[1].Value.Equal(decimal.NewFromFloat(124.00)))
}

// TestParseLocalPrices_FallbackCommodity verifies that the fallbackCommodity
// argument is used when neither the file nor the entry specifies a commodity.
func TestParseLocalPrices_FallbackCommodity(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"entries": [
			{"date": "2024-03-01", "value": "50.00", "currency": "USD"}
		]
	}`)

	prices, err := parseLocalPrices(data, "MYSTOCK")
	require.NoError(t, err)
	require.Len(t, prices, 1)

	assert.Equal(t, "MYSTOCK", prices[0].CommodityName)
	assert.Equal(t, "USD", prices[0].QuoteCommodity)
}

// TestParseLocalPrices_EntryOverrides verifies that per-entry commodity and
// currency fields override the file-level defaults.
func TestParseLocalPrices_EntryOverrides(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"commodity": "DEFAULT",
		"currency": "INR",
		"entries": [
			{"date": "2024-04-01", "value": "200.00", "commodity": "OVERRIDE", "currency": "USD"}
		]
	}`)

	prices, err := parseLocalPrices(data, "FALLBACK")
	require.NoError(t, err)
	require.Len(t, prices, 1)

	assert.Equal(t, "OVERRIDE", prices[0].CommodityName)
	assert.Equal(t, "USD", prices[0].QuoteCommodity)
}

// TestParseLocalPrices_EmptyEntries verifies that a file with no entries
// returns an empty (non-nil) slice without error.
func TestParseLocalPrices_EmptyEntries(t *testing.T) {
	data := []byte(`{"version":1,"commodity":"X","currency":"INR","entries":[]}`)
	prices, err := parseLocalPrices(data, "X")
	require.NoError(t, err)
	assert.NotNil(t, prices)
	assert.Empty(t, prices)
}

// TestParseLocalPrices_InvalidJSON verifies that malformed JSON returns an error.
func TestParseLocalPrices_InvalidJSON(t *testing.T) {
	_, err := parseLocalPrices([]byte(`not json`), "X")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local-json")
}

// TestParseLocalPrices_InvalidDate verifies that a bad date value returns an error.
func TestParseLocalPrices_InvalidDate(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"entries": [{"date": "not-a-date", "value": "1.0"}]
	}`)
	_, err := parseLocalPrices(data, "X")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local-json")
}

// TestParseLocalPrices_InvalidValue verifies that a non-numeric value returns an error.
func TestParseLocalPrices_InvalidValue(t *testing.T) {
	data := []byte(`{
		"version": 1,
		"entries": [{"date": "2024-01-01", "value": "abc"}]
	}`)
	_, err := parseLocalPrices(data, "X")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local-json")
}

// TestParseLocalPrices_UnsupportedVersion verifies that a future version
// number returns an informative error.
func TestParseLocalPrices_UnsupportedVersion(t *testing.T) {
	data := []byte(`{"version": 99, "entries": []}`)
	_, err := parseLocalPrices(data, "X")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format version")
}

// TestProviderMetadata verifies the stable Code/Label/Description values.
func TestProviderMetadata(t *testing.T) {
	p := &PriceProvider{}
	assert.Equal(t, "local-json", p.Code())
	assert.NotEmpty(t, p.Label())
	assert.NotEmpty(t, p.Description())
	assert.NotEmpty(t, p.AutoCompleteFields())
	assert.NotNil(t, p.AutoComplete(nil, "", nil))
}
