package service

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/google/btree"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// GetNativeUnitPrice returns the latest price for a commodity on or before
// date, preferring a non-default-currency quote when one exists (e.g. AAPL
// quoted in USD rather than INR).  Falls back to the default-currency price if
// no foreign-currency price is available.  Returns (value, quoteCurrency, ok).
func GetNativeUnitPrice(db *gorm.DB, commodity string, date time.Time) (decimal.Decimal, string, bool) {
	pcache.Do(func() { loadPriceCache(db) })

	pcacheMu.RLock()
	defer pcacheMu.RUnlock()

	pt := pcache.pricesTree[commodity]
	if pt == nil {
		return decimal.Zero, "", false
	}

	dc := config.DefaultCurrency()
	pivot := utils.EndOfDay(date)
	var bestNative, bestDC price.Price

	pt.Descend(func(item btree.Item) bool {
		p := item.(price.Price)
		if p.Date.After(pivot) {
			return true
		}
		if !isDefaultCurrency(p.QuoteCommodity, dc) && bestNative.Value.IsZero() {
			bestNative = p
		}
		if isDefaultCurrency(p.QuoteCommodity, dc) && bestDC.Value.IsZero() {
			bestDC = p
		}
		if !bestNative.Value.IsZero() && !bestDC.Value.IsZero() {
			return false
		}
		return true
	})

	if !bestNative.Value.IsZero() {
		return bestNative.Value, bestNative.QuoteCommodity, true
	}
	if !bestDC.Value.IsZero() {
		return bestDC.Value, dc, true
	}
	return decimal.Zero, "", false
}

// IsSecurity reports whether commodity should be treated as a financial
// instrument (stock/fund/metal/etc.) rather than a foreign-currency cash
// holding.  A commodity is a security when:
//   - it has price entries with a non-Unknown CommodityType (scraped via a
//     configured provider), or
//   - it has price entries with a QuoteCommodity that differs from the default
//     currency (e.g. AAPL priced in USD in an INR ledger).
func IsSecurity(db *gorm.DB, commodity string) bool {
	pcache.Do(func() { loadPriceCache(db) })

	pcacheMu.RLock()
	defer pcacheMu.RUnlock()

	pt := pcache.pricesTree[commodity]
	if pt == nil {
		return false
	}

	dc := config.DefaultCurrency()
	isSecurity := false
	pt.Ascend(func(item btree.Item) bool {
		p := item.(price.Price)
		if p.CommodityType != config.Unknown {
			isSecurity = true
			return false
		}
		if !isDefaultCurrency(p.QuoteCommodity, dc) {
			isSecurity = true
			return false
		}
		return true
	})
	return isSecurity
}

// IsForeignCurrency reports whether commodity is a foreign-currency cash
// holding (not a security). A commodity is a foreign currency if it appears in
// the configured currencies list (config.GetCurrencies()), which always
// includes the default currency plus any user-configured currencies.
func IsForeignCurrency(commodity string) bool {
	normalizedCommodity := strings.ToUpper(strings.TrimSpace(commodity))
	if normalizedCommodity == "" {
		return false
	}

	if normalizedCommodity == strings.ToUpper(strings.TrimSpace(config.DefaultCurrency())) {
		return false
	}
	for _, c := range config.GetCurrencies() {
		if normalizedCommodity == strings.ToUpper(strings.TrimSpace(c)) {
			return true
		}
	}
	return false
}

// FXRate represents an exchange rate with metadata indicating if it was
// resolved via a direct pair or synthesized via a one-hop cross rate.
type FXRate struct {
	Date    time.Time       `json:"date"`
	Rate    decimal.Decimal `json:"rate"`
	Derived bool            `json:"derived"`
}

// GetRate resolves the exchange rate that converts one unit of base into quote
// on the given date.  It attempts resolution in the following order:
//
//  1. Direct pair (base → quote) on or before date.
//  2. Inverse pair (quote → base) on or before date, inverted.
//  3. One-hop cross via each anchor currency: rate(base→anchor) * rate(anchor→quote).
//     (step 3 is skipped when EnableMultiCurrencyPrices is false)
//
// Journal-sourced prices take precedence over provider-sourced prices when both
// exist for the same (base, quote, date) tuple.  The result is deterministic
// for a given database state.
//
// Returns (rate, true) when a rate can be resolved, or (decimal.Zero, false)
// when no matching price data exists.
func GetRate(db *gorm.DB, base, quote string, date time.Time) (decimal.Decimal, bool) {
	fx, ok := GetFXRate(db, base, quote, date)
	return fx.Rate, ok
}

// GetFXRate is the same as GetRate but returns a full FXRate struct including
// the derived flag.
func GetFXRate(db *gorm.DB, base, quote string, date time.Time) (FXRate, bool) {
	if base == quote {
		return FXRate{Date: date, Rate: decimal.NewFromInt(1), Derived: false}, true
	}

	rcache.Do(func() { loadRateCache(db) })

	// 1. Direct pair.
	if rate, ok := lookupRateBetween(base, quote, date); ok {
		return FXRate{Date: date, Rate: rate, Derived: false}, true
	}

	// 2 & 3 are handled by lookupRateBetween already for the direct and
	// inverse cases.  Now attempt one-hop cross via each anchor currency
	// (only when the multi-currency feature flag is on).
	if config.IsMultiCurrencyPricesEnabled() {
		for _, anchor := range anchorCurrencies() {
			if anchor == base || anchor == quote {
				continue
			}
			r1, ok1 := lookupRateBetween(base, anchor, date)
			r2, ok2 := lookupRateBetween(anchor, quote, date)
			if ok1 && ok2 {
				return FXRate{Date: date, Rate: r1.Mul(r2), Derived: true}, true
			}
		}
	}

	log.WithFields(log.Fields{
		"base":  base,
		"quote": quote,
		"date":  date.Format("2006-01-02"),
	}).Debug("GetRate: no price data found for pair")

	return FXRate{}, false
}

// GetFXRates returns the daily exchange rates for the given month and pair.
func GetFXRates(db *gorm.DB, base, quote string, year, month int) []FXRate {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0).Add(-time.Nanosecond)

	var rates []FXRate
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if rate, ok := GetFXRate(db, base, quote, d); ok {
			rates = append(rates, rate)
		}
	}
	return rates
}

// anchorCurrencies returns the ordered list of intermediate currencies used
// for one-hop cross-rate resolution.  The configured default currency is
// always included; it falls back to "INR" when config is not initialised.
func anchorCurrencies() []string {
	dc := config.DefaultCurrency()
	if dc == "" {
		dc = "INR"
	}
	return []string{dc}
}

// lookupRateBetween resolves the rate from c1 to c2 by trying the direct pair
// first and then the inverse pair.  It is used both for the initial direct/inverse
// resolution in GetRate and for computing each leg of a one-hop cross rate.
func lookupRateBetween(c1, c2 string, date time.Time) (decimal.Decimal, bool) {
	if p, ok := lookupDirectRate(c1, c2, date); ok {
		return p.Value, true
	}
	if p, ok := lookupDirectRate(c2, c1, date); ok && !p.Value.IsZero() {
		return decimal.NewFromInt(1).Div(p.Value), true
	}
	return decimal.Zero, false
}

// lookupDirectRate returns the most-recent price for (base → quote) on or
// before date.  It queries only the in-memory pair trees; the cache must be
// loaded before calling this function.
func lookupDirectRate(base, quote string, date time.Time) (price.Price, bool) {
	rcacheMu.RLock()
	defer rcacheMu.RUnlock()

	k := ratePairKey{Base: base, Quote: quote}
	pt := rcache.pairTrees[k]
	if pt == nil {
		return price.Price{}, false
	}
	// For exchange rates, we use the end of the day as pivot to handle
	// fetch race conditions between commodities and FX rates on the same day.
	pivot := utils.EndOfDay(date)
	p := utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: pivot, QuoteCommodity: quote})
	if p.Value.IsZero() {
		return price.Price{}, false
	}
	return p, true
}
