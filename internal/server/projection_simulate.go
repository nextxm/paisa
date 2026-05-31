package server

import (
	"github.com/ananthakumaran/paisa/internal/projection/dna"
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// SimulateRequest allows the frontend to override simulation parameters.
// Zero values use defaults derived from the user's Financial DNA.
type SimulateRequest struct {
	Iterations       int     `json:"iterations"`
	MonthsToProject  int     `json:"months_to_project"`
	MonthlyContrib   float64 `json:"monthly_contribution"`
	ContribGrowthPct float64 `json:"contribution_growth_rate"`
	ExpectedReturn   float64 `json:"expected_return"`
	ReturnVolatility float64 `json:"return_volatility"`
	InflationRate    float64 `json:"inflation_rate"`
	SWR              float64 `json:"swr"`
}

// GetProjectionSimulate runs a Monte Carlo simulation using the user's
// Financial DNA as baseline parameters, with optional overrides from the
// request.
func GetProjectionSimulate(db *gorm.DB, req SimulateRequest) gin.H {
	profile := dna.ExtractProfile(db)

	cfg := simulator.DefaultConfig()
	cfg.CurrentNetworth = profile.CurrentNetworth
	cfg.MonthlyContrib = profile.MonthlyContribution
	cfg.AnnualExpenses = profile.AnnualExpenses

	// Use DNA-derived values as defaults
	if !profile.HistoricalReturn.IsZero() {
		cfg.ExpectedReturn = profile.HistoricalReturn
	}
	if !profile.ReturnVolatility.IsZero() {
		cfg.ReturnVolatility = profile.ReturnVolatility
	}
	if !profile.IncomeGrowthRate.IsZero() {
		cfg.ContribGrowthPct = profile.IncomeGrowthRate
	}

	// Apply request overrides (non-zero values override DNA defaults)
	if req.Iterations > 0 {
		cfg.Iterations = req.Iterations
	}
	if req.MonthsToProject > 0 {
		cfg.MonthsToProject = req.MonthsToProject
	}
	if req.MonthlyContrib != 0 {
		cfg.MonthlyContrib = decimal.NewFromFloat(req.MonthlyContrib)
	}
	if req.ContribGrowthPct != 0 {
		cfg.ContribGrowthPct = decimal.NewFromFloat(req.ContribGrowthPct)
	}
	if req.ExpectedReturn != 0 {
		cfg.ExpectedReturn = decimal.NewFromFloat(req.ExpectedReturn)
	}
	if req.ReturnVolatility != 0 {
		cfg.ReturnVolatility = decimal.NewFromFloat(req.ReturnVolatility)
	}
	if req.InflationRate != 0 {
		cfg.InflationRate = decimal.NewFromFloat(req.InflationRate)
	}
	if req.SWR != 0 {
		cfg.SWR = decimal.NewFromFloat(req.SWR)
	}

	result := simulator.Run(cfg)

	return gin.H{
		"profile":    profile,
		"simulation": result,
	}
}
