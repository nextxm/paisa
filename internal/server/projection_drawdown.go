package server

import (
	projectiondrawdown "github.com/ananthakumaran/paisa/internal/projection/drawdown"
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type DrawdownRequest struct {
	Amount                  float64                             `json:"amount"`
	Buckets                 []projectiondrawdown.DrawdownBucket `json:"buckets"`
	IncludeProjectionImpact bool                                `json:"include_projection_impact"`
	Baseline                SimulateRequest                     `json:"baseline"`
}

func GetProjectionDrawdown(db *gorm.DB, req DrawdownRequest) map[string]any {
	analysis := projectiondrawdown.Analyze(db, decimal.NewFromFloat(req.Amount), req.Buckets)
	availableAssets := projectiondrawdown.AvailableAssetMetadata(db)
	response := map[string]any{
		"drawdown":           analysis,
		"available_accounts": projectiondrawdown.AvailableAssetAccounts(db),
		"available_assets":   availableAssets,
	}

	if !req.IncludeProjectionImpact {
		return response
	}

	profile, baselineCfg, lifeGoalMetadata := buildProjectionSimulationConfig(db, req.Baseline)
	baseline := simulator.Run(baselineCfg)

	postDrawdownCfg := baselineCfg
	drawdownOutflow := analysis.Recommended.
		Add(analysis.TotalEstimatedTax.ShortTerm).
		Add(analysis.TotalEstimatedTax.LongTerm).
		Add(analysis.TotalEstimatedTax.Slab)
	postDrawdownCfg.CurrentNetworth = baselineCfg.CurrentNetworth.Sub(drawdownOutflow)
	if postDrawdownCfg.CurrentNetworth.LessThan(decimal.Zero) {
		postDrawdownCfg.CurrentNetworth = decimal.Zero
	}

	postDrawdown := simulator.Run(postDrawdownCfg)

	fireProbabilityDelta := decimal.NewFromFloat(postDrawdown.FIREProbability).
		Sub(decimal.NewFromFloat(baseline.FIREProbability)).
		Round(4)

	fireYearP50Delta := 0
	if baseline.FIREYearP50 > 0 && postDrawdown.FIREYearP50 > 0 {
		fireYearP50Delta = postDrawdown.FIREYearP50 - baseline.FIREYearP50
	}

	response["impact"] = map[string]any{
		"profile": profile,
		"drawdown_outflow": map[string]any{
			"withdrawal":    analysis.Recommended,
			"estimated_tax": analysis.TotalEstimatedTax,
			"total":         drawdownOutflow.Round(2),
		},
		"baseline": map[string]any{
			"simulation": baseline,
			"goals":      buildLifeGoalResponse(lifeGoalMetadata, baseline.GoalProbabilities),
		},
		"post_drawdown": map[string]any{
			"simulation": postDrawdown,
			"goals":      buildLifeGoalResponse(lifeGoalMetadata, postDrawdown.GoalProbabilities),
		},
		"delta": map[string]any{
			"fire_probability": fireProbabilityDelta,
			"fire_year_p50":    fireYearP50Delta,
		},
	}

	return response
}
