package server

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"

	"github.com/ananthakumaran/paisa/internal/cache"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// priceQueryLayout is the date format accepted by the price filter query parameters.
const priceQueryLayout = "2006-01-02"

// PriceQuery holds the optional query parameters supported by GET /api/price.
// All fields are optional; omitting all of them activates backward-compatible mode.
type PriceQuery struct {
	Base           string `form:"base"`
	Quote          string `form:"quote"`
	From           string `form:"from"`
	To             string `form:"to"`
	Source         string `form:"source"`
	ReportCurrency string `form:"report_currency"`
}

// isFiltered returns true when at least one filter or conversion parameter has
// been set.  Note that report_currency is intentionally included: specifying
// only a report_currency (with no base/quote/date/source filters) is a valid
// use-case that returns all prices converted to the requested currency, and
// that response is returned as a flat list rather than the legacy map format.
func (q PriceQuery) isFiltered() bool {
	return q.Base != "" || q.Quote != "" || q.From != "" || q.To != "" || q.Source != "" || q.ReportCurrency != ""
}

// GetPricesHandler is the unified handler for GET /api/price.
//
// Backward-compatible mode (no query parameters):
//
//	Returns {"prices": {"<commodity>": [Price, ...]}} exactly as before.
//
// Filtered mode (any query parameter present):
//
//	Returns {"prices": [Price, ...]} as a deterministically-ordered flat list.
//	Optional filters: base, quote, from (YYYY-MM-DD), to (YYYY-MM-DD), source.
//	Optional conversion: report_currency converts every price's value to the
//	requested currency using GetRate; unconvertible prices are returned unchanged.
//
// Error responses use the standard error envelope (see apierror.go).
func GetPricesHandler(db *gorm.DB, c *gin.Context) {
	var q PriceQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
		return
	}

	// No filters → backward-compatible map response.
	if !q.isFiltered() {
		c.JSON(http.StatusOK, GetPrices(db))
		return
	}

	// Build the model-layer filter, parsing date strings.
	filter := price.PriceFilter{
		Base:   q.Base,
		Quote:  q.Quote,
		Source: q.Source,
	}

	if q.From != "" {
		t, err := time.Parse(priceQueryLayout, q.From)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'from' date: expected YYYY-MM-DD format")
			return
		}
		filter.From = t
	}
	if q.To != "" {
		t, err := time.Parse(priceQueryLayout, q.To)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'to' date: expected YYYY-MM-DD format")
			return
		}
		filter.To = t
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			"'from' date must not be after 'to' date")
		return
	}

	prices, err := price.FindFiltered(db, filter)
	if err != nil {
		log.WithError(err).Error("GetPricesHandler: FindFiltered failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError,
			"failed to query prices")
		return
	}

	// Apply report-currency conversion when requested and the feature flag is on.
	if q.ReportCurrency != "" && config.IsMultiCurrencyPricesEnabled() {
		prices = convertToReportCurrency(db, prices, q.ReportCurrency)
	}

	c.JSON(http.StatusOK, gin.H{"prices": prices})
}

// convertToReportCurrency converts each price's value to the given report
// currency using GetRate.  When the price is already in the report currency no
// conversion is applied.  When a conversion rate cannot be resolved, the
// original price is kept unchanged so the caller always receives a complete
// (if unconverted) result set rather than a partial error.
func convertToReportCurrency(db *gorm.DB, prices []price.Price, reportCurrency string) []price.Price {
	result := make([]price.Price, 0, len(prices))
	for _, p := range prices {
		if p.QuoteCommodity == reportCurrency {
			result = append(result, p)
			continue
		}
		rate, ok := service.GetRate(db, p.QuoteCommodity, reportCurrency, p.Date)
		if !ok {
			// Cannot convert; include the price unchanged and log for observability.
			log.WithFields(log.Fields{
				"from": p.QuoteCommodity,
				"to":   reportCurrency,
				"date": p.Date.Format("2006-01-02"),
			}).Debug("convertToReportCurrency: no rate found, keeping original price")
			result = append(result, p)
			continue
		}
		converted := p
		converted.Value = p.Value.Mul(rate)
		converted.QuoteCommodity = reportCurrency
		result = append(result, converted)
	}
	return result
}

func GetPrices(db *gorm.DB) gin.H {
	var commodities []string
	result := db.Model(&posting.Posting{}).Where("commodity != ?", config.DefaultCurrency()).Distinct().Pluck("commodity", &commodities)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	var prices = make(map[string][]price.Price)
	for _, commodity := range commodities {
		prices[commodity] = service.GetAllPrices(db, commodity)
	}
	return gin.H{"prices": prices}
}

// GetPriceCurrencies returns the distinct quote commodities (currencies) present
// in the prices table.  This is used by the UI to populate report-currency
// selectors on valuation pages.
func GetPriceCurrencies(db *gorm.DB, c *gin.Context) {
	var currencies []string
	if err := db.Model(&price.Price{}).Distinct().Pluck("quote_commodity", &currencies).Error; err != nil {
		log.WithError(err).Error("GetPriceCurrencies: query failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to query currencies")
		return
	}

	// Filter out empty strings; sort for a deterministic response.
	filtered := make([]string, 0, len(currencies))
	for _, cur := range currencies {
		if cur != "" {
			filtered = append(filtered, cur)
		}
	}
	sort.Strings(filtered)

	c.JSON(http.StatusOK, gin.H{"currencies": filtered})
}

type AutoCompleteRequest struct {
	Provider string            `json:"provider"`
	Field    string            `json:"field"`
	Filters  map[string]string `json:"filters"`
}

func GetPriceProviders(db *gorm.DB) gin.H {
	providers := scraper.GetAllProviders()
	return gin.H{
		"providers": lo.Map(providers, func(provider price.PriceProvider, _ int) gin.H {
			return gin.H{
				"code":        provider.Code(),
				"label":       provider.Label(),
				"description": provider.Description(),
				"fields":      provider.AutoCompleteFields(),
			}
		}),
	}

}

func ClearPriceCache(db *gorm.DB) gin.H {
	err := price.DeleteAll(db)
	if err != nil {
		return gin.H{"success": false, "message": err.Error()}
	}

	cache.Clear()

	result, err := model.SyncJournal(db)
	if err != nil {
		return gin.H{"success": false, "message": result.Message}
	}

	return gin.H{"success": true}
}

func ClearPriceProviderCache(db *gorm.DB, code string) gin.H {
	provider := scraper.GetProviderByCode(code)
	provider.ClearCache(db)
	return gin.H{}
}

func GetPriceAutoCompletions(db *gorm.DB, request AutoCompleteRequest) gin.H {
	provider := scraper.GetProviderByCode(request.Provider)
	completions := provider.AutoComplete(db, request.Field, request.Filters)

	completions = lo.Filter(completions, func(completion price.AutoCompleteItem, _ int) bool {
		item := completion.Label
		item = strings.Replace(strings.ToLower(item), " ", "", -1)
		words := strings.Split(strings.ToLower(request.Filters[request.Field]), " ")
		for _, word := range words {
			if strings.TrimSpace(word) != "" && !strings.Contains(item, word) {
				return false
			}
		}
		return true
	})

	return gin.H{"completions": completions}
}

// exportFormatToExtension maps an ExportFormat to a suggested file extension.
var exportFormatToExtension = map[price.ExportFormat]string{
	price.FormatLedger:    "ledger",
	price.FormatHLedger:   "journal",
	price.FormatBeancount: "beancount",
}

// ExportPricesHandler handles GET /api/price/export.
//
// Query parameters:
//
//   - format (optional): one of "ledger" (default), "hledger", "beancount".
//   - base   (optional): filter by base commodity name.
//   - quote  (optional): filter by quote commodity.
//   - from   (optional): inclusive lower date bound, YYYY-MM-DD.
//   - to     (optional): inclusive upper date bound, YYYY-MM-DD.
//   - source (optional): filter by price source (e.g. "journal", "com-yahoo").
//
// The response is text/plain in the requested dialect, suitable for appending
// directly to a ledger/hledger/beancount journal file.
func ExportPricesHandler(db *gorm.DB, c *gin.Context) {
	// --- format parameter ---
	formatStr := c.DefaultQuery("format", "ledger")
	format := price.ExportFormat(formatStr)
	if !price.IsValidExportFormat(format) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			fmt.Sprintf("invalid format %q: must be one of ledger, hledger, beancount", formatStr))
		return
	}

	// --- optional filter parameters ---
	filter := price.PriceFilter{
		Base:   c.Query("base"),
		Quote:  c.Query("quote"),
		Source: c.Query("source"),
	}

	if fromStr := c.Query("from"); fromStr != "" {
		t, err := time.Parse(priceQueryLayout, fromStr)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'from' date: expected YYYY-MM-DD format")
			return
		}
		filter.From = t
	}
	if toStr := c.Query("to"); toStr != "" {
		t, err := time.Parse(priceQueryLayout, toStr)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'to' date: expected YYYY-MM-DD format")
			return
		}
		filter.To = t
	}
	if !filter.From.IsZero() && !filter.To.IsZero() && filter.From.After(filter.To) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			"'from' date must not be after 'to' date")
		return
	}

	// --- query DB ---
	prices, err := price.FindFiltered(db, filter)
	if err != nil {
		log.WithError(err).Error("ExportPricesHandler: FindFiltered failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError,
			"failed to query prices")
		return
	}

	// --- render ---
	text, err := price.FormatPrices(prices, format)
	if err != nil {
		log.WithError(err).Error("ExportPricesHandler: FormatPrices failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError,
			"failed to format prices")
		return
	}

	ext := exportFormatToExtension[format]
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="prices.%s"`, ext))
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(text))
}
