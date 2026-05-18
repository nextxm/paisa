package server

import (
	"encoding/json"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
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

func TestGetNetworthProjection_LiveFallbackMatchesSnapshot(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	t.Cleanup(utils.UnsetNow)

	db := openTestDB(t)
	seedProjectionTestPostings(t, db)

	req := NetworthProjectionRequest{
		Years:            15,
		ConservativeCAGR: decimal.NewFromInt(8),
		ExpectedCAGR:     decimal.NewFromInt(12),
		OptimisticCAGR:   decimal.NewFromInt(16),
		SWR:              decimal.NewFromInt(4),
	}

	livePayload, err := json.Marshal(GetNetworthProjection(db, req))
	require.NoError(t, err)

	require.NoError(t, RefreshNetworthProjectionSnapshot(db))

	snapshotPayload, err := json.Marshal(GetNetworthProjection(db, req))
	require.NoError(t, err)

	assert.JSONEq(t, string(livePayload), string(snapshotPayload))
}

func TestGetNetworthProjection_PrefersSnapshotWhenPresent(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	t.Cleanup(utils.UnsetNow)

	db := openTestDB(t)
	seedProjectionTestPostings(t, db)
	require.NoError(t, RefreshNetworthProjectionSnapshot(db))
	require.NoError(t, db.Exec("DELETE FROM postings").Error)

	result := GetNetworthProjection(db, NetworthProjectionRequest{
		Years:            15,
		ConservativeCAGR: decimal.NewFromInt(8),
		ExpectedCAGR:     decimal.NewFromInt(12),
		OptimisticCAGR:   decimal.NewFromInt(16),
		SWR:              decimal.NewFromInt(4),
	})

	assert.True(t, result["current_networth"].(decimal.Decimal).Equal(decimal.NewFromInt(60000)))
	assert.True(t, result["derived_contribution"].(decimal.Decimal).Equal(decimal.NewFromInt(3000)))
	assert.True(t, result["annual_expenses"].(decimal.Decimal).Equal(decimal.NewFromInt(60000)))
	assert.True(t, result["savings_rate"].(decimal.Decimal).Equal(decimal.NewFromInt(30)))
}

func seedProjectionTestPostings(t *testing.T, db *gorm.DB) {
	t.Helper()

	postings := make([]posting.Posting, 0, 36)
	for month := 0; month < 12; month++ {
		baseDate := mustParseDate("2024-04-10").AddDate(0, month, 0)
		idPrefix := baseDate.Format("2006-01")

		postings = append(postings,
			posting.Posting{
				TransactionID:  idPrefix + "-salary",
				Date:           baseDate,
				Account:        "Assets:Checking",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(10000),
				Amount:         decimal.NewFromInt(10000),
				OriginalAmount: decimal.NewFromInt(10000),
			},
			posting.Posting{
				TransactionID:  idPrefix + "-salary",
				Date:           baseDate,
				Account:        "Income:Salary",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(-10000),
				Amount:         decimal.NewFromInt(-10000),
				OriginalAmount: decimal.NewFromInt(-10000),
			},
			posting.Posting{
				TransactionID:  idPrefix + "-expense",
				Date:           baseDate.AddDate(0, 0, 2),
				Account:        "Expenses:Household",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(5000),
				Amount:         decimal.NewFromInt(5000),
				OriginalAmount: decimal.NewFromInt(5000),
			},
			posting.Posting{
				TransactionID:  idPrefix + "-expense",
				Date:           baseDate.AddDate(0, 0, 2),
				Account:        "Assets:Checking",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(-5000),
				Amount:         decimal.NewFromInt(-5000),
				OriginalAmount: decimal.NewFromInt(-5000),
			},
			posting.Posting{
				TransactionID:  idPrefix + "-invest",
				Date:           baseDate.AddDate(0, 0, 5),
				Account:        "Assets:Investments:Index",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(3000),
				Amount:         decimal.NewFromInt(3000),
				OriginalAmount: decimal.NewFromInt(3000),
			},
			posting.Posting{
				TransactionID:  idPrefix + "-invest",
				Date:           baseDate.AddDate(0, 0, 5),
				Account:        "Assets:Checking",
				Commodity:      "INR",
				Quantity:       decimal.NewFromInt(-3000),
				Amount:         decimal.NewFromInt(-3000),
				OriginalAmount: decimal.NewFromInt(-3000),
			},
		)
	}

	require.NoError(t, db.Create(&postings).Error)
}
