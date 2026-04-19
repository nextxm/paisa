package service

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// ---------------------------------------------------------------------------
// Compatibility-mode tests (disable_multi_currency_prices = true)
// ---------------------------------------------------------------------------

// loadMarketTestConfig loads a minimal config with the given
// disable_multi_currency_prices value.  It restores the previous config via
// t.Cleanup.
func loadMarketTestConfig(t *testing.T, disableMultiCurrency bool) {
	t.Helper()
	orig := config.GetConfig()

	disableStr := "false"
	if disableMultiCurrency {
		disableStr = "true"
	}
	yaml := "journal_path: main.ledger\ndb_path: paisa.db\ndisable_multi_currency_prices: " + disableStr
	require.NoError(t, config.LoadConfig([]byte(yaml), ""), "loadMarketTestConfig: LoadConfig failed")

	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

// TestGetRate_CrossRate_DisabledByFlag verifies that when
// disable_multi_currency_prices is true, GetRate does NOT resolve cross-rate
// hops and returns (zero, false) for a pair that can only be resolved via an
// anchor currency.
func TestGetRate_CrossRate_DisabledByFlag(t *testing.T) {
	loadMarketTestConfig(t, true) // flag ON → multi-currency disabled
	db := openTestDB(t)
	ClearRateCache()

	// Seed legs so a cross-rate would be possible if enabled.
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0, config.Unknown)

	_, ok := GetRate(db, "USD", "EUR", mustParseDate("2024-06-01"))
	assert.False(t, ok, "cross-rate must not be resolved when disable_multi_currency_prices is true")
}

// TestGetRate_DirectPair_StillWorksWhenFlagDisabled verifies that disabling the
// multi-currency flag does not break direct/inverse pair resolution.
func TestGetRate_DirectPair_StillWorksWhenFlagDisabled(t *testing.T) {
	loadMarketTestConfig(t, true) // flag ON → multi-currency disabled
	db := openTestDB(t)
	ClearRateCache()

	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)

	rate, ok := GetRate(db, "USD", "INR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "direct pair must still resolve when disable_multi_currency_prices is true")
	assert.True(t, decimal.NewFromFloat(83.0).Equal(rate))
}

// TestGetRate_CrossRate_EnabledByDefault verifies that cross-rate resolution
// is active when disable_multi_currency_prices is false (the default).
func TestGetRate_CrossRate_EnabledByDefault(t *testing.T) {
	loadMarketTestConfig(t, false) // default behaviour: multi-currency enabled
	db := openTestDB(t)
	ClearRateCache()

	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)
	seedPrice(t, db, "EUR", "INR", "journal", "2024-01-01", 90.0, config.Unknown)

	rate, ok := GetRate(db, "USD", "EUR", mustParseDate("2024-06-01"))
	assert.True(t, ok, "cross-rate must be resolved when disable_multi_currency_prices is false")
	expected := decimal.NewFromFloat(83.0).Mul(decimal.NewFromInt(1).Div(decimal.NewFromFloat(90.0)))
	diff := expected.Sub(rate).Abs()
	assert.True(t, diff.LessThan(decimal.NewFromFloat(0.0001)))
}

// ---------------------------------------------------------------------------
// synthesizeDefaultCurrencyPrices tests
// ---------------------------------------------------------------------------

// seedPosting inserts a minimal posting record for the given commodity.
func seedPosting(t *testing.T, db *gorm.DB, commodity string) {
	t.Helper()
	p := posting.Posting{
		TransactionID: commodity + "-txn",
		Date:          mustParseDate("2024-01-01"),
		Payee:         "test",
		Account:       "Assets:Test",
		Commodity:     commodity,
		Quantity:      decimal.NewFromFloat(1),
		Amount:        decimal.NewFromFloat(100),
	}
	require.NoError(t, db.Create(&p).Error)
}

// TestGetUnitPrice_SynthesizesProviderPriceFromNativeCurrency verifies that when
// a commodity has only provider prices in a non-default currency (e.g. USD), and
// an exchange rate to the default currency (INR) is available, GetUnitPrice
// returns a price denominated in the default currency.
func TestGetUnitPrice_SynthesizesProviderPriceFromNativeCurrency(t *testing.T) {
	loadMarketTestConfig(t, false)
	db := openTestDB(t)
	ClearPriceCache()
	ClearRateCache()

	// AAPL has a provider price in USD only.
	seedPrice(t, db, "AAPL", "USD", "provider", "2024-01-01", 150.0, config.Stock)
	// Exchange rate: 1 USD = 83 INR.
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)

	pc := GetUnitPrice(db, "AAPL", mustParseDate("2024-06-01"))
	expected := decimal.NewFromFloat(150.0).Mul(decimal.NewFromFloat(83.0))
	assert.Equal(t, "INR", pc.QuoteCommodity, "synthesized price must be quoted in default currency")
	diff := expected.Sub(pc.Value).Abs()
	assert.Truef(t, diff.LessThan(decimal.NewFromFloat(0.01)),
		"synthesized price must equal native price × exchange rate, got %s want %s", pc.Value, expected)
}

// TestGetUnitPrice_NoSynthesisWhenAlreadyInDefaultCurrency verifies that when a
// commodity already has prices in the default currency, GetUnitPrice returns the
// original price unchanged.
func TestGetUnitPrice_NoSynthesisWhenAlreadyInDefaultCurrency(t *testing.T) {
	loadMarketTestConfig(t, false)
	db := openTestDB(t)
	ClearPriceCache()
	ClearRateCache()

	// NIFTY has a provider price in INR (the default currency).
	seedPrice(t, db, "NIFTY", "INR", "provider", "2024-01-01", 21500.0, config.Stock)

	pc := GetUnitPrice(db, "NIFTY", mustParseDate("2024-06-01"))
	assert.Equal(t, "INR", pc.QuoteCommodity)
	assert.True(t, decimal.NewFromFloat(21500.0).Equal(pc.Value),
		"price already in default currency must be returned unchanged")
}

// TestGetUnitPrice_PreservesNativePriceWhenNoRateAvailable verifies that when a
// commodity has only non-default-currency prices and no exchange rate is
// available, the original native price is preserved (not dropped).
func TestGetUnitPrice_PreservesNativePriceWhenNoRateAvailable(t *testing.T) {
	loadMarketTestConfig(t, false)
	db := openTestDB(t)
	ClearPriceCache()
	ClearRateCache()

	// AAPL has a provider price in USD, but no USD→INR rate is seeded.
	seedPrice(t, db, "AAPL", "USD", "provider", "2024-01-01", 150.0, config.Stock)

	pc := GetUnitPrice(db, "AAPL", mustParseDate("2024-06-01"))
	// Without a rate the original native price must be preserved.
	assert.True(t, decimal.NewFromFloat(150.0).Equal(pc.Value),
		"native price must be preserved when no exchange rate is available")
}

// TestGetUnitPrice_SynthesizesJournalPriceFromNativeCurrency verifies that when
// a commodity has only journal prices in a non-default currency (no provider
// prices) and an exchange rate is available, GetUnitPrice synthesizes a
// default-currency price.
func TestGetUnitPrice_SynthesizesJournalPriceFromNativeCurrency(t *testing.T) {
	loadMarketTestConfig(t, false)
	db := openTestDB(t)
	ClearPriceCache()
	ClearRateCache()

	// A posting for AAPL is required to trigger the journal-price loading loop.
	seedPosting(t, db, "AAPL")
	// Journal price for AAPL in USD only (no INR price).
	seedPrice(t, db, "AAPL", "USD", "journal", "2024-01-01", 150.0, config.Unknown)
	// Exchange rate: 1 USD = 83 INR.
	seedPrice(t, db, "USD", "INR", "journal", "2024-01-01", 83.0, config.Unknown)

	pc := GetUnitPrice(db, "AAPL", mustParseDate("2024-06-01"))
	expected := decimal.NewFromFloat(150.0).Mul(decimal.NewFromFloat(83.0))
	assert.Equal(t, "INR", pc.QuoteCommodity, "synthesized journal price must be quoted in default currency")
	diff := expected.Sub(pc.Value).Abs()
	assert.Truef(t, diff.LessThan(decimal.NewFromFloat(0.01)),
		"synthesized journal price must equal native price × exchange rate, got %s want %s", pc.Value, expected)
}

// TestGetAllPrices_MultiCurrency verifies that GetAllPrices returns all prices
// for a commodity, including those in different quote currencies.
func TestGetAllPrices_MultiCurrency(t *testing.T) {
	db := openTestDB(t)
	ClearPriceCache()

	// Seed prices in different currencies for the same commodity.
	seedPrice(t, db, "AAPL", "USD", "journal", "2024-01-01", 150.0, config.Unknown)
	seedPrice(t, db, "AAPL", "INR", "journal", "2024-01-01", 12500.0, config.Unknown)
	seedPrice(t, db, "AAPL", "USD", "journal", "2024-01-02", 155.0, config.Unknown)

	prices := GetAllPrices(db, "AAPL")
	assert.Len(t, prices, 3, "should return all seeded prices")

	// Order should be date DESC, quote_commodity ASC.
	assert.Equal(t, mustParseDate("2024-01-02"), prices[0].Date)
	assert.Equal(t, "USD", prices[0].QuoteCommodity)

	assert.Equal(t, mustParseDate("2024-01-01"), prices[1].Date)
	assert.Equal(t, "INR", prices[1].QuoteCommodity)

	assert.Equal(t, mustParseDate("2024-01-01"), prices[2].Date)
	assert.Equal(t, "USD", prices[2].QuoteCommodity)
}
