package server

import (
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProjectNetworth_ZeroCAGRUsesLinearContribution(t *testing.T) {
	start := mustParseDate("2025-01-01")
	points := projectNetworth(
		start,
		decimal.NewFromInt(1000),
		decimal.NewFromInt(100),
		decimal.Zero,
		12,
	)

	require.Len(t, points, 12)
	assert.True(t, points[11].BalanceAmount.Equal(decimal.NewFromInt(2200)))
	assert.Equal(t, "2026-01-01", points[11].Date.Format("2006-01-02"))
}

func TestProjectNetworth_ScenarioOrdering(t *testing.T) {
	start := mustParseDate("2025-01-01")
	monthly := decimal.NewFromInt(1000)
	current := decimal.NewFromInt(100000)

	conservative := projectNetworth(start, current, monthly, decimal.NewFromInt(6), 120)
	expected := projectNetworth(start, current, monthly, decimal.NewFromInt(10), 120)
	optimistic := projectNetworth(start, current, monthly, decimal.NewFromInt(14), 120)

	require.NotEmpty(t, conservative)
	require.NotEmpty(t, expected)
	require.NotEmpty(t, optimistic)

	cLast := conservative[len(conservative)-1].BalanceAmount
	eLast := expected[len(expected)-1].BalanceAmount
	oLast := optimistic[len(optimistic)-1].BalanceAmount

	assert.True(t, cLast.LessThan(eLast))
	assert.True(t, eLast.LessThan(oLast))
}

func TestProjectionMilestones_IncludesOneCroreAndFireTarget(t *testing.T) {
	points := []NetworthProjectionPoint{
		{Date: mustParseDate("2026-01-01"), BalanceAmount: decimal.NewFromInt(9000000)},
		{Date: mustParseDate("2026-06-01"), BalanceAmount: decimal.NewFromInt(10000000)},
		{Date: mustParseDate("2027-01-01"), BalanceAmount: decimal.NewFromInt(12000000)},
	}

	milestones := projectionMilestones(points, decimal.NewFromInt(11000000))
	require.Len(t, milestones, 2)
	assert.Equal(t, "You will hit 1Cr", milestones[0].Label)
	assert.Equal(t, "FIRE target reached", milestones[1].Label)
	assert.Equal(t, "2026-06-01", milestones[0].Date.Format("2006-01-02"))
	assert.Equal(t, "2027-01-01", milestones[1].Date.Format("2006-01-02"))
}
