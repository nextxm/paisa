package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// mustParseDate parses a YYYY-MM-DD string and panics on failure.
func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// seedPricesIntoDB inserts the given price rows into the DB.
func seedPricesIntoDB(t *testing.T, db *gorm.DB, prices []price.Price) {
	t.Helper()
	for i := range prices {
		require.NoError(t, db.Create(&prices[i]).Error)
	}
}

// buildPricesRouter constructs a minimal Gin engine wired to GetPricesHandler.
func buildPricesRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/price", func(c *gin.Context) {
		GetPricesHandler(db, c)
	})
	return r
}

// ---------------------------------------------------------------------------
// Backward compatibility: no query parameters → map-keyed response
// ---------------------------------------------------------------------------

// TestGetPricesHandler_NoFilters_MapFormat verifies that GET /api/price without
// any query parameters returns the legacy map-keyed format for backward compat.
func TestGetPricesHandler_NoFilters_MapFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	_, hasPrices := body["prices"]
	assert.True(t, hasPrices, "response must contain 'prices' key")

	// In compatibility mode the value must be a JSON object (map), not an array.
	var pricesObj map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body["prices"], &pricesObj),
		"unfiltered 'prices' must be a JSON object for backward compatibility")
}

// ---------------------------------------------------------------------------
// Filtered mode: any query parameter → deterministic array response
// ---------------------------------------------------------------------------

// TestGetPricesHandler_BaseFilter_ListFormat verifies that adding a base filter
// switches the response to a JSON array with only matching rows.
func TestGetPricesHandler_BaseFilter_ListFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))

	var pricesArr []json.RawMessage
	require.NoError(t, json.Unmarshal(body["prices"], &pricesArr),
		"filtered 'prices' must be a JSON array")
	assert.Len(t, pricesArr, 1, "only the USD price should be returned")
}

// TestGetPricesHandler_QuoteFilter verifies filtering by quote commodity.
func TestGetPricesHandler_QuoteFilter(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "EUR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(0.92), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?quote=EUR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []json.RawMessage
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	assert.Len(t, arr, 1)
}

// TestGetPricesHandler_SourceFilter verifies filtering by source field.
func TestGetPricesHandler_SourceFilter(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-02-01"),
			Value: decimal.NewFromFloat(82.5), Source: "com-yahoo"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?source=journal", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []json.RawMessage
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	assert.Len(t, arr, 1, "only the journal price should be returned")
}

// TestGetPricesHandler_DateRange verifies that from/to filters restrict results.
func TestGetPricesHandler_DateRange(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2023-06-01"),
			Value: decimal.NewFromFloat(81.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2025-01-01"),
			Value: decimal.NewFromFloat(85.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&from=2024-01-01&to=2024-12-31", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []json.RawMessage
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	assert.Len(t, arr, 1, "only the 2024-01-01 row should be returned")
}

// ---------------------------------------------------------------------------
// Error cases: invalid inputs return 400 INVALID_REQUEST
// ---------------------------------------------------------------------------

// TestGetPricesHandler_InvalidFromDate returns 400 for a malformed 'from' date.
func TestGetPricesHandler_InvalidFromDate(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.NotEmpty(t, detail.Message)
}

// TestGetPricesHandler_InvalidToDate returns 400 for a malformed 'to' date.
func TestGetPricesHandler_InvalidToDate(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?to=31/12/2024", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestGetPricesHandler_FromAfterTo returns 400 when from > to.
func TestGetPricesHandler_FromAfterTo(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?from=2024-12-31&to=2024-01-01", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "from")
}

// ---------------------------------------------------------------------------
// Report currency conversion
// ---------------------------------------------------------------------------

// TestGetPricesHandler_ReportCurrency_SameQuote verifies that prices already in
// the report currency are returned unchanged.
func TestGetPricesHandler_ReportCurrency_SameQuote(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=INR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []struct {
		Value          json.Number `json:"value"`
		QuoteCommodity string      `json:"quote_commodity"`
	}
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	require.Len(t, arr, 1)
	assert.Equal(t, "INR", arr[0].QuoteCommodity,
		"quote must remain INR when report_currency matches the existing quote")
	v, _ := arr[0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001)
}

// TestGetPricesHandler_ReportCurrency_Converts verifies conversion when a direct
// cross-rate from the price's quote to the report currency is available.
func TestGetPricesHandler_ReportCurrency_Converts(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	service.ClearRateCache()

	// USD→INR = 83.0 (the price we want to convert)
	// INR→EUR = 0.011 (the conversion rate; so USD→EUR ≈ 83 * 0.011 = 0.913)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "INR", CommodityName: "INR",
			QuoteCommodity: "EUR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(0.011), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=EUR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []struct {
		Value          json.Number `json:"value"`
		QuoteCommodity string      `json:"quote_commodity"`
	}
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	// Only the USD price should come back (we filter base=USD).
	require.Len(t, arr, 1)
	assert.Equal(t, "EUR", arr[0].QuoteCommodity,
		"quote must be EUR after conversion")
	v, _ := arr[0].Value.Float64()
	assert.InDelta(t, 0.913, v, 0.01,
		"converted value must be approximately 83 * 0.011")
}

// TestGetPricesHandler_ReportCurrency_NoRateUnchanged verifies that when no
// conversion rate is available, the price is returned unchanged rather than
// being dropped or causing an error.
func TestGetPricesHandler_ReportCurrency_NoRateUnchanged(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	service.ClearRateCache()

	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	// Request report_currency=JPY but no JPY rate exists → price unchanged.
	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=JPY", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []struct {
		Value          json.Number `json:"value"`
		QuoteCommodity string      `json:"quote_commodity"`
	}
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	require.Len(t, arr, 1, "price must still be present even when conversion fails")
	// Value and quote are unchanged because there was no conversion rate.
	assert.Equal(t, "INR", arr[0].QuoteCommodity,
		"quote must remain INR when no conversion rate exists")
	v, _ := arr[0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001)
}

// ---------------------------------------------------------------------------
// Compatibility-mode tests (disable_multi_currency_prices = true)
// ---------------------------------------------------------------------------

// loadServerTestConfig loads a minimal config with optional readonly and
// disable_multi_currency_prices flags, restoring the previous config via
// t.Cleanup.
func loadServerTestConfig(t *testing.T, readonly bool, disableMultiCurrency bool) {
	t.Helper()
	orig := config.GetConfig()

	readonlyStr := "false"
	if readonly {
		readonlyStr = "true"
	}
	disableStr := "false"
	if disableMultiCurrency {
		disableStr = "true"
	}
	yaml := "journal_path: main.ledger\ndb_path: paisa.db\nreadonly: " + readonlyStr +
		"\ndisable_multi_currency_prices: " + disableStr
	require.NoError(t, config.LoadConfig([]byte(yaml), ""), "loadServerTestConfig: LoadConfig failed")

	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

// TestGetPricesHandler_ReportCurrency_SkippedWhenFlagDisabled verifies that
// when disable_multi_currency_prices is true, a report_currency query param
// is silently ignored and prices are returned in their original quote currency.
func TestGetPricesHandler_ReportCurrency_SkippedWhenFlagDisabled(t *testing.T) {
	loadServerTestConfig(t, false, true) // disable_multi_currency_prices = true
	db := openTestDB(t)
	service.ClearRateCache()

	// Seed USD→INR = 83.0 and a EUR→INR rate for potential conversion.
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	// Request report_currency=EUR; with flag disabled no conversion must happen.
	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=EUR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []struct {
		Value          json.Number `json:"value"`
		QuoteCommodity string      `json:"quote_commodity"`
	}
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	require.Len(t, arr, 1)
	// Conversion is skipped → quote remains INR, value remains 83.0.
	assert.Equal(t, "INR", arr[0].QuoteCommodity,
		"quote must remain INR when disable_multi_currency_prices is true")
	v, _ := arr[0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001,
		"value must remain unconverted when disable_multi_currency_prices is true")
}

// TestGetPricesHandler_ReportCurrency_EnabledByDefault verifies that
// report_currency conversion is active when disable_multi_currency_prices is
// false (the default).
func TestGetPricesHandler_ReportCurrency_EnabledByDefault(t *testing.T) {
	loadServerTestConfig(t, false, false) // disable_multi_currency_prices = false (default)
	db := openTestDB(t)
	service.ClearRateCache()

	// Seed USD→INR = 83.0 and EUR→INR = 90.0 (so INR→EUR ≈ 0.0111).
	// Requesting report_currency=EUR should convert INR value to EUR.
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=EUR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	var arr []struct {
		Value          json.Number `json:"value"`
		QuoteCommodity string      `json:"quote_commodity"`
	}
	require.NoError(t, json.Unmarshal(body["prices"], &arr))
	require.Len(t, arr, 1)
	assert.Equal(t, "EUR", arr[0].QuoteCommodity,
		"quote must be EUR after conversion when disable_multi_currency_prices is false")
	v, _ := arr[0].Value.Float64()
	// 83 INR * (1/90 EUR/INR) ≈ 0.922
	assert.InDelta(t, 0.922, v, 0.01,
		"converted value must be approximately 83/90")
}
