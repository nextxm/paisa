package server

import (
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type InvestmentIncomeHolding struct {
	Type           string                     `json:"type"`
	Holding        string                     `json:"holding"`
	Postings       []posting.Posting          `json:"postings"`
	TotalIncome    decimal.Decimal            `json:"total_income"`
	TTMIncome      decimal.Decimal            `json:"ttm_income"`
	TTMYield       decimal.Decimal            `json:"ttm_yield"`
	CurrentBalance decimal.Decimal            `json:"current_balance"`
	YearlyIncome   map[string]decimal.Decimal `json:"yearly_income"`
}

type InvestmentIncomeTimelinePoint struct {
	Date         time.Time       `json:"date"`
	Dividend     decimal.Decimal `json:"dividend"`
	Interest     decimal.Decimal `json:"interest"`
	Distribution decimal.Decimal `json:"distribution"`
	Total        decimal.Decimal `json:"total"`
}

func investmentIncomeLikePatterns() []string {
	return []string{
		"Income:Dividend%",
		"Income:Interest%",
		"Income:Distribution%",
		"Income:Distributions%",
	}
}

func investmentIncomeKind(account string) (string, bool) {
	switch {
	case strings.HasPrefix(account, "Income:Dividend"):
		return "Dividend", true
	case strings.HasPrefix(account, "Income:Interest"):
		return "Interest", true
	case strings.HasPrefix(account, "Income:Distribution"), strings.HasPrefix(account, "Income:Distributions"):
		return "Distribution", true
	default:
		return "", false
	}
}

func investmentIncomeHoldingAccount(account string) string {
	parts := strings.Split(account, ":")
	if len(parts) < 3 {
		return account
	}
	holdingPath := strings.Join(parts[2:], ":")
	if holdingPath == "" {
		return account
	}
	if strings.HasPrefix(holdingPath, "Assets:") {
		return holdingPath
	}
	return "Assets:" + holdingPath
}

func isInvestmentIncomePosting(p posting.Posting) bool {
	_, ok := investmentIncomeKind(p.Account)
	return ok
}

func trailing12MonthsStart(asOfDate time.Time) time.Time {
	return utils.BeginningOfMonth(asOfDate).AddDate(0, -11, 0)
}

func computeInvestmentIncomeAmount(postings []posting.Posting) decimal.Decimal {
	return utils.SumBy(postings, func(p posting.Posting) decimal.Decimal { return p.Amount.Neg() })
}

func computeInvestmentIncomeByHolding(db *gorm.DB, asOfDate time.Time) []InvestmentIncomeHolding {
	patterns := investmentIncomeLikePatterns()
	incomes := query.Init(db).UntilDate(asOfDate).Like(patterns...).All()
	if len(incomes) == 0 {
		return []InvestmentIncomeHolding{}
	}

	ttmStart := trailing12MonthsStart(asOfDate)
	currentBalances := map[string]decimal.Decimal{}

	grouped := map[string][]posting.Posting{}
	for _, income := range incomes {
		kind, ok := investmentIncomeKind(income.Account)
		if !ok {
			continue
		}
		holding := investmentIncomeHoldingAccount(income.Account)
		key := kind + "|" + holding
		grouped[key] = append(grouped[key], income)
	}

	keys := utils.SortedKeys(grouped)
	result := make([]InvestmentIncomeHolding, 0, len(keys))
	for _, key := range keys {
		parts := strings.SplitN(key, "|", 2)
		kind, holding := parts[0], parts[1]
		incomePostings := grouped[key]
		totalIncome := computeInvestmentIncomeAmount(incomePostings)

		ttmPostings := make([]posting.Posting, 0, len(incomePostings))
		for _, p := range incomePostings {
			if !p.Date.Before(ttmStart) {
				ttmPostings = append(ttmPostings, p)
			}
		}
		ttmIncome := computeInvestmentIncomeAmount(ttmPostings)

		balance, found := currentBalances[holding]
		if !found {
			networth := computeNetworth(db, query.Init(db).UntilDate(asOfDate).AccountPrefix(holding).All())
			balance = networth.BalanceAmount
			currentBalances[holding] = balance
		}

		ttmYield := decimal.Zero
		if balance.GreaterThan(decimal.Zero) {
			ttmYield = ttmIncome.Div(balance).Mul(decimal.NewFromInt(100))
		}

		yearlyIncome := map[string]decimal.Decimal{}
		for _, p := range incomePostings {
			year := p.Date.Format("2006")
			yearlyIncome[year] = yearlyIncome[year].Add(p.Amount.Neg())
		}

		result = append(result, InvestmentIncomeHolding{
			Type:           kind,
			Holding:        holding,
			Postings:       incomePostings,
			TotalIncome:    totalIncome,
			TTMIncome:      ttmIncome,
			TTMYield:       ttmYield,
			CurrentBalance: balance,
			YearlyIncome:   yearlyIncome,
		})
	}
	return result
}

func computeInvestmentIncomeTimeline(holdings []InvestmentIncomeHolding) []InvestmentIncomeTimelinePoint {
	type timelineTotals struct {
		dividend     decimal.Decimal
		interest     decimal.Decimal
		distribution decimal.Decimal
	}

	monthTotals := map[string]timelineTotals{}
	for _, holding := range holdings {
		for _, p := range holding.Postings {
			month := p.Date.Format("2006-01")
			value := p.Amount.Neg()
			current := monthTotals[month]
			switch holding.Type {
			case "Dividend":
				current.dividend = current.dividend.Add(value)
			case "Interest":
				current.interest = current.interest.Add(value)
			default:
				current.distribution = current.distribution.Add(value)
			}
			monthTotals[month] = current
		}
	}

	months := utils.SortedKeys(monthTotals)
	timeline := make([]InvestmentIncomeTimelinePoint, 0, len(months))
	for _, month := range months {
		date, err := time.Parse("2006-01", month)
		if err != nil {
			continue
		}
		monthData := monthTotals[month]
		total := monthData.dividend.Add(monthData.interest).Add(monthData.distribution)
		timeline = append(timeline, InvestmentIncomeTimelinePoint{
			Date:         date,
			Dividend:     monthData.dividend,
			Interest:     monthData.interest,
			Distribution: monthData.distribution,
			Total:        total,
		})
	}

	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Date.Before(timeline[j].Date)
	})
	return timeline
}

func GetInvestmentIncome(db *gorm.DB) gin.H {
	holdings := computeInvestmentIncomeByHolding(db, utils.ToDate(utils.Now()))
	timeline := computeInvestmentIncomeTimeline(holdings)
	ttmTotal := utils.SumBy(holdings, func(h InvestmentIncomeHolding) decimal.Decimal {
		return h.TTMIncome
	})

	byType := map[string][]InvestmentIncomeHolding{
		"Dividend":     {},
		"Interest":     {},
		"Distribution": {},
	}
	for _, holding := range holdings {
		byType[holding.Type] = append(byType[holding.Type], holding)
	}

	return gin.H{
		"income_by_type": byType,
		"holdings":       holdings,
		"timeline":       timeline,
		"ttm_total":      ttmTotal,
	}
}
