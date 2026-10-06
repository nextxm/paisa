package server

import (
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/projection/dna"
	projectiongoals "github.com/ananthakumaran/paisa/internal/projection/goals"
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/ananthakumaran/paisa/internal/utils"
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
	profile, cfg, lifeGoalMetadata := buildProjectionSimulationConfig(db, req)
	result := simulator.Run(cfg)

	return gin.H{
		"profile":    profile,
		"simulation": result,
		"goals":      buildLifeGoalResponse(lifeGoalMetadata, result.GoalProbabilities),
	}
}

func buildProjectionSimulationConfig(db *gorm.DB, req SimulateRequest) (dna.FinancialProfile, simulator.SimulationConfig, []projectiongoals.LifeGoalMetadata) {
	profile := dna.ExtractProfile(db)

	cfg := simulator.DefaultConfig()
	cfg.StartDate = utils.ToDate(utils.Now())
	cfg.CurrentNetworth = profile.CurrentNetworth
	cfg.MonthlyContrib = profile.MonthlyContribution
	cfg.AnnualExpenses = profile.AnnualExpenses

	if !profile.HistoricalReturn.IsZero() {
		cfg.ExpectedReturn = profile.HistoricalReturn
	}
	if !profile.ReturnVolatility.IsZero() {
		cfg.ReturnVolatility = profile.ReturnVolatility
	}
	if !profile.IncomeGrowthRate.IsZero() {
		cfg.ContribGrowthPct = profile.IncomeGrowthRate
	}

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

	lifeGoalCashflows, lifeGoalMetadata, err := projectiongoals.ExpandLifeGoals(config.GetConfig().Goals.Life, cfg.StartDate)
	if err == nil {
		cfg.Goals = lifeGoalCashflows
	}

	return profile, cfg, lifeGoalMetadata
}

type projectionLifeGoal struct {
	GoalID            string   `json:"goal_id"`
	Name              string   `json:"name"`
	Icon              string   `json:"icon"`
	Type              string   `json:"type"`
	TargetAmount      float64  `json:"target_amount"`
	TargetDate        string   `json:"target_date"`
	StartDate         string   `json:"start_date"`
	EndDate           string   `json:"end_date"`
	Frequency         string   `json:"frequency"`
	InflationRate     *float64 `json:"inflation_rate,omitempty"`
	Priority          int      `json:"priority"`
	FundedBy          []string `json:"funded_by"`
	MonthlyAllocation float64  `json:"monthly_allocation"`
	CashflowCount     int      `json:"cashflow_count"`
	Probability       float64  `json:"probability"`
}

func buildLifeGoalResponse(metadata []projectiongoals.LifeGoalMetadata, probabilities []simulator.GoalProbability) []projectionLifeGoal {
	probabilityByGoal := make(map[string]float64, len(probabilities))
	for _, probability := range probabilities {
		probabilityByGoal[probability.GoalID] = probability.Probability
	}

	goals := make([]projectionLifeGoal, 0, len(metadata))
	for _, meta := range metadata {
		goals = append(goals, projectionLifeGoal{
			GoalID:            meta.GoalID,
			Name:              meta.Name,
			Icon:              meta.Icon,
			Type:              meta.Type,
			TargetAmount:      meta.TargetAmount,
			TargetDate:        meta.TargetDate,
			StartDate:         meta.StartDate,
			EndDate:           meta.EndDate,
			Frequency:         meta.Frequency,
			InflationRate:     meta.InflationRate,
			Priority:          meta.Priority,
			FundedBy:          meta.FundedBy,
			MonthlyAllocation: meta.MonthlyAllocation,
			CashflowCount:     meta.CashflowCount,
			Probability:       probabilityByGoal[meta.GoalID],
		})
	}

	return goals
}
