package assets

import (
	"strings"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AssetBreakdown struct {
	Group            string          `json:"group"`
	InvestmentAmount decimal.Decimal `json:"investmentAmount"`
	WithdrawalAmount decimal.Decimal `json:"withdrawalAmount"`
	MarketAmount     decimal.Decimal `json:"marketAmount"`
	BalanceUnits     decimal.Decimal `json:"balanceUnits"`
	LatestPrice      decimal.Decimal `json:"latestPrice"`
	XIRR             decimal.Decimal `json:"xirr"`
	GainAmount       decimal.Decimal `json:"gainAmount"`
	AbsoluteReturn   decimal.Decimal `json:"absoluteReturn"`
}

func GetCheckingBalance(db *gorm.DB, reportCurrency string) gin.H {
	return doGetBalance(db, "Assets:Checking:%", false, reportCurrency)
}

func GetBalance(db *gorm.DB, reportCurrency string) gin.H {
	return doGetBalance(db, "Assets:%", true, reportCurrency)
}

func doGetBalance(db *gorm.DB, pattern string, rollup bool, reportCurrency string) gin.H {
	postings := query.Init(db).Like(pattern, "Income:CapitalGains:%").All()
	postings = service.PopulateMarketPrice(db, postings)
	breakdowns := ComputeBreakdowns(db, postings, rollup)
	if reportCurrency != "" && reportCurrency != config.DefaultCurrency() {
		breakdowns = convertBreakdownsToReportCurrency(db, breakdowns, reportCurrency)
	}
	return gin.H{"asset_breakdowns": breakdowns}
}

// convertBreakdownsToReportCurrency multiplies all amount fields in each
// AssetBreakdown by the current exchange rate from the default currency to
// reportCurrency.  Rate-insensitive fields (XIRR, AbsoluteReturn, BalanceUnits)
// are left unchanged.  When no rate is available, breakdowns are returned as-is.
func convertBreakdownsToReportCurrency(db *gorm.DB, breakdowns map[string]AssetBreakdown, reportCurrency string) map[string]AssetBreakdown {
	today := utils.EndOfToday()
	rate, ok := service.GetRate(db, config.DefaultCurrency(), reportCurrency, today)
	if !ok {
		return breakdowns
	}
	result := make(map[string]AssetBreakdown, len(breakdowns))
	for k, v := range breakdowns {
		result[k] = AssetBreakdown{
			Group:            v.Group,
			InvestmentAmount: v.InvestmentAmount.Mul(rate),
			WithdrawalAmount: v.WithdrawalAmount.Mul(rate),
			MarketAmount:     v.MarketAmount.Mul(rate),
			BalanceUnits:     v.BalanceUnits,
			LatestPrice:      v.LatestPrice.Mul(rate),
			XIRR:             v.XIRR,
			GainAmount:       v.GainAmount.Mul(rate),
			AbsoluteReturn:   v.AbsoluteReturn,
		}
	}
	return result
}

func ComputeBreakdowns(db *gorm.DB, postings []posting.Posting, rollup bool) map[string]AssetBreakdown {
	accounts := make(map[string]bool)
	for _, p := range postings {
		if service.IsCapitalGains(p) {
			continue
		}

		if rollup {
			var parts []string
			for _, part := range strings.Split(p.Account, ":") {
				parts = append(parts, part)
				accounts[strings.Join(parts, ":")] = false
			}
		}
		accounts[p.Account] = true

	}

	result := make(map[string]AssetBreakdown)

	for group, leaf := range accounts {
		ps := lo.Filter(postings, func(p posting.Posting, _ int) bool {
			account := p.Account
			if service.IsCapitalGains(p) {
				account = service.CapitalGainsSourceAccount(p.Account)
			}
			return utils.IsSameOrParent(account, group)
		})
		result[group] = ComputeBreakdown(db, ps, leaf, group)
	}

	return result
}

func ComputeBreakdown(db *gorm.DB, ps []posting.Posting, leaf bool, group string) AssetBreakdown {
	investmentAmount := lo.Reduce(ps, func(acc decimal.Decimal, p posting.Posting, _ int) decimal.Decimal {
		if utils.IsCheckingAccount(p.Account) || p.Amount.LessThan(decimal.Zero) || service.IsInterest(db, p) || service.IsStockSplit(db, p) || service.IsCapitalGains(p) {
			return acc
		} else {
			return acc.Add(p.Amount)
		}
	}, decimal.Zero)
	withdrawalAmount := lo.Reduce(ps, func(acc decimal.Decimal, p posting.Posting, _ int) decimal.Decimal {
		if !service.IsCapitalGains(p) && (utils.IsCheckingAccount(p.Account) || p.Amount.GreaterThan(decimal.Zero) || service.IsInterest(db, p) || service.IsStockSplit(db, p)) {
			return acc
		} else {
			return acc.Add(p.Amount.Neg())
		}
	}, decimal.Zero)
	psWithoutCapitalGains := lo.Filter(ps, func(p posting.Posting, _ int) bool {
		return !service.IsCapitalGains(p)
	})
	marketAmount := accounting.CurrentBalance(psWithoutCapitalGains)
	var balanceUnits decimal.Decimal
	if leaf {
		balanceUnits = lo.Reduce(ps, func(acc decimal.Decimal, p posting.Posting, _ int) decimal.Decimal {
			if !utils.IsCurrency(p.Commodity) {
				return acc.Add(p.Quantity)
			}
			return decimal.Zero
		}, decimal.Zero)
	}

	xirr := service.XIRR(db, ps)
	netInvestment := investmentAmount.Sub(withdrawalAmount)
	gainAmount := marketAmount.Sub(netInvestment)
	absoluteReturn := decimal.Zero
	if !investmentAmount.IsZero() {
		absoluteReturn = marketAmount.Sub(netInvestment).Div(investmentAmount)
	}
	return AssetBreakdown{
		InvestmentAmount: investmentAmount,
		WithdrawalAmount: withdrawalAmount,
		MarketAmount:     marketAmount,
		XIRR:             xirr,
		Group:            group,
		BalanceUnits:     balanceUnits,
		GainAmount:       gainAmount,
		AbsoluteReturn:   absoluteReturn,
	}
}
