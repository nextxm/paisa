package server

import (
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/server/assets"
	"github.com/ananthakumaran/paisa/internal/server/goal"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type dashboardStageTimings map[string]int64

func buildDashboardWithTimings(db *gorm.DB) (gin.H, dashboardStageTimings) {
	timings := dashboardStageTimings{}

	measure := func(name string, fn func() any) any {
		start := time.Now()
		v := fn()
		timings[name] = time.Since(start).Milliseconds()
		return v
	}

	result := gin.H{
		"checkingBalances":     measure("checkingBalances", func() any { return assets.GetCheckingBalance(db, "") }),
		"networth":             measure("networth", func() any { return GetCurrentNetworth(db) }),
		"expenses":             measure("expenses", func() any { return GetCurrentExpense(db) }),
		"cashFlows":            measure("cashFlows", func() any { return GetCurrentCashFlow(db) }),
		"transactionSequences": measure("transactionSequences", func() any { return ComputeRecurringTransactions(query.Init(db).All()) }),
		"transactions":         measure("transactions", func() any { return GetLatestTransactions(db) }),
		"budget":               measure("budget", func() any { return GetCurrentBudget(db) }),
		"goalSummaries":        measure("goalSummaries", func() any { return goal.GetGoalSummaries(db) }),
	}

	return result, timings
}

func encodeDashboardStageTimings(timings dashboardStageTimings) string {
	if len(timings) == 0 {
		return ""
	}

	keys := make([]string, 0, len(timings))
	for k := range timings {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+strconv.FormatInt(timings[k], 10))
	}
	return strings.Join(parts, ",")
}

func GetDashboard(db *gorm.DB) gin.H {
	result, _ := buildDashboardWithTimings(db)
	return result
}
