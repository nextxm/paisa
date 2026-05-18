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

var (
	rcache   rateCache
	rcacheMu sync.RWMutex
)

type priceCache struct {
	sync.Once
	pricesTree        map[string]*btree.BTree
	postingPricesTree map[string]*btree.BTree
	dcPricesTree      map[string]*btree.BTree
}

var (
	pcache   priceCache
	pcacheMu sync.RWMutex
)

func loadPriceCache(db *gorm.DB) {
	pricesTree := make(map[string]*btree.BTree)
	postingPricesTree := make(map[string]*btree.BTree)

	dc := config.DefaultCurrency()
	var prices []price.Price
	if err := db.Find(&prices).Error; err != nil {
		log.Fatal(err)
	}

	for _, p := range prices {
		if p.QuoteCommodity == "" {
			p.QuoteCommodity = dc
		}

		if pricesTree[p.CommodityName] == nil {
			pricesTree[p.CommodityName] = btree.New(2)
		}
		pricesTree[p.CommodityName].ReplaceOrInsert(p)

		// postingPricesTree is used specifically for native journal prices
		// already in the default currency.
		if p.CommodityType == config.Unknown && p.QuoteCommodity == dc {
			if postingPricesTree[p.CommodityName] == nil {
				postingPricesTree[p.CommodityName] = btree.New(2)
			}
			postingPricesTree[p.CommodityName].ReplaceOrInsert(p)
		}
	}

	dcPricesTree, missingFXCount := synthesizeDefaultCurrencyPrices(db, dc, pricesTree, postingPricesTree)

	if missingFXCount > 0 {
		log.WithField("missing_fx_count", missingFXCount).Warn("Some prices could not be converted to the default currency due to missing FX rates")
	}

	pcacheMu.Lock()
	defer pcacheMu.Unlock()
	pcache.pricesTree = pricesTree
	pcache.postingPricesTree = postingPricesTree
	pcache.dcPricesTree = dcPricesTree
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
func synthesizeDefaultCurrencyPrices(db *gorm.DB, dc string, pricesTree, postingPricesTree map[string]*btree.BTree) (map[string]*btree.BTree, int) {
	dcPricesTree := make(map[string]*btree.BTree)
	missingFXCount := 0

	for commodityName, tree := range pricesTree {
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
			} else {
				missingFXCount++
			}
			return true
		})

		// If the commodity still has no entries in out, we keep the original
		// tree as a last-resort fallback (it will still contain native prices).
		if out.Len() > 0 {
			dcPricesTree[commodityName] = out
		} else {
			dcPricesTree[commodityName] = tree
		}
	}

	// Also handle postingPricesTree (specifically used for ledger cost basis)
	for commodityName, tree := range postingPricesTree {
		if dcPricesTree[commodityName] == nil {
			dcPricesTree[commodityName] = tree
		}
	}
	return dcPricesTree, missingFXCount
}

func ClearPriceCache() {
	pcacheMu.Lock()
	defer pcacheMu.Unlock()
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

	pcacheMu.RLock()
	defer pcacheMu.RUnlock()

	pt := pcache.dcPricesTree[commodity]
	if pt == nil {
		if !config.IsMissingPriceLoggingDisabled() {
			log.WithFields(log.Fields{
				"commodity": commodity,
				"date":      date.Format("2006-01-02"),
			}).Warn("Price not found, using 0")
		}
		return price.Price{}
	}

	dc := config.DefaultCurrency()
	// Provider rows can carry an intra-day timestamp for a trading date (for
	// example, Yahoo daily candles for LSE symbols). Query with end-of-day so a
	// same-calendar-day price is still found when callers pass a date at midnight.
	pivot := utils.EndOfDay(date)
	pc := utils.BTreeDescendFirstLessOrEqual(pt, price.Price{Date: pivot, QuoteCommodity: dc})
	if !pc.Value.Equal(decimal.Zero) {
		return pc
	}

	if !config.IsMissingPriceLoggingDisabled() {
		log.WithFields(log.Fields{
			"commodity": commodity,
			"date":      date.Format("2006-01-02"),
		}).Warn("Price not found, using 0")
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
	dc := config.DefaultCurrency()
	if p.Commodity == dc {
		return p.Amount
	}

	pc := GetUnitPrice(db, p.Commodity, date)
	if !pc.Value.Equal(decimal.Zero) {
		if utils.IsSameDate(p.Date, date) && !p.Amount.Equal(p.Quantity) && !p.Amount.IsZero() {
			return p.Amount
		}
		return p.Quantity.Mul(pc.Value)
	}

	return p.Amount
}

func GetPrice(db *gorm.DB, commodity string, quantity decimal.Decimal, date time.Time) decimal.Decimal {
	dc := config.DefaultCurrency()
	if commodity == dc {
		return quantity
	}

	pc := GetUnitPrice(db, commodity, date)
	if !pc.Value.Equal(decimal.Zero) {
		return quantity.Mul(pc.Value)
	}

	return quantity
}

func PopulateMarketPrice(db *gorm.DB, ps []posting.Posting) []posting.Posting {
	return PopulateMarketPriceAt(db, ps, utils.ToDate(utils.Now()))
}

func PopulateMarketPriceAt(db *gorm.DB, ps []posting.Posting, date time.Time) []posting.Posting {
	asOf := utils.EndOfDay(date)
	return lo.Map(ps, func(p posting.Posting, _ int) posting.Posting {
		p.MarketAmount = GetMarketPrice(db, p, asOf)
		return p
	})
}

// loadRateCache populates rcache.pairTrees from the database.
// Provider prices are loaded first, then journal prices are inserted on top so
// that journal values take precedence over provider values for the same date.
func loadRateCache(db *gorm.DB) {
	pairTrees := make(map[ratePairKey]*btree.BTree)

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
		if pairTrees[k] == nil {
			pairTrees[k] = btree.New(rateCacheBTreeDegree)
		}
		pairTrees[k].ReplaceOrInsert(p)
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

	rcacheMu.Lock()
	defer rcacheMu.Unlock()
	rcache.pairTrees = pairTrees
}

// ClearRateCache invalidates the pair-aware rate cache so it is rebuilt on the
// next call to GetRate.
func ClearRateCache() {
	rcacheMu.Lock()
	defer rcacheMu.Unlock()
	rcache = rateCache{}
}
