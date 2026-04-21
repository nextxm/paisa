package service

import (
	"sync"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/google/btree"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// rateCacheBTreeDegree is the btree branching factor used for pair-indexed price
// trees.  A degree of 2 gives a 2-3 tree, which is a good default for
// in-memory structures with moderate data volumes.
const rateCacheBTreeDegree = 2

// ratePairKey identifies a directed currency pair for cache indexing.
type ratePairKey struct {
	Base  string
	Quote string
}

// rateCache holds pair-indexed btree structures for efficient GetRate lookups.
// It is populated lazily on first use and reset by ClearRateCache.
type rateCache struct {
	sync.Once
	pairTrees map[ratePairKey]*btree.BTree
}

var rcache rateCache

type priceCache struct {
	sync.Once
	pricesTree        map[string]*btree.BTree
	postingPricesTree map[string]*btree.BTree
	dcPricesTree      map[string]*btree.BTree
}

var pcache priceCache

func loadPriceCache(db *gorm.DB) {
	pcache.pricesTree = make(map[string]*btree.BTree)
	pcache.postingPricesTree = make(map[string]*btree.BTree)
	pcache.dcPricesTree = make(map[string]*btree.BTree)

	dc := config.DefaultCurrency()
	var prices []price.Price
	if err := db.Find(&prices).Error; err != nil {
		log.Fatal(err)
	}

	for _, p := range prices {
		if p.QuoteCommodity == "" {
			p.QuoteCommodity = dc
		}

		if pcache.pricesTree[p.CommodityName] == nil {
			pcache.pricesTree[p.CommodityName] = btree.New(2)
		}
		pcache.pricesTree[p.CommodityName].ReplaceOrInsert(p)

		// postingPricesTree is used specifically for native journal prices
		// already in the default currency.
		if p.CommodityType == config.Unknown && p.QuoteCommodity == dc {
			if pcache.postingPricesTree[p.CommodityName] == nil {
				pcache.postingPricesTree[p.CommodityName] = btree.New(2)
			}
			pcache.postingPricesTree[p.CommodityName].ReplaceOrInsert(p)
		}
	}

	synthesizeDefaultCurrencyPrices(db, dc)
}

// isDefaultCurrency reports whether quote matches the configured default
// currency dc.  A blank quote is treated as equivalent to dc because some
// legacy price rows omit the quote field and were always assumed to be
// denominated in the default currency.
func isDefaultCurrency(quote, dc string) bool {
	return quote == dc || quote == ""
}

// synthesizeDefaultCurrencyPrices iterates over every commodity in
// pcache.pricesTree and, for any tree that contains no price quoted in dc,
// builds a replacement tree whose values are expressed in dc by multiplying
// each native price by GetRate(nativeQuote, dc, date).
// Commodities that already have at least one dc-denominated price are left
// unchanged.  When no exchange rate can be resolved for a particular entry it
// is kept in the tree unchanged so that existing fallback behaviour is
// preserved.
func synthesizeDefaultCurrencyPrices(db *gorm.DB, dc string) {
	for commodityName, tree := range pcache.pricesTree {
		// Build a unified tree in the default currency.
		// Start by inserting all native prices that are already in dc.
		out := btree.New(2)
		tree.Ascend(func(item btree.Item) bool {
			p := item.(price.Price)
			if isDefaultCurrency(p.QuoteCommodity, dc) {
				out.ReplaceOrInsert(p)
			}
			return true
		})

		// Now iterate over every native price again. If it's not in dc,
		// try to synthesize a dc value.
		tree.Ascend(func(item btree.Item) bool {
			p := item.(price.Price)
			if isDefaultCurrency(p.QuoteCommodity, dc) {
				return true
			}

			rate, ok := GetRate(db, p.QuoteCommodity, dc, p.Date)
			if ok && !rate.IsZero() {
				syn := p
				syn.QuoteCommodity = dc
				syn.Value = p.Value.Mul(rate)

				// Only insert synthesized price if there isn't already a native
				// DC price for this exact date.
				if exists := out.Get(syn); exists == nil {
					out.ReplaceOrInsert(syn)
				}
			}
			return true
		})

		// If the commodity still has no entries in out, we keep the original
		// tree as a last-resort fallback (it will still contain native prices).
		if out.Len() > 0 {
			pcache.dcPricesTree[commodityName] = out
		} else {
			pcache.dcPricesTree[commodityName] = tree
		}
	}

	// Also handle postingPricesTree (specifically used for ledger cost basis)
	for commodityName, tree := range pcache.postingPricesTree {
		if pcache.dcPricesTree[commodityName] == nil {
			pcache.dcPricesTree[commodityName] = tree
		}
	}
}

func ClearPriceCache() {
	pcache = priceCache{}
}

func WarmCache(db *gorm.DB) {
	go func() {
		pcache.Do(func() { loadPriceCache(db) })
		rcache.Do(func() { loadRateCache(db) })
	}()
}

func GetUnitPrice(db *gorm.DB, commodity string, date time.Time) price.Price {
	pcache.Do(func() { loadPriceCache(db) })

	pt := pcache.dcPricesTree[commodity]
	if pt == nil {
		log.Fatal("Price not found ", commodity)
	}

	dc := config.DefaultCurrency()
	// Use a 'maximum' pivot for the date to find the latest quote currency entry.
	// However, since dcPricesTree is synthesized to be DC-only, the simple pivot works.
	pc := utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: date, QuoteCommodity: dc})
	if !pc.Value.Equal(decimal.Zero) {
		return pc
	}

	return price.Price{}
}

func GetAllPrices(db *gorm.DB, commodity string) []price.Price {
	var prices []price.Price
	if err := db.Where("commodity_name = ?", commodity).
		Order("date DESC, quote_commodity ASC").
		Find(&prices).Error; err != nil {
		log.WithError(err).Error("GetAllPrices: query failed")
		return []price.Price{}
	}
	return prices
}

func GetMarketPrice(db *gorm.DB, p posting.Posting, date time.Time) decimal.Decimal {
	if utils.IsCurrency(p.Commodity) {
		return p.Amount
	}

	pc := GetUnitPrice(db, p.Commodity, date)
	if !pc.Value.Equal(decimal.Zero) {
		if p.Date.Equal(date) && !p.Amount.Equal(p.Quantity) && !p.Amount.IsZero() {
			return p.Amount
		}
		return p.Quantity.Mul(pc.Value)
	}

	return p.Amount
}

func GetPrice(db *gorm.DB, commodity string, quantity decimal.Decimal, date time.Time) decimal.Decimal {
	if utils.IsCurrency(commodity) {
		return quantity
	}

	pc := GetUnitPrice(db, commodity, date)
	if !pc.Value.Equal(decimal.Zero) {
		return quantity.Mul(pc.Value)
	}

	return quantity
}

func PopulateMarketPrice(db *gorm.DB, ps []posting.Posting) []posting.Posting {
	date := utils.EndOfToday()
	return lo.Map(ps, func(p posting.Posting, _ int) posting.Posting {
		p.MarketAmount = GetMarketPrice(db, p, date)
		return p
	})
}

// loadRateCache populates rcache.pairTrees from the database.
// Provider prices are loaded first, then journal prices are inserted on top so
// that journal values take precedence over provider values for the same date.
func loadRateCache(db *gorm.DB) {
	rcache.pairTrees = make(map[ratePairKey]*btree.BTree)

	dc := config.DefaultCurrency()
	// Helper that inserts a price into the pair-indexed tree.
	insert := func(p price.Price) {
		quote := p.QuoteCommodity
		if quote == "" {
			quote = dc
		}
		if quote == "" {
			return
		}
		k := ratePairKey{Base: p.CommodityName, Quote: quote}
		if rcache.pairTrees[k] == nil {
			rcache.pairTrees[k] = btree.New(rateCacheBTreeDegree)
		}
		rcache.pairTrees[k].ReplaceOrInsert(p)
	}

	// Load provider prices first (lower precedence).
	var providerPrices []price.Price
	if err := db.Where("commodity_type != ?", config.Unknown).Find(&providerPrices).Error; err != nil {
		log.Fatal(err)
	}
	for _, p := range providerPrices {
		insert(p)
	}

	// Load journal prices second so they overwrite provider prices on the same date.
	var journalPrices []price.Price
	if err := db.Where("commodity_type = ?", config.Unknown).Find(&journalPrices).Error; err != nil {
		log.Fatal(err)
	}
	for _, p := range journalPrices {
		insert(p)
	}
}

// ClearRateCache invalidates the pair-aware rate cache so it is rebuilt on the
// next call to GetRate.
func ClearRateCache() {
	rcache = rateCache{}
}

// lookupDirectRate returns the most-recent price for (base → quote) on or
// before date.  It queries only the in-memory pair trees; the cache must be
// loaded before calling this function.
func lookupDirectRate(base, quote string, date time.Time) (price.Price, bool) {
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
	if base == quote {
		return decimal.NewFromInt(1), true
	}

	rcache.Do(func() { loadRateCache(db) })

	// 1. Direct pair.
	if rate, ok := lookupRateBetween(base, quote, date); ok {
		return rate, true
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
				return r1.Mul(r2), true
			}
		}
	}

	log.WithFields(log.Fields{
		"base":  base,
		"quote": quote,
		"date":  date.Format("2006-01-02"),
	}).Debug("GetRate: no price data found for pair")

	return decimal.Zero, false
}
