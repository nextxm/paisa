package server

import (
	"sort"
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
	Account           string            `json:"account"`
	Networth          Networth          `json:"networth"`
	XIRR              decimal.Decimal   `json:"xirr"`
	Postings          []posting.Posting `json:"postings"`
	IncomeReceived    decimal.Decimal   `json:"income_received"`
	PriceAppreciation decimal.Decimal   `json:"price_appreciation"`
	TotalReturn       decimal.Decimal   `json:"total_return"`
	TTMYield          decimal.Decimal   `json:"ttm_yield"`
}

type AccountGain struct {
	Account           string            `json:"account"`
	NetworthTimeline  []Networth        `json:"networthTimeline"`
	XIRR              decimal.Decimal   `json:"xirr"`
	Postings          []posting.Posting `json:"postings"`
	IncomeReceived    decimal.Decimal   `json:"income_received"`
	PriceAppreciation decimal.Decimal   `json:"price_appreciation"`
	TotalReturn       decimal.Decimal   `json:"total_return"`
	TTMYield          decimal.Decimal   `json:"ttm_yield"`
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
	asOfDate := utils.ToDate(utils.Now())
	for _, account := range utils.SortedKeys(byAccount) {
		ps := byAccount[account]
		networth := computeNetworth(db, ps)
		incomeReceived, ttmIncome := computeInvestmentIncomeForAccount(db, account, asOfDate)
		ttmYield := decimal.Zero
		if networth.BalanceAmount.GreaterThan(decimal.Zero) {
			ttmYield = ttmIncome.Div(networth.BalanceAmount).Mul(decimal.NewFromInt(100))
		}
		priceAppreciation := networth.GainAmount
		totalReturn := priceAppreciation.Add(incomeReceived)
		gains = append(gains, Gain{
			Account:           account,
			XIRR:              service.XIRR(db, ps),
			Networth:          networth,
			Postings:          ps,
			IncomeReceived:    incomeReceived,
			PriceAppreciation: priceAppreciation,
			TotalReturn:       totalReturn,
			TTMYield:          ttmYield,
		})
	}

	return gin.H{"gain_breakdown": gains}
}

func GetAccountGain(db *gorm.DB, account string, asOfDate time.Time) gin.H {
	capitalGainsAccount := strings.Replace(account, "Assets", "Income:CapitalGains", 1)
	postings := query.Init(db).UntilDate(asOfDate).AccountPrefix(account, capitalGainsAccount).All()
	sort.Slice(postings, func(i, j int) bool {
		if postings[i].Date.Equal(postings[j].Date) {
			return postings[i].Amount.GreaterThan(postings[j].Amount)
		}
		return postings[i].Date.Before(postings[j].Date)
	})
	postings = service.PopulateMarketPriceAt(db, postings, asOfDate)
	networthTimeline := computeNetworthTimeline(db, postings, accounting.IsLeafAccount(db, account), asOfDate)
	networth := Networth{}
	if len(networthTimeline) > 0 {
		networth = networthTimeline[len(networthTimeline)-1]
	}
	incomeReceived, ttmIncome := computeInvestmentIncomeForAccount(db, account, asOfDate)
	ttmYield := decimal.Zero
	if networth.BalanceAmount.GreaterThan(decimal.Zero) {
		ttmYield = ttmIncome.Div(networth.BalanceAmount).Mul(decimal.NewFromInt(100))
	}
	priceAppreciation := networth.GainAmount
	totalReturn := priceAppreciation.Add(incomeReceived)
	gain := AccountGain{
		Account:           account,
		XIRR:              service.XIRR(db, postings),
		NetworthTimeline:  networthTimeline,
		Postings:          postings,
		IncomeReceived:    incomeReceived,
		PriceAppreciation: priceAppreciation,
		TotalReturn:       totalReturn,
		TTMYield:          ttmYield,
	}

	commodities := lo.Uniq(lo.Map(postings, func(p posting.Posting, _ int) string { return p.Commodity }))
	var portfolio_groups PortfolioAllocationGroups
	portfolio_groups = GetAccountPortfolioAllocation(db, account)
	if !(len(commodities) > 0 && len(portfolio_groups.Commomdities) == len(commodities)) {
		portfolio_groups = PortfolioAllocationGroups{Commomdities: []string{}, NameAndSecurityType: []PortfolioAggregate{}, SecurityType: []PortfolioAggregate{}, Rating: []PortfolioAggregate{}, Industry: []PortfolioAggregate{}}
	}

	assetBreakdown := assets.ComputeBreakdownAt(db, postings, false, account, asOfDate)

	return gin.H{"gain_timeline_breakdown": gain, "portfolio_allocation": portfolio_groups, "asset_breakdown": assetBreakdown}
}

func computeInvestmentIncomeForAccount(db *gorm.DB, account string, asOfDate time.Time) (decimal.Decimal, decimal.Decimal) {
	incomePostings := query.Init(db).UntilDate(asOfDate).Like(investmentIncomeLikePatterns()...).All()
	ttmStart := trailing12MonthsStart(asOfDate)
	total := decimal.Zero
	ttm := decimal.Zero
	for _, p := range incomePostings {
		if investmentIncomeHoldingAccount(p.Account) != account {
			continue
		}
		amount := p.Amount.Neg()
		total = total.Add(amount)
		if !p.Date.Before(ttmStart) {
			ttm = ttm.Add(amount)
		}
	}
	return total, ttm
}
