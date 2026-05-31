package goals

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
)

func TestExpandLifeGoalsBuildsMilestonesAndRecurringCashflows(t *testing.T) {
	start := time.Date(2026, time.January, 15, 0, 0, 0, 0, time.UTC)
	inflation := 8.0

	cashflows, metadata, err := ExpandLifeGoals([]config.LifeGoal{
		{
			Name:         "House Down Payment",
			Type:         "milestone",
			TargetAmount: 5000000,
			TargetDate:   "2027-06",
			Priority:     10,
		},
		{
			Name:          "Family Vacation",
			Type:          "recurring",
			StartDate:     "2026-03",
			EndDate:       "2026-09",
			Frequency:     "quarterly",
			MonthlyAlloc:  120000,
			InflationRate: &inflation,
			Priority:      2,
		},
	}, start)
	if err != nil {
		t.Fatalf("ExpandLifeGoals returned error: %v", err)
	}

	if len(cashflows) != 4 {
		t.Fatalf("len(cashflows) = %d, want 4", len(cashflows))
	}
	if cashflows[0].GoalID != "Family Vacation" || cashflows[0].Month != 2 {
		t.Fatalf("first recurring cashflow = %+v, want Family Vacation at month 2", cashflows[0])
	}
	if cashflows[1].Month != 5 || cashflows[2].Month != 8 || cashflows[3].Month != 17 {
		t.Fatalf("unexpected cashflow months: %+v", cashflows)
	}
	if len(metadata) != 2 {
		t.Fatalf("len(metadata) = %d, want 2", len(metadata))
	}
	if metadata[0].Name != "House Down Payment" || metadata[0].CashflowCount != 1 {
		t.Fatalf("unexpected milestone metadata: %+v", metadata[0])
	}
	if metadata[1].CashflowCount != 3 {
		t.Fatalf("recurring cashflow count = %d, want 3", metadata[1].CashflowCount)
	}
}

func TestExpandLifeGoalsRejectsInvalidRecurringWindow(t *testing.T) {
	start := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	_, _, err := ExpandLifeGoals([]config.LifeGoal{{
		Name:      "Broken Goal",
		Type:      "recurring",
		StartDate: "2028-01",
		EndDate:   "2027-01",
		Frequency: "monthly",
	}}, start)
	if err == nil {
		t.Fatal("ExpandLifeGoals error = nil, want error")
	}
}
