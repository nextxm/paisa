package price

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	dbutil "github.com/ananthakumaran/paisa/internal/db"
	sqlcdb "github.com/ananthakumaran/paisa/internal/db/sqlc"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/google/btree"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
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
	b := o.(Price)
	if !p.Date.Equal(b.Date) {
		return p.Date.Before(b.Date)
	}
	return p.QuoteCommodity < b.QuoteCommodity
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

func upsertPriceParams(price *Price) sqlcdb.UpsertPriceParams {
	return sqlcdb.UpsertPriceParams{
		Date:           dbutil.NullTime(price.Date),
		CommodityType:  price.CommodityType,
		CommodityID:    dbutil.NullString(price.CommodityID),
		CommodityName:  dbutil.NullString(price.CommodityName),
		QuoteCommodity: dbutil.NullString(price.QuoteCommodity),
		Value:          price.Value,
		Source:         dbutil.NullString(price.Source),
	}
}

func UpsertAllByTypeNameAndID(db *gorm.DB, commodityType config.CommodityType, commodityName string, commodityID string, prices []*Price) error {
	return db.Transaction(func(tx *gorm.DB) error {
		queries := dbutil.Queries(tx)
		dc := defaultQuoteCommodity()
		for _, price := range deduplicatePricePointers(prices) {
			if price.QuoteCommodity == "" {
				price.QuoteCommodity = dc
			}
			if err := queries.UpsertPrice(context.Background(), upsertPriceParams(price)); err != nil {
				return err
			}
		}

		return nil
	})
}

// FilterSince returns only the prices whose date falls on or after the
// start-of-day (UTC) of since.  When since is the zero time all prices are
// returned unchanged, so callers may pass a zero value to disable filtering.
func FilterSince(prices []*Price, since time.Time) []*Price {
	if since.IsZero() {
		return prices
	}
	sinceDay := since.UTC().Truncate(24 * time.Hour)
	out := make([]*Price, 0, len(prices))
	for _, p := range prices {
		// !Before is equivalent to (After || Equal): include prices on sinceDay
		// itself as well as any later date.
		if !p.Date.UTC().Before(sinceDay) {
			out = append(out, p)
		}
	}
	return out
}

// deduplicatePricePointers returns a new slice with only the last price seen
// for each unique DB key tuple. Providers can emit duplicate rows for the same
// (commodity_type, date, commodity_name, quote_commodity) combination in a
// single fetch; keeping only the last row avoids violating the unique index
// while preserving the latest provider value.
func deduplicatePricePointers(prices []*Price) []*Price {
	type key struct {
		commodityType config.CommodityType
		name          string
		unix          int64
		quote         string
	}
	indexByKey := make(map[key]int, len(prices))
	out := make([]*Price, 0, len(prices))
	for _, p := range prices {
		p.Date = p.Date.UTC().Truncate(time.Second)
		k := key{p.CommodityType, p.CommodityName, p.Date.Unix(), p.QuoteCommodity}
		if idx, ok := indexByKey[k]; ok {
			out[idx] = p
			continue
		}
		indexByKey[k] = len(out)
		out = append(out, p)
	}
	return out
}

// deduplicatePrices returns a new slice with only the last price seen for each
// DB key tuple. Ledger CLIs that infer implicit prices from transaction cost
// annotations (e.g. hledger --infer-market-prices) can emit multiple
// identical entries for the same date; keeping only the last one avoids
// duplicate inserts while preserving the final observed value.
func deduplicatePrices(prices []Price) []Price {
	type key struct {
		commodityType config.CommodityType
		name          string
		unix          int64
		quote         string
	}
	indexByKey := make(map[key]int, len(prices))
	out := make([]Price, 0, len(prices))
	for i := range prices {
		prices[i].Date = prices[i].Date.UTC().Truncate(time.Second)
		k := key{prices[i].CommodityType, prices[i].CommodityName, prices[i].Date.Unix(), prices[i].QuoteCommodity}
		if idx, dup := indexByKey[k]; dup {
			out[idx] = prices[i]
			continue
		}
		indexByKey[k] = len(out)
		out = append(out, prices[i])
	}
	return out
}

func UpsertAllByType(db *gorm.DB, commodityType config.CommodityType, prices []Price) error {
	return db.Transaction(func(tx *gorm.DB) error {
		queries := dbutil.Queries(tx)
		if err := queries.DeletePricesByType(context.Background(), commodityType); err != nil {
			return err
		}
		dc := defaultQuoteCommodity()
		for i := range prices {
			if prices[i].QuoteCommodity == "" {
				prices[i].QuoteCommodity = dc
			}
		}
		for _, price := range deduplicatePrices(prices) {
			if err := queries.UpsertPrice(context.Background(), upsertPriceParams(&price)); err != nil {
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

func listPricesParams(filter PriceFilter) sqlcdb.ListPricesParams {
	return sqlcdb.ListPricesParams{
		Column1: filter.Base,
		Column2: filter.Quote,
		Column3: filter.Source,
		Column4: dbutil.BoolFlag(!filter.From.IsZero()),
		Date:    dbutil.NullTime(filter.From),
		Column6: dbutil.BoolFlag(!filter.To.IsZero()),
		Date_2:  dbutil.NullTime(filter.To),
	}
}

func listLatestPricesParams(filter PriceFilter) sqlcdb.ListLatestPricesParams {
	return sqlcdb.ListLatestPricesParams{
		Column1: filter.Base,
		Column2: filter.Quote,
		Column3: filter.Source,
		Column4: dbutil.BoolFlag(!filter.From.IsZero()),
		Date:    dbutil.NullTime(filter.From),
		Column6: dbutil.BoolFlag(!filter.To.IsZero()),
		Date_2:  dbutil.NullTime(filter.To),
	}
}

func mapPrice(row sqlcdb.Price) Price {
	price := Price{
		ID:             uint(row.ID),
		CommodityType:  row.CommodityType,
		CommodityID:    row.CommodityID.String,
		CommodityName:  row.CommodityName.String,
		QuoteCommodity: row.QuoteCommodity.String,
		Value:          row.Value,
		Source:         row.Source.String,
	}
	if row.Date.Valid {
		price.Date = utils.ToDate(row.Date.Time)
	}
	return price
}

// FindFiltered queries the prices table using the given filter and returns
// results ordered deterministically by (date ASC, commodity_name ASC,
// quote_commodity ASC, source ASC).
func FindFiltered(db *gorm.DB, filter PriceFilter) ([]Price, error) {
	queries := dbutil.Queries(db)
	var (
		rows []sqlcdb.Price
		err  error
	)
	if filter.LatestOnly {
		rows, err = queries.ListLatestPrices(context.Background(), listLatestPricesParams(filter))
	} else {
		rows, err = queries.ListPrices(context.Background(), listPricesParams(filter))
	}
	if err != nil {
		return nil, err
	}
	prices := make([]Price, 0, len(rows))
	for _, row := range rows {
		prices = append(prices, mapPrice(row))
	}
	return prices, nil
}

// FindByDateBaseQuote returns the most-recent price on or before date for the
// given base/quote commodity pair.  The second return value is false only when
// no matching row exists; any other database error is returned as-is.
func FindByDateBaseQuote(db *gorm.DB, date time.Time, baseCommodity, quoteCommodity string) (Price, bool, error) {
	row, err := dbutil.Queries(db).FindPriceByDateBaseQuote(context.Background(), sqlcdb.FindPriceByDateBaseQuoteParams{
		BaseCommodity:  dbutil.NullString(baseCommodity),
		QuoteCommodity: dbutil.NullString(quoteCommodity),
		AtOrBefore:     dbutil.NullTime(date),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Price{}, false, nil
		}
		return Price{}, false, err
	}
	return mapPrice(row), true, nil
}
