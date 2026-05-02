package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type AddTransactionRequest struct {
	Date         string `json:"date" binding:"required"`
	Payee        string `json:"payee"`
	Narration    string `json:"narration"`
	FromAccount  string `json:"from_account" binding:"required"`
	ToAccount    string `json:"to_account" binding:"required"`
	Amount       string `json:"amount" binding:"required"`
	Commodity    string `json:"commodity" binding:"required"`
	ToAmount     string `json:"to_amount"`
	ToCommodity  string `json:"to_commodity"`
	ExchangeRate string `json:"exchange_rate"`
}

func matchAccount(db *gorm.DB, query string) (string, error) {
	accounts := accounting.AllAccounts(db)
	if len(accounts) == 0 {
		return query, nil // If no accounts in DB, just return the query
	}

	lowerQuery := strings.ToLower(query)

	// 1. Exact match (case-insensitive)
	for _, acc := range accounts {
		if strings.ToLower(acc) == lowerQuery {
			return acc, nil
		}
	}

	// 2. Contains match
	var matches []string
	for _, acc := range accounts {
		if strings.Contains(strings.ToLower(acc), lowerQuery) {
			matches = append(matches, acc)
		}
	}

	if len(matches) == 1 {
		return matches[0], nil
	} else if len(matches) > 1 {
		// Pick shortest match as best guess, or maybe error out?
		// We'll pick shortest match as the most "direct" hit.
		best := matches[0]
		for _, m := range matches[1:] {
			if len(m) < len(best) {
				best = m
			}
		}
		return best, nil
	}

	// 3. If no matches, return query as new account
	return query, nil
}

func formatTransaction(req AddTransactionRequest) string {
	var sb strings.Builder
	dialect := config.GetConfig().LedgerCli

	// Clean strings
	payee := strings.TrimSpace(req.Payee)
	narration := strings.TrimSpace(req.Narration)

	// Header line
	if dialect == "beancount" {
		sb.WriteString(req.Date)
		sb.WriteString(" * ")
		if payee != "" {
			sb.WriteString(fmt.Sprintf("%q ", payee))
		}
		if narration != "" {
			sb.WriteString(fmt.Sprintf("%q", narration))
		}
		sb.WriteString("\n")
	} else {
		// ledger or hledger
		dateStr := strings.ReplaceAll(req.Date, "-", "/")
		sb.WriteString(dateStr)
		sb.WriteString(" * ")
		if payee != "" && narration != "" {
			sb.WriteString(payee + " | " + narration)
		} else if payee != "" {
			sb.WriteString(payee)
		} else if narration != "" {
			sb.WriteString(narration)
		}
		sb.WriteString("\n")
	}

	// Write FromAccount posting (negative amount)
	sb.WriteString("  ")
	sb.WriteString(req.FromAccount)
	sb.WriteString("  -")
	sb.WriteString(req.Amount)
	sb.WriteString(" ")
	sb.WriteString(req.Commodity)

	// Add Exchange rate or ToAmount notation to the FromAccount if provided
	if req.ToAmount != "" && req.ToCommodity != "" && req.ToCommodity != req.Commodity {
		if dialect == "beancount" {
			sb.WriteString(fmt.Sprintf(" @@ %s %s", req.ToAmount, req.ToCommodity))
		} else {
			sb.WriteString(fmt.Sprintf(" @@ %s %s", req.ToAmount, req.ToCommodity))
		}
	} else if req.ExchangeRate != "" {
		targetCmdty := req.ToCommodity
		if targetCmdty == "" {
			targetCmdty = req.Commodity
		}
		sb.WriteString(fmt.Sprintf(" @ %s %s", req.ExchangeRate, targetCmdty))
	}
	sb.WriteString("\n")

	// Write ToAccount posting (positive amount)
	sb.WriteString("  ")
	sb.WriteString(req.ToAccount)

	if req.ToAmount != "" && req.ToCommodity != "" {
		sb.WriteString("   ")
		sb.WriteString(req.ToAmount)
		sb.WriteString(" ")
		sb.WriteString(req.ToCommodity)
	} else {
		// Just the positive amount in the same commodity
		sb.WriteString("   ")
		sb.WriteString(req.Amount)
		sb.WriteString(" ")
		sb.WriteString(req.Commodity)
	}
	sb.WriteString("\n")

	// Add an extra newline for separation
	sb.WriteString("\n")

	return sb.String()
}

func appendTransactionAndSync(db *gorm.DB, req AddTransactionRequest) (string, error) {
	addPath := config.GetAddJournalPath()
	if addPath == "" {
		return "", fmt.Errorf("add_journal_path is not configured")
	}

	// Resolve accounts
	fromAcc, _ := matchAccount(db, req.FromAccount)
	req.FromAccount = fromAcc

	toAcc, _ := matchAccount(db, req.ToAccount)
	req.ToAccount = toAcc

	// Format
	entryText := formatTransaction(req)

	// Ensure directory exists
	err := os.MkdirAll(filepath.Dir(addPath), 0750)
	if err != nil {
		log.Warn("Failed to create add journal directory: ", err)
		return "", fmt.Errorf("failed to create directory")
	}

	// Append to file
	f, err := os.OpenFile(addPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Warn("Failed to open add journal file: ", err)
		return "", fmt.Errorf("failed to open file")
	}
	defer f.Close()

	if _, err := f.WriteString(entryText); err != nil {
		log.Warn("Failed to write to add journal file: ", err)
		return "", fmt.Errorf("failed to write file")
	}

	// Trigger sync
	Sync(db, SyncRequest{Journal: true})

	return entryText, nil
}

func AddTransactionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req AddTransactionRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		entryText, err := appendTransactionAndSync(db, req)
		if err != nil {
			if err.Error() == "add_journal_path is not configured" {
				RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			} else {
				RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			}
			return
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "entry": entryText})
	}
}
