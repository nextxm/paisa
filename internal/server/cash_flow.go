package server

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/monthly_account_balance"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

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
	if rows, err := monthly_account_balance.All(db); err == nil && len(rows) > 0 {
		return gin.H{"cash_flows": computeCashFlowFromMonthlyBalances(rows, decimal.Zero)}
	}
	return gin.H{"cash_flows": computeCashFlow(query.Init(db).NotInactive(), decimal.Zero)}
}

func isInactiveAccount(account string, conf config.Config) bool {
	if matchesAnyAccountPrefix(account, conf.InactiveAccounts) {
		return true
	}
	for _, a := range conf.Accounts {
		if a.Inactive && a.Name == account {
			return true
		}
	}
	return false
}

type cashFlowMonthlySums struct {
	income      decimal.Decimal
	expenses    decimal.Decimal
	liabilities decimal.Decimal
	investment  decimal.Decimal
	tax         decimal.Decimal
	checking    decimal.Decimal
}

func accumulateCashFlowBucket(ms *cashFlowMonthlySums, account string, amount decimal.Decimal, checkingAccounts []string) {
	switch {
	case isAccountPrefix(account, "Expenses:Tax"):
		ms.tax = ms.tax.Add(amount)
	case strings.HasPrefix(account, "Expenses:"):
		ms.expenses = ms.expenses.Add(amount)
	case strings.HasPrefix(account, "Income:"):
		ms.income = ms.income.Add(amount)
	case strings.HasPrefix(account, "Liabilities:"):
		ms.liabilities = ms.liabilities.Add(amount)
	case len(checkingAccounts) > 0 && matchesAnyAccountPrefix(account, checkingAccounts):
		ms.checking = ms.checking.Add(amount)
	case strings.HasPrefix(account, "Assets:"):
		ms.investment = ms.investment.Add(amount)
	}
}

func buildCashFlowSeries(firstMonth, lastMonth time.Time, byMonth map[string]*cashFlowMonthlySums, balance decimal.Decimal) []CashFlow {
	var cashFlows []CashFlow
	end := utils.MaxTime(utils.EndOfToday(), lastMonth)
	for start := firstMonth; start.Before(end); start = start.AddDate(0, 1, 0) {
		cashFlow := CashFlow{Date: start}
		if ms, ok := byMonth[start.Format("2006-01")]; ok {
			cashFlow.Expenses = ms.expenses
			cashFlow.Income = ms.income.Neg()
			cashFlow.Liabilities = ms.liabilities.Neg()
			cashFlow.Investment = ms.investment
			cashFlow.Tax = ms.tax
			cashFlow.Checking = ms.checking
		}
		balance = balance.Add(cashFlow.Checking)
		cashFlow.Balance = balance
		cashFlows = append(cashFlows, cashFlow)
	}
	return cashFlows
}

func computeCashFlowFromMonthlyBalances(rows []monthly_account_balance.MonthlyAccountBalance, balance decimal.Decimal) []CashFlow {
	conf := config.GetConfig()
	checkingAccounts := conf.CheckingAccounts
	byMonth := make(map[string]*cashFlowMonthlySums)
	var firstMonth, lastMonth time.Time

	for _, r := range rows {
		if isInactiveAccount(r.Account, conf) {
			continue
		}
		monthDate, err := time.Parse("2006-01", r.Month)
		if err != nil {
			continue
		}
		if firstMonth.IsZero() || monthDate.Before(firstMonth) {
			firstMonth = monthDate
		}
		if lastMonth.IsZero() || monthDate.After(lastMonth) {
			lastMonth = monthDate
		}

		ms := byMonth[r.Month]
		if ms == nil {
			ms = &cashFlowMonthlySums{}
			byMonth[r.Month] = ms
		}
		accumulateCashFlowBucket(ms, r.Account, r.Amount, checkingAccounts)
	}

	if firstMonth.IsZero() {
		return []CashFlow{}
	}
	return buildCashFlowSeries(firstMonth, lastMonth, byMonth, balance)
}

func GetCurrentCashFlow(db *gorm.DB) []CashFlow {
	checkingAccounts := config.GetConfig().CheckingAccounts
	var balance decimal.Decimal
	if len(checkingAccounts) > 0 {
		for _, s := range query.Init(db).NotInactive().BeforeNMonths(3).AccountPrefix(checkingAccounts...).GroupSum() {
			balance = balance.Add(s.Amount)
		}
	}
	return computeCashFlow(query.Init(db).NotInactive().LastNMonths(3), balance)
}

func isAccountPrefix(account, prefix string) bool {
	return account == prefix || strings.HasPrefix(account, prefix+":")
}

func matchesAnyAccountPrefix(account string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if isAccountPrefix(account, prefix) {
			return true
		}
	}
	return false
}

func computeCashFlow(q *query.Query, balance decimal.Decimal) []CashFlow {
	postings := q.All()
	if len(postings) == 0 {
		return []CashFlow{}
	}

	checkingAccounts := config.GetConfig().CheckingAccounts
	byMonth := make(map[string]*cashFlowMonthlySums)

	for _, p := range postings {
		key := p.Date.Format("2006-01")
		ms := byMonth[key]
		if ms == nil {
			ms = &cashFlowMonthlySums{}
			byMonth[key] = ms
		}
		accumulateCashFlowBucket(ms, p.Account, p.Amount, checkingAccounts)
	}

	return buildCashFlowSeries(utils.BeginningOfMonth(postings[0].Date), postings[len(postings)-1].Date, byMonth, balance)
}
