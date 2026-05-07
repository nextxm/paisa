package server

import (
	"net/http"

	"github.com/ananthakumaran/paisa/internal/model/import_preset"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ImportPresetRequest struct {
	Name            string            `json:"name" binding:"required"`
	ColumnMappings  map[string]string `json:"column_mappings"`
	DateFormat      string            `json:"date_format"`
	DefaultAccounts map[string]string `json:"default_accounts"`
	Delimiter       string            `json:"delimiter"`
}

type ImportPresetDeleteRequest struct {
	Name string `json:"name" binding:"required"`
}

func handleGetImportPresets(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		presets, err := import_preset.All(db)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"presets": presets})
	}
}

func handleUpsertImportPreset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportPresetRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		preset, err := import_preset.Upsert(db, import_preset.ImportPreset{
			Name:            req.Name,
			ColumnMappings:  req.ColumnMappings,
			DateFormat:      req.DateFormat,
			DefaultAccounts: req.DefaultAccounts,
			Delimiter:       req.Delimiter,
		})
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"preset": preset, "saved": true})
	}
}

func handleDeleteImportPreset(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ImportPresetDeleteRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		if err := import_preset.Delete(db, req.Name); err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}
