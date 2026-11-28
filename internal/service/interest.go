package service

import (
	"strings"
	"sync"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type interestCache struct {
	mu       sync.RWMutex
	loadMu   sync.Mutex
	loaded   bool
	postings map[int64][]posting.Posting
}

var icache interestCache

func ensureInterestCacheLoaded(db *gorm.DB) {
	icache.mu.RLock()
	loaded := icache.loaded
	icache.mu.RUnlock()
	if loaded {
		return
	}

	icache.loadMu.Lock()
	defer icache.loadMu.Unlock()

	icache.mu.RLock()
	loaded = icache.loaded
	icache.mu.RUnlock()
	if loaded {
		return
	}

	postings := query.Init(db).Like("Income:Interest:%").All()
	grouped := lo.GroupBy(postings, func(p posting.Posting) int64 { return p.Date.Unix() })

	icache.mu.Lock()
	icache.postings = grouped
	icache.loaded = true
	icache.mu.Unlock()
}

type interestRepaymentCache struct {
	mu       sync.RWMutex
	loadMu   sync.Mutex
	loaded   bool
	postings map[int64][]posting.Posting
}

var irepaymentCache interestRepaymentCache

func ensureInterestRepaymentCacheLoaded(db *gorm.DB) {
	irepaymentCache.mu.RLock()
	loaded := irepaymentCache.loaded
	irepaymentCache.mu.RUnlock()
	if loaded {
		return
	}

	irepaymentCache.loadMu.Lock()
	defer irepaymentCache.loadMu.Unlock()

	irepaymentCache.mu.RLock()
	loaded = irepaymentCache.loaded
	irepaymentCache.mu.RUnlock()
	if loaded {
		return
	}

	postings := query.Init(db).Like("Expenses:Interest:%").All()
	grouped := lo.GroupBy(postings, func(p posting.Posting) int64 { return p.Date.Unix() })

	irepaymentCache.mu.Lock()
	irepaymentCache.postings = grouped
	irepaymentCache.loaded = true
	irepaymentCache.mu.Unlock()
}

func ClearInterestCache() {
	icache.mu.Lock()
	icache.loaded = false
	icache.postings = nil
	icache.mu.Unlock()

	irepaymentCache.mu.Lock()
	irepaymentCache.loaded = false
	irepaymentCache.postings = nil
	irepaymentCache.mu.Unlock()
}

func CapitalGainsSourceAccount(account string) string {
	parts := strings.Split(account, ":")
	return "Assets:" + strings.Join(parts[2:], ":")
}

func CapitalGainsAccount(account string) string {
	parts := strings.Split(account, ":")
	return "Income:CapitalGains:" + strings.Join(parts[1:], ":")
}

func IsCapitalGains(p posting.Posting) bool {
	if utils.IsParent(p.Account, "Income:CapitalGains") {
		return true
	}

	return false
}

func InvestmentIncomeType(account string) (string, bool) {
	switch {
	case strings.HasPrefix(account, "Income:Dividend"):
		return "Dividend", true
	case strings.HasPrefix(account, "Income:Interest"):
		return "Interest", true
	case strings.HasPrefix(account, "Income:Distribution"), strings.HasPrefix(account, "Income:Distributions"):
		return "Distribution", true
	default:
		return "", false
	}
}

func IsInvestmentIncome(p posting.Posting) bool {
	_, ok := InvestmentIncomeType(p.Account)
	return ok
}

func IsRefund(p posting.Posting) bool {
	if utils.IsParent(p.Account, "Income:Refund") {
		return true
	}

	return false
}

func IsStockSplit(db *gorm.DB, p posting.Posting) bool {
	if utils.IsCurrency(p.Commodity) {
		return false
	}

	t, found := transaction.GetById(db, p.TransactionID)
	if !found {
		return false
	}

	for _, tp := range t.Postings {
		if utils.IsCurrency(tp.Commodity) || tp.Account != p.Account {
			return false
		}
	}
	return true
}

func IsSellWithCapitalGains(db *gorm.DB, p posting.Posting) bool {
	if utils.IsCurrency(p.Commodity) {
		return false
	}

	t, found := transaction.GetById(db, p.TransactionID)
	if !found {
		return false
	}

	for _, tp := range t.Postings {
		if IsCapitalGains(tp) {
			return true
		}
	}
	return false
}

func IsContraPostingRefund(db *gorm.DB, p posting.Posting) bool {
	t, found := transaction.GetById(db, p.TransactionID)
	if !found {
		return false
	}

	for _, tp := range t.Postings {
		if IsRefund(tp) {
			return true
		}
	}
	return false
}

func IsInterestRepayment(db *gorm.DB, p posting.Posting) bool {
	if !utils.IsCurrency(p.Commodity) {
		return false
	}

	if strings.HasPrefix(p.Account, "Expenses:Interest:") {
		return true
	}

	ensureInterestRepaymentCacheLoaded(db)
	irepaymentCache.mu.RLock()
	candidates := irepaymentCache.postings[p.Date.Unix()]
	irepaymentCache.mu.RUnlock()

	for _, ip := range candidates {
		if ip.Date.Equal(p.Date) &&
			ip.Amount.Neg().Equal(p.Amount) &&
			ip.Payee == p.Payee {
			return true
		}
	}

	return false
}

func IsInterest(db *gorm.DB, p posting.Posting) bool {
	if !utils.IsCurrency(p.Commodity) {
		return false
	}

	ensureInterestCacheLoaded(db)
	icache.mu.RLock()
	candidates := icache.postings[p.Date.Unix()]
	icache.mu.RUnlock()

	for _, ip := range candidates {
		if ip.Date.Equal(p.Date) &&
			ip.Amount.Neg().Equal(p.Amount) &&
			ip.Payee == p.Payee {
			return true
		}
	}

	return false
}
