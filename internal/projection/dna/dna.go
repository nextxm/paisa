package dna

import (
	"math"
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// FinancialProfile contains inferred financial parameters derived from the
// user's ledger history. These parameters feed into the projection/simulation
// engine. All rates are expressed as percentages (e.g., 8.5 means 8.5%).
type FinancialProfile struct {
	// Current state
	CurrentNetworth     decimal.Decimal `json:"current_networth"`
	MonthlyContribution decimal.Decimal `json:"monthly_contribution"`
	SavingsRate         decimal.Decimal `json:"savings_rate"`
	AnnualExpenses      decimal.Decimal `json:"annual_expenses"`
	AnnualIncome        decimal.Decimal `json:"annual_income"`

	// Growth rates (annualized, percentage)
	IncomeGrowthRate  decimal.Decimal `json:"income_growth_rate"`
	ExpenseGrowthRate decimal.Decimal `json:"expense_growth_rate"`

	// Investment characteristics (annualized, percentage)
	HistoricalReturn decimal.Decimal `json:"historical_return"`
	ReturnVolatility decimal.Decimal `json:"return_volatility"`

	// Data quality indicators
	IncomeYearsCovered  int `json:"income_years_covered"`
	ExpenseYearsCovered int `json:"expense_years_covered"`
	PriceMonthsCovered  int `json:"price_months_covered"`
}

// ExtractProfile analyzes the user's ledger data and derives a FinancialProfile.
func ExtractProfile(db *gorm.DB) FinancialProfile {
	profile := FinancialProfile{}

	now := utils.Now()
	profile.CurrentNetworth = computeCurrentNetworth(db)
	profile.AnnualIncome, profile.IncomeGrowthRate, profile.IncomeYearsCovered = computeIncomeMetrics(db, now)
	profile.AnnualExpenses, profile.ExpenseGrowthRate, profile.ExpenseYearsCovered = computeExpenseMetrics(db, now)
	profile.MonthlyContribution, profile.SavingsRate = computeSavingsMetrics(db, now)
	profile.HistoricalReturn, profile.ReturnVolatility, profile.PriceMonthsCovered = computeReturnMetrics(db, now)

	return profile
}

// computeCurrentNetworth calculates the current net worth from all assets,
// liabilities, and unrealized capital gains, with market prices applied.
func computeCurrentNetworth(db *gorm.DB) decimal.Decimal {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()
	postings = service.PopulateMarketPrice(db, postings)

	return utils.SumBy(postings, func(p posting.Posting) decimal.Decimal {
		return p.MarketAmount
	})
}

// computeIncomeMetrics calculates annual income and year-over-year growth.
// It uses up to the last 3 complete financial years of income data.
func computeIncomeMetrics(db *gorm.DB, now time.Time) (annualIncome, growthRate decimal.Decimal, yearsCovered int) {
	// Get the last 3 years of income postings
	threeYearsAgo := utils.BeginningOfMonth(now).AddDate(-3, 0, 0)
	incomePostings := query.Init(db).Like("Income:%").
		Where("date between ? AND ?", threeYearsAgo, now).All()

	if len(incomePostings) == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	// Group by financial year
	yearlyIncome := groupByYear(incomePostings, now)
	yearsCovered = len(yearlyIncome)

	if yearsCovered == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	// Most recent complete year's income (negated since Income postings are negative)
	years := sortedYearKeys(yearlyIncome)
	latestYear := years[len(years)-1]
	annualIncome = yearlyIncome[latestYear].Neg()

	// Compute CAGR if we have at least 2 years
	if yearsCovered >= 2 {
		oldestYear := years[0]
		oldestIncome := yearlyIncome[oldestYear].Neg()
		if oldestIncome.GreaterThan(decimal.Zero) {
			growthRate = computeCAGR(oldestIncome, annualIncome, yearsCovered-1)
		}
	}

	return annualIncome, growthRate, yearsCovered
}

// computeExpenseMetrics calculates annual expenses and year-over-year growth.
// Excludes tax expenses for a cleaner lifestyle cost picture.
func computeExpenseMetrics(db *gorm.DB, now time.Time) (annualExpenses, growthRate decimal.Decimal, yearsCovered int) {
	threeYearsAgo := utils.BeginningOfMonth(now).AddDate(-3, 0, 0)
	expensePostings := query.Init(db).Like("Expenses:%").
		NotAccountPrefix("Expenses:Tax").
		Where("date between ? AND ?", threeYearsAgo, now).All()

	if len(expensePostings) == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	yearlyExpenses := groupByYear(expensePostings, now)
	yearsCovered = len(yearlyExpenses)

	if yearsCovered == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	years := sortedYearKeys(yearlyExpenses)
	latestYear := years[len(years)-1]
	annualExpenses = yearlyExpenses[latestYear]

	if yearsCovered >= 2 {
		oldestYear := years[0]
		oldestExpenses := yearlyExpenses[oldestYear]
		if oldestExpenses.GreaterThan(decimal.Zero) {
			growthRate = computeCAGR(oldestExpenses, annualExpenses, yearsCovered-1)
		}
	}

	return annualExpenses, growthRate, yearsCovered
}

// computeSavingsMetrics calculates the monthly contribution and savings rate
// from the last 12 months of data, reusing the pattern from the existing
// projection engine.
func computeSavingsMetrics(db *gorm.DB, now time.Time) (monthlyContribution, savingsRate decimal.Decimal) {
	incomePostings := query.Init(db).Like("Income:%").LastNMonths(12).All()
	assetPostings := query.Init(db).
		Like("Assets:%").
		NotAccountPrefix("Assets:Checking").
		Where("transaction_id not in (select transaction_id from postings p where p.account like ? and p.transaction_id = transaction_id)", "Liabilities:%").
		LastNMonths(12).
		All()

	assetPostings = filterStockSplits(db, assetPostings)
	netIncome := accounting.CostSum(incomePostings).Neg()
	netInvestment := accounting.CostSum(assetPostings)

	monthsCovered := maxInt(monthsCoveredByPostings(incomePostings, now), monthsCoveredByPostings(assetPostings, now))
	if monthsCovered < 1 {
		monthsCovered = 1
	}

	if netIncome.GreaterThan(decimal.Zero) {
		savingsRate = netInvestment.Div(netIncome).Mul(decimal.NewFromInt(100))
	}

	monthlyContribution = netInvestment.Div(decimal.NewFromInt(int64(monthsCovered)))
	if savingsRate.GreaterThan(decimal.Zero) && netIncome.GreaterThan(decimal.Zero) {
		avgMonthlyIncome := netIncome.Div(decimal.NewFromInt(int64(monthsCovered)))
		monthlyContribution = avgMonthlyIncome.Mul(savingsRate).Div(decimal.NewFromInt(100))
	}

	return monthlyContribution, savingsRate
}

// computeReturnMetrics calculates portfolio-weighted historical return and
// volatility from commodity price history.
//
// All non-checking asset postings up to now are loaded with a single query and
// then sliced in memory for each month of the 24-month window. Market values
// are computed once per month boundary (the end of month N is the start of
// month N+1) using the in-memory price cache, so no per-month SQLite queries
// are issued.
func computeReturnMetrics(db *gorm.DB, now time.Time) (annualReturn, volatility decimal.Decimal, monthsCovered int) {
	// Without any commodity prices there is nothing to derive returns from.
	if !hasAnyPrice(db) {
		return decimal.Zero, decimal.Zero, 0
	}

	// Compute monthly returns across all commodities (portfolio-level)
	// Use a simple approach: compute monthly returns for the overall portfolio
	// by looking at net worth changes
	twoYearsAgo := utils.BeginningOfMonth(now).AddDate(-2, 0, 0)
	boundaries := monthBoundaries(twoYearsAgo, now)
	if len(boundaries) < 2 {
		return decimal.Zero, decimal.Zero, 0
	}

	// Single query: every non-checking asset posting up to the last boundary,
	// ordered by date ascending. UntilDate is used (rather than
	// "date between time.Time{} AND x") because a zero lower bound is mapped
	// to NULL by the sqlc query path and would match no rows.
	assetPostings := query.Init(db).Like("Assets:%").
		NotAccountPrefix("Assets:Checking").
		UntilDate(boundaries[len(boundaries)-1]).All()

	valueAt := func(ps []posting.Posting, date time.Time) decimal.Decimal {
		return accounting.CurrentBalance(service.PopulateMarketPriceAt(db, ps, date))
	}
	monthlyReturns := monthlyReturnsFromPostings(assetPostings, boundaries, valueAt)

	monthsCovered = len(monthlyReturns)
	if monthsCovered < 3 {
		return decimal.Zero, decimal.Zero, monthsCovered
	}

	// Mean monthly return
	sum := 0.0
	for _, r := range monthlyReturns {
		sum += r
	}
	meanMonthly := sum / float64(monthsCovered)

	// Annualize: (1 + mean_monthly)^12 - 1
	annualReturnFloat := math.Pow(1+meanMonthly, 12) - 1
	annualReturn = decimal.NewFromFloat(annualReturnFloat * 100).Round(2)

	// Volatility: annualized std deviation
	sumSqDiff := 0.0
	for _, r := range monthlyReturns {
		diff := r - meanMonthly
		sumSqDiff += diff * diff
	}
	stdMonthly := math.Sqrt(sumSqDiff / float64(monthsCovered-1))
	volatilityFloat := stdMonthly * math.Sqrt(12)
	volatility = decimal.NewFromFloat(volatilityFloat * 100).Round(2)

	return annualReturn, volatility, monthsCovered
}

// --- Helper functions ---

// hasAnyPrice reports whether at least one commodity price row exists. Query
// errors are treated as "no prices" so callers fall back to zero metrics.
func hasAnyPrice(db *gorm.DB) bool {
	var ids []uint
	if err := db.Model(&price.Price{}).Limit(1).Pluck("id", &ids).Error; err != nil {
		return false
	}
	return len(ids) > 0
}

// monthBoundaries returns the month-start dates from start up to and including
// the last month end that does not exceed now. Consecutive pairs
// (boundaries[i], boundaries[i+1]) describe the complete months in the window.
func monthBoundaries(start, now time.Time) []time.Time {
	var boundaries []time.Time
	for monthStart := start; monthStart.Before(now); monthStart = monthStart.AddDate(0, 1, 0) {
		monthEnd := monthStart.AddDate(0, 1, 0)
		if monthEnd.After(now) {
			break
		}
		if len(boundaries) == 0 {
			boundaries = append(boundaries, monthStart)
		}
		boundaries = append(boundaries, monthEnd)
	}
	return boundaries
}

// monthlyReturnsFromPostings computes time-weighted monthly returns for each
// consecutive pair of boundaries using a pre-fetched slice of postings.
//
// For month i (boundaries[i] .. boundaries[i+1]):
//   - start value: market value of postings with date <= boundaries[i]
//   - end value:   market value of postings with date <= boundaries[i+1]
//   - contribution: cost of postings with boundaries[i] <= date <= boundaries[i+1]
//     (inclusive on both ends, matching SQL BETWEEN semantics)
//
// valueAt is evaluated exactly once per boundary because the end of one month
// is the start of the next.
func monthlyReturnsFromPostings(postings []posting.Posting, boundaries []time.Time, valueAt func([]posting.Posting, time.Time) decimal.Decimal) []float64 {
	if len(boundaries) < 2 {
		return nil
	}

	sorted := make([]posting.Posting, len(postings))
	copy(sorted, postings)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].Date.Before(sorted[j].Date) })

	// upTo returns the number of postings with date <= d.
	upTo := func(d time.Time) int {
		return sort.Search(len(sorted), func(i int) bool { return sorted[i].Date.After(d) })
	}
	// from returns the index of the first posting with date >= d.
	from := func(d time.Time) int {
		return sort.Search(len(sorted), func(i int) bool { return !sorted[i].Date.Before(d) })
	}

	values := make([]decimal.Decimal, len(boundaries))
	ends := make([]int, len(boundaries))
	for i, b := range boundaries {
		ends[i] = upTo(b)
		values[i] = valueAt(sorted[:ends[i]], b)
	}

	var monthlyReturns []float64
	for i := 0; i+1 < len(boundaries); i++ {
		startValue := values[i]
		endValue := values[i+1]
		contribAmount := accounting.CostSum(sorted[from(boundaries[i]):ends[i+1]])

		if startValue.GreaterThan(decimal.Zero) {
			// Time-weighted return: (end - start - contributions) / start
			returnPct := endValue.Sub(startValue).Sub(contribAmount).Div(startValue)
			r, _ := returnPct.Float64()
			monthlyReturns = append(monthlyReturns, r)
		}
	}
	return monthlyReturns
}

// groupByYear groups postings by calendar year and sums their amounts.
func groupByYear(postings []posting.Posting, now time.Time) map[int]decimal.Decimal {
	result := make(map[int]decimal.Decimal)
	currentYear := now.Year()
	currentMonth := now.Month()

	for _, p := range postings {
		year := p.Date.Year()
		// Skip the current year if we're less than 6 months in (incomplete data)
		if year == currentYear && currentMonth < 7 {
			continue
		}
		result[year] = result[year].Add(p.Amount)
	}
	return result
}

// sortedYearKeys returns the keys of a year→amount map in ascending order.
func sortedYearKeys(m map[int]decimal.Decimal) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	// Simple sort for small slices
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[i] > keys[j] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	return keys
}

// computeCAGR calculates compound annual growth rate.
// CAGR = ((end/start)^(1/years) - 1) * 100
func computeCAGR(start, end decimal.Decimal, years int) decimal.Decimal {
	if start.IsZero() || years <= 0 {
		return decimal.Zero
	}
	ratio, _ := end.Div(start).Float64()
	if ratio <= 0 {
		return decimal.Zero
	}
	cagr := math.Pow(ratio, 1.0/float64(years)) - 1
	return decimal.NewFromFloat(cagr * 100).Round(2)
}

// filterStockSplits removes stock split postings that would skew contribution
// calculations.
func filterStockSplits(db *gorm.DB, postings []posting.Posting) []posting.Posting {
	filtered := make([]posting.Posting, 0, len(postings))
	for _, p := range postings {
		if !service.IsStockSplit(db, p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

// monthsCoveredByPostings returns the number of months between the first
// posting and now, capped at 12.
func monthsCoveredByPostings(postings []posting.Posting, now time.Time) int {
	if len(postings) == 0 {
		return 0
	}
	start := utils.BeginningOfMonth(postings[0].Date)
	end := utils.BeginningOfMonth(now)
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
