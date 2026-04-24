package liabilities

import (
	"github.com/nextxm/paisa/internal/query"
	"github.com/nextxm/paisa/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetRepayment(db *gorm.DB) gin.H {
	postings := query.Init(db).Like("Liabilities:%").Credit().All()
	postings = service.PopulateMarketPrice(db, postings)
	expenses := query.Init(db).Like("Expenses:Interest:%").All()
	postings = append(postings, expenses...)
	return gin.H{"repayments": postings}
}
