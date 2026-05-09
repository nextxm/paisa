package server

import (
	"github.com/ananthakumaran/paisa/internal/cache"
	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncRequest struct {
	Journal     bool `json:"journal"`
	Prices      bool `json:"prices"`
	ForcePrices bool `json:"force_prices"`
	Portfolios  bool `json:"portfolios"`
}

// Sync executes the requested sync stages synchronously and returns the
// response payload together with a (possibly empty) slice of per-step
// diagnostic messages that should be surfaced to operators via the job
// Details field.
//
// progressFn, when non-nil, is forwarded to [model.SyncCommodities] so that
// the caller receives incremental "X of Y commodities" updates while the
// price-scraper stage is in progress.  Pass nil when progress reporting is
// not needed (e.g. the one-shot /api/init bootstrap).
func Sync(db *gorm.DB, request SyncRequest, progressFn func(completed, total int)) (gin.H, []string) {
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
		commoditiesResult, err := model.SyncCommodities(db, request.ForcePrices, progressFn)
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
	// request after sync is served quickly.  WarmCache is intentionally
	// unconditional: the in-memory caches are always cleared by cache.Clear()
	// at the top of Sync, so they must be re-populated regardless of which
	// stages ran.
	service.WarmCache(db)

	// Wrap XIRR calculations in the job flow: pre-compute XIRR for every
	// investment account and store the results in the SQLite computation cache.
	// WarmXIRRCache is conditional because it only makes sense when investment
	// data may have changed (journal or price sync was requested).  Any accounts
	// whose XIRR solver did not converge are recorded as Details so operators
	// can investigate without having to inspect server logs.
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
