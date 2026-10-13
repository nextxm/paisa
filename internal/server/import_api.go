package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	BaseAccount  string                       `json:"base_account"` // The account being statemented, e.g. Assets:Checking:Chase
	Commodity    string                       `json:"commodity"`
	GroupByDate  bool                         `json:"group_by_date"`
	RawContent   string                       `json:"raw_content"`
	Transactions []CommitCandidateTransaction `json:"transactions"`
}

// CommitCandidateTransaction represents an individual entry to append.
type CommitCandidateTransaction struct {
	Date            string   `json:"date" binding:"required"`
	Payee           string   `json:"payee"`
	Memo            string   `json:"memo"`
	Amount          float64  `json:"amount" binding:"required"`
	Currency        string   `json:"currency"`
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

		trimmedRaw := strings.TrimSpace(req.RawContent)
		if len(req.Transactions) == 0 && trimmedRaw == "" {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "no transactions to commit")
			return
		}

		defaultCommodity := req.Commodity
		if defaultCommodity == "" {
			defaultCommodity = config.DefaultCurrency()
		}
		if defaultCommodity == "" {
			defaultCommodity = "INR"
		}

		targetFile := config.GetAddJournalPath()
		if targetFile == "" {
			targetFile = config.GetJournalPath()
		}

		var contentToAppend string
		if trimmedRaw != "" {
			contentToAppend = "\n" + trimmedRaw + "\n"
		} else {
			contentToAppend = formatTransactionsForCommit(db, req, defaultCommodity)
		}

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

		committedCount := len(req.Transactions)
		if committedCount == 0 && trimmedRaw != "" {
			committedCount = 1
		}

		c.JSON(http.StatusOK, gin.H{
			"success":         true,
			"committed_count": committedCount,
			"target_file":     targetFile,
			"errors":          errors,
		})
	}
}

func formatTransactionsForCommit(db *gorm.DB, req CommitImportRequest, defaultCommodity string) string {
	var sb strings.Builder
	dialect := config.GetConfig().LedgerCli
	baseAcc, _ := matchAccount(db, req.BaseAccount)
	if baseAcc == "" {
		baseAcc = req.BaseAccount
	}

	if req.GroupByDate && dialect != "beancount" {
		// Group debit entries by (Date, Currency) while preserving chronological appearance
		type groupKey struct {
			date     string
			currency string
		}
		var debitKeys []groupKey
		debitGroups := make(map[groupKey][]CommitCandidateTransaction)
		var creditTxs []CommitCandidateTransaction

		for _, tx := range req.Transactions {
			curr := strings.TrimSpace(tx.Currency)
			if curr == "" {
				curr = defaultCommodity
			}
			tx.Currency = curr

			if tx.IsDebit {
				k := groupKey{date: tx.Date, currency: curr}
				if _, exists := debitGroups[k]; !exists {
					debitKeys = append(debitKeys, k)
				}
				debitGroups[k] = append(debitGroups[k], tx)
			} else {
				creditTxs = append(creditTxs, tx)
			}
		}

		for _, k := range debitKeys {
			txs := debitGroups[k]
			dateStr := strings.ReplaceAll(k.date, "-", "/")
			if len(txs) == 1 {
				writeSingleLedgerEntry(&sb, db, baseAcc, txs[0], k.currency, dialect)
				continue
			}

			sb.WriteString(fmt.Sprintf("%s * Consolidated Debits (%d items)\n", dateStr, len(txs)))
			var total float64
			for _, tx := range txs {
				offsetAcc, _ := matchAccount(db, tx.SelectedAccount)
				if offsetAcc == "" {
					offsetAcc = tx.SelectedAccount
				}
				amtStr := fmt.Sprintf("%.2f", tx.Amount)
				total += tx.Amount
				note := extractShortNote(tx.Payee, tx.Memo)
				if note != "" {
					sb.WriteString(fmt.Sprintf("  %-40s   %s %s  ; %s\n", offsetAcc, amtStr, k.currency, note))
				} else {
					sb.WriteString(fmt.Sprintf("  %-40s   %s %s\n", offsetAcc, amtStr, k.currency))
				}
			}
			sb.WriteString(fmt.Sprintf("  %-40s  -%.2f %s\n\n", baseAcc, total, k.currency))
		}

		for _, tx := range creditTxs {
			writeSingleLedgerEntry(&sb, db, baseAcc, tx, tx.Currency, dialect)
		}
		return sb.String()
	}

	for _, tx := range req.Transactions {
		curr := strings.TrimSpace(tx.Currency)
		if curr == "" {
			curr = defaultCommodity
		}
		writeSingleLedgerEntry(&sb, db, baseAcc, tx, curr, dialect)
	}

	return sb.String()
}

var (
	txPrefixRe = regexp.MustCompile(`(?i)^(UPI|NEFT|IMPS|RTGS|POS|ATM|ACH|BIL|MMT|BBPS|ECOM|VPS|IPS)[-/\s:*]+`)
	refIdRe    = regexp.MustCompile(`(?i)\b(REF|TXN|TRN|UTR|RRN|ID|NO|NUM|CHQ)[\s.:#-]*[A-Z0-9]{5,}\b|\b\d{6,}\b|\b[A-Z0-9]*\d{4,}[A-Z0-9]*\b`)
	vpaRe      = regexp.MustCompile(`\S+@\S+`)
	multiSpace = regexp.MustCompile(`\s+`)
)

// extractShortNote strips transaction prefixes, reference/transaction IDs, and VPA handles to produce a concise note.
func extractShortNote(payee, memo string) string {
	raw := strings.TrimSpace(payee)
	if raw == "" {
		raw = strings.TrimSpace(memo)
	}
	if raw == "" {
		return ""
	}

	// Strip common payment prefixes like UPI-, NEFT/, POS etc.
	cleaned := txPrefixRe.ReplaceAllString(raw, "")

	// Split by common delimiters (/, -, |, *) and keep meaningful non-ID segments
	segments := strings.FieldsFunc(cleaned, func(r rune) bool {
		return r == '/' || r == '-' || r == '|' || r == '*'
	})

	var meaningful []string
	for _, seg := range segments {
		s := strings.TrimSpace(seg)
		if s == "" {
			continue
		}
		// Remove VPA handles and long reference IDs
		s = vpaRe.ReplaceAllString(s, "")
		s = refIdRe.ReplaceAllString(s, "")
		s = strings.TrimSpace(multiSpace.ReplaceAllString(s, " "))
		// Skip pure noise tokens
		upper := strings.ToUpper(s)
		if s == "" || upper == "UPI" || upper == "NEFT" || upper == "IMPS" || upper == "RTGS" || upper == "DR" || upper == "CR" || upper == "IN" || upper == "IND" {
			continue
		}
		meaningful = append(meaningful, s)
	}

	result := strings.Join(meaningful, " ")
	if result == "" {
		// Fallback to basic reference number stripping on raw string
		result = refIdRe.ReplaceAllString(cleaned, "")
		result = strings.TrimSpace(multiSpace.ReplaceAllString(result, " "))
	}

	if len(result) > 32 {
		result = strings.TrimSpace(result[:32])
	}
	return result
}

func writeSingleLedgerEntry(sb *strings.Builder, db *gorm.DB, baseAcc string, tx CommitCandidateTransaction, commodity string, dialect string) {
	offsetAcc, _ := matchAccount(db, tx.SelectedAccount)
	if offsetAcc == "" {
		offsetAcc = tx.SelectedAccount
	}

	shortNote := extractShortNote(tx.Payee, tx.Memo)
	if shortNote == "" {
		shortNote = strings.TrimSpace(tx.Payee)
	}
	amtStr := fmt.Sprintf("%.2f", tx.Amount)

	if dialect == "beancount" {
		sb.WriteString(tx.Date)
		sb.WriteString(" * ")
		if shortNote != "" {
			sb.WriteString(fmt.Sprintf("%q", shortNote))
		}
		for _, tag := range tx.Tags {
			cleanTag := strings.TrimPrefix(tag, "#")
			sb.WriteString(" #" + cleanTag)
		}
		sb.WriteString("\n")
	} else {
		dateStr := strings.ReplaceAll(tx.Date, "-", "/")
		sb.WriteString(dateStr)
		sb.WriteString(" * ")
		sb.WriteString(shortNote)
		if len(tx.Tags) > 0 {
			sb.WriteString("  ; " + strings.Join(tx.Tags, " "))
		}
		sb.WriteString("\n")
	}

	if tx.IsDebit {
		sb.WriteString(fmt.Sprintf("  %-40s  -%s %s\n", baseAcc, amtStr, commodity))
		sb.WriteString(fmt.Sprintf("  %-40s   %s %s\n\n", offsetAcc, amtStr, commodity))
	} else {
		sb.WriteString(fmt.Sprintf("  %-40s   %s %s\n", baseAcc, amtStr, commodity))
		sb.WriteString(fmt.Sprintf("  %-40s  -%s %s\n\n", offsetAcc, amtStr, commodity))
	}
}

// Ensure accounting import is referenced
var _ = accounting.AllAccounts
