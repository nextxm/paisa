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
}

var pcache priceCache

func loadPriceCache(db *gorm.DB) {
	var prices []price.Price
	result := db.Where("commodity_type != ?", config.Unknown).Find(&prices)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	pcache.pricesTree = make(map[string]*btree.BTree)
	pcache.postingPricesTree = make(map[string]*btree.BTree)

	for _, price := range prices {
		if pcache.pricesTree[price.CommodityName] == nil {
			pcache.pricesTree[price.CommodityName] = btree.New(2)
		}

		pcache.pricesTree[price.CommodityName].ReplaceOrInsert(price)
	}

	var postings []posting.Posting
	result = db.Find(&postings)
	if result.Error != nil {
		log.Fatal(result.Error)
	}

	dc := config.DefaultCurrency()
	for commodityName, postings := range lo.GroupBy(postings, func(p posting.Posting) string { return p.Commodity }) {
		if !utils.IsCurrency(postings[0].Commodity) {
			result := db.Where("commodity_type = ? and commodity_name = ? and (quote_commodity = ? or quote_commodity = '')", config.Unknown, commodityName, dc).Find(&prices)
			if result.Error != nil {
				log.Fatal(result.Error)
			}

			postingPricesTree := btree.New(2)
			for _, price := range prices {
				postingPricesTree.ReplaceOrInsert(price)
			}
			pcache.postingPricesTree[commodityName] = postingPricesTree

			if pcache.pricesTree[commodityName] == nil {
				// No provider prices: load all journal prices (any quote currency)
				// so that synthesizeDefaultCurrencyPrices can convert them below.
				var allJournalPrices []price.Price
				result2 := db.Where("commodity_type = ? and commodity_name = ?", config.Unknown, commodityName).Find(&allJournalPrices)
				if result2.Error != nil {
					log.Fatal(result2.Error)
				}
				nativeTree := btree.New(2)
				for _, p := range allJournalPrices {
					nativeTree.ReplaceOrInsert(p)
				}
				if nativeTree.Len() > 0 {
					pcache.pricesTree[commodityName] = nativeTree
				} else {
					pcache.pricesTree[commodityName] = postingPricesTree
				}
			}
		}
	}

	// For commodities whose price tree has no entry in the default currency,
	// synthesize virtual default-currency prices by multiplying each native
	// price by GetRate(nativeCurrency, dc, date).  This allows GetUnitPrice to
	// always return values denominated in dc regardless of what currency the
	// underlying prices are stored in.
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
		// Check whether the tree already has any price in the default currency.
		hasDC := false
		tree.Ascend(func(item btree.Item) bool {
			p := item.(price.Price)
			if isDefaultCurrency(p.QuoteCommodity, dc) {
				hasDC = true
				return false
			}
			return true
		})
		if hasDC {
			continue
		}

		// No dc prices found: build a synthetic dc tree.
		out := btree.New(2)
		tree.Ascend(func(item btree.Item) bool {
			p := item.(price.Price)
			if isDefaultCurrency(p.QuoteCommodity, dc) {
				// Treat blank quote as already-dc; keep as-is.
				out.ReplaceOrInsert(p)
				return true
			}
			rate, ok := GetRate(db, p.QuoteCommodity, dc, p.Date)
			if ok && !rate.IsZero() {
				syn := p
				syn.QuoteCommodity = dc
				syn.Value = p.Value.Mul(rate)
				out.ReplaceOrInsert(syn)
			} else {
				// Rate unavailable: keep original entry so callers can still
				// fall back to the native price rather than getting a zero.
				out.ReplaceOrInsert(p)
			}
			return true
		})

		if out.Len() > 0 {
			pcache.pricesTree[commodityName] = out
		}
	}
}

func ClearPriceCache() {
	pcache = priceCache{}
}

// WarmCaches pre-loads the price and rate in-memory BTree indexes in the
// background so that the first API request after a startup or sync does not
// pay the full cold-start cost.  It must be called after any database write
// that changes prices or postings (i.e., after SyncJournal / SyncCommodities).
// The function resets the existing caches first so stale data is not served
// during the warm-up window.
func WarmCaches(db *gorm.DB) {
	// Reset both caches so they are rebuilt from fresh DB state.
	pcache = priceCache{}
	rcache = rateCache{}

	go func() {
		// Trigger the sync.Once initializers by calling the cheapest public
		// function that uses each cache.  The cost is paid once here in a
		// goroutine rather than blocking the next HTTP request.
		pcache.Do(func() { loadPriceCache(db) })
		rcache.Do(func() { loadRateCache(db) })
		log.Info("WarmCaches: price and rate caches are ready")
	}()
}


func GetUnitPrice(db *gorm.DB, commodity string, date time.Time) price.Price {
	pcache.Do(func() { loadPriceCache(db) })

	pt := pcache.pricesTree[commodity]
	if pt == nil {
		log.Fatal("Price not found ", commodity)
	}

	pc := utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: date})
	if !pc.Value.Equal(decimal.Zero) {
		return pc
	}

	pt = pcache.postingPricesTree[commodity]
	if pt == nil {
		log.Fatal("Price not found ", commodity)
	}
	return utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: date})

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

	// Helper that inserts a price into the pair-indexed tree.
	insert := func(p price.Price) {
		if p.QuoteCommodity == "" {
			return
		}
		k := ratePairKey{Base: p.CommodityName, Quote: p.QuoteCommodity}
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
	p := utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: date})
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
