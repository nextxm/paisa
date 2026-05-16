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
	patterns := append([]string{"Assets:%", "Income:CapitalGains:%"}, investmentIncomeLikePatterns()...)
	postings := query.Init(db).Like(patterns...).NotAccountPrefix("Assets:Checking").All()
	postings = service.PopulateMarketPrice(db, postings)
	byAccount := lo.GroupBy(postings, func(p posting.Posting) string {
		if service.IsCapitalGains(p) {
			return service.CapitalGainsSourceAccount(p.Account)
		}
		if isInvestmentIncomePosting(p) {
			return investmentIncomeHoldingAccount(p.Account)
		}
		return p.Account
	})
	var gains []Gain
	ttmStart := trailing12MonthsStart(utils.ToDate(utils.Now()))
	for _, account := range utils.SortedKeys(byAccount) {
		ps := byAccount[account]
		networth := computeNetworth(db, ps)
		incomeReceived := utils.SumBy(ps, func(p posting.Posting) decimal.Decimal {
			if isInvestmentIncomePosting(p) {
				return p.Amount.Neg()
			}
			return decimal.Zero
		})
		ttmIncome := utils.SumBy(ps, func(p posting.Posting) decimal.Decimal {
			if isInvestmentIncomePosting(p) && !p.Date.Before(ttmStart) {
				return p.Amount.Neg()
			}
			return decimal.Zero
		})
		ttmYield := decimal.Zero
		if networth.BalanceAmount.GreaterThan(decimal.Zero) {
			ttmYield = ttmIncome.Div(networth.BalanceAmount).Mul(decimal.NewFromInt(100))
		}
		totalReturn := networth.GainAmount
		gains = append(gains, Gain{
			Account:           account,
			XIRR:              service.XIRR(db, ps),
			Networth:          networth,
			Postings:          ps,
			IncomeReceived:    incomeReceived,
			PriceAppreciation: totalReturn.Sub(incomeReceived),
			TotalReturn:       totalReturn,
			TTMYield:          ttmYield,
		})
	}

	return gin.H{"gain_breakdown": gains}
}

func GetAccountGain(db *gorm.DB, account string, asOfDate time.Time) gin.H {
	capitalGainsAccount := strings.Replace(account, "Assets", "Income:CapitalGains", 1)
	assetAndGains := query.Init(db).UntilDate(asOfDate).AccountPrefix(account, capitalGainsAccount).All()
	investmentIncome := query.Init(db).UntilDate(asOfDate).Like(investmentIncomeLikePatterns()...).All()
	investmentIncome = lo.Filter(investmentIncome, func(p posting.Posting, _ int) bool {
		return investmentIncomeHoldingAccount(p.Account) == account
	})
	postings := append(assetAndGains, investmentIncome...)
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
	incomeReceived := utils.SumBy(postings, func(p posting.Posting) decimal.Decimal {
		if isInvestmentIncomePosting(p) {
			return p.Amount.Neg()
		}
		return decimal.Zero
	})
	ttmStart := trailing12MonthsStart(asOfDate)
	ttmIncome := utils.SumBy(postings, func(p posting.Posting) decimal.Decimal {
		if isInvestmentIncomePosting(p) && !p.Date.Before(ttmStart) {
			return p.Amount.Neg()
		}
		return decimal.Zero
	})
	ttmYield := decimal.Zero
	if networth.BalanceAmount.GreaterThan(decimal.Zero) {
		ttmYield = ttmIncome.Div(networth.BalanceAmount).Mul(decimal.NewFromInt(100))
	}
	totalReturn := networth.GainAmount
	gain := AccountGain{
		Account:           account,
		XIRR:              service.XIRR(db, postings),
		NetworthTimeline:  networthTimeline,
		Postings:          postings,
		IncomeReceived:    incomeReceived,
		PriceAppreciation: totalReturn.Sub(incomeReceived),
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
