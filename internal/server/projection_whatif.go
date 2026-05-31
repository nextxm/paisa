package server

import (
	projectionwhatif "github.com/ananthakumaran/paisa/internal/projection/whatif"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type WhatIfScenarioRequest struct {
	Name      string          `json:"name"`
	Overrides SimulateRequest `json:"overrides"`
}

type WhatIfRequest struct {
	Baseline  SimulateRequest         `json:"baseline"`
	Scenarios []WhatIfScenarioRequest `json:"scenarios"`
}

type projectionWhatIfScenario struct {
	Name       string               `json:"name"`
	Simulation any                  `json:"simulation"`
	Goals      []projectionLifeGoal `json:"goals"`
}

func GetProjectionWhatIf(db *gorm.DB, req WhatIfRequest) map[string]any {
	profile, baseCfg, lifeGoalMetadata := buildProjectionSimulationConfig(db, req.Baseline)
	scenarios := make([]projectionwhatif.Scenario, 0, len(req.Scenarios))

	for _, scenario := range req.Scenarios {
		scenarios = append(scenarios, projectionwhatif.Scenario{
			Name:      scenario.Name,
			Overrides: scenarioOverridesToMap(scenario.Overrides),
		})
	}

	comparison := projectionwhatif.Compare(baseCfg, scenarios)
	responseScenarios := make([]projectionWhatIfScenario, 0, len(comparison.Scenarios))
	for _, scenario := range comparison.Scenarios {
		responseScenarios = append(responseScenarios, projectionWhatIfScenario{
			Name:       scenario.Name,
			Simulation: scenario.Result,
			Goals:      buildLifeGoalResponse(lifeGoalMetadata, scenario.Result.GoalProbabilities),
		})
	}

	return map[string]any{
		"profile": profile,
		"baseline": map[string]any{
			"simulation": comparison.Baseline,
			"goals":      buildLifeGoalResponse(lifeGoalMetadata, comparison.Baseline.GoalProbabilities),
		},
		"scenarios": responseScenarios,
	}
}

func scenarioOverridesToMap(req SimulateRequest) map[string]decimal.Decimal {
	overrides := map[string]decimal.Decimal{}
	if req.Iterations > 0 {
		overrides["iterations"] = decimal.NewFromInt(int64(req.Iterations))
	}
	if req.MonthsToProject > 0 {
		overrides["months_to_project"] = decimal.NewFromInt(int64(req.MonthsToProject))
	}
	if req.MonthlyContrib != 0 {
		overrides["monthly_contribution"] = decimal.NewFromFloat(req.MonthlyContrib)
	}
	if req.ContribGrowthPct != 0 {
		overrides["contribution_growth_rate"] = decimal.NewFromFloat(req.ContribGrowthPct)
	}
	if req.ExpectedReturn != 0 {
		overrides["expected_return"] = decimal.NewFromFloat(req.ExpectedReturn)
	}
	if req.ReturnVolatility != 0 {
		overrides["return_volatility"] = decimal.NewFromFloat(req.ReturnVolatility)
	}
	if req.InflationRate != 0 {
		overrides["inflation_rate"] = decimal.NewFromFloat(req.InflationRate)
	}
	if req.SWR != 0 {
		overrides["swr"] = decimal.NewFromFloat(req.SWR)
	}
	return overrides
}
