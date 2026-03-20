package server

import (
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InvestmentYearlyCard struct {
	StartDate         time.Time         `json:"start_date"`
	EndDate           time.Time         `json:"end_date"`
	Postings          []posting.Posting `json:"postings"`
	GrossSalaryIncome decimal.Decimal   `json:"gross_salary_income"`
	GrossOtherIncome  decimal.Decimal   `json:"gross_other_income"`
	NetTax            decimal.Decimal   `json:"net_tax"`
	NetIncome         decimal.Decimal   `json:"net_income"`
	NetInvestment     decimal.Decimal   `json:"net_investment"`
	NetExpense        decimal.Decimal   `json:"net_expense"`
	SavingsRate       decimal.Decimal   `json:"savings_rate"`
}

func GetInvestment(db *gorm.DB, reportCurrency string) gin.H {
	assets := query.Init(db).Like("Assets:%").NotAccountPrefix("Assets:Checking").
		Where("transaction_id not in (select transaction_id from postings p where p.account like ? and p.transaction_id = transaction_id)", "Liabilities:%").
		All()
	incomes := query.Init(db).Like("Income:%").All()
	expenses := query.Init(db).Like("Expenses:%").All()
	p := query.Init(db).First()

	if p == nil {
		return gin.H{"assets": []posting.Posting{}, "yearly_cards": []InvestmentYearlyCard{}}
	}

	assets = lo.Filter(assets, func(p posting.Posting, _ int) bool { return !service.IsStockSplit(db, p) })
	if reportCurrency != "" && reportCurrency != config.DefaultCurrency() {
		assets = convertPostingsToReportCurrency(db, assets, reportCurrency)
		incomes = convertPostingsToReportCurrency(db, incomes, reportCurrency)
		expenses = convertPostingsToReportCurrency(db, expenses, reportCurrency)
	}
	return gin.H{"assets": assets, "yearly_cards": computeInvestmentYearlyCard(p.Date, assets, expenses, incomes)}
}

// convertPostingsToReportCurrency converts each posting's Amount (and MarketAmount)
// to the report currency by applying the exchange rate from the default currency
// on the posting's date.  Posting amounts in Paisa are always denominated in the
// default currency regardless of the p.Commodity field.  When no rate is available
// for a given date, the posting is kept unchanged so the caller always receives a
// complete result set.
func convertPostingsToReportCurrency(db *gorm.DB, postings []posting.Posting, reportCurrency string) []posting.Posting {
	defaultCurrency := config.DefaultCurrency()
	result := make([]posting.Posting, 0, len(postings))
	for _, p := range postings {
		rate, ok := service.GetRate(db, defaultCurrency, reportCurrency, p.Date)
		if !ok {
			result = append(result, p)
			continue
		}
		converted := p
		converted.Amount = p.Amount.Mul(rate)
		if !p.MarketAmount.IsZero() {
			converted.MarketAmount = p.MarketAmount.Mul(rate)
		}
		result = append(result, converted)
	}
	return result
}

func computeInvestmentYearlyCard(start time.Time, assets []posting.Posting, expenses []posting.Posting, incomes []posting.Posting) []InvestmentYearlyCard {
	var yearlyCards []InvestmentYearlyCard = make([]InvestmentYearlyCard, 0)

	if len(assets) == 0 {
		return yearlyCards
	}

	var p posting.Posting
	end := utils.EndOfToday()
	for start = utils.BeginningOfFinancialYear(start); start.Before(end); start = start.AddDate(1, 0, 0) {
		yearEnd := utils.EndOfFinancialYear(start)
		var currentYearPostings []posting.Posting = make([]posting.Posting, 0)
		for len(assets) > 0 && utils.IsWithDate(assets[0].Date, start, yearEnd) {
			p, assets = assets[0], assets[1:]
			currentYearPostings = append(currentYearPostings, p)
		}

		var currentYearTaxes []posting.Posting = make([]posting.Posting, 0)
		var currentYearExpenses []posting.Posting = make([]posting.Posting, 0)

		for len(expenses) > 0 && utils.IsWithDate(expenses[0].Date, start, yearEnd) {
			p, expenses = expenses[0], expenses[1:]
			if utils.IsSameOrParent(p.Account, "Expenses:Tax") {
				currentYearTaxes = append(currentYearTaxes, p)
			} else {
				currentYearExpenses = append(currentYearExpenses, p)
			}
		}

		netTax := accounting.CostSum(currentYearTaxes)
		netExpense := accounting.CostSum(currentYearExpenses)

		var currentYearIncomes []posting.Posting = make([]posting.Posting, 0)
		for len(incomes) > 0 && utils.IsWithDate(incomes[0].Date, start, yearEnd) {
			p, incomes = incomes[0], incomes[1:]
			currentYearIncomes = append(currentYearIncomes, p)
		}

		grossSalaryIncome := utils.SumBy(currentYearIncomes, func(p posting.Posting) decimal.Decimal {
			if strings.HasPrefix(p.Account, "Income:Salary") {
				return p.Amount.Neg()
			} else {
				return decimal.Zero
			}
		})
		grossOtherIncome := utils.SumBy(currentYearIncomes, func(p posting.Posting) decimal.Decimal {
			if !strings.HasPrefix(p.Account, "Income:Salary") {
				return p.Amount.Neg()
			} else {
				return decimal.Zero
			}
		})

		netInvestment := accounting.CostSum(currentYearPostings)

		netIncome := grossSalaryIncome.Add(grossOtherIncome).Sub(netTax)
		var savingsRate decimal.Decimal = decimal.Zero
		if !netIncome.Equal(decimal.Zero) {
			savingsRate = (netInvestment.Div(netIncome)).Mul(decimal.NewFromInt(100))
		}

		yearlyCards = append(yearlyCards, InvestmentYearlyCard{
			StartDate:         start,
			EndDate:           yearEnd,
			Postings:          currentYearPostings,
			NetTax:            netTax,
			GrossSalaryIncome: grossSalaryIncome,
			GrossOtherIncome:  grossOtherIncome,
			NetIncome:         netIncome,
			NetInvestment:     netInvestment,
			NetExpense:        netExpense,
			SavingsRate:       savingsRate,
		})

	}
	return yearlyCards
}
