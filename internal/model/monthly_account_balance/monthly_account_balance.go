// Package monthly_account_balance provides a materialized summary table that
// stores pre-computed per-(month, account, commodity) totals refreshed
// atomically during every journal sync.
package monthly_account_balance

import (
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// MonthlyAccountBalance stores the pre-computed balance delta for a single
// (month, account, commodity) tuple, refreshed atomically on every journal sync.
type MonthlyAccountBalance struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Month     string          `gorm:"uniqueIndex:idx_monthly_account_balances_key;index;not null" json:"month"`
	Account   string          `gorm:"uniqueIndex:idx_monthly_account_balances_key;not null" json:"account"`
	Commodity string          `gorm:"uniqueIndex:idx_monthly_account_balances_key;not null" json:"commodity"`
	Quantity  decimal.Decimal `json:"quantity"`
	Amount    decimal.Decimal `json:"amount"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (MonthlyAccountBalance) TableName() string {
	return "monthly_account_balances"
}

// All returns every row in monthly_account_balances ordered by month, account, commodity.
func All(db *gorm.DB) ([]MonthlyAccountBalance, error) {
	var rows []MonthlyAccountBalance
	if err := db.Order("month asc, account asc, commodity asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// RefreshFromPostings computes per-(month, account, commodity) sums from the
// supplied postings slice and replaces monthly_account_balances within tx.
// Only non-forecast postings are included.
func RefreshFromPostings(tx *gorm.DB, postings []*posting.Posting) error {
	type key struct{ Month, Account, Commodity string }
	sums := make(map[key]*MonthlyAccountBalance)
	now := time.Now()

	for _, p := range postings {
		if p.Forecast {
			continue
		}
		month := p.Date.Format("2006-01")
		k := key{month, p.Account, p.Commodity}
		if b, ok := sums[k]; ok {
			b.Quantity = b.Quantity.Add(p.Quantity)
			b.Amount = b.Amount.Add(p.Amount)
		} else {
			sums[k] = &MonthlyAccountBalance{
				Month:     month,
				Account:   p.Account,
				Commodity: p.Commodity,
				Quantity:  p.Quantity,
				Amount:    p.Amount,
				UpdatedAt: now,
			}
		}
	}

	rows := make([]*MonthlyAccountBalance, 0, len(sums))
	for _, b := range sums {
		rows = append(rows, b)
	}
	return replaceAll(tx, rows)
}

func replaceAll(tx *gorm.DB, balances []*MonthlyAccountBalance) error {
	if err := tx.Exec("DELETE FROM monthly_account_balances").Error; err != nil {
		return err
	}
	if len(balances) == 0 {
		return nil
	}
	const batchSize = 500
	return tx.CreateInBatches(balances, batchSize).Error
}
