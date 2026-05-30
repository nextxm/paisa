package server

import (
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type CurrencyExposure struct {
	Currency   string          `json:"currency"`
	Amount     decimal.Decimal `json:"amount"`
	Percentage decimal.Decimal `json:"percentage"`
}

func GetCurrencyExposure(db *gorm.DB) gin.H {
	postings := query.Init(db).Like("Assets:%").UntilToday().All()
	return gin.H{"currency_exposure": computeCurrencyExposure(db, postings, utils.EndOfToday())}
}

func computeCurrencyExposure(db *gorm.DB, postings []posting.Posting, asOfDate time.Time) []CurrencyExposure {
	byCommodity := make(map[string]decimal.Decimal)
	for _, p := range postings {
		byCommodity[p.Commodity] = byCommodity[p.Commodity].Add(service.GetMarketPrice(db, p, asOfDate))
	}

	byCurrency := make(map[string]decimal.Decimal)
	total := decimal.Zero
	for commodity, amount := range byCommodity {
		if amount.LessThanOrEqual(decimal.Zero) {
			continue
		}
		currency := commodityDenominationCurrency(db, commodity, asOfDate)
		if currency == "" {
			currency = config.DefaultCurrency()
		}
		byCurrency[currency] = byCurrency[currency].Add(amount)
		total = total.Add(amount)
	}

	exposures := make([]CurrencyExposure, 0, len(byCurrency))
	for currency, amount := range byCurrency {
		percentage := decimal.Zero
		if !total.IsZero() {
			percentage = amount.Div(total).Mul(decimal.NewFromInt(100))
		}
		exposures = append(exposures, CurrencyExposure{
			Currency:   currency,
			Amount:     amount,
			Percentage: percentage,
		})
	}

	sort.Slice(exposures, func(i, j int) bool {
		return exposures[i].Amount.GreaterThan(exposures[j].Amount)
	})

	return exposures
}
