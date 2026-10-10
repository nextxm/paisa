package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/importer"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/import_rule"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// HandleParseStatement parses statement uploads (CSV/OFX/QFX) and applies rules & duplicate scoring.
func HandleParseStatement(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req importer.IngestionRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		result, err := importer.Ingest(db, req)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		c.JSON(http.StatusOK, result)
	}
}

// HandleGetRules lists all user classification rules.
func HandleGetRules(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		rules, err := import_rule.All(db)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"rules": rules})
	}
}

// HandleSaveRule creates or updates an import rule.
func HandleSaveRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r import_rule.Rule
		if !BindJSONOrError(c, &r) {
			return
		}

		if err := import_rule.Save(db, &r); err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"rule": r, "saved": true})
	}
}

// HandleDeleteRule deletes an import rule by ID.
func HandleDeleteRule(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "invalid rule id")
			return
		}

		if err := import_rule.Delete(db, uint(id)); err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true})
	}
}

// CommitImportRequest defines candidate items to commit directly to the ledger.
type CommitImportRequest struct {
	BaseAccount  string                       `json:"base_account" binding:"required"` // The account being statemented, e.g. Assets:Checking:Chase
	Commodity    string                       `json:"commodity"`
	Transactions []CommitCandidateTransaction `json:"transactions" binding:"required"`
}

// CommitCandidateTransaction represents an individual entry to append.
type CommitCandidateTransaction struct {
	Date            string   `json:"date" binding:"required"`
	Payee           string   `json:"payee"`
	Memo            string   `json:"memo"`
	Amount          float64  `json:"amount" binding:"required"`
	IsDebit         bool     `json:"is_debit"`
	SelectedAccount string   `json:"selected_account" binding:"required"`
	Tags            []string `json:"tags"`
}

// HandleCommitImport writes the approved staged transactions to the journal file.
func HandleCommitImport(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CommitImportRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		if len(req.Transactions) == 0 {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "no transactions to commit")
			return
		}

		commodity := req.Commodity
		if commodity == "" {
			commodity = config.DefaultCurrency()
		}
		if commodity == "" {
			commodity = "USD"
		}

		targetFile := config.GetAddJournalPath()
		if targetFile == "" {
			targetFile = config.GetJournalPath()
		}

		var sb strings.Builder
		dialect := config.GetConfig().LedgerCli

		for _, tx := range req.Transactions {
			// Resolve accounts
			baseAcc, _ := matchAccount(db, req.BaseAccount)
			offsetAcc, _ := matchAccount(db, tx.SelectedAccount)

			payee := strings.TrimSpace(tx.Payee)
			memo := strings.TrimSpace(tx.Memo)
			amtStr := fmt.Sprintf("%.2f", tx.Amount)

			// Header line
			if dialect == "beancount" {
				sb.WriteString(tx.Date)
				sb.WriteString(" * ")
				if payee != "" {
					sb.WriteString(fmt.Sprintf("%q ", payee))
				}
				if memo != "" {
					sb.WriteString(fmt.Sprintf("%q", memo))
				}
				for _, tag := range tx.Tags {
					cleanTag := strings.TrimPrefix(tag, "#")
					sb.WriteString(" #" + cleanTag)
				}
				sb.WriteString("\n")
			} else {
				// ledger / hledger format
				dateStr := strings.ReplaceAll(tx.Date, "-", "/")
				sb.WriteString(dateStr)
				sb.WriteString(" * ")
				if payee != "" && memo != "" {
					sb.WriteString(payee)
					sb.WriteString(" | ")
					sb.WriteString(memo)
				} else if payee != "" {
					sb.WriteString(payee)
				} else if memo != "" {
					sb.WriteString(memo)
				}
				if len(tx.Tags) > 0 {
					sb.WriteString("  ; " + strings.Join(tx.Tags, " "))
				}
				sb.WriteString("\n")
			}

			// Posting lines:
			// If Debit (money spent): BaseAccount is reduced (-Amount), OffsetAccount is increased (+Amount)
			// If Credit (money received): BaseAccount is increased (+Amount), OffsetAccount is reduced (-Amount)
			if tx.IsDebit {
				sb.WriteString(fmt.Sprintf("  %-40s  -%s %s\n", baseAcc, amtStr, commodity))
				sb.WriteString(fmt.Sprintf("  %-40s   %s %s\n\n", offsetAcc, amtStr, commodity))
			} else {
				sb.WriteString(fmt.Sprintf("  %-40s   %s %s\n", baseAcc, amtStr, commodity))
				sb.WriteString(fmt.Sprintf("  %-40s  -%s %s\n\n", offsetAcc, amtStr, commodity))
			}
		}

		contentToAppend := sb.String()

		// Write to journal file
		if err := os.MkdirAll(filepath.Dir(targetFile), 0750); err != nil {
			log.Warn("Failed to create destination directory: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to create directory")
			return
		}

		f, err := os.OpenFile(targetFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Warn("Failed to open journal file for append: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to open journal file")
			return
		}
		defer f.Close()

		if _, err := f.WriteString(contentToAppend); err != nil {
			log.Warn("Failed to write to journal file: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to append to journal")
			return
		}

		// Validate ledger journal
		errors, _, _ := ledger.Cli().ValidateFile(config.GetJournalPath())

		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"committed_count": len(req.Transactions),
			"target_file":     targetFile,
			"errors":          errors,
		})
	}
}

// Ensure accounting import is referenced
var _ = accounting.AllAccounts
