package server

import (
	"github.com/ananthakumaran/paisa/internal/budget"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetBudget(db *gorm.DB) gin.H {
	forecastPostings := query.Init(db).Like("Expenses:%").Forecast().All()
	expenses := query.Init(db).Like("Expenses:%").All()
	result := budget.Compute(db, forecastPostings, expenses)
	return gin.H{
		"budgetsByMonth":        result.BudgetsByMonth,
		"checkingBalance":       result.CheckingBalance,
		"availableForBudgeting": result.AvailableForBudgeting,
	}
}

func GetCurrentBudget(db *gorm.DB) gin.H {
	forecastPostings := query.Init(db).Like("Expenses:%").Forecast().UntilThisMonthEnd().All()
	expenses := query.Init(db).Like("Expenses:%").UntilThisMonthEnd().All()
	result := budget.Compute(db, forecastPostings, expenses)
	return gin.H{
		"budgetsByMonth":        result.BudgetsByMonth,
		"checkingBalance":       result.CheckingBalance,
		"availableForBudgeting": result.AvailableForBudgeting,
	}
}
