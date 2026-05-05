package server

import (
	"net/http"

	"github.com/ananthakumaran/paisa/internal/model/account_note"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type accountNoteRequest struct {
	Account string `json:"account" binding:"required"`
	Note    string `json:"note"`
}

// GetAllAccountNotes returns all stored account notes.
func GetAllAccountNotes(db *gorm.DB, c *gin.Context) {
	notes, err := account_note.GetAll(db)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_notes": notes})
}

// GetAccountNote returns the note for the account name given in the URL parameter.
func GetAccountNote(db *gorm.DB, c *gin.Context) {
	accountName := c.Param("account")
	note, err := account_note.Get(db, accountName)
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusOK, gin.H{"account_note": nil, "found": false})
			return
		}
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_note": note, "found": true})
}

// UpsertAccountNote creates or updates the note for an account.
func UpsertAccountNote(db *gorm.DB, c *gin.Context) {
	var req accountNoteRequest
	if !BindJSONOrError(c, &req) {
		return
	}

	note, err := account_note.Upsert(db, req.Account, req.Note)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"account_note": note, "saved": true})
}

// DeleteAccountNote removes the note for the account supplied in the request body.
func DeleteAccountNote(db *gorm.DB, c *gin.Context) {
	var req struct {
		Account string `json:"account" binding:"required"`
	}
	if !BindJSONOrError(c, &req) {
		return
	}

	if err := account_note.Delete(db, req.Account); err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}
