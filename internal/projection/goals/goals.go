package goals

import (
	"fmt"
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/projection/simulator"
	"github.com/shopspring/decimal"
)

type LifeGoalMetadata struct {
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
}

func ExpandLifeGoals(lifeGoals []config.LifeGoal, startDate time.Time) ([]simulator.GoalCashflow, []LifeGoalMetadata, error) {
	startMonth := monthStart(startDate)
	var cashflows []simulator.GoalCashflow
	metadata := make([]LifeGoalMetadata, 0, len(lifeGoals))

	for _, goal := range lifeGoals {
		meta := LifeGoalMetadata{
			GoalID:            goal.Name,
			Name:              goal.Name,
			Icon:              goal.Icon,
			Type:              goal.Type,
			TargetAmount:      goal.TargetAmount,
			TargetDate:        goal.TargetDate,
			StartDate:         goal.StartDate,
			EndDate:           goal.EndDate,
			Frequency:         goal.Frequency,
			InflationRate:     goal.InflationRate,
			Priority:          goal.Priority,
			FundedBy:          append([]string{}, goal.FundedBy...),
			MonthlyAllocation: goal.MonthlyAlloc,
		}

		switch goal.Type {
		case "", "milestone":
			month, err := parseMonthOffset(startMonth, goal.TargetDate)
			if err != nil {
				return nil, nil, fmt.Errorf("life goal %q: %w", goal.Name, err)
			}
			if month >= 0 {
				cashflows = append(cashflows, simulator.GoalCashflow{
					GoalID:        goal.Name,
					Name:          goal.Name,
					Month:         month,
					Amount:        decimal.NewFromFloat(goal.TargetAmount),
					InflationRate: inflationDecimal(goal.InflationRate),
					Priority:      goal.Priority,
				})
				meta.CashflowCount = 1
			}
		case "recurring":
			months, err := expandRecurringMonths(startMonth, goal.StartDate, goal.EndDate, goal.Frequency)
			if err != nil {
				return nil, nil, fmt.Errorf("life goal %q: %w", goal.Name, err)
			}
			for _, month := range months {
				cashflows = append(cashflows, simulator.GoalCashflow{
					GoalID:        goal.Name,
					Name:          goal.Name,
					Month:         month,
					Amount:        decimal.NewFromFloat(goal.MonthlyAlloc),
					InflationRate: inflationDecimal(goal.InflationRate),
					Priority:      goal.Priority,
				})
			}
			meta.CashflowCount = len(months)
		default:
			return nil, nil, fmt.Errorf("life goal %q: unsupported type %q", goal.Name, goal.Type)
		}

		metadata = append(metadata, meta)
	}

	sort.Slice(cashflows, func(i, j int) bool {
		if cashflows[i].Month != cashflows[j].Month {
			return cashflows[i].Month < cashflows[j].Month
		}
		if cashflows[i].Priority != cashflows[j].Priority {
			return cashflows[i].Priority > cashflows[j].Priority
		}
		return cashflows[i].GoalID < cashflows[j].GoalID
	})

	sort.Slice(metadata, func(i, j int) bool {
		if metadata[i].Priority != metadata[j].Priority {
			return metadata[i].Priority > metadata[j].Priority
		}
		return metadata[i].Name < metadata[j].Name
	})

	return cashflows, metadata, nil
}

func parseMonthOffset(startMonth time.Time, raw string) (int, error) {
	if raw == "" {
		return -1, fmt.Errorf("missing month")
	}
	goalMonth, err := time.Parse("2006-01", raw)
	if err != nil {
		return -1, fmt.Errorf("invalid month %q", raw)
	}
	goalMonth = monthStart(goalMonth)
	return (goalMonth.Year()-startMonth.Year())*12 + int(goalMonth.Month()-startMonth.Month()), nil
}

func expandRecurringMonths(startMonth time.Time, startRaw string, endRaw string, frequency string) ([]int, error) {
	if startRaw == "" || endRaw == "" {
		return nil, fmt.Errorf("recurring goals require start_date and end_date")
	}
	startOffset, err := parseMonthOffset(startMonth, startRaw)
	if err != nil {
		return nil, err
	}
	endOffset, err := parseMonthOffset(startMonth, endRaw)
	if err != nil {
		return nil, err
	}
	if endOffset < startOffset {
		return nil, fmt.Errorf("end_date must not be before start_date")
	}

	step := 1
	switch frequency {
	case "", "monthly":
		step = 1
	case "quarterly":
		step = 3
	case "yearly":
		step = 12
	default:
		return nil, fmt.Errorf("unsupported frequency %q", frequency)
	}

	if endOffset < 0 {
		return []int{}, nil
	}
	if startOffset < 0 {
		startOffset = 0
	}

	months := []int{}
	for month := startOffset; month <= endOffset; month += step {
		months = append(months, month)
	}
	return months, nil
}

func monthStart(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func inflationDecimal(rate *float64) decimal.Decimal {
	if rate == nil {
		return decimal.Zero
	}
	return decimal.NewFromFloat(*rate)
}
