package server

import (
	"net/http"
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

	// Apply report-currency conversion when requested.
	if q.ReportCurrency != "" {
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
			// Cannot convert; include the price unchanged.
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
