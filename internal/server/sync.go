package server

import (
	"github.com/ananthakumaran/paisa/internal/cache"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncRequest struct {
	Journal    bool `json:"journal"`
	Prices     bool `json:"prices"`
	Portfolios bool `json:"portfolios"`
}

// Sync executes the requested sync stages synchronously and returns the
// response payload together with a (possibly empty) slice of per-step
// diagnostic messages that should be surfaced to operators via the job
// Details field.
func Sync(db *gorm.DB, request SyncRequest) (gin.H, []string) {
	cache.Clear()

	var journalResult model.SyncResult
	var details []string

	if request.Journal {
		var err error
		journalResult, err = model.SyncJournal(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": journalResult.FailedStage,
				"message":      journalResult.Message,
			}, details
		}
	}

	if request.Prices {
		commoditiesResult, err := model.SyncCommodities(db)
		// Accumulate per-commodity failures into details regardless of whether
		// the overall sync succeeded or failed.
		details = append(details, commoditiesResult.Failures...)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "commodities",
				"message":      err.Error(),
			}, details
		}
		err = model.SyncCII(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "cii",
				"message":      err.Error(),
			}, details
		}
	}

	if request.Portfolios {
		err := model.SyncPortfolios(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "portfolios",
				"message":      err.Error(),
			}, details
		}
	}

	// Warm the market price and FX-rate in-memory caches so the first API
	// request after sync is served quickly.
	service.WarmCache(db)

	// Wrap XIRR calculations in the job flow: pre-compute XIRR for every
	// investment account and store the results in the SQLite computation cache.
	// Any accounts whose XIRR solver did not converge are recorded as Details
	// so operators can investigate without having to inspect server logs.
	if request.Journal || request.Prices {
		xirrWarnings := service.WarmXIRRCache(db)
		details = append(details, xirrWarnings...)
	}

	return gin.H{
		"success":       true,
		"posting_count": journalResult.PostingCount,
		"price_count":   journalResult.PriceCount,
	}, details
}
