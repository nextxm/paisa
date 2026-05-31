package drawdown

import (
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	c "github.com/ananthakumaran/paisa/internal/model/commodity"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/taxation"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type DrawdownBucket struct {
	AccountGlob         string                 `json:"account_glob"`
	TaxCategory         config.TaxCategoryType `json:"tax_category"`
	HoldingPeriodMonths int                    `json:"holding_period_months"`
}

type Recommendation struct {
	Account             string                 `json:"account"`
	Commodity           string                 `json:"commodity"`
	TaxCategory         config.TaxCategoryType `json:"tax_category"`
	Units               decimal.Decimal        `json:"units"`
	Amount              decimal.Decimal        `json:"amount"`
	EstimatedTax        taxation.Tax           `json:"estimated_tax"`
	HoldingPeriodDays   int                    `json:"holding_period_days"`
	HoldingPeriodMonths int                    `json:"holding_period_months"`
	PurchaseDate        time.Time              `json:"purchase_date"`
	CurrentUnitPrice    decimal.Decimal        `json:"current_unit_price"`
	EffectiveTaxRate    decimal.Decimal        `json:"effective_tax_rate"`
}

type Analysis struct {
	RequestedAmount   decimal.Decimal  `json:"requested_amount"`
	Recommended       decimal.Decimal  `json:"recommended_amount"`
	RemainingAmount   decimal.Decimal  `json:"remaining_amount"`
	TotalEstimatedTax taxation.Tax     `json:"total_estimated_tax"`
	Recommendations   []Recommendation `json:"recommendations"`
}

func Analyze(db *gorm.DB, amount decimal.Decimal, buckets []DrawdownBucket) Analysis {
	analysis := Analysis{
		RequestedAmount: amount,
		RemainingAmount: amount,
		Recommendations: []Recommendation{},
	}
	if !amount.GreaterThan(decimal.Zero) {
		return analysis
	}

	availableLots := collectAvailableLots(db, buckets)
	remaining := amount
	totalTax := taxation.Tax{}
	recommended := decimal.Zero

	for _, lot := range availableLots {
		if !remaining.GreaterThan(decimal.Zero) {
			break
		}

		currentValue := lot.CurrentUnitPrice.Mul(lot.Posting.Quantity)
		if !currentValue.GreaterThan(decimal.Zero) {
			continue
		}

		quantity := lot.Posting.Quantity
		amountToUse := currentValue
		if currentValue.GreaterThan(remaining) {
			quantity = remaining.Div(lot.CurrentUnitPrice)
			amountToUse = remaining
		}

		tax := taxation.EstimateSale(db, lot.Posting, lot.Commodity, quantity, lot.CurrentUnitPrice, lot.PriceDate)
		recommendation := Recommendation{
			Account:             lot.Posting.Account,
			Commodity:           lot.Posting.Commodity,
			TaxCategory:         lot.Commodity.TaxCategory,
			Units:               quantity,
			Amount:              amountToUse.Round(2),
			EstimatedTax:        tax,
			HoldingPeriodDays:   int(lot.PriceDate.Sub(lot.Posting.Date).Hours() / 24),
			HoldingPeriodMonths: monthsBetween(lot.Posting.Date, lot.PriceDate),
			PurchaseDate:        lot.Posting.Date,
			CurrentUnitPrice:    lot.CurrentUnitPrice,
			EffectiveTaxRate:    effectiveTaxRate(tax, amountToUse),
		}

		analysis.Recommendations = append(analysis.Recommendations, recommendation)
		totalTax = taxation.Add(totalTax, tax)
		recommended = recommended.Add(amountToUse)
		remaining = remaining.Sub(amountToUse)
	}

	analysis.Recommended = recommended.Round(2)
	if remaining.GreaterThan(decimal.Zero) {
		analysis.RemainingAmount = remaining.Round(2)
	} else {
		analysis.RemainingAmount = decimal.Zero
	}
	analysis.TotalEstimatedTax = totalTax
	return analysis
}

type availableLot struct {
	Posting          posting.Posting
	Commodity        config.Commodity
	CurrentUnitPrice decimal.Decimal
	PriceDate        time.Time
	SortTaxRate      decimal.Decimal
	SortTaxAmount    decimal.Decimal
	HoldingDays      int
}

func collectAvailableLots(db *gorm.DB, buckets []DrawdownBucket) []availableLot {
	commodities := c.All()
	eligible := make([]config.Commodity, 0, len(commodities))
	for _, commodity := range commodities {
		if isSupportedTaxCategory(commodity.TaxCategory) {
			eligible = append(eligible, commodity)
		}
	}

	postings := query.Init(db).Like("Assets:%").Commodities(eligible).All()
	byAccount := map[string][]posting.Posting{}
	for _, p := range postings {
		if p.Quantity.GreaterThan(decimal.Zero) && matchesBucket(p.Account, buckets) {
			byAccount[p.Account] = append(byAccount[p.Account], p)
		}
	}

	priceDate := utils.EndOfToday()
	lots := []availableLot{}
	for _, accountLots := range byAccount {
		fifo := accounting.FIFO(accountLots)
		if len(fifo) == 0 {
			continue
		}
		commodity := c.FindByName(fifo[0].Commodity)
		currentPrice := service.GetUnitPrice(db, commodity.Name, priceDate)
		if !currentPrice.Value.GreaterThan(decimal.Zero) {
			continue
		}
		for _, lot := range fifo {
			tax := taxation.EstimateSale(db, lot, commodity, lot.Quantity, currentPrice.Value, currentPrice.Date)
			currentValue := currentPrice.Value.Mul(lot.Quantity)
			lots = append(lots, availableLot{
				Posting:          lot,
				Commodity:        commodity,
				CurrentUnitPrice: currentPrice.Value,
				PriceDate:        currentPrice.Date,
				SortTaxRate:      effectiveTaxRate(tax, currentValue),
				SortTaxAmount:    totalTaxAmount(tax),
				HoldingDays:      int(currentPrice.Date.Sub(lot.Date).Hours() / 24),
			})
		}
	}

	sort.Slice(lots, func(i, j int) bool {
		if !lots[i].SortTaxRate.Equal(lots[j].SortTaxRate) {
			return lots[i].SortTaxRate.LessThan(lots[j].SortTaxRate)
		}
		if !lots[i].SortTaxAmount.Equal(lots[j].SortTaxAmount) {
			return lots[i].SortTaxAmount.LessThan(lots[j].SortTaxAmount)
		}
		return lots[i].HoldingDays > lots[j].HoldingDays
	})

	return lots
}

func effectiveTaxRate(tax taxation.Tax, amount decimal.Decimal) decimal.Decimal {
	if !amount.GreaterThan(decimal.Zero) {
		return decimal.Zero
	}
	return totalTaxAmount(tax).Div(amount)
}

func totalTaxAmount(tax taxation.Tax) decimal.Decimal {
	return tax.ShortTerm.Add(tax.LongTerm).Add(tax.Slab)
}

func monthsBetween(start, end time.Time) int {
	return (end.Year()-start.Year())*12 + int(end.Month()-start.Month())
}

func isSupportedTaxCategory(category config.TaxCategoryType) bool {
	return category == config.Debt || category == config.Equity || category == config.Equity65 || category == config.Equity35 || category == config.UnlistedEquity
}

func matchesBucket(account string, buckets []DrawdownBucket) bool {
	if len(buckets) == 0 {
		return true
	}
	for _, bucket := range buckets {
		if bucket.AccountGlob != "" && globMatch(account, bucket.AccountGlob) {
			return true
		}
	}
	return false
}

func globMatch(account, glob string) bool {
	if glob == "" || glob == "*" {
		return true
	}
	if strings.HasSuffix(glob, "*") {
		return strings.HasPrefix(account, strings.TrimSuffix(glob, "*"))
	}
	return account == glob
}
