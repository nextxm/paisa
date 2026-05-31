package server

import (
	"github.com/ananthakumaran/paisa/internal/projection/dna"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetFinancialDNA(db *gorm.DB) gin.H {
	profile := dna.ExtractProfile(db)
	return gin.H{
		"profile": profile,
	}
}
