package server

import (
	"sync"

	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/server/assets"
	"github.com/ananthakumaran/paisa/internal/server/goal"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetDashboard(db *gorm.DB) gin.H {
	// Each sub-computation is independent (read-only), so we run them
	// concurrently to reduce overall dashboard latency.
	var (
		wg sync.WaitGroup

		checkingBalances     interface{}
		networth             interface{}
		expenses             interface{}
		cashFlows            interface{}
		transactionSequences interface{}
		transactions         interface{}
		budget               interface{}
		goalSummaries        interface{}
	)

	wg.Add(8)
	go func() { defer wg.Done(); checkingBalances = assets.GetCheckingBalance(db, "") }()
	go func() { defer wg.Done(); networth = GetCurrentNetworth(db) }()
	go func() { defer wg.Done(); expenses = GetCurrentExpense(db) }()
	go func() { defer wg.Done(); cashFlows = GetCurrentCashFlow(db) }()
	go func() {
		defer wg.Done()
		transactionSequences = ComputeRecurringTransactions(query.Init(db).All())
	}()
	go func() { defer wg.Done(); transactions = GetLatestTransactions(db) }()
	go func() { defer wg.Done(); budget = GetCurrentBudget(db) }()
	go func() { defer wg.Done(); goalSummaries = goal.GetGoalSummaries(db) }()
	wg.Wait()

	return gin.H{
		"checkingBalances":     checkingBalances,
		"networth":             networth,
		"expenses":             expenses,
		"cashFlows":            cashFlows,
		"transactionSequences": transactionSequences,
		"transactions":         transactions,
		"budget":               budget,
		"goalSummaries":        goalSummaries,
	}
}

