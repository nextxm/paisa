package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// buildExportRouter constructs a minimal Gin engine wired to ExportPricesHandler.
func buildExportRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/price/export", func(c *gin.Context) {
		ExportPricesHandler(db, c)
	})
	return r
}

// ---------------------------------------------------------------------------
// Happy-path: valid format parameters
// ---------------------------------------------------------------------------

// TestExportPricesHandler_LedgerFormat verifies that ?format=ledger returns
// the expected plain-text price directives with the ledger date/syntax.
func TestExportPricesHandler_LedgerFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
	})
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=ledger", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Header().Get("Content-Type"), "text/plain")

	body, _ := io.ReadAll(rec.Body)
	// Ledger uses slash-separated dates and includes a time component.
	assert.Contains(t, string(body), "P 2024/01/01 00:00:00 USD 83 INR")
}

// TestExportPricesHandler_HLedgerFormat verifies the hledger output format.
func TestExportPricesHandler_HLedgerFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
	})
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=hledger", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	assert.Contains(t, string(body), "P 2024-01-01 USD 83 INR")
	// hledger format must NOT include a time component.
	assert.NotContains(t, string(body), "00:00:00")
}

// TestExportPricesHandler_BeancountFormat verifies the beancount output format.
func TestExportPricesHandler_BeancountFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "USD", CommodityName: "USD",
			QuoteCommodity: "INR", Date: mustParseDate("2024-01-01"),
			Value: decimal.NewFromFloat(83.0), Source: "journal"},
	})
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=beancount", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	// Beancount uses "YYYY-MM-DD price BASE VALUE QUOTE" syntax.
	assert.Contains(t, string(body), "2024-01-01 price USD 83 INR")
}

// TestExportPricesHandler_DefaultFormat_IsLedger verifies that omitting the
// format parameter defaults to ledger output.
func TestExportPricesHandler_DefaultFormat_IsLedger(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPricesIntoDB(t, db, []price.Price{
		{CommodityType: config.Unknown, CommodityID: "EUR", CommodityName: "EUR",
			QuoteCommodity: "INR", Date: mustParseDate("2024-02-01"),
			Value: decimal.NewFromFloat(90.0), Source: "journal"},
	})
	r := buildExportRouter(t, db)

	// No format query parameter → must default to ledger.
	req := httptest.NewRequest(http.MethodGet, "/api/price/export", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body, _ := io.ReadAll(rec.Body)
	// Ledger output uses slashes in the date.
	assert.Contains(t, string(body), "P 2024/02/01 00:00:00 EUR 90 INR")
}

// ---------------------------------------------------------------------------
// Filter parameters
// ---------------------------------------------------------------------------

// TestExportPricesHandler_BaseFilter verifies that the base= parameter restricts output.
func TestExportPricesHandler_BaseFilter(t *testing.T) {
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
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=hledger&base=USD", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	body := rec.Body.String()
	assert.Contains(t, body, "USD")
	assert.NotContains(t, body, "EUR")
}

// TestExportPricesHandler_EmptyResult_ReturnsEmptyBody verifies that when no
// prices match the filter, the response is 200 with an empty body.
func TestExportPricesHandler_EmptyResult_ReturnsEmptyBody(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=ledger&base=NONEXISTENT", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	assert.Empty(t, strings.TrimSpace(rec.Body.String()),
		"no matching prices → empty response body")
}

// TestExportPricesHandler_DeterministicOrder verifies that two identical
// requests produce byte-for-byte identical output (stable export).
func TestExportPricesHandler_DeterministicOrder(t *testing.T) {
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
	r := buildExportRouter(t, db)

	doRequest := func() string {
		req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=hledger", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
		return rec.Body.String()
	}

	first := doRequest()
	second := doRequest()
	assert.Equal(t, first, second,
		"repeated exports on unchanged data must produce identical output")
}

// ---------------------------------------------------------------------------
// Error cases
// ---------------------------------------------------------------------------

// TestExportPricesHandler_InvalidFormat returns 400 for an unknown format.
func TestExportPricesHandler_InvalidFormat(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?format=csv", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "csv")
}

// TestExportPricesHandler_InvalidFromDate returns 400 for a malformed 'from' date.
func TestExportPricesHandler_InvalidFromDate(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestExportPricesHandler_FromAfterTo returns 400 when from > to.
func TestExportPricesHandler_FromAfterTo(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildExportRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/price/export?from=2024-12-31&to=2024-01-01", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "from")
}

// TestExportPricesHandler_ContentDisposition verifies that the response
// includes a Content-Disposition header suggesting a file name.
func TestExportPricesHandler_ContentDisposition(t *testing.T) {
	formats := []struct {
		format string
		ext    string
	}{
		{"ledger", "ledger"},
		{"hledger", "journal"},
		{"beancount", "beancount"},
	}

	for _, tc := range formats {
		tc := tc
		t.Run(tc.format, func(t *testing.T) {
			loadTestConfig(t, false)
			db := openTestDB(t)
			r := buildExportRouter(t, db)

			req := httptest.NewRequest(http.MethodGet,
				"/api/price/export?format="+tc.format, nil)
			rec := httptest.NewRecorder()
			r.ServeHTTP(rec, req)

			require.Equal(t, http.StatusOK, rec.Code)
			cd := rec.Header().Get("Content-Disposition")
			assert.Contains(t, cd, "attachment")
			assert.Contains(t, cd, "prices."+tc.ext)
		})
	}
}
