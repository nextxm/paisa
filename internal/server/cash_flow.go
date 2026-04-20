package server

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	cashFlowForecastMonths   = 12
	cashFlowHistoricalMonths = 3
)

// CashFlowForecast represents a single month's projected income, expense, and
// running checking-account balance.
type CashFlowForecast struct {
	Date    time.Time       `json:"date"`
	Income  decimal.Decimal `json:"income"`
	Expense decimal.Decimal `json:"expense"`
	Balance decimal.Decimal `json:"balance"`
}

// GetCashFlowForecast combines recurring TransactionSequence schedules with
// trailing 3-month averages to produce 12 months of projected cash flow.
func GetCashFlowForecast(db *gorm.DB) gin.H {
	return gin.H{"forecasts": computeCashFlowForecast(db)}
}

func computeCashFlowForecast(db *gorm.DB) []CashFlowForecast {
	now := utils.Now()

	// Current total checking balance.
	balance := accounting.CostSum(query.Init(db).AccountPrefix("Assets:Checking").All())

	// Fetch historical income and expense postings for the trailing window.
	histQuery := query.Init(db).LastNMonths(cashFlowHistoricalMonths)
	incomePostings := histQuery.Clone().Like("Income:%").All()
	expensePostings := histQuery.Clone().Like("Expenses:%").NotAccountPrefix("Expenses:Tax").All()

	incomeByMonth := utils.GroupByMonth(incomePostings)
	expenseByMonth := utils.GroupByMonth(expensePostings)

	// Get all recurring sequences.
	sequences := ComputeRecurringTransactions(query.Init(db).All())

	// Accumulate the recurring portion of income/expense per historical month.
	histStart := utils.BeginningOfMonth(now).AddDate(0, -(cashFlowHistoricalMonths - 1), 0)
	recurringIncByMonth := make(map[string]decimal.Decimal)
	recurringExpByMonth := make(map[string]decimal.Decimal)

	for _, seq := range sequences {
		for _, t := range seq.Transactions {
			if t.Date.Before(histStart) {
				continue
			}
			key := t.Date.Format("2006-01")
			for _, p := range t.Postings {
				if strings.HasPrefix(p.Account, "Income:") {
					recurringIncByMonth[key] = recurringIncByMonth[key].Add(p.Amount.Neg())
				} else if strings.HasPrefix(p.Account, "Expenses:") &&
					!strings.HasPrefix(p.Account, "Expenses:Tax") {
					recurringExpByMonth[key] = recurringExpByMonth[key].Add(p.Amount)
				}
			}
		}
	}

	// Sum totals over the historical period.
	var totalIncome, totalExpense decimal.Decimal
	var totalRecurringInc, totalRecurringExp decimal.Decimal

	for i := 0; i < cashFlowHistoricalMonths; i++ {
		m := histStart.AddDate(0, i, 0)
		key := m.Format("2006-01")
		if ps, ok := incomeByMonth[key]; ok {
			totalIncome = totalIncome.Add(accounting.CostSum(ps).Neg())
		}
		if ps, ok := expenseByMonth[key]; ok {
			totalExpense = totalExpense.Add(accounting.CostSum(ps))
		}
		totalRecurringInc = totalRecurringInc.Add(recurringIncByMonth[key])
		totalRecurringExp = totalRecurringExp.Add(recurringExpByMonth[key])
	}

	// Average non-recurring monthly income/expense (clamped to zero).
	n := decimal.NewFromInt(int64(cashFlowHistoricalMonths))
	avgNonRecInc := maxDecimal(totalIncome.Sub(totalRecurringInc).Div(n), decimal.Zero)
	avgNonRecExp := maxDecimal(totalExpense.Sub(totalRecurringExp).Div(n), decimal.Zero)

	// Project forward cashFlowForecastMonths months starting from next month.
	forecastStart := utils.BeginningOfMonth(now).AddDate(0, 1, 0)
	forecasts := make([]CashFlowForecast, 0, cashFlowForecastMonths)

	for i := 0; i < cashFlowForecastMonths; i++ {
		m := forecastStart.AddDate(0, i, 0)

		projInc := decimal.Zero
		projExp := decimal.Zero

		for _, seq := range sequences {
			if seq.Interval <= 0 || len(seq.Transactions) == 0 {
				continue
			}
			// ComputeRecurringTransactions sorts Transactions descending by date,
			// so index 0 is always the most recent occurrence.
			mostRecent := seq.Transactions[0]
			count := recurringOccurrencesInMonth(mostRecent.Date, seq.Interval, m)
			if count == 0 {
				continue
			}
			mult := decimal.NewFromInt(int64(count))
			for _, p := range mostRecent.Postings {
				if strings.HasPrefix(p.Account, "Income:") {
					projInc = projInc.Add(p.Amount.Neg().Mul(mult))
				} else if strings.HasPrefix(p.Account, "Expenses:") &&
					!strings.HasPrefix(p.Account, "Expenses:Tax") {
					projExp = projExp.Add(p.Amount.Mul(mult))
				}
			}
		}

		income := avgNonRecInc.Add(projInc)
		expense := avgNonRecExp.Add(projExp)
		balance = balance.Add(income).Sub(expense)

		forecasts = append(forecasts, CashFlowForecast{
			Date:    m,
			Income:  income,
			Expense: expense,
			Balance: balance,
		})
	}

	return forecasts
}

// recurringOccurrencesInMonth counts how many times a recurring transaction
// (identified by its last occurrence date and day-interval) falls within the
// given calendar month.
func recurringOccurrencesInMonth(lastDate time.Time, interval int, monthStart time.Time) int {
	if interval <= 0 {
		return 0
	}
	monthEnd := monthStart.AddDate(0, 1, 0)
	next := lastDate
	for next.Before(monthStart) {
		next = next.AddDate(0, 0, interval)
	}
	count := 0
	for next.Before(monthEnd) {
		count++
		next = next.AddDate(0, 0, interval)
	}
	return count
}

// maxDecimal returns the larger of two decimal.Decimal values.
func maxDecimal(a, b decimal.Decimal) decimal.Decimal {
	if a.GreaterThanOrEqual(b) {
		return a
	}
	return b
}

type CashFlow struct {
	Date        time.Time       `json:"date"`
	Income      decimal.Decimal `json:"income"`
	Expenses    decimal.Decimal `json:"expenses"`
	Liabilities decimal.Decimal `json:"liabilities"`
	Investment  decimal.Decimal `json:"investment"`
	Tax         decimal.Decimal `json:"tax"`
	Checking    decimal.Decimal `json:"checking"`
	Balance     decimal.Decimal `json:"balance"`
}

func (c CashFlow) GroupDate() time.Time {
	return c.Date
}

func GetCashFlow(db *gorm.DB) gin.H {
	return gin.H{"cash_flows": computeCashFlow(db, query.Init(db), decimal.Zero)}
}

func GetCurrentCashFlow(db *gorm.DB) []CashFlow {
	balance := accounting.CostSum(query.Init(db).BeforeNMonths(3).AccountPrefix("Assets:Checking").All())
	return computeCashFlow(db, query.Init(db).LastNMonths(3), balance)
}

func computeCashFlow(db *gorm.DB, q *query.Query, balance decimal.Decimal) []CashFlow {
	var cashFlows []CashFlow

	expenses := utils.GroupByMonth(q.Clone().Like("Expenses:%").NotAccountPrefix("Expenses:Tax").All())
	incomes := utils.GroupByMonth(q.Clone().Like("Income:%").All())
	liabilities := utils.GroupByMonth(q.Clone().Like("Liabilities:%").All())
	investments := utils.GroupByMonth(q.Clone().Like("Assets:%").NotAccountPrefix("Assets:Checking").All())
	taxes := utils.GroupByMonth(q.Clone().AccountPrefix("Expenses:Tax").All())
	checkings := utils.GroupByMonth(q.Clone().AccountPrefix("Assets:Checking").All())
	postings := q.Clone().All()

	if len(postings) == 0 {
		return []CashFlow{}
	}

	end := utils.MaxTime(utils.EndOfToday(), postings[len(postings)-1].Date)
	for start := utils.BeginningOfMonth(postings[0].Date); start.Before(end); start = start.AddDate(0, 1, 0) {
		cashFlow := CashFlow{Date: start}

		key := start.Format("2006-01")
		ps, ok := expenses[key]
		if ok {
			cashFlow.Expenses = accounting.CostSum(ps)
		}

		ps, ok = incomes[key]
		if ok {
			cashFlow.Income = accounting.CostSum(ps).Neg()
		}

		ps, ok = liabilities[key]
		if ok {
			cashFlow.Liabilities = accounting.CostSum(ps).Neg()
		}

		ps, ok = investments[key]
		if ok {
			cashFlow.Investment = accounting.CostSum(ps)
		}

		ps, ok = taxes[key]
		if ok {
			cashFlow.Tax = accounting.CostSum(ps)
		}

		ps, ok = checkings[key]
		if ok {
			cashFlow.Checking = accounting.CostSum(ps)
		}

		balance = balance.Add(cashFlow.Checking)
		cashFlow.Balance = balance

		cashFlows = append(cashFlows, cashFlow)
	}

	return cashFlows
}
