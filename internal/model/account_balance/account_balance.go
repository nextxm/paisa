// Package account_balance provides a materialized summary table that stores
// pre-computed per-(account, commodity) balance totals.  The table is refreshed
// atomically during every journal sync so that balance lookups are O(1)
// index-seeks rather than O(N) full-table scans over the postings table.
package account_balance

import (
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AccountBalance stores the pre-computed balance for a single
// (account, commodity) pair, refreshed atomically on every journal sync.
type AccountBalance struct {
	ID        uint            `gorm:"primaryKey" json:"id"`
	Account   string          `gorm:"uniqueIndex:idx_account_balances_account_commodity;not null" json:"account"`
	Commodity string          `gorm:"uniqueIndex:idx_account_balances_account_commodity;not null" json:"commodity"`
	Quantity  decimal.Decimal `json:"quantity"`
	Amount    decimal.Decimal `json:"amount"`
	UpdatedAt time.Time       `json:"updated_at"`
}

func (AccountBalance) TableName() string {
	return "account_balances"
}

// All returns every row in the account_balances table ordered by account then commodity.
func All(db *gorm.DB) ([]AccountBalance, error) {
	var balances []AccountBalance
	if err := db.Order("account asc, commodity asc").Find(&balances).Error; err != nil {
		return nil, err
	}
	return balances, nil
}

// ByAccount returns all balance rows whose account equals the given account name.
// For an O(1) single-account lookup use this rather than loading all postings.
func ByAccount(db *gorm.DB, account string) ([]AccountBalance, error) {
	var balances []AccountBalance
	if err := db.Where("account = ?", account).Order("commodity asc").Find(&balances).Error; err != nil {
		return nil, err
	}
	return balances, nil
}

// RefreshFromPostings computes per-(account, commodity) sums from the supplied
// postings slice and replaces the entire account_balances table within the
// given (already-open) transaction tx.
//
// Only non-forecast postings are included, matching the semantics of
// query.Init(db).GroupSum().
func RefreshFromPostings(tx *gorm.DB, postings []*posting.Posting) error {
	type key struct{ Account, Commodity string }
	sums := make(map[key]*AccountBalance)
	now := time.Now()

	for _, p := range postings {
		if p.Forecast {
			continue
		}
		k := key{p.Account, p.Commodity}
		if b, ok := sums[k]; ok {
			b.Quantity = b.Quantity.Add(p.Quantity)
			b.Amount = b.Amount.Add(p.Amount)
		} else {
			sums[k] = &AccountBalance{
				Account:   p.Account,
				Commodity: p.Commodity,
				Quantity:  p.Quantity,
				Amount:    p.Amount,
				UpdatedAt: now,
			}
		}
	}

	rows := make([]*AccountBalance, 0, len(sums))
	for _, b := range sums {
		rows = append(rows, b)
	}
	return replaceAll(tx, rows)
}

// replaceAll atomically replaces all rows in account_balances within the given
// transaction: it deletes all existing rows then inserts the new set in batches.
func replaceAll(tx *gorm.DB, balances []*AccountBalance) error {
	if err := tx.Exec("DELETE FROM account_balances").Error; err != nil {
		return err
	}
	if len(balances) == 0 {
		return nil
	}
	const batchSize = 500
	return tx.CreateInBatches(balances, batchSize).Error
}
