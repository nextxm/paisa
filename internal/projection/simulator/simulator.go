package simulator

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/shopspring/decimal"
)

// SimulationConfig holds all parameters for a Monte Carlo simulation run.
type SimulationConfig struct {
	Iterations       int             `json:"iterations"`
	MonthsToProject  int             `json:"months_to_project"`
	StartDate        time.Time       `json:"start_date"`
	CurrentNetworth  decimal.Decimal `json:"current_networth"`
	MonthlyContrib   decimal.Decimal `json:"monthly_contribution"`
	ContribGrowthPct decimal.Decimal `json:"contribution_growth_rate"` // annual %, e.g. 5.0
	ExpectedReturn   decimal.Decimal `json:"expected_return"`          // annual %, e.g. 12.0
	ReturnVolatility decimal.Decimal `json:"return_volatility"`        // annual %, e.g. 18.0
	InflationRate    decimal.Decimal `json:"inflation_rate"`           // annual %, e.g. 6.0
	SWR              decimal.Decimal `json:"swr"`                      // %, e.g. 4.0
	AnnualExpenses   decimal.Decimal `json:"annual_expenses"`          // current annual expenses
	Goals            []GoalCashflow  `json:"goals"`
}

// GoalCashflow represents a scheduled outflow for a goal at a specific month.
type GoalCashflow struct {
	Name          string          `json:"name"`
	Month         int             `json:"month"`
	Amount        decimal.Decimal `json:"amount"`
	InflationRate decimal.Decimal `json:"inflation_rate"` // per-goal override; zero = use global
}

// MonthlyPoint represents a single month's projected balance.
type MonthlyPoint struct {
	Date          time.Time       `json:"date"`
	BalanceAmount decimal.Decimal `json:"balance_amount"`
}

// GoalProbability holds the probability of meeting a specific goal.
type GoalProbability struct {
	Name        string  `json:"name"`
	Month       int     `json:"month"`
	Probability float64 `json:"probability"`
}

// SimulationResult holds the aggregated results of a Monte Carlo simulation.
type SimulationResult struct {
	Bands             map[string][]MonthlyPoint `json:"bands"`
	FIREProbability   float64                   `json:"fire_probability"`
	FIREYearP50       int                       `json:"fire_year_p50"`
	FIREYearP25       int                       `json:"fire_year_p25"`
	FIREYearP75       int                       `json:"fire_year_p75"`
	TargetCorpus      decimal.Decimal           `json:"target_corpus"`
	GoalProbabilities []GoalProbability         `json:"goal_probabilities"`
	Iterations        int                       `json:"iterations"`
	MonthsProjected   int                       `json:"months_projected"`
}

// DefaultConfig returns a SimulationConfig with sensible defaults.
func DefaultConfig() SimulationConfig {
	return SimulationConfig{
		Iterations:       1000,
		MonthsToProject:  360,
		StartDate:        time.Now(),
		SWR:              decimal.NewFromFloat(4.0),
		InflationRate:    decimal.NewFromFloat(6.0),
		ExpectedReturn:   decimal.NewFromFloat(12.0),
		ReturnVolatility: decimal.NewFromFloat(18.0),
	}
}

// Run executes a Monte Carlo simulation and returns aggregated results with
// percentile bands (P10/P25/P50/P75/P90) and FIRE probability.
func Run(cfg SimulationConfig) SimulationResult {
	if cfg.Iterations <= 0 {
		cfg.Iterations = 1000
	}
	if cfg.MonthsToProject <= 0 {
		cfg.MonthsToProject = 360
	}

	// Pre-compute monthly parameters from annual rates
	annualReturn, _ := cfg.ExpectedReturn.Div(decimal.NewFromInt(100)).Float64()
	annualVol, _ := cfg.ReturnVolatility.Div(decimal.NewFromInt(100)).Float64()

	// Log-normal monthly parameters:
	//   μ_monthly = ln(1 + r_annual) / 12
	//   σ_monthly = σ_annual / sqrt(12)
	monthlyMu := math.Log(1+annualReturn) / 12.0
	monthlySigma := annualVol / math.Sqrt(12.0)

	monthlyContrib, _ := cfg.MonthlyContrib.Float64()
	contribGrowthAnnual, _ := cfg.ContribGrowthPct.Div(decimal.NewFromInt(100)).Float64()
	monthlyContribGrowth := math.Pow(1+contribGrowthAnnual, 1.0/12.0) - 1.0

	globalInflationAnnual, _ := cfg.InflationRate.Div(decimal.NewFromInt(100)).Float64()
	currentNW, _ := cfg.CurrentNetworth.Float64()
	annualExpenses, _ := cfg.AnnualExpenses.Float64()
	swrPct, _ := cfg.SWR.Div(decimal.NewFromInt(100)).Float64()

	// Compute FIRE target corpus
	var targetCorpus float64
	if swrPct > 0 && annualExpenses > 0 {
		targetCorpus = annualExpenses / swrPct
	}

	// Pre-index goal outflows by month for fast lookup
	type goalEntry struct {
		name          string
		amount        float64
		inflationRate float64
	}
	goalsByMonth := make(map[int][]goalEntry)
	goalNames := make(map[string]bool)
	for _, g := range cfg.Goals {
		ir, _ := g.InflationRate.Float64()
		if ir == 0 {
			ir = globalInflationAnnual
		} else {
			ir = ir / 100.0
		}
		goalsByMonth[g.Month] = append(goalsByMonth[g.Month], goalEntry{
			name:          g.Name,
			amount:        g.Amount.InexactFloat64(),
			inflationRate: ir,
		})
		goalNames[g.Name] = true
	}

	// Allocate storage: allBalances[iteration][month]
	allBalances := make([][]float64, cfg.Iterations)
	fireMonths := make([]int, cfg.Iterations)
	goalSuccessCount := make(map[string]int)

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < cfg.Iterations; i++ {
		balances := make([]float64, cfg.MonthsToProject)
		balance := currentNW
		contrib := monthlyContrib
		fireMonth := -1
		goalFunded := make(map[string]bool)

		for m := 0; m < cfg.MonthsToProject; m++ {
			// Sample monthly return from log-normal distribution
			z := rng.NormFloat64()
			monthlyReturn := math.Exp(monthlyMu-0.5*monthlySigma*monthlySigma+monthlySigma*z) - 1.0

			// Apply return
			balance = balance * (1.0 + monthlyReturn)

			// Add contribution (grows monthly)
			balance += contrib
			contrib *= (1.0 + monthlyContribGrowth)

			// Subtract goal outflows
			if goals, ok := goalsByMonth[m]; ok {
				for _, g := range goals {
					yearsFromStart := float64(m) / 12.0
					inflatedAmount := g.amount * math.Pow(1+g.inflationRate, yearsFromStart)
					if balance >= inflatedAmount {
						goalFunded[g.name] = true
					}
					balance -= inflatedAmount
				}
			}

			balances[m] = balance

			// Check FIRE: balance >= target corpus (inflated)
			if fireMonth == -1 && targetCorpus > 0 {
				yearsFromStart := float64(m) / 12.0
				inflatedTarget := targetCorpus * math.Pow(1+globalInflationAnnual, yearsFromStart)
				if balance >= inflatedTarget {
					fireMonth = m
				}
			}
		}

		allBalances[i] = balances
		fireMonths[i] = fireMonth
		for name := range goalFunded {
			goalSuccessCount[name]++
		}
	}

	// --- Aggregation ---
	result := SimulationResult{
		Bands:           make(map[string][]MonthlyPoint),
		Iterations:      cfg.Iterations,
		MonthsProjected: cfg.MonthsToProject,
		TargetCorpus:    decimal.NewFromFloat(targetCorpus).Round(0),
	}

	percentiles := []struct {
		label string
		pct   float64
	}{
		{"p10", 0.10},
		{"p25", 0.25},
		{"p50", 0.50},
		{"p75", 0.75},
		{"p90", 0.90},
	}

	// For each month, collect all iteration values, sort, and pick percentiles
	monthBalances := make([]float64, cfg.Iterations)
	for _, p := range percentiles {
		points := make([]MonthlyPoint, cfg.MonthsToProject)
		for m := 0; m < cfg.MonthsToProject; m++ {
			for i := 0; i < cfg.Iterations; i++ {
				monthBalances[i] = allBalances[i][m]
			}
			sort.Float64s(monthBalances)
			idx := int(math.Round(p.pct * float64(cfg.Iterations-1)))
			if idx >= cfg.Iterations {
				idx = cfg.Iterations - 1
			}
			points[m] = MonthlyPoint{
				Date:          cfg.StartDate.AddDate(0, m+1, 0),
				BalanceAmount: decimal.NewFromFloat(monthBalances[idx]).Round(0),
			}
		}
		result.Bands[p.label] = points
	}

	// FIRE probability and year percentiles
	fireCount := 0
	var fireMonthValues []int
	for _, fm := range fireMonths {
		if fm >= 0 {
			fireCount++
			fireMonthValues = append(fireMonthValues, fm)
		}
	}
	result.FIREProbability = math.Round(float64(fireCount)/float64(cfg.Iterations)*1000) / 1000

	if len(fireMonthValues) > 0 {
		sort.Ints(fireMonthValues)
		n := len(fireMonthValues)
		result.FIREYearP25 = (fireMonthValues[n*25/100] + 6) / 12
		result.FIREYearP50 = (fireMonthValues[n*50/100] + 6) / 12
		result.FIREYearP75 = (fireMonthValues[n*75/100] + 6) / 12
	}

	// Goal probabilities
	for name := range goalNames {
		result.GoalProbabilities = append(result.GoalProbabilities, GoalProbability{
			Name:        name,
			Probability: math.Round(float64(goalSuccessCount[name])/float64(cfg.Iterations)*1000) / 1000,
		})
	}
	// Sort for deterministic output
	sort.Slice(result.GoalProbabilities, func(i, j int) bool {
		return result.GoalProbabilities[i].Name < result.GoalProbabilities[j].Name
	})

	return result
}
