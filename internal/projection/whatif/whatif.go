package whatif

import (
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/shopspring/decimal"
)

type Scenario struct {
	Name      string                     `json:"name"`
	Overrides map[string]decimal.Decimal `json:"overrides"`
}

type ScenarioResult struct {
	Name   string                     `json:"name"`
	Result simulator.SimulationResult `json:"result"`
}

type ComparisonResult struct {
	Baseline  simulator.SimulationResult `json:"baseline"`
	Scenarios []ScenarioResult           `json:"scenarios"`
}

func Compare(baseConfig simulator.SimulationConfig, scenarios []Scenario) ComparisonResult {
	result := ComparisonResult{
		Baseline:  simulator.Run(baseConfig),
		Scenarios: make([]ScenarioResult, 0, len(scenarios)),
	}

	for _, scenario := range scenarios {
		cfg := ApplyOverrides(baseConfig, scenario.Overrides)
		result.Scenarios = append(result.Scenarios, ScenarioResult{
			Name:   scenario.Name,
			Result: simulator.Run(cfg),
		})
	}

	return result
}

func ApplyOverrides(base simulator.SimulationConfig, overrides map[string]decimal.Decimal) simulator.SimulationConfig {
	cfg := base
	for key, value := range overrides {
		switch key {
		case "iterations":
			cfg.Iterations = int(value.IntPart())
		case "months_to_project":
			cfg.MonthsToProject = int(value.IntPart())
		case "monthly_contribution":
			cfg.MonthlyContrib = value
		case "contribution_growth_rate":
			cfg.ContribGrowthPct = value
		case "expected_return":
			cfg.ExpectedReturn = value
		case "return_volatility":
			cfg.ReturnVolatility = value
		case "inflation_rate":
			cfg.InflationRate = value
		case "swr":
			cfg.SWR = value
		}
	}
	return cfg
}
