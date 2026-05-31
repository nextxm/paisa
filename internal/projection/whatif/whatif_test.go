package whatif

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/shopspring/decimal"
)

func TestApplyOverridesUpdatesSelectedFieldsOnly(t *testing.T) {
	base := simulator.DefaultConfig()
	base.StartDate = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	base.MonthlyContrib = decimal.NewFromInt(10000)
	base.ExpectedReturn = decimal.NewFromFloat(10)
	base.InflationRate = decimal.NewFromFloat(6)

	overridden := ApplyOverrides(base, map[string]decimal.Decimal{
		"monthly_contribution": decimal.NewFromInt(25000),
		"expected_return":      decimal.NewFromFloat(14),
	})

	if !overridden.MonthlyContrib.Equal(decimal.NewFromInt(25000)) {
		t.Fatalf("MonthlyContrib = %s, want 25000", overridden.MonthlyContrib)
	}
	if !overridden.ExpectedReturn.Equal(decimal.NewFromFloat(14)) {
		t.Fatalf("ExpectedReturn = %s, want 14", overridden.ExpectedReturn)
	}
	if !overridden.InflationRate.Equal(base.InflationRate) {
		t.Fatalf("InflationRate changed unexpectedly: %s", overridden.InflationRate)
	}
}

func TestCompareReturnsBaselineAndScenarioResults(t *testing.T) {
	base := simulator.DefaultConfig()
	base.Iterations = 25
	base.MonthsToProject = 24
	base.StartDate = time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	base.CurrentNetworth = decimal.NewFromInt(1000000)
	base.MonthlyContrib = decimal.NewFromInt(10000)
	base.ExpectedReturn = decimal.NewFromFloat(10)
	base.ReturnVolatility = decimal.NewFromFloat(0.1)
	base.InflationRate = decimal.NewFromFloat(6)

	result := Compare(base, []Scenario{
		{Name: "Increase SIP", Overrides: map[string]decimal.Decimal{"monthly_contribution": decimal.NewFromInt(20000)}},
		{Name: "Lower Return", Overrides: map[string]decimal.Decimal{"expected_return": decimal.NewFromFloat(6)}},
	})

	if result.Baseline.MonthsProjected != 24 {
		t.Fatalf("baseline months = %d, want 24", result.Baseline.MonthsProjected)
	}
	if len(result.Scenarios) != 2 {
		t.Fatalf("len(result.Scenarios) = %d, want 2", len(result.Scenarios))
	}
	if result.Scenarios[0].Name != "Increase SIP" {
		t.Fatalf("first scenario name = %q, want Increase SIP", result.Scenarios[0].Name)
	}
	if len(result.Scenarios[0].Result.Bands["p50"]) != 24 {
		t.Fatalf("first scenario p50 length = %d, want 24", len(result.Scenarios[0].Result.Bands["p50"]))
	}
}
