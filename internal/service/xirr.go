package service

import (
	"fmt"
	"sort"

	"github.com/ananthakumaran/paisa/internal/model/cache"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/ananthakumaran/paisa/internal/xirr"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// buildXIRRCashflows constructs the cashflow slice used by both XIRR and
// WarmXIRRCache so the two functions stay consistent.
func buildXIRRCashflows(db *gorm.DB, ps []posting.Posting) []xirr.Cashflow {
	today := utils.EndOfToday()
	marketAmount := utils.SumBy(ps, func(p posting.Posting) decimal.Decimal {
		if IsCapitalGains(p) {
			return decimal.Zero
		}
		return p.MarketAmount
	})
	cashflows := lo.Reverse(lo.Map(ps, func(p posting.Posting, _ int) xirr.Cashflow {
		if IsInterest(db, p) || IsInterestRepayment(db, p) {
			return xirr.Cashflow{Date: p.Date, Amount: 0}
		}
		return xirr.Cashflow{Date: p.Date, Amount: GetMarketPrice(db, p, p.Date).Neg().Round(4).InexactFloat64()}
	}))
	cashflows = append(cashflows, xirr.Cashflow{Date: today, Amount: marketAmount.Round(4).InexactFloat64()})
	return cashflows
}

func XIRR(db *gorm.DB, ps []posting.Posting) decimal.Decimal {
	cashflows := buildXIRRCashflows(db, ps)
	return cache.Lookup(db, cashflows, func() decimal.Decimal {
		return xirr.XIRR(cashflows)
	})
}

func APR(db *gorm.DB, ps []posting.Posting) decimal.Decimal {
	today := utils.EndOfToday()
	marketAmount := utils.SumBy(ps, func(p posting.Posting) decimal.Decimal {
		return p.MarketAmount
	})
	cashflows := lo.Map(ps, func(p posting.Posting, _ int) xirr.Cashflow {
		return xirr.Cashflow{Date: p.Date, Amount: GetMarketPrice(db, p, p.Date).Round(4).InexactFloat64()}
	})
	cashflows = append(cashflows, xirr.Cashflow{Date: today, Amount: marketAmount.Neg().Round(4).InexactFloat64()})

	return cache.Lookup(db, cashflows, func() decimal.Decimal {
		return xirr.XIRR(cashflows)
	})
}

// WarmXIRRCache pre-computes XIRR for every investment account and stores the
// results in the SQLite computation cache so that subsequent API calls are
// served from cache rather than recomputing the expensive Newton-Raphson
// iteration on each request.
//
// It returns a (possibly empty) slice of human-readable diagnostic messages –
// one entry per account whose XIRR calculation did not converge.  Callers
// (e.g. the background sync job) can surface these as job Details so operators
// can identify data problems without having to inspect server logs.
func WarmXIRRCache(db *gorm.DB) []string {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%").NotAccountPrefix("Assets:Checking").All()
	postings = PopulateMarketPrice(db, postings)

	byAccount := lo.GroupBy(postings, func(p posting.Posting) string {
		if IsCapitalGains(p) {
			return CapitalGainsSourceAccount(p.Account)
		}
		return p.Account
	})

	// Process accounts in a deterministic order so the warnings slice is stable
	// across runs.
	accounts := make([]string, 0, len(byAccount))
	for account := range byAccount {
		accounts = append(accounts, account)
	}
	sort.Strings(accounts)

	var warnings []string
	for _, account := range accounts {
		ps := byAccount[account]
		cashflows := buildXIRRCashflows(db, ps)

		// Use cache.Lookup with a closure so that:
		//   (a) if the result is already cached the value is returned as-is, and
		//   (b) if not yet cached we call XIRRWithConvergence so we can detect
		//       convergence failures and still populate the cache in one pass.
		var didNotConverge bool
		_ = cache.Lookup(db, cashflows, func() decimal.Decimal {
			result, ok := xirr.XIRRWithConvergence(cashflows)
			if !ok {
				didNotConverge = true
			}
			return result
		})

		if didNotConverge {
			warnings = append(warnings, fmt.Sprintf("XIRR did not converge for account: %s", account))
		}
	}

	return warnings
}
