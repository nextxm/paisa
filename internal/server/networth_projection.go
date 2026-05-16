package server

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	defaultProjectionYears            = 15
	defaultProjectionConservativeCAGR = 8.0
	defaultProjectionExpectedCAGR     = 12.0
	defaultProjectionOptimisticCAGR   = 16.0
	defaultProjectionSWR              = 4.0
	oneCrore                          = 10000000
)

type NetworthProjectionPoint struct {
	Date          time.Time       `json:"date"`
	BalanceAmount decimal.Decimal `json:"balanceAmount"`
}

type NetworthProjectionMilestone struct {
	Label  string          `json:"label"`
	Date   time.Time       `json:"date"`
	Amount decimal.Decimal `json:"amount"`
}

type NetworthProjectionScenario struct {
	Conservative []NetworthProjectionPoint `json:"conservative"`
	Expected     []NetworthProjectionPoint `json:"expected"`
	Optimistic   []NetworthProjectionPoint `json:"optimistic"`
}

type NetworthProjectionRequest struct {
	Years               int
	ConservativeCAGR    decimal.Decimal
	ExpectedCAGR        decimal.Decimal
	OptimisticCAGR      decimal.Decimal
	MonthlyContribution decimal.Decimal
	SWR                 decimal.Decimal
}

type historicalProjectionInputs struct {
	MonthlyContribution decimal.Decimal
	SavingsRate         decimal.Decimal
	AnnualExpenses      decimal.Decimal
}

func parseNetworthProjectionRequest(c *gin.Context) (NetworthProjectionRequest, bool) {
	req := NetworthProjectionRequest{
		Years:            defaultProjectionYears,
		ConservativeCAGR: decimal.NewFromFloat(defaultProjectionConservativeCAGR),
		ExpectedCAGR:     decimal.NewFromFloat(defaultProjectionExpectedCAGR),
		OptimisticCAGR:   decimal.NewFromFloat(defaultProjectionOptimisticCAGR),
		SWR:              decimal.NewFromFloat(defaultProjectionSWR),
	}

	if yearsStr := c.Query("years"); yearsStr != "" {
		years, err := strconv.Atoi(yearsStr)
		if err != nil || years < 1 || years > 50 {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "years must be between 1 and 50")
			return NetworthProjectionRequest{}, false
		}
		req.Years = years
	}

	parseDecimal := func(name string, min decimal.Decimal, max decimal.Decimal, target *decimal.Decimal) bool {
		raw := c.Query(name)
		if raw == "" {
			return true
		}
		value, err := decimal.NewFromString(raw)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, fmt.Sprintf("%s must be a number", name))
			return false
		}
		if value.LessThan(min) || value.GreaterThan(max) {
			RespondError(
				c,
				http.StatusBadRequest,
				ErrCodeInvalidRequest,
				fmt.Sprintf("%s must be between %s and %s", name, min.String(), max.String()),
			)
			return false
		}
		*target = value
		return true
	}

	if !parseDecimal("conservative_cagr", decimal.NewFromInt(-50), decimal.NewFromInt(100), &req.ConservativeCAGR) {
		return NetworthProjectionRequest{}, false
	}
	if !parseDecimal("expected_cagr", decimal.NewFromInt(-50), decimal.NewFromInt(100), &req.ExpectedCAGR) {
		return NetworthProjectionRequest{}, false
	}
	if !parseDecimal("optimistic_cagr", decimal.NewFromInt(-50), decimal.NewFromInt(100), &req.OptimisticCAGR) {
		return NetworthProjectionRequest{}, false
	}
	if !parseDecimal("monthly_contribution", decimal.NewFromInt(-1000000000), decimal.NewFromInt(1000000000), &req.MonthlyContribution) {
		return NetworthProjectionRequest{}, false
	}
	if !parseDecimal("swr", decimal.NewFromFloat(0.1), decimal.NewFromInt(20), &req.SWR) {
		return NetworthProjectionRequest{}, false
	}

	return req, true
}

func GetNetworthProjection(db *gorm.DB, req NetworthProjectionRequest) gin.H {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()
	postings = service.PopulateMarketPrice(db, postings)
	currentNetworth := computeNetworth(db, postings).BalanceAmount

	derived := deriveProjectionInputs(db)
	monthlyContribution := req.MonthlyContribution
	if monthlyContribution.IsZero() {
		monthlyContribution = derived.MonthlyContribution
	}

	months := req.Years * 12
	now := utils.ToDate(utils.Now())
	conservative := projectNetworth(now, currentNetworth, monthlyContribution, req.ConservativeCAGR, months)
	expected := projectNetworth(now, currentNetworth, monthlyContribution, req.ExpectedCAGR, months)
	optimistic := projectNetworth(now, currentNetworth, monthlyContribution, req.OptimisticCAGR, months)

	targetCorpus := decimal.Zero
	fireProgress := decimal.Zero
	var yearsToFIRE *decimal.Decimal
	if req.SWR.GreaterThan(decimal.Zero) {
		targetCorpus = derived.AnnualExpenses.Div(req.SWR.Div(decimal.NewFromInt(100)))
		if targetCorpus.GreaterThan(decimal.Zero) {
			fireProgress = currentNetworth.Div(targetCorpus).Mul(decimal.NewFromInt(100))
			if fireProgress.GreaterThan(decimal.NewFromInt(100)) {
				fireProgress = decimal.NewFromInt(100)
			}
			if crossedDate, ok := firstCrossingDate(expected, targetCorpus); ok {
				monthsToFire := monthDiff(now, crossedDate)
				years := decimal.NewFromInt(int64(monthsToFire)).Div(decimal.NewFromInt(12)).Round(2)
				yearsToFIRE = &years
			}
		}
	}

	return gin.H{
		"current_networth":      currentNetworth,
		"savings_rate":          derived.SavingsRate.Round(2),
		"monthly_contribution":  monthlyContribution.Round(2),
		"derived_contribution":  derived.MonthlyContribution.Round(2),
		"annual_expenses":       derived.AnnualExpenses.Round(2),
		"swr":                   req.SWR.Round(2),
		"target_corpus":         targetCorpus.Round(2),
		"years_to_fire":         yearsToFIRE,
		"fire_progress_percent": fireProgress.Round(2),
		"projection":            NetworthProjectionScenario{Conservative: conservative, Expected: expected, Optimistic: optimistic},
		"milestones":            projectionMilestones(expected, targetCorpus),
		"conservative_cagr":     req.ConservativeCAGR.Round(2),
		"expected_cagr":         req.ExpectedCAGR.Round(2),
		"optimistic_cagr":       req.OptimisticCAGR.Round(2),
	}
}

func deriveProjectionInputs(db *gorm.DB) historicalProjectionInputs {
	incomePostings := query.Init(db).Like("Income:%").LastNMonths(12).All()
	assetPostings := query.Init(db).
		Like("Assets:%").
		NotAccountPrefix("Assets:Checking").
		Where("transaction_id not in (select transaction_id from postings p where p.account like ? and p.transaction_id = transaction_id)", "Liabilities:%").
		LastNMonths(12).
		All()
	expensePostings := query.Init(db).Like("Expenses:%").NotAccountPrefix("Expenses:Tax").LastNMonths(12).All()

	assetPostings = filterStockSplitPostings(db, assetPostings)
	netIncome := accounting.CostSum(incomePostings).Neg()
	netInvestment := accounting.CostSum(assetPostings)

	monthsForSavings := maxInt(monthsCovered(incomePostings), monthsCovered(assetPostings))
	if monthsForSavings < 1 {
		monthsForSavings = 1
	}
	monthsForExpenses := monthsCovered(expensePostings)
	if monthsForExpenses < 1 {
		monthsForExpenses = 1
	}

	savingsRate := decimal.Zero
	if netIncome.GreaterThan(decimal.Zero) {
		savingsRate = netInvestment.Div(netIncome).Mul(decimal.NewFromInt(100))
	}

	monthlyContribution := netInvestment.Div(decimal.NewFromInt(int64(monthsForSavings)))
	if savingsRate.GreaterThan(decimal.Zero) && netIncome.GreaterThan(decimal.Zero) {
		avgMonthlyIncome := netIncome.Div(decimal.NewFromInt(int64(monthsForSavings)))
		monthlyContribution = avgMonthlyIncome.Mul(savingsRate).Div(decimal.NewFromInt(100))
	}

	totalExpenses := accounting.CostSum(expensePostings)
	annualExpenses := totalExpenses.
		Div(decimal.NewFromInt(int64(monthsForExpenses))).
		Mul(decimal.NewFromInt(12))

	return historicalProjectionInputs{
		MonthlyContribution: monthlyContribution,
		SavingsRate:         savingsRate,
		AnnualExpenses:      annualExpenses,
	}
}

func filterStockSplitPostings(db *gorm.DB, postings []posting.Posting) []posting.Posting {
	filtered := make([]posting.Posting, 0, len(postings))
	for _, p := range postings {
		if !service.IsStockSplit(db, p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func projectNetworth(
	startDate time.Time,
	currentNetworth decimal.Decimal,
	monthlyContribution decimal.Decimal,
	cagrPercent decimal.Decimal,
	months int,
) []NetworthProjectionPoint {
	if months <= 0 {
		return []NetworthProjectionPoint{}
	}

	cagr, _ := cagrPercent.Div(decimal.NewFromInt(100)).Float64()
	monthlyRate := decimal.NewFromFloat(math.Pow(1+cagr, 1.0/12) - 1)
	points := make([]NetworthProjectionPoint, 0, months)
	current := currentNetworth
	for i := 1; i <= months; i++ {
		current = current.Mul(decimal.NewFromInt(1).Add(monthlyRate)).Add(monthlyContribution)
		points = append(points, NetworthProjectionPoint{
			Date:          startDate.AddDate(0, i, 0),
			BalanceAmount: current.Round(2),
		})
	}
	return points
}

func projectionMilestones(expected []NetworthProjectionPoint, fireTarget decimal.Decimal) []NetworthProjectionMilestone {
	milestones := make([]NetworthProjectionMilestone, 0)
	if date, ok := firstCrossingDate(expected, decimal.NewFromInt(oneCrore)); ok {
		milestones = append(milestones, NetworthProjectionMilestone{
			Label:  "You will hit 1Cr",
			Date:   date,
			Amount: decimal.NewFromInt(oneCrore),
		})
	}
	if fireTarget.GreaterThan(decimal.Zero) {
		if date, ok := firstCrossingDate(expected, fireTarget); ok {
			milestones = append(milestones, NetworthProjectionMilestone{
				Label:  "FIRE target reached",
				Date:   date,
				Amount: fireTarget.Round(2),
			})
		}
	}
	return milestones
}

func firstCrossingDate(points []NetworthProjectionPoint, threshold decimal.Decimal) (time.Time, bool) {
	for _, p := range points {
		if p.BalanceAmount.GreaterThanOrEqual(threshold) {
			return p.Date, true
		}
	}
	return time.Time{}, false
}

func monthDiff(from, to time.Time) int {
	months := (to.Year()-from.Year())*12 + int(to.Month()-from.Month())
	if months < 0 {
		return 0
	}
	return months
}

func monthsCovered(postings []posting.Posting) int {
	if len(postings) == 0 {
		return 0
	}
	start := utils.BeginningOfMonth(postings[0].Date)
	end := utils.BeginningOfMonth(utils.Now())
	months := (end.Year()-start.Year())*12 + int(end.Month()-start.Month()) + 1
	if months < 1 {
		return 1
	}
	if months > 12 {
		return 12
	}
	return months
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
