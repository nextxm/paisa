package accounting

import (
	"sync"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"golang.org/x/exp/slices"
	"gorm.io/gorm"
)

type accountCache struct {
	sync.Once
	accounts []string
}

type commodityCache struct {
	sync.Once
	commodities []string
}

var acache accountCache
var ccache commodityCache

func loadAccountCache(db *gorm.DB) {
	db.Model(&posting.Posting{}).Distinct().Pluck("Account", &acache.accounts)
}

func loadCommodityCache(db *gorm.DB) {
	db.Model(&posting.Posting{}).Distinct().Pluck("Commodity", &ccache.commodities)
}

func AllAccounts(db *gorm.DB) []string {
	acache.Do(func() {
		loadAccountCache(db)
		slices.Sort(acache.accounts)
	})
	return acache.accounts
}

func AllCommodities(db *gorm.DB) []string {
	ccache.Do(func() {
		loadCommodityCache(db)
		slices.Sort(ccache.commodities)
	})
	return ccache.commodities
}

func IsLeafAccount(db *gorm.DB, account string) bool {
	return slices.Contains(AllAccounts(db), account)
}

func ClearCache() {
	acache = accountCache{}
	ccache = commodityCache{}
}
