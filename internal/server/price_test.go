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

type priceResponse struct {
	Prices      map[string][]priceRow `json:"prices"`
	HistoryMode string                `json:"history_mode"`
}

type priceRow struct {
	Date           string      `json:"date"`
	CommodityName  string      `json:"commodity_name"`
	QuoteCommodity string      `json:"quote_commodity"`
	Value          json.Number `json:"value"`
	Source         string      `json:"source"`
}

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

// buildPricesRouter constructs a minimal Gin engine wired to price handlers.
func buildPricesRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/price", func(c *gin.Context) {
		GetPricesHandler(db, c)
	})
	r.GET("/api/price/filters", func(c *gin.Context) {
		GetPriceFilters(db, c)
	})
	return r
}

func decodePriceResponse(t *testing.T, rec *httptest.ResponseRecorder) priceResponse {
	t.Helper()
	var body priceResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body
}

// TestGetPricesHandler_DefaultsToLatestGroupedFormat verifies that the default
// response returns one latest row per commodity in grouped format.
func TestGetPricesHandler_DefaultsToLatestGroupedFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-06-01"),
			Value: decimal.NewFromFloat(84.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-02-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodePriceResponse(t, rec)
	assert.Equal(t, "latest", body.HistoryMode)
	require.Len(t, body.Prices, 2)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "2024-06-01T00:00:00Z", body.Prices["USD"][0].Date)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 84.0, v, 0.001)
}

// TestGetPricesHandler_HistoryAllIncludesFullSeries verifies that history=all
// returns the full series for each matching base in descending date order.
func TestGetPricesHandler_HistoryAllIncludesFullSeries(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-06-01"),
			Value: decimal.NewFromFloat(84.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&history=all", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodePriceResponse(t, rec)
	assert.Equal(t, "all", body.HistoryMode)
	require.Len(t, body.Prices["USD"], 2)
	assert.Equal(t, "2024-06-01T00:00:00Z", body.Prices["USD"][0].Date)
	assert.Equal(t, "2024-01-01T00:00:00Z", body.Prices["USD"][1].Date)
}

// TestGetPricesHandler_FiltersRemainGrouped verifies that filters narrow the
// grouped response instead of switching to a flat list.
func TestGetPricesHandler_FiltersRemainGrouped(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "EUR", Date: mustParseDate("2024-02-01"),
			Value: decimal.NewFromFloat(0.92), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?quote=INR", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices, 2)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "INR", body.Prices["USD"][0].QuoteCommodity)
}

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

// TestGetPricesHandler_InvalidHistory returns 400 for unsupported history mode.
func TestGetPricesHandler_InvalidHistory(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price?history=weekly", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "history")
}

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
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "INR", body.Prices["USD"][0].QuoteCommodity)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001)
}

// TestGetPricesHandler_ReportCurrency_Converts verifies conversion when a rate
// from the price quote to the report currency is available.
func TestGetPricesHandler_ReportCurrency_Converts(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	service.ClearRateCache()

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
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "EUR", body.Prices["USD"][0].QuoteCommodity)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 0.913, v, 0.01)
}

// TestGetPricesHandler_ReportCurrency_NoRateUnchanged verifies that when no
// conversion rate is available, the price is returned unchanged.
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

	req := httptest.NewRequest(http.MethodGet, "/api/price?base=USD&report_currency=JPY", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "INR", body.Prices["USD"][0].QuoteCommodity)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001)
}

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
// when disable_multi_currency_prices is true, report_currency is ignored.
func TestGetPricesHandler_ReportCurrency_SkippedWhenFlagDisabled(t *testing.T) {
	loadServerTestConfig(t, false, true)
	db := openTestDB(t)
	service.ClearRateCache()

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
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "INR", body.Prices["USD"][0].QuoteCommodity)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 83.0, v, 0.001)
}

// TestGetPricesHandler_ReportCurrency_EnabledByDefault verifies that
// report_currency conversion is active when disable_multi_currency_prices is
// false (the default).
func TestGetPricesHandler_ReportCurrency_EnabledByDefault(t *testing.T) {
	loadServerTestConfig(t, false, false)
	db := openTestDB(t)
	service.ClearRateCache()

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
	body := decodePriceResponse(t, rec)
	require.Len(t, body.Prices["USD"], 1)
	assert.Equal(t, "EUR", body.Prices["USD"][0].QuoteCommodity)
	v, _ := body.Prices["USD"][0].Value.Float64()
	assert.InDelta(t, 0.922, v, 0.01)
}

// TestGetPriceFilters returns distinct sorted filter options for the price page.
func TestGetPriceFilters(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "USD", Date: mustParseDate("2024-01-02"),
			Value: decimal.NewFromFloat(1.08), Source: "com-yahoo"},
	})
	r := buildPricesRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/filters", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	var body struct {
		Bases   []string `json:"bases"`
		Quotes  []string `json:"quotes"`
		Sources []string `json:"sources"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	assert.Equal(t, []string{"EUR", "USD"}, body.Bases)
	assert.Equal(t, []string{"INR", "USD"}, body.Quotes)
	assert.Equal(t, []string{"com-yahoo", "journal"}, body.Sources)
}
