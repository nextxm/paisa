package server

import (
	"github.com/nextxm/paisa/internal/accounting"
	"github.com/nextxm/paisa/internal/query"
	"github.com/nextxm/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetLedger(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	postings = service.PopulateMarketPrice(db, postings)
	postings = accounting.PopulateBalance(postings)
	accounting.SortDesc(postings)
	return gin.H{"postings": postings}
}
