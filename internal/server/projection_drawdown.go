package server

import (
	projectiondrawdown "github.com/ananthakumaran/paisa/internal/projection/drawdown"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type DrawdownRequest struct {
	Amount  float64                             `json:"amount"`
	Buckets []projectiondrawdown.DrawdownBucket `json:"buckets"`
}

func GetProjectionDrawdown(db *gorm.DB, req DrawdownRequest) map[string]any {
	analysis := projectiondrawdown.Analyze(db, decimal.NewFromFloat(req.Amount), req.Buckets)
	return map[string]any{"drawdown": analysis}
}
