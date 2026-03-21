package price

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/google/btree"
	"github.com/shopspring/decimal"
)

type Price struct {
	ID             uint                 `gorm:"primaryKey" json:"id"`
	Date           time.Time            `json:"date"`
	CommodityType  config.CommodityType `json:"commodity_type"`
	CommodityID    string               `json:"commodity_id"`
	CommodityName  string               `json:"commodity_name"`
	QuoteCommodity string               `json:"quote_commodity"`
	Value          decimal.Decimal      `json:"value"`
	Source         string               `json:"source"`
}

func (p Price) Less(o btree.Item) bool {
	return p.Date.Before(o.(Price).Date)
}

func DeleteAll(db *gorm.DB) error {
	err := db.Exec("DELETE FROM prices").Error
	if err != nil {
		return err
	}
	return nil
}

// defaultQuoteCommodity returns the default_currency from config, falling back
// to "INR" if the config has not been initialised (e.g., in tests).
func defaultQuoteCommodity() string {
	dc := config.DefaultCurrency()
	if dc == "" {
		return "INR"
	}
	return dc
}

func UpsertAllByTypeNameAndID(db *gorm.DB, commodityType config.CommodityType, commodityName string, commodityID string, prices []*Price) error {
	return db.Transaction(func(tx *gorm.DB) error {
		// Build the complete set of names and IDs to delete.  Providers may
		// return companion entries (e.g., exchange-rate rows) alongside the
		// main commodity rows; including them here ensures stale companion
		// rows are removed on each resync rather than accumulating.
		nameSet := map[string]struct{}{commodityName: {}}
		idSet := map[string]struct{}{commodityID: {}}
		for _, p := range prices {
			if p.CommodityName != "" {
				nameSet[p.CommodityName] = struct{}{}
			}
			if p.CommodityID != "" {
				idSet[p.CommodityID] = struct{}{}
			}
		}
		names := make([]string, 0, len(nameSet))
		for n := range nameSet {
			names = append(names, n)
		}
		ids := make([]string, 0, len(idSet))
		for id := range idSet {
			ids = append(ids, id)
		}

		err := tx.Delete(&Price{}, "commodity_type = ? and (commodity_name IN ? or commodity_id IN ?)", commodityType, names, ids).Error
		if err != nil {
			return err
		}

		dc := defaultQuoteCommodity()
		for _, price := range prices {
			if price.QuoteCommodity == "" {
				price.QuoteCommodity = dc
			}
			err := tx.Create(price).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// deduplicatePrices returns a new slice with only the last price seen for each
// (CommodityName, Date, QuoteCommodity) triple.  Ledger CLIs that infer
// implicit prices from transaction cost annotations (e.g. hledger
// --infer-market-prices) can emit multiple identical entries for the same
// date; keeping only one is safe because they carry the same value.
func deduplicatePrices(prices []Price) []Price {
	type key struct {
		name  string
		date  time.Time
		quote string
	}
	seen := make(map[key]struct{}, len(prices))
	out := make([]Price, 0, len(prices))
	for _, p := range prices {
		k := key{p.CommodityName, p.Date, p.QuoteCommodity}
		if _, dup := seen[k]; dup {
			continue
		}
		seen[k] = struct{}{}
		out = append(out, p)
	}
	return out
}

func UpsertAllByType(db *gorm.DB, commodityType config.CommodityType, prices []Price) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := tx.Delete(&Price{}, "commodity_type = ?", commodityType).Error
		if err != nil {
			return err
		}
		dc := defaultQuoteCommodity()
		for i := range prices {
			if prices[i].QuoteCommodity == "" {
				prices[i].QuoteCommodity = dc
			}
		}
		for _, price := range deduplicatePrices(prices) {
			err := tx.Create(&price).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
}

// PriceFilter holds optional criteria for querying the prices table.
// Zero values are treated as "no constraint" for each field.
type PriceFilter struct {
	// Base filters by commodity_name (base commodity). Empty = no filter.
	Base string
	// Quote filters by quote_commodity. Empty = no filter.
	Quote string
	// From is an inclusive lower bound on the date column. Zero = no lower bound.
	From time.Time
	// To is an inclusive upper bound on the date column. Zero = no upper bound.
	To time.Time
	// Source filters by the source field ("journal" or a provider code). Empty = no filter.
	Source string
	// LatestOnly returns only the newest matching row per base commodity.
	LatestOnly bool
}

// FindFiltered queries the prices table using the given filter and returns
// results ordered deterministically by (date ASC, commodity_name ASC,
// quote_commodity ASC, source ASC).
func FindFiltered(db *gorm.DB, filter PriceFilter) ([]Price, error) {
	q := applyPriceFilter(db.Model(&Price{}), filter)
	if filter.LatestOnly {
		ranked := q.Select(`prices.*, ROW_NUMBER() OVER (
			PARTITION BY commodity_name
			ORDER BY date DESC, quote_commodity ASC, source ASC, id DESC
		) AS row_number`)
		q = db.Table("(?) as ranked_prices", ranked).
			Select("id, date, commodity_type, commodity_id, commodity_name, quote_commodity, value, source").
			Where("row_number = 1")
	}
	q = q.Order("date ASC, commodity_name ASC, quote_commodity ASC, source ASC")
	var prices []Price
	if err := q.Find(&prices).Error; err != nil {
		return nil, err
	}
	return prices, nil
}

func applyPriceFilter(q *gorm.DB, filter PriceFilter) *gorm.DB {
	if filter.Base != "" {
		q = q.Where("commodity_name = ?", filter.Base)
	}
	if filter.Quote != "" {
		q = q.Where("quote_commodity = ?", filter.Quote)
	}
	if !filter.From.IsZero() {
		q = q.Where("date >= ?", filter.From)
	}
	if !filter.To.IsZero() {
		q = q.Where("date <= ?", filter.To)
	}
	if filter.Source != "" {
		q = q.Where("source = ?", filter.Source)
	}
	return q
}

// FindByDateBaseQuote returns the most-recent price on or before date for the
// given base/quote commodity pair.  The second return value is false only when
// no matching row exists; any other database error is returned as-is.
func FindByDateBaseQuote(db *gorm.DB, date time.Time, baseCommodity, quoteCommodity string) (Price, bool, error) {
	var p Price
	result := db.Where("commodity_name = ? AND quote_commodity = ? AND date <= ?", baseCommodity, quoteCommodity, date).
		Order("date DESC").
		First(&p)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return Price{}, false, nil
		}
		return Price{}, false, result.Error
	}
	return p, true, nil
}
