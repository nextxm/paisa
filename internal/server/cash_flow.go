package server

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
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
	return gin.H{"cash_flows": computeCashFlow(query.Init(db).NotInactive(), decimal.Zero)}
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
	type monthlySums struct {
		income      decimal.Decimal
		expenses    decimal.Decimal
		liabilities decimal.Decimal
		investment  decimal.Decimal
		tax         decimal.Decimal
		checking    decimal.Decimal
	}
	byMonth := make(map[string]*monthlySums)

	for _, p := range postings {
		key := p.Date.Format("2006-01")
		ms := byMonth[key]
		if ms == nil {
			ms = &monthlySums{}
			byMonth[key] = ms
		}

		switch {
		case isAccountPrefix(p.Account, "Expenses:Tax"):
			ms.tax = ms.tax.Add(p.Amount)
		case strings.HasPrefix(p.Account, "Expenses:"):
			ms.expenses = ms.expenses.Add(p.Amount)
		case strings.HasPrefix(p.Account, "Income:"):
			ms.income = ms.income.Add(p.Amount)
		case strings.HasPrefix(p.Account, "Liabilities:"):
			ms.liabilities = ms.liabilities.Add(p.Amount)
		case len(checkingAccounts) > 0 && matchesAnyAccountPrefix(p.Account, checkingAccounts):
			ms.checking = ms.checking.Add(p.Amount)
		case strings.HasPrefix(p.Account, "Assets:"):
			ms.investment = ms.investment.Add(p.Amount)
		}
	}

	var cashFlows []CashFlow
	end := utils.MaxTime(utils.EndOfToday(), postings[len(postings)-1].Date)
	for start := utils.BeginningOfMonth(postings[0].Date); start.Before(end); start = start.AddDate(0, 1, 0) {
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
