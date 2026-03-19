package service

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and runs all migrations.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

func seedPrice(t *testing.T, db *gorm.DB, base, quote, source string, date string, val float64, ct config.CommodityType) {
	t.Helper()
	p := price.Price{
		Date:           mustParseDate(date),
		CommodityType:  ct,
		CommodityID:    base,
		CommodityName:  base,
		QuoteCommodity: quote,
		Value:          decimal.NewFromFloat(val),
		Source:         source,
	}
	require.NoError(t, db.Create(&p).Error)
}

// TestGetRate_DirectPair verifies that GetRate resolves a direct (base→quote) pair.
func TestGetRate_DirectPair(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)

	rate, ok := GetRate(db, "USD", "INR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "direct pair must be found")
	assert.True(t, decimal.NewFromFloat(83.0).Equal(rate), "direct pair rate must match seeded value")
}

// TestGetRate_InversePair verifies that GetRate resolves the rate via the inverse pair.
func TestGetRate_InversePair(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	// Only store INR→USD; GetRate(USD, INR) should compute 1/(INR→USD).
	seedPrice(t, db, "INR", "USD", "journal", "2024-01-01", 0.012, config.Unknown)

	rate, ok := GetRate(db, "USD", "INR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "inverse pair must be found")
	expected := decimal.NewFromInt(1).Div(decimal.NewFromFloat(0.012))
	diff := expected.Sub(rate).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.0001)), "inverse pair rate must be 1/(INR→USD)")
}

// TestGetRate_SameCommodity verifies that GetRate(x, x, date) always returns 1.
func TestGetRate_SameCommodity(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	rate, ok := GetRate(db, "USD", "USD", mustParseDate("2024-01-01"))
	assert.True(t, ok)
	assert.True(t, decimal.NewFromInt(1).Equal(rate))
}

// TestGetRate_NotFound verifies that GetRate returns false when no price data exists.
func TestGetRate_NotFound(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	_, ok := GetRate(db, "GBP", "JPY", mustParseDate("2024-01-01"))
	assert.False(t, ok, "must return false when no price data exists for the pair")
}

// TestGetRate_LatestOnOrBeforeDate verifies that GetRate returns the latest price
// on or before the requested date (not a future price).
func TestGetRate_LatestOnOrBeforeDate(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0, config.Unknown)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-06-01", 95.0, config.Unknown)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-12-01", 100.0, config.Unknown)

	// Query on 2024-07-01 must return the 2024-06-01 value (closest prior).
	rate, ok := GetRate(db, "EUR", "INR", mustParseDate("2024-07-01"))
	assert.True(t, ok)
	assert.True(t, decimal.NewFromFloat(95.0).Equal(rate), "must return most recent price on or before date")
}

// TestGetRate_JournalOverridesProvider verifies that when both journal and provider
// prices exist for the same (base, quote, date), the journal value is returned.
func TestGetRate_JournalOverridesProvider(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	// Provider price (lower precedence): USD→INR = 82.0
	seedPrice(t, db, "USD", "INR", "provider", "2024-03-01", 82.0, config.Stock)

	// Journal price (higher precedence): USD→INR = 83.5
	seedPrice(t, db, "USD", "INR", "journal", "2024-03-01", 83.5, config.Unknown)

	rate, ok := GetRate(db, "USD", "INR", mustParseDate("2024-03-01"))
	assert.True(t, ok)
	assert.True(t, decimal.NewFromFloat(83.5).Equal(rate), "journal price must override provider price on same date")
}

// TestGetRate_CrossRateViaAnchor verifies that GetRate resolves a one-hop cross
// rate through the default currency anchor (INR).
func TestGetRate_CrossRateViaAnchor(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	// USD→INR = 83.0, EUR→INR = 90.0  ⟹  GetRate(USD, EUR) = 83.0/90.0
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0, config.Unknown)

	rate, ok := GetRate(db, "USD", "EUR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "cross rate via anchor must be found")

	// USD/EUR = USD/INR * INR/EUR = 83.0 * (1/90.0)
	expected := decimal.NewFromFloat(83.0).Mul(decimal.NewFromInt(1).Div(decimal.NewFromFloat(90.0)))
	diff := expected.Sub(rate).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.0001)), "cross rate must equal USD/INR * (1/(EUR/INR))")
}

// TestGetRate_CrossRateDirectLegs verifies a cross rate where both legs of the
// hop are direct (base→anchor and anchor→quote both present directly).
func TestGetRate_CrossRateDirectLegs(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	// GBP→INR = 105.0, INR→JPY = 1.78  ⟹  GetRate(GBP, JPY) = 105.0 * 1.78
	seedPrice(t, db, "GBP", "INR", "journal", "2024-01-01", 105.0, config.Unknown)
	seedPrice(t, db, "INR", "JPY", "journal", "2024-01-01", 1.78, config.Unknown)

	rate, ok := GetRate(db, "GBP", "JPY", mustParseDate("2024-06-01"))
	assert.True(t, ok, "cross rate via both direct legs must be found")
	expected := decimal.NewFromFloat(105.0).Mul(decimal.NewFromFloat(1.78))
	diff := expected.Sub(rate).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.0001)), "cross rate value must be product of both legs")
}

// TestGetRate_ProviderOnlyIsUsedWhenNoJournal verifies that provider prices are
// used when no journal price exists for the pair.
func TestGetRate_ProviderOnlyIsUsedWhenNoJournal(t *testing.T) {
	db := openTestDB(t)
	ClearRateCache()

	seedPrice(t, db, "NIFTY", "INR", "provider", "2024-01-01", 21500.0, config.Stock)

	rate, ok := GetRate(db, "NIFTY", "INR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "provider-only price must still be resolved")
	assert.True(t, decimal.NewFromFloat(21500.0).Equal(rate))
}
