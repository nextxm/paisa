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
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// priceQueryLayout is the date format accepted by the price filter query parameters.
const priceQueryLayout = "2006-01-02"

const (
	priceHistoryLatest = "latest"
	priceHistoryAll    = "all"
)

// PriceQuery holds the optional query parameters supported by GET /api/price.
// All fields are optional; omitting all of them activates backward-compatible mode.
type PriceQuery struct {
	Base           string `form:"base"`
	Quote          string `form:"quote"`
	From           string `form:"from"`
	To             string `form:"to"`
	Source         string `form:"source"`
	ReportCurrency string `form:"report_currency"`
	History        string `form:"history"`
}

func (q PriceQuery) historyMode() string {
	if q.History == "" {
		return priceHistoryLatest
	}
	return q.History
}

// GetPricesHandler is the unified handler for GET /api/price.
//
// Returns grouped prices keyed by base commodity for both default and filtered
// requests. By default, only the latest matching row per commodity is loaded;
// pass history=all to load full history for each matching commodity.
//
// Error responses use the standard error envelope (see apierror.go).
func GetPricesHandler(db *gorm.DB, c *gin.Context) {
	var q PriceQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
		return
	}
	if q.historyMode() != priceHistoryLatest && q.historyMode() != priceHistoryAll {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			"invalid 'history' value: expected latest or all")
		return
	}

	// Build the model-layer filter, parsing date strings.
	filter := price.PriceFilter{
		Base:       q.Base,
		Quote:      q.Quote,
		Source:     q.Source,
		LatestOnly: q.historyMode() != priceHistoryAll,
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

	c.JSON(http.StatusOK, gin.H{
		"prices":      groupPricesByCommodity(prices),
		"history_mode": q.historyMode(),
	})
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

func groupPricesByCommodity(prices []price.Price) map[string][]price.Price {
	grouped := make(map[string][]price.Price)
	for _, p := range prices {
		grouped[p.CommodityName] = append(grouped[p.CommodityName], p)
	}
	for commodity := range grouped {
		sort.Slice(grouped[commodity], func(i, j int) bool {
			left := grouped[commodity][i]
			right := grouped[commodity][j]
			if !left.Date.Equal(right.Date) {
				return left.Date.After(right.Date)
			}
			if left.QuoteCommodity != right.QuoteCommodity {
				return left.QuoteCommodity < right.QuoteCommodity
			}
			if left.Source != right.Source {
				return left.Source < right.Source
			}
			return left.ID > right.ID
		})
	}
	return grouped
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

// GetPriceFilters returns the distinct base commodities, quote currencies, and
// sources present in the prices table for populating the price page filters.
func GetPriceFilters(db *gorm.DB, c *gin.Context) {
	var bases []string
	if err := db.Model(&price.Price{}).Distinct().Pluck("commodity_name", &bases).Error; err != nil {
		log.WithError(err).Error("GetPriceFilters: base query failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to query price filters")
		return
	}

	var quotes []string
	if err := db.Model(&price.Price{}).Distinct().Pluck("quote_commodity", &quotes).Error; err != nil {
		log.WithError(err).Error("GetPriceFilters: quote query failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to query price filters")
		return
	}

	var sources []string
	if err := db.Model(&price.Price{}).Distinct().Pluck("source", &sources).Error; err != nil {
		log.WithError(err).Error("GetPriceFilters: source query failed")
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to query price filters")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bases":   nonEmptySortedStrings(bases),
		"quotes":  nonEmptySortedStrings(quotes),
		"sources": nonEmptySortedStrings(sources),
	})
}

func nonEmptySortedStrings(values []string) []string {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	sort.Strings(filtered)
	return filtered
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
