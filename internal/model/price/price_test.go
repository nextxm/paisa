package price

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite DB and runs AutoMigrate for the Price
// model only.  We cannot import the migration package here (it imports price),
// so we migrate the table directly.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Price{}))
	require.NoError(t, db.Exec("CREATE UNIQUE INDEX IF NOT EXISTS idx_prices_type_date_base_quote ON prices(commodity_type, date, commodity_name, quote_commodity)").Error)
	return db
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func seedPrice(t *testing.T, db *gorm.DB, base, quote, source, date string, val float64) {
	t.Helper()
	p := Price{
		Date:           mustParseDate(date),
		CommodityType:  config.Unknown,
		CommodityID:    base,
		CommodityName:  base,
		QuoteCommodity: quote,
		Value:          decimal.NewFromFloat(val),
		Source:         source,
	}
	require.NoError(t, db.Create(&p).Error)
}

// TestFindFiltered_NoFilter verifies that an empty filter returns all prices.
func TestFindFiltered_NoFilter(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0)

	prices, err := FindFiltered(db, PriceFilter{})
	require.NoError(t, err)
	assert.Len(t, prices, 2)
}

// TestFindFiltered_ByBase verifies that filtering by base commodity works.
func TestFindFiltered_ByBase(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0)
	seedPrice(t, db, "USD", "INR", "journal", "2024-06-01", 84.0)

	prices, err := FindFiltered(db, PriceFilter{Base: "USD"})
	require.NoError(t, err)
	assert.Len(t, prices, 2)
	for _, p := range prices {
		assert.Equal(t, "USD", p.CommodityName)
	}
}

// TestFindFiltered_ByQuote verifies that filtering by quote commodity works.
func TestFindFiltered_ByQuote(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "USD", "EUR", "journal", "2024-01-01", 0.92)

	prices, err := FindFiltered(db, PriceFilter{Quote: "EUR"})
	require.NoError(t, err)
	assert.Len(t, prices, 1)
	assert.Equal(t, "EUR", prices[0].QuoteCommodity)
}

// TestFindFiltered_BySource verifies that filtering by source works.
func TestFindFiltered_BySource(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "USD", "INR", "com-yahoo", "2024-01-02", 82.5)

	prices, err := FindFiltered(db, PriceFilter{Source: "journal"})
	require.NoError(t, err)
	assert.Len(t, prices, 1)
	assert.Equal(t, "journal", prices[0].Source)
}

// TestFindFiltered_DateRange verifies that from/to date filtering works.
func TestFindFiltered_DateRange(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2023-12-01", 81.0)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "USD", "INR", "journal", "2024-06-01", 84.0)
	seedPrice(t, db, "USD", "INR", "journal", "2025-01-01", 85.0)

	prices, err := FindFiltered(db, PriceFilter{
		From: mustParseDate("2024-01-01"),
		To:   mustParseDate("2024-12-31"),
	})
	require.NoError(t, err)
	assert.Len(t, prices, 2)
	assert.True(t, !prices[0].Date.Before(mustParseDate("2024-01-01")))
	assert.True(t, !prices[len(prices)-1].Date.After(mustParseDate("2024-12-31")))
}

// TestFindFiltered_DeterministicOrder verifies that results are ordered by
// (date ASC, commodity_name ASC, quote_commodity ASC, source ASC) using only
// rows that are valid under the production unique index.
func TestFindFiltered_DeterministicOrder(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "EUR", "provider", "2024-01-01", 0.92)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)

	prices, err := FindFiltered(db, PriceFilter{})
	require.NoError(t, err)
	assert.Len(t, prices, 3)

	// EUR before USD (commodity_name ASC), then USD/EUR before USD/INR (quote_commodity ASC).
	assert.Equal(t, "EUR", prices[0].CommodityName)
	assert.Equal(t, "USD", prices[1].CommodityName)
	assert.Equal(t, "EUR", prices[1].QuoteCommodity)
	assert.Equal(t, "USD", prices[2].CommodityName)
	assert.Equal(t, "INR", prices[2].QuoteCommodity)
}

// TestFindFiltered_BaseAndQuoteCombined verifies that combining base and quote
// filters applies both constraints simultaneously.
func TestFindFiltered_BaseAndQuoteCombined(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "USD", "EUR", "journal", "2024-01-01", 0.92)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0)

	prices, err := FindFiltered(db, PriceFilter{Base: "USD", Quote: "INR"})
	require.NoError(t, err)
	assert.Len(t, prices, 1)
	assert.Equal(t, "USD", prices[0].CommodityName)
	assert.Equal(t, "INR", prices[0].QuoteCommodity)
}

// TestFindFiltered_EmptyResult verifies that filtering with no match returns an empty slice.
func TestFindFiltered_EmptyResult(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)

	prices, err := FindFiltered(db, PriceFilter{Base: "GBP"})
	require.NoError(t, err)
	assert.Empty(t, prices)
}

// TestFindFiltered_LatestOnly verifies that latest-only mode returns one row
// per base commodity using the newest matching price.
func TestFindFiltered_LatestOnly(t *testing.T) {
	db := openTestDB(t)
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0)
	seedPrice(t, db, "USD", "INR", "journal", "2024-06-01", 84.0)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-02-01", 90.0)

	prices, err := FindFiltered(db, PriceFilter{LatestOnly: true})
	require.NoError(t, err)
	assert.Len(t, prices, 2)
	assert.Equal(t, "EUR", prices[0].CommodityName)
	assert.Equal(t, "USD", prices[1].CommodityName)
	assert.Equal(t, mustParseDate("2024-06-01"), prices[1].Date)
}

// TestUpsertAllByTypeNameAndID_DeduplicatesProviderBatch verifies that a
// single provider sync can include duplicate rows for the same DB key without
// tripping the production unique index.
func TestUpsertAllByTypeNameAndID_DeduplicatesProviderBatch(t *testing.T) {
	db := openTestDB(t)
	prices := []*Price{
		{
			Date:           mustParseDate("2003-12-01"),
			CommodityType:  config.Stock,
			CommodityID:    "USDINR=X",
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(45.70),
			Source:         "com-yahoo",
		},
		{
			Date:           mustParseDate("2003-12-01"),
			CommodityType:  config.Stock,
			CommodityID:    "USDINR=X",
			CommodityName:  "USD",
			QuoteCommodity: "INR",
			Value:          decimal.NewFromFloat(45.71),
			Source:         "com-yahoo",
		},
	}

	require.NoError(t, UpsertAllByTypeNameAndID(db, config.Stock, "QQQ", "QQQ", prices))

	var stored []Price
	require.NoError(t, db.Where("commodity_name = ? AND quote_commodity = ?", "USD", "INR").Find(&stored).Error)
	require.Len(t, stored, 1)
	assert.True(t, stored[0].Value.Equal(decimal.NewFromFloat(45.71)))
}
