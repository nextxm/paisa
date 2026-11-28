package accounting

import (
	"sync"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type accountCache struct {
	mu       sync.RWMutex
	loadMu   sync.Mutex
	loaded   bool
	accounts []string
}

type commodityCache struct {
	mu          sync.RWMutex
	loadMu      sync.Mutex
	loaded      bool
	commodities []string
}

var acache accountCache
var ccache commodityCache

func ensureAccountCacheLoaded(db *gorm.DB) {
	acache.mu.RLock()
	loaded := acache.loaded
	acache.mu.RUnlock()
	if loaded {
		return
	}

	acache.loadMu.Lock()
	defer acache.loadMu.Unlock()

	acache.mu.RLock()
	loaded = acache.loaded
	acache.mu.RUnlock()
	if loaded {
		return
	}

	var accounts []string
	db.Model(&posting.Posting{}).Distinct().Pluck("Account", &accounts)
	slices.Sort(accounts)

	acache.mu.Lock()
	acache.accounts = accounts
	acache.loaded = true
	acache.mu.Unlock()
}

func ensureCommodityCacheLoaded(db *gorm.DB) {
	ccache.mu.RLock()
	loaded := ccache.loaded
	ccache.mu.RUnlock()
	if loaded {
		return
	}

	ccache.loadMu.Lock()
	defer ccache.loadMu.Unlock()

	ccache.mu.RLock()
	loaded = ccache.loaded
	ccache.mu.RUnlock()
	if loaded {
		return
	}

	var commodities []string
	db.Model(&posting.Posting{}).Distinct().Pluck("Commodity", &commodities)
	slices.Sort(commodities)

	ccache.mu.Lock()
	ccache.commodities = commodities
	ccache.loaded = true
	ccache.mu.Unlock()
}

func AllAccounts(db *gorm.DB) []string {
	ensureAccountCacheLoaded(db)
	acache.mu.RLock()
	defer acache.mu.RUnlock()
	return acache.accounts
}

func AllCommodities(db *gorm.DB) []string {
	ensureCommodityCacheLoaded(db)
	ccache.mu.RLock()
	defer ccache.mu.RUnlock()
	return ccache.commodities
}

func IsLeafAccount(db *gorm.DB, account string) bool {
	accounts := AllAccounts(db)
	_, found := slices.BinarySearch(accounts, account)
	return found
}

func ClearCache() {
	acache.mu.Lock()
	acache.loaded = false
	acache.accounts = nil
	acache.mu.Unlock()

	ccache.mu.Lock()
	ccache.loaded = false
	ccache.commodities = nil
	ccache.mu.Unlock()
}
