package budget

import (
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// AccountBudget holds the budget details for a single account in a given month.
type AccountBudget struct {
	Account   string            `json:"account"`
	Forecast  decimal.Decimal   `json:"forecast"`
	Actual    decimal.Decimal   `json:"actual"`
	Rollover  decimal.Decimal   `json:"rollover"`
	Available decimal.Decimal   `json:"available"`
	Date      time.Time         `json:"date"`
	Expenses  []posting.Posting `json:"expenses"`
}

// Budget holds the aggregated budget for a single month.
type Budget struct {
	Date               time.Time       `json:"date"`
	Accounts           []AccountBudget `json:"accounts"`
	AvailableThisMonth decimal.Decimal `json:"availableThisMonth"`
	EndOfMonthBalance  decimal.Decimal `json:"endOfMonthBalance"`
	Forecast           decimal.Decimal `json:"forecast"`
}

// Summary is the result returned by Compute.
type Summary struct {
	BudgetsByMonth        map[string]Budget
	CheckingBalance       decimal.Decimal
	AvailableForBudgeting decimal.Decimal
}

// Compute orchestrates the full budget calculation for the given forecast
// and expense postings.
func Compute(db *gorm.DB, forecastPostings, expensesPostings []posting.Posting) Summary {
	checkingBalance := accounting.CostSum(query.Init(db).AccountPrefix("Assets:Checking").All())
	availableForBudgeting := checkingBalance

	forecasts := utils.GroupByMonth(forecastPostings)
	expenses := utils.GroupByMonth(expensesPostings)

	accounts := lo.Uniq(lo.Map(forecastPostings, func(p posting.Posting, _ int) string {
		return p.Account
	}))
	sort.Strings(accounts)

	budgetsByMonth := make(map[string]Budget)
	balance := make(map[string]decimal.Decimal)

	currentMonth := lo.Must(time.ParseInLocation("2006-01", utils.Now().Format("2006-01"), config.TimeZone()))

	if len(forecastPostings) > 0 {
		firstMonth := utils.BeginningOfMonth(forecastPostings[0].Date)
		end := utils.EndOfMonth(forecastPostings[len(forecastPostings)-1].Date)

		for month := firstMonth; month.Before(end) || month.Equal(end); month = month.AddDate(0, 1, 0) {
			monthKey := month.Format("2006-01")
			var accountBudgets []AccountBudget

			forecastsByMonth := forecasts[monthKey]
			date := lo.Must(time.ParseInLocation("2006-01", monthKey, config.TimeZone()))
			expensesByMonth, ok := expenses[monthKey]
			if !ok {
				expensesByMonth = []posting.Posting{}
			}

			forecastsByAccount := accounting.GroupByAccount(forecastsByMonth)
			expensesByAccount := accounting.GroupByAccount(expensesByMonth)

			for _, account := range accounts {
				fs := forecastsByAccount[account]
				es := PopExpenses(account, expensesByAccount)

				b := BuildAccountBudget(date, account, balance[account], fs, es, date.Before(currentMonth))
				if b.Available.IsPositive() {
					balance[account] = b.Available
				} else {
					balance[account] = decimal.Zero
				}

				accountBudgets = append(accountBudgets, b)
			}

			availableThisMonth := utils.SumBy(
				accountBudgets, func(b AccountBudget) decimal.Decimal {
					if b.Available.IsPositive() {
						return b.Available
					}
					return decimal.Zero
				})

			forecast := utils.SumBy(
				accountBudgets, func(b AccountBudget) decimal.Decimal {
					if b.Forecast.IsPositive() {
						return b.Forecast
					}
					return decimal.Zero
				})

			availableForBudgeting = availableForBudgeting.Sub(availableThisMonth)
			endOfMonthBalance := availableForBudgeting

			budgetsByMonth[monthKey] = Budget{
				Date:               date,
				Accounts:           accountBudgets,
				EndOfMonthBalance:  endOfMonthBalance,
				AvailableThisMonth: availableThisMonth,
				Forecast:           forecast,
			}
		}
	}

	return Summary{
		BudgetsByMonth:        budgetsByMonth,
		CheckingBalance:       checkingBalance,
		AvailableForBudgeting: availableForBudgeting,
	}
}

// BuildAccountBudget constructs an AccountBudget for the given account,
// applying rollover logic based on the configuration.
func BuildAccountBudget(date time.Time, account string, balance decimal.Decimal, forecasts []posting.Posting, expenses []posting.Posting, past bool) AccountBudget {
	forecast := accounting.CostSum(forecasts)
	actual := accounting.CostSum(expenses)

	rollover := decimal.Zero
	available := forecast.Sub(actual)
	if past {
		available = decimal.Zero
	}
	if config.GetConfig().Budget.Rollover == config.Yes {
		rollover = balance
		available = balance.Add(forecast.Sub(actual))
	}

	return AccountBudget{
		Account:   account,
		Forecast:  forecast,
		Actual:    actual,
		Rollover:  rollover,
		Available: available,
		Date:      date,
		Expenses:  expenses,
	}
}

// PopExpenses extracts from expensesByAccount the postings belonging to
// forecastAccount or any of its sub-accounts, removing them from the map.
func PopExpenses(forecastAccount string, expensesByAccount map[string][]posting.Posting) []posting.Posting {
	expenses := []posting.Posting{}
	for account, es := range expensesByAccount {
		if utils.IsSameOrParent(account, forecastAccount) {
			expenses = append(expenses, es...)
			delete(expensesByAccount, account)
		}
	}
	return expenses
}
