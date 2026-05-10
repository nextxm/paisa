package server

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/server/assets"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Gain struct {
	Account  string            `json:"account"`
	Networth Networth          `json:"networth"`
	XIRR     decimal.Decimal   `json:"xirr"`
	Postings []posting.Posting `json:"postings"`
}

type AccountGain struct {
	Account          string            `json:"account"`
	NetworthTimeline []Networth        `json:"networthTimeline"`
	XIRR             decimal.Decimal   `json:"xirr"`
	Postings         []posting.Posting `json:"postings"`
}

func GetGain(db *gorm.DB) gin.H {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%").NotAccountPrefix("Assets:Checking").All()
	postings = service.PopulateMarketPrice(db, postings)
	byAccount := lo.GroupBy(postings, func(p posting.Posting) string {
		if service.IsCapitalGains(p) {
			return service.CapitalGainsSourceAccount(p.Account)
		}
		return p.Account
	})
	var gains []Gain
	for _, account := range utils.SortedKeys(byAccount) {
		ps := byAccount[account]
		gains = append(gains, Gain{Account: account, XIRR: service.XIRR(db, ps), Networth: computeNetworth(db, ps), Postings: ps})
	}

	return gin.H{"gain_breakdown": gains}
}

func GetAccountGain(db *gorm.DB, account string, asOfDate time.Time) gin.H {
	capitalGainsAccount := strings.Replace(account, "Assets", "Income:CapitalGains", 1)
	postings := query.Init(db).UntilDate(asOfDate).AccountPrefix(account, capitalGainsAccount).All()
	postings = service.PopulateMarketPriceAt(db, postings, asOfDate)
	gain := AccountGain{Account: account, XIRR: service.XIRR(db, postings), NetworthTimeline: computeNetworthTimeline(db, postings, accounting.IsLeafAccount(db, account), asOfDate), Postings: postings}

	commodities := lo.Uniq(lo.Map(postings, func(p posting.Posting, _ int) string { return p.Commodity }))
	var portfolio_groups PortfolioAllocationGroups
	portfolio_groups = GetAccountPortfolioAllocation(db, account)
	if !(len(commodities) > 0 && len(portfolio_groups.Commomdities) == len(commodities)) {
		portfolio_groups = PortfolioAllocationGroups{Commomdities: []string{}, NameAndSecurityType: []PortfolioAggregate{}, SecurityType: []PortfolioAggregate{}, Rating: []PortfolioAggregate{}, Industry: []PortfolioAggregate{}}
	}

	assetBreakdown := assets.ComputeBreakdownAt(db, postings, false, account, asOfDate)

	return gin.H{"gain_timeline_breakdown": gain, "portfolio_allocation": portfolio_groups, "asset_breakdown": assetBreakdown}
}
