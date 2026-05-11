package posting

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	ASSETS               = "assets"
	ASSETS_CASH          = "assets:cash"
	INCOME               = "income"
	INCOME_INTEREST      = "income:interest"
	INCOME_DIVIDEND      = "income:dividend"
	INCOME_CAPITAL_GAINS = "income:capital_gains"
	EXPENSES             = "expenses"
	EXPENSES_CHARGES     = "expenses:charges"
	EXPENSES_TAXES       = "expenses:taxes"
	LIABILITIES          = "liabilities"
)

type Posting struct {
	ID                   uint            `gorm:"primaryKey" json:"id"`
	TransactionID        string          `json:"transaction_id"`
	Date                 time.Time       `json:"date"`
	Payee                string          `json:"payee"`
	Account              string          `json:"account"`
	Commodity            string          `json:"commodity"`
	Quantity             decimal.Decimal `json:"quantity"`
	Amount               decimal.Decimal `json:"amount"`
	Status               string          `json:"status"`
	TagRecurring         string          `json:"tag_recurring"`
	TagPeriod            string          `json:"tag_period"`
	TransactionBeginLine uint64          `json:"transaction_begin_line"`
	TransactionEndLine   uint64          `json:"transaction_end_line"`
	FileName             string          `json:"file_name"`
	Forecast             bool            `json:"forecast"`
	Note                 string          `json:"note"`
	TransactionNote      string          `json:"transaction_note"`
	// TransactionHash is a SHA-256 content hash of all postings that belong to
	// the same transaction.  It is identical for every posting row in the same
	// transaction and is used by DeltaUpsert to skip unchanged transactions.
	TransactionHash string `gorm:"index" json:"-"`

	MarketAmount decimal.Decimal `gorm:"-:all" json:"market_amount"`
	Balance      decimal.Decimal `gorm:"-:all" json:"balance"`

	behaviours []string `gorm:"-:all"`
}

func (p Posting) GroupDate() time.Time {
	return p.Date
}

func (p *Posting) RestName(level int) string {
	return strings.Join(strings.Split(p.Account, ":")[level:], ":")
}

func (p Posting) Negate() Posting {
	clone := p
	clone.Quantity = p.Quantity.Neg()
	clone.Amount = p.Amount.Neg()
	return clone
}

func (p *Posting) Price() decimal.Decimal {
	if p.Quantity.IsZero() {
		return decimal.Zero
	}
	return p.Amount.Div(p.Quantity)
}

func (p *Posting) AddAmount(amount decimal.Decimal) {
	p.Amount = p.Amount.Add(amount)
}

func (p *Posting) AddQuantity(quantity decimal.Decimal) {
	price := p.Price()
	p.Quantity = p.Quantity.Add(quantity)
	p.Amount = p.Quantity.Mul(price)
}

func (p Posting) WithQuantity(quantity decimal.Decimal) Posting {
	clone := p
	clone.Quantity = quantity
	clone.Amount = quantity.Mul(p.Price())
	return clone
}

func (p Posting) WithAmount(amount decimal.Decimal) Posting {
	clone := p
	clone.Amount = amount
	clone.Quantity = amount.Div(p.Price())
	return clone
}

func (p Posting) Split(amount decimal.Decimal) (Posting, Posting) {
	return p.WithAmount(amount), p.WithAmount(p.Amount.Sub(amount))
}

func (p Posting) Behaviours() []string {
	if p.behaviours == nil {
		p.behaviours = Behaviours(p.Account)
	}
	return p.behaviours
}

func (p Posting) HasBehaviour(behaviour string) bool {
	for _, b := range p.Behaviours() {
		if b == behaviour {
			return true
		}
	}
	return false
}

// transactionHashRow is a lightweight projection used to load existing
// (transaction_id, transaction_hash) pairs from the database without
// fetching the full posting rows.
type transactionHashRow struct {
	TransactionID   string
	TransactionHash string
}

// contentHash computes a deterministic SHA-256 digest of the stable fields
// that make up a single posting.  It is called once per posting; callers
// accumulate the digests of all postings in a transaction and combine them
// (via ComputeTransactionHash) into a single per-transaction hash.
func contentHash(p *Posting) string {
	h := sha256.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%s|%s|%s|%s|%d|%d|%v|%s|%s",
		p.TransactionID,
		p.Date.Format("2006-01-02"),
		p.Payee,
		p.Account,
		p.Commodity,
		p.Quantity.String(),
		p.Amount.String(),
		p.Status,
		p.TransactionBeginLine,
		p.TransactionEndLine,
		p.Forecast,
		p.Note,
		p.TransactionNote,
	)
	return hex.EncodeToString(h.Sum(nil))
}

// ComputeTransactionHash returns a deterministic hash representing the full set
// of postings in a single transaction.  The postings are sorted by account name
// before hashing so that re-ordering within a transaction does not produce a
// spurious "changed" signal.
func ComputeTransactionHash(postings []*Posting) string {
	// Sort a copy by account to get a stable order within the transaction.
	sorted := make([]*Posting, len(postings))
	copy(sorted, postings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Account < sorted[j].Account
	})
	h := sha256.New()
	for _, p := range sorted {
		fmt.Fprintf(h, "%s;", contentHash(p))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// StampTransactionHash groups postings by TransactionID, computes
// ComputeTransactionHash for each group, and sets the TransactionHash field
// on every posting row.  This must be called before persisting postings to the
// database.
func StampTransactionHash(postings []*Posting) {
	groups := make(map[string][]*Posting, len(postings))
	for _, p := range postings {
		groups[p.TransactionID] = append(groups[p.TransactionID], p)
	}
	for _, group := range groups {
		hash := ComputeTransactionHash(group)
		for _, p := range group {
			p.TransactionHash = hash
		}
	}
}

// DeltaUpsert performs an incremental update of the postings table.
// It stamps a per-transaction content hash on every new posting, then:
//   - Deletes rows for transaction IDs that are no longer present.
//   - Inserts rows for brand-new transaction IDs.
//   - Replaces (delete + insert) rows for transaction IDs whose content hash
//     changed since the last sync.
//   - Skips posting rows whose transaction hash is identical to what is
//     already stored, avoiding unnecessary I/O for unchanged transactions.
//
// The whole operation runs inside a single SQLite transaction so the postings
// table is never in a partially-updated state.
func DeltaUpsert(db *gorm.DB, newPostings []*Posting) (added, updated, removed, unchanged int, err error) {
	// Stamp per-transaction hashes before any DB work.
	StampTransactionHash(newPostings)

	// Group incoming postings by TransactionID.
	newGroups := make(map[string][]*Posting, len(newPostings))
	for _, p := range newPostings {
		newGroups[p.TransactionID] = append(newGroups[p.TransactionID], p)
	}

	// Load existing (transaction_id, transaction_hash) pairs.
	var rows []transactionHashRow
	if err2 := db.Model(&Posting{}).
		Select("DISTINCT transaction_id, transaction_hash").
		Scan(&rows).Error; err2 != nil {
		err = err2
		return
	}

	existing := make(map[string]string, len(rows))
	for _, r := range rows {
		existing[r.TransactionID] = r.TransactionHash
	}

	// Classify each existing transaction.
	var txIDsToDelete []string
	for txID := range existing {
		if _, ok := newGroups[txID]; !ok {
			txIDsToDelete = append(txIDsToDelete, txID)
			removed++
		}
	}

	// Classify each incoming transaction.
	var postingsToInsert []*Posting
	for txID, group := range newGroups {
		existingHash, inDB := existing[txID]
		switch {
		case !inDB:
			// Brand new transaction.
			added++
			postingsToInsert = append(postingsToInsert, group...)
		case group[0].TransactionHash != existingHash:
			// Transaction content changed — delete old rows, insert new ones.
			updated++
			txIDsToDelete = append(txIDsToDelete, txID)
			postingsToInsert = append(postingsToInsert, group...)
		default:
			unchanged++
		}
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if len(txIDsToDelete) > 0 {
			const chunkSize = 500
			for i := 0; i < len(txIDsToDelete); i += chunkSize {
				end := i + chunkSize
				if end > len(txIDsToDelete) {
					end = len(txIDsToDelete)
				}
				if e := tx.Where("transaction_id IN ?", txIDsToDelete[i:end]).Delete(&Posting{}).Error; e != nil {
					return e
				}
			}
		}
		if len(postingsToInsert) > 0 {
			if e := tx.CreateInBatches(postingsToInsert, 500).Error; e != nil {
				return e
			}
		}
		return nil
	})
	return
}

// UpsertAll replaces the entire postings table with the supplied rows.  It
// wraps the operation in a transaction so the table is never partially written.
// Use DeltaUpsert for incremental syncs; UpsertAll is provided for contexts
// (such as tests) that need a complete replace.
func UpsertAll(db *gorm.DB, postings []*Posting) error {
	const batchSize = 500

	return db.Transaction(func(tx *gorm.DB) error {
		err := tx.Exec("DELETE FROM postings").Error
		if err != nil {
			return err
		}

		if len(postings) == 0 {
			return nil
		}

		return tx.CreateInBatches(postings, batchSize).Error
	})
}

func Behaviours(account string) []string {
	var behaviours []string
	if utils.IsParent(account, "Assets") {
		behaviours = append(behaviours, ASSETS)
	}

	if utils.IsSameOrParent(account, "Assets:Checking") {
		behaviours = append(behaviours, ASSETS_CASH)
	}

	if utils.IsParent(account, "Income") {
		behaviours = append(behaviours, INCOME)
	}

	if utils.IsSameOrParent(account, "Income:Interest") {
		behaviours = append(behaviours, INCOME_INTEREST)
	}

	if utils.IsSameOrParent(account, "Income:Dividend") {
		behaviours = append(behaviours, INCOME_DIVIDEND)
	}

	if utils.IsSameOrParent(account, "Income:Capital Gains") {
		behaviours = append(behaviours, INCOME_CAPITAL_GAINS)
	}

	if utils.IsParent(account, "Expenses") {
		behaviours = append(behaviours, EXPENSES)
	}

	if utils.IsSameOrParent(account, "Expenses:Charges") {
		behaviours = append(behaviours, EXPENSES_CHARGES)
	}

	if utils.IsSameOrParent(account, "Expenses:Tax") {
		behaviours = append(behaviours, EXPENSES_TAXES)
	}

	if utils.IsParent(account, "Liabilities") {
		behaviours = append(behaviours, LIABILITIES)
	}
	return behaviours
}
