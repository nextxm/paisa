package server

import (
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type IncomeStatement struct {
	StartingBalance decimal.Decimal            `json:"startingBalance"`
	EndingBalance   decimal.Decimal            `json:"endingBalance"`
	Date            time.Time                  `json:"date"`
	Income          map[string]decimal.Decimal `json:"income"`
	Interest        map[string]decimal.Decimal `json:"interest"`
	Equity          map[string]decimal.Decimal `json:"equity"`
	Pnl             map[string]decimal.Decimal `json:"pnl"`
	Liabilities     map[string]decimal.Decimal `json:"liabilities"`
	Tax             map[string]decimal.Decimal `json:"tax"`
	Expenses        map[string]decimal.Decimal `json:"expenses"`
}

func GetIncomeStatement(db *gorm.DB) gin.H {
	postings := query.Init(db).All()
	statements := computeStatement(db, postings)
	return gin.H{"yearly": statements}
}

// networthAt computes the total net worth (assets at market value + liabilities
// at book value, with liabilities being naturally negative) from all postings
// on or before `date`.  This mirrors the Networth page so that startingBalance
// and endingBalance always agree with the Networth page values.
//
// For currency (default-currency) postings: p.Quantity == p.Amount, and
// GetPrice returns quantity as-is, so cash accounts sum correctly.
// For non-currency postings: units are accumulated and priced at `date`.
// Liability accounts have negative amounts/quantities in ledger double-entry,
// so their contribution is naturally a negative number that reduces net worth.
func networthAt(db *gorm.DB, allPostings []posting.Posting, date time.Time) decimal.Decimal {
	// Clamp to today so future dates don't project beyond available prices.
	today := utils.EndOfToday()
	if date.After(today) {
		date = today
	}
	endOfDay := utils.EndOfDay(date)

	// account → commodity → net quantity
	quantities := make(map[string]map[string]decimal.Decimal)

	for _, p := range allPostings {
		if p.Date.After(endOfDay) {
			continue
		}
		category := utils.FirstName(p.Account)
		if category != "Assets" && category != "Liabilities" {
			continue
		}
		if quantities[p.Account] == nil {
			quantities[p.Account] = make(map[string]decimal.Decimal)
		}
		quantities[p.Account][p.Commodity] = quantities[p.Account][p.Commodity].Add(p.Quantity)
	}

	total := decimal.Zero
	for _, commodities := range quantities {
		for commodity, qty := range commodities {
			if qty.IsZero() {
				continue
			}
			// GetPrice returns qty for currencies, market value for others.
			// Liabilities have negative quantities, so they naturally reduce
			// total — no special-casing needed.
			total = total.Add(service.GetPrice(db, commodity, qty, date))
		}
	}
	return total
}

func computeStatement(db *gorm.DB, postings []posting.Posting) map[string]IncomeStatement {
	statements := make(map[string]IncomeStatement)

	grouped := utils.GroupByFY(postings)
	fys := make([]string, 0, len(grouped))
	for fy := range grouped {
		fys = append(fys, fy)
	}
	sort.Strings(fys)

	// runnings tracks the market-value-adjusted cost basis and unit count per
	// asset account across all FYs so that per-FY unrealised Pnl is accurate.
	type runningBalance struct {
		amount   decimal.Decimal            // running market-adjusted cost basis
		quantity map[string]decimal.Decimal // non-currency commodity → net units
	}
	runnings := make(map[string]runningBalance)

	// Cap the effective end of the current (in-progress) FY to today so that
	// endingBalance and the Pnl snapshot match the Networth page.
	today := utils.EndOfToday()

	for _, fy := range fys {
		incomeStatement := IncomeStatement{}
		start, end := utils.ParseFY(fy)
		if end.After(today) {
			end = today
		}
		incomeStatement.Date = start

		// startingBalance / endingBalance are always computed directly from the
		// balance sheet rather than being derived from income flows.  This makes
		// them immune to categorisation choices and sign-convention bugs, and
		// guarantees they match the Networth page.
		incomeStatement.StartingBalance = networthAt(db, postings, start.Add(-1))
		incomeStatement.EndingBalance = networthAt(db, postings, end)

		type categoryQuantities map[string]map[string]decimal.Decimal
		acc := struct {
			Income      categoryQuantities
			Interest    categoryQuantities
			Equity      categoryQuantities
			Liabilities categoryQuantities
			Tax         categoryQuantities
			Expenses    categoryQuantities
		}{
			Income:      make(categoryQuantities),
			Interest:    make(categoryQuantities),
			Equity:      make(categoryQuantities),
			Liabilities: make(categoryQuantities),
			Tax:         make(categoryQuantities),
			Expenses:    make(categoryQuantities),
		}

		updateAcc := func(cat categoryQuantities, account, commodity string, qty decimal.Decimal) {
			if cat[account] == nil {
				cat[account] = make(map[string]decimal.Decimal)
			}
			cat[account][commodity] = cat[account][commodity].Add(qty)
		}

		incomeStatement.Income = make(map[string]decimal.Decimal)
		incomeStatement.Interest = make(map[string]decimal.Decimal)
		incomeStatement.Equity = make(map[string]decimal.Decimal)
		incomeStatement.Pnl = make(map[string]decimal.Decimal)
		incomeStatement.Liabilities = make(map[string]decimal.Decimal)
		incomeStatement.Tax = make(map[string]decimal.Decimal)
		incomeStatement.Expenses = make(map[string]decimal.Decimal)

		for _, p := range grouped[fy] {
			category := utils.FirstName(p.Account)

			switch category {
			case "Income":
				if service.IsCapitalGains(p) {
					// Realised capital gains: update the cost-basis running total
					// for the source asset account so that subsequent-year
					// unrealised Pnl remains accurate.
					sourceAccount := service.CapitalGainsSourceAccount(p.Account)
					r := runnings[sourceAccount]
					if r.quantity == nil {
						r.quantity = make(map[string]decimal.Decimal)
					}
					r.amount = r.amount.Add(p.Amount)
					runnings[sourceAccount] = r
				} else if strings.HasPrefix(p.Account, "Income:Interest") {
					updateAcc(acc.Interest, p.Account, p.Commodity, p.Quantity)
				} else {
					updateAcc(acc.Income, p.Account, p.Commodity, p.Quantity)
				}
			case "Equity":
				updateAcc(acc.Equity, p.Account, p.Commodity, p.Quantity)
			case "Expenses":
				if strings.HasPrefix(p.Account, "Expenses:Tax") {
					updateAcc(acc.Tax, p.Account, p.Commodity, p.Quantity)
				} else {
					updateAcc(acc.Expenses, p.Account, p.Commodity, p.Quantity)
				}
			case "Liabilities":
				updateAcc(acc.Liabilities, p.Account, p.Commodity, p.Quantity)
			case "Assets":
				r := runnings[p.Account]
				if r.quantity == nil {
					r.quantity = make(map[string]decimal.Decimal)
				}
				if !utils.IsCurrency(p.Commodity) {
					r.amount = r.amount.Add(p.Amount)
					r.quantity[p.Commodity] = r.quantity[p.Commodity].Add(p.Quantity)
				}
				runnings[p.Account] = r
			default:
				// ignore
			}
		}

		finalizeAcc := func(target map[string]decimal.Decimal, cat categoryQuantities) {
			for account, commodities := range cat {
				for commodity, qty := range commodities {
					target[account] = target[account].Add(service.GetPrice(db, commodity, qty, end))
				}
			}
		}

		finalizeAcc(incomeStatement.Income, acc.Income)
		finalizeAcc(incomeStatement.Interest, acc.Interest)
		finalizeAcc(incomeStatement.Equity, acc.Equity)
		finalizeAcc(incomeStatement.Liabilities, acc.Liabilities)
		finalizeAcc(incomeStatement.Tax, acc.Tax)
		finalizeAcc(incomeStatement.Expenses, acc.Expenses)

		// Compute per-account unrealised Pnl: market value of currently-held
		// units at `end` minus the running cash cost-basis.
		for account, r := range runnings {
			diff := r.amount.Neg()
			for commodity, quantity := range r.quantity {
				diff = diff.Add(service.GetPrice(db, commodity, quantity, end))
			}
			incomeStatement.Pnl[account] = diff

			// Carry forward the market-value-adjusted basis into the next FY
			// so subsequent-year Pnl represents only the incremental gain.
			r.amount = r.amount.Add(diff)
			runnings[account] = r
		}

		// Strip zero-value entries (cosmetic clean-up).
		sumBalance(incomeStatement.Income)
		sumBalance(incomeStatement.Interest)
		sumBalance(incomeStatement.Equity)
		sumBalance(incomeStatement.Pnl)
		sumBalance(incomeStatement.Liabilities)
		sumBalance(incomeStatement.Tax)
		sumBalance(incomeStatement.Expenses)

		statements[fy] = incomeStatement
	}

	return statements
}

func sumBalance(breakdown map[string]decimal.Decimal) decimal.Decimal {
	total := decimal.Zero
	for k, v := range breakdown {
		total = total.Add(v)

		if v.Equal(decimal.Zero) {
			delete(breakdown, k)
		}
	}
	return total
}
