package server

import (
	"github.com/ananthakumaran/paisa/internal/accounting"
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

func Sync(db *gorm.DB, request SyncRequest) gin.H {
	cache.Clear()

	var journalResult model.SyncResult

	if request.Journal {
		var err error
		journalResult, err = model.SyncJournal(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": journalResult.FailedStage,
				"message":      journalResult.Message,
			}
		}
		// Pre-populate the accounting cache with sorted data immediately
		// after the journal write so that /api/editor/files and similar
		// endpoints always see a deterministically-ordered accounts list.
		accounting.AllAccounts(db)
	}

	if request.Prices {
		err := model.SyncCommodities(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "commodities",
				"message":      err.Error(),
			}
		}
		err = model.SyncCII(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "cii",
				"message":      err.Error(),
			}
		}
	}

	if request.Portfolios {
		err := model.SyncPortfolios(db)
		if err != nil {
			return gin.H{
				"success":      false,
				"failed_stage": "portfolios",
				"message":      err.Error(),
			}
		}
	}

	// Rebuild in-memory price/rate caches in the background from the freshly
	// written data.  cache.Clear() above reset them; this restores them
	// without blocking the HTTP response.
	service.WarmCaches(db)

	return gin.H{
		"success":       true,
		"posting_count": journalResult.PostingCount,
		"price_count":   journalResult.PriceCount,
	}
}
