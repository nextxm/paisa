package transaction

import (
	"sync"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type Transaction struct {
	ID           string            `json:"id"`
	Date         time.Time         `json:"date"`
	Payee        string            `json:"payee"`
	Postings     []posting.Posting `json:"postings"`
	TagRecurring string            `json:"tag_recurring"`
	TagPeriod    string            `json:"tag_period"`
	BeginLine    uint64            `json:"beginLine"`
	EndLine      uint64            `json:"endLine"`
	FileName     string            `json:"fileName"`
	Note         string            `json:"note"`
}

type transactionCache struct {
	mu           sync.RWMutex
	loadMu       sync.Mutex
	loaded       bool
	transactions map[string]Transaction
}

var tcache transactionCache

func ensureTransactionCacheLoaded(db *gorm.DB) {
	tcache.mu.RLock()
	loaded := tcache.loaded
	tcache.mu.RUnlock()
	if loaded {
		return
	}

	tcache.loadMu.Lock()
	defer tcache.loadMu.Unlock()

	tcache.mu.RLock()
	loaded = tcache.loaded
	tcache.mu.RUnlock()
	if loaded {
		return
	}

	postings := query.Init(db).All()
	txs := make(map[string]Transaction)
	for _, t := range Build(postings) {
		txs[t.ID] = t
	}

	tcache.mu.Lock()
	tcache.transactions = txs
	tcache.loaded = true
	tcache.mu.Unlock()
}

func GetById(db *gorm.DB, id string) (Transaction, bool) {
	ensureTransactionCacheLoaded(db)
	tcache.mu.RLock()
	defer tcache.mu.RUnlock()
	t, found := tcache.transactions[id]
	return t, found
}

func ClearCache() {
	tcache.mu.Lock()
	tcache.loaded = false
	tcache.transactions = nil
	tcache.mu.Unlock()
}

func Build(postings []posting.Posting) []Transaction {
	grouped := lo.GroupBy(postings, func(p posting.Posting) string { return p.TransactionID })
	return lo.Map(lo.Values(grouped), func(ps []posting.Posting, _ int) Transaction {
		sample := ps[0]
		var tagRecurring string
		var tagPeriod string
		for _, p := range ps {
			if p.TagRecurring != "" {
				tagRecurring = p.TagRecurring
			}

			if p.TagPeriod != "" {
				tagPeriod = p.TagPeriod
			}
		}
		return Transaction{
			ID:           sample.TransactionID,
			Date:         sample.Date,
			Payee:        sample.Payee,
			Postings:     ps,
			TagRecurring: tagRecurring,
			TagPeriod:    tagPeriod,
			BeginLine:    sample.TransactionBeginLine,
			EndLine:      sample.TransactionEndLine,
			Note:         sample.TransactionNote,
			FileName:     sample.FileName,
		}
	})

}
