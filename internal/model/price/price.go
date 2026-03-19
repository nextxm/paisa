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
		err := tx.Delete(&Price{}, "commodity_type = ? and (commodity_id = ? or commodity_name = ?)", commodityType, commodityID, commodityName).Error
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

func UpsertAllByType(db *gorm.DB, commodityType config.CommodityType, prices []Price) error {
	return db.Transaction(func(tx *gorm.DB) error {
		err := tx.Delete(&Price{}, "commodity_type = ?", commodityType).Error
		if err != nil {
			return err
		}
		dc := defaultQuoteCommodity()
		for _, price := range prices {
			if price.QuoteCommodity == "" {
				price.QuoteCommodity = dc
			}
			err := tx.Create(&price).Error
			if err != nil {
				return err
			}
		}

		return nil
	})
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
