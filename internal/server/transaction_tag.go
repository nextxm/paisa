package server

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/ananthakumaran/paisa/internal/model/transaction_tag"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type transactionTagRequest struct {
	Tag string `json:"tag" binding:"required"`
}

func GetTransactionTagsHandler(db *gorm.DB, c *gin.Context) {
	tags, err := transaction_tag.ListByTransactionID(db, c.Param("id"))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func AddTransactionTagHandler(db *gorm.DB, c *gin.Context) {
	var req transactionTagRequest
	if !BindJSONOrError(c, &req) {
		return
	}
	if strings.TrimSpace(req.Tag) == "" {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "tag cannot be empty")
		return
	}
	if _, err := transaction_tag.Add(db, c.Param("id"), req.Tag); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	tags, err := transaction_tag.ListByTransactionID(db, c.Param("id"))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func DeleteTransactionTagHandler(db *gorm.DB, c *gin.Context) {
	if err := transaction_tag.Delete(db, c.Param("id"), c.Param("tag")); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	tags, err := transaction_tag.ListByTransactionID(db, c.Param("id"))
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

func GetTagAutocompleteHandler(db *gorm.DB, c *gin.Context) {
	limit := 10
	if rawLimit := c.Query("limit"); rawLimit != "" {
		if n, err := strconv.Atoi(rawLimit); err == nil && n > 0 {
			limit = n
		}
	}
	tags, err := transaction_tag.Autocomplete(db, c.Query("q"), limit)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}
