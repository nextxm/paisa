package assets

import (
	"sort"
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

// OriginalCurrencyBalance holds an asset's balance expressed in a single
// native currency without any FX conversion to the default currency.
type OriginalCurrencyBalance struct {
	Currency string          `json:"currency"`
	Amount   decimal.Decimal `json:"amount"`
}

type AssetBreakdown struct {
	Group            string                    `json:"group"`
	InvestmentAmount decimal.Decimal           `json:"investmentAmount"`
	WithdrawalAmount decimal.Decimal           `json:"withdrawalAmount"`
	MarketAmount     decimal.Decimal           `json:"marketAmount"`
	BalanceUnits     decimal.Decimal           `json:"balanceUnits"`
	LatestPrice      decimal.Decimal           `json:"latestPrice"`
	XIRR             decimal.Decimal           `json:"xirr"`
	AbsoluteReturn   decimal.Decimal           `json:"absoluteReturn"`
	GainAmount       decimal.Decimal           `json:"gainAmount"`
	OriginalBalances []OriginalCurrencyBalance `json:"originalBalances"`
	Inactive         bool                      `json:"inactive"`
}

// GetCheckingBalance returns a breakdown of the current balance for all
// configured checking accounts.
func GetCheckingBalance(db *gorm.DB, reportCurrency string) gin.H {
	result := doGetBalance(db, config.GetConfig().CheckingAccounts, false, reportCurrency)
	if breakdowns, ok := result["asset_breakdowns"].(map[string]AssetBreakdown); ok {
		filtered := make(map[string]AssetBreakdown)
		for k, v := range breakdowns {
			if !v.Inactive || !v.MarketAmount.IsZero() {
				filtered[k] = v
			}
		}
		result["asset_breakdowns"] = filtered
	}
	return result
}

func GetBalance(db *gorm.DB, reportCurrency string) gin.H {
	return GetBalanceByMode(db, reportCurrency, false)
}

func GetBalanceByMode(db *gorm.DB, reportCurrency string, flat bool) gin.H {
	return doGetBalance(db, []string{"Assets"}, !flat, reportCurrency)
}

func doGetBalance(db *gorm.DB, patterns []string, rollup bool, reportCurrency string) gin.H {
	var dbPatterns []string
	for _, p := range patterns {
		if strings.HasPrefix(p, "regex:") {
			dbPatterns = append(dbPatterns, "Assets", "Liabilities")
		} else {
			dbPatterns = append(dbPatterns, p)
		}
	}
	dbPatterns = append(dbPatterns, "Income:CapitalGains")
	postings := query.Init(db).AccountPrefix(dbPatterns...).All()
	postings = service.PopulateMarketPrice(db, postings)

	var breakdowns map[string]AssetBreakdown
	if !rollup {
		breakdowns = make(map[string]AssetBreakdown)
		for _, p := range patterns {
			if strings.HasPrefix(p, "regex:") {
				matchedAccounts := make(map[string]bool)
				for _, pos := range postings {
					account := pos.Account
					if service.IsCapitalGains(pos) {
						account = service.CapitalGainsSourceAccount(pos.Account)
					}
					if utils.MatchAccount(account, p) {
						matchedAccounts[account] = true
					}
				}
				for account := range matchedAccounts {
					ps := lo.Filter(postings, func(pos posting.Posting, _ int) bool {
						acc := pos.Account
						if service.IsCapitalGains(pos) {
							acc = service.CapitalGainsSourceAccount(pos.Account)
						}
						return acc == account
					})
					breakdowns[account] = ComputeBreakdown(db, ps, true, account)
				}
			} else {
				group := strings.TrimSuffix(p, ":%")
				ps := lo.Filter(postings, func(pos posting.Posting, _ int) bool {
					account := pos.Account
					if service.IsCapitalGains(pos) {
						account = service.CapitalGainsSourceAccount(pos.Account)
					}
					return utils.MatchAccount(account, p)
				})
				if len(ps) > 0 {
					breakdowns[group] = ComputeBreakdown(db, ps, true, group)
				}
			}
		}
	} else {
		breakdowns = ComputeBreakdowns(db, postings, rollup)

		// Filter breakdowns to only include those that match the requested patterns.
		// This prevents internal query accounts like Income:CapitalGains from
		// appearing in the final output.
		filtered := make(map[string]AssetBreakdown)
		for k, v := range breakdowns {
			for _, p := range patterns {
				if utils.MatchAccount(k, p) {
					filtered[k] = v
					break
				}
			}
		}
		breakdowns = filtered
	}

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
			OriginalBalances: v.OriginalBalances,
			Inactive:         v.Inactive,
		}
	}
	return result
}

// ComputeBreakdowns builds a per-account AssetBreakdown map for every account
// (and optionally its parent groups when rollup is true) present in postings.
//
// Performance: postings are first grouped by their effective account in O(N),
// then for each breakdown group the relevant postings are collected in O(A×C)
// where A is the number of groups and C is the number of distinct leaf
// accounts.  This is substantially better than the naïve O(A×N) approach of
// filtering the full postings slice once per group.
func ComputeBreakdowns(db *gorm.DB, postings []posting.Posting, rollup bool) map[string]AssetBreakdown {
	// Phase 1: identify all breakdown groups and classify leaf vs. parent.
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

	// Phase 2: pre-group postings by effective account (O(N)).
	// Capital-gains postings are remapped to their source account so they
	// aggregate alongside the corresponding asset postings.
	byEffectiveAccount := make(map[string][]posting.Posting, len(accounts))
	for _, p := range postings {
		effAcc := p.Account
		if service.IsCapitalGains(p) {
			effAcc = service.CapitalGainsSourceAccount(p.Account)
		}
		byEffectiveAccount[effAcc] = append(byEffectiveAccount[effAcc], p)
	}

	// Phase 3: for each breakdown group collect postings from all matching
	// child accounts (O(A×C)) and compute the breakdown.
	result := make(map[string]AssetBreakdown)
	for group, leaf := range accounts {
		var ps []posting.Posting
		for effAcc, accPostings := range byEffectiveAccount {
			if utils.IsSameOrParent(effAcc, group) {
				ps = append(ps, accPostings...)
			}
		}
		result[group] = ComputeBreakdown(db, ps, leaf, group)
	}

	return result
}

func ComputeBreakdown(db *gorm.DB, ps []posting.Posting, leaf bool, group string) AssetBreakdown {
	investmentAmount := lo.Reduce(ps, func(acc decimal.Decimal, p posting.Posting, _ int) decimal.Decimal {
		if utils.IsCheckingAccount(p.Account) || p.Amount.LessThan(decimal.Zero) || service.IsInterest(db, p) || service.IsStockSplit(db, p) || service.IsCapitalGains(p) {
			return acc
		} else {
			return acc.Add(service.GetMarketPrice(db, p, p.Date))
		}
	}, decimal.Zero)
	withdrawalAmount := lo.Reduce(ps, func(acc decimal.Decimal, p posting.Posting, _ int) decimal.Decimal {
		if !service.IsCapitalGains(p) && (utils.IsCheckingAccount(p.Account) || p.Amount.GreaterThan(decimal.Zero) || service.IsInterest(db, p) || service.IsStockSplit(db, p)) {
			return acc
		} else {
			return acc.Add(service.GetMarketPrice(db, p, p.Date).Neg())
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
	originalBalances := computeOriginalBalances(db, psWithoutCapitalGains)
	return AssetBreakdown{
		InvestmentAmount: investmentAmount,
		WithdrawalAmount: withdrawalAmount,
		MarketAmount:     marketAmount,
		XIRR:             xirr,
		Group:            group,
		BalanceUnits:     balanceUnits,
		GainAmount:       gainAmount,
		AbsoluteReturn:   absoluteReturn,
		OriginalBalances: originalBalances,
		Inactive:         utils.IsInactiveAccount(group),
	}
}

// computeOriginalBalances aggregates postings into per-currency amounts
// without applying any FX conversion to the default currency.
//
//   - Default-currency postings contribute their Amount to the default currency
//     bucket.
//   - Foreign-currency postings (foreign cash) contribute their Quantity to the
//     commodity-name bucket, without any conversion.
//   - Security postings contribute Quantity × native-unit-price to the price's
//     quote currency bucket.
func computeOriginalBalances(db *gorm.DB, ps []posting.Posting) []OriginalCurrencyBalance {
	dc := config.DefaultCurrency()
	date := utils.EndOfToday()

	// Map commodity → net quantity for securities (priced assets).
	securityQty := make(map[string]decimal.Decimal)
	// Map currency → amount for cash/currency holdings.
	currencyAmt := make(map[string]decimal.Decimal)

	for _, p := range ps {
		if utils.IsCurrency(p.Commodity) {
			// Default currency: use Amount field (already in default currency)
			currencyAmt[dc] = currencyAmt[dc].Add(p.Amount)
		} else if service.IsForeignCurrency(p.Commodity) {
			// Foreign cash: use Quantity in the commodity's own currency
			currencyAmt[p.Commodity] = currencyAmt[p.Commodity].Add(p.Quantity)
		} else if utils.IsSameOrParent(p.Account, "Assets:Equity") || service.IsSecurity(db, p.Commodity) {
			// Securities (especially equity holdings) are valued via native unit prices.
			securityQty[p.Commodity] = securityQty[p.Commodity].Add(p.Quantity)
		} else {
			// Non-equity non-default commodities without explicit currency config
			// are tracked in their own native units (e.g. crypto-like holdings).
			currencyAmt[p.Commodity] = currencyAmt[p.Commodity].Add(p.Quantity)
		}
	}

	// Convert security quantities to their native-currency amounts.
	for commodity, qty := range securityQty {
		if qty.IsZero() {
			continue
		}
		unitPrice, quoteCurrency, ok := service.GetNativeUnitPrice(db, commodity, date)
		if ok && !unitPrice.IsZero() {
			currencyAmt[quoteCurrency] = currencyAmt[quoteCurrency].Add(qty.Mul(unitPrice))
		}
	}

	// Convert map to a deterministically sorted slice, omitting zero balances.
	var result []OriginalCurrencyBalance
	for currency, amount := range currencyAmt {
		if !amount.IsZero() {
			result = append(result, OriginalCurrencyBalance{Currency: currency, Amount: amount})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Currency < result[j].Currency
	})
	return result
}
