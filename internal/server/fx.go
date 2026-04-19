package server

import (
	"net/http"
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetFXRatesHandler(db *gorm.DB, c *gin.Context) {
	base := c.Query("base")
	quote := c.Query("quote")

	if base == "" || quote == "" {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "base and quote currencies are required")
		return
	}

	// 1. Collect all relevant dates where prices exist for this pair or its legs.
	datesMap := make(map[int64]bool)

	var prices []price.Price
	// Direct & Inverse
	db.Model(&price.Price{}).
		Where("(commodity_name = ? AND quote_commodity = ?) OR (commodity_name = ? AND quote_commodity = ?)", base, quote, quote, base).
		Distinct().Pluck("date", &prices)
	for _, p := range prices {
		datesMap[p.Date.Unix()] = true
	}

	// Cross-rate legs
	for _, anchor := range []string{"INR", "USD", "EUR"} { // Common anchors + default
		db.Model(&price.Price{}).
			Where("(commodity_name = ? AND quote_commodity = ?) OR (commodity_name = ? AND quote_commodity = ?)", base, anchor, anchor, base).
			Distinct().Pluck("date", &prices)
		for _, p := range prices {
			datesMap[p.Date.Unix()] = true
		}
		db.Model(&price.Price{}).
			Where("(commodity_name = ? AND quote_commodity = ?) OR (commodity_name = ? AND quote_commodity = ?)", quote, anchor, anchor, quote).
			Distinct().Pluck("date", &prices)
		for _, p := range prices {
			datesMap[p.Date.Unix()] = true
		}
	}

	var dates []time.Time
	for unix := range datesMap {
		dates = append(dates, time.Unix(unix, 0).UTC())
	}
	sort.Slice(dates, func(i, j int) bool { return dates[i].After(dates[j]) })

	// 2. Resolve rate for each date.
	type RateEntry struct {
		Date time.Time `json:"date"`
		service.ResolvedRate
	}
	var results []RateEntry

	for _, date := range dates {
		if res, ok := service.GetRateDetails(db, base, quote, date); ok {
			results = append(results, RateEntry{Date: date, ResolvedRate: res})
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"base":  base,
		"quote": quote,
		"rates": results,
	})
}
