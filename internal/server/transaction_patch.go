package server

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TransactionPatchRequest struct {
	FileName  string `json:"file_name" binding:"required"`
	BeginLine int    `json:"begin_line" binding:"required"`
	EndLine   int    `json:"end_line" binding:"required"`
	Action    string `json:"action" binding:"required"` // "replace" or "delete"
	Content   string `json:"content"`
}

func resolveJournalSubPath(fileName string) (string, error) {
	journalPath := config.GetJournalPath()
	if journalPath == "" {
		return "", fmt.Errorf("journal_path is not configured")
	}
	dir := filepath.Dir(journalPath)
	return utils.BuildSubPath(dir, fileName)
}

func splitNormalizedLines(content string) []string {
	normalized := strings.ReplaceAll(content, "\r\n", "\n")
	return strings.Split(normalized, "\n")
}

func validateWithTempFile(filePath string, content string) ([]ledger.LedgerFileError, string, error) {
	tmpfile, err := os.CreateTemp(filepath.Dir(filePath), "paisa-tmp-")
	if err != nil {
		return nil, "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpfile.Name())

	if _, err := tmpfile.Write([]byte(content)); err != nil {
		tmpfile.Close()
		return nil, "", fmt.Errorf("failed to write temp file: %w", err)
	}
	if err := tmpfile.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close temp file: %w", err)
	}

	return ledger.Cli().ValidateFile(tmpfile.Name())
}

// clampEndLine adjusts endLine when Beancount's lastPostingLine+1 overshoots a file with no trailing newline.
func clampEndLine(beginLine, endLine, lineCount int) int {
	if endLine == lineCount+1 && beginLine >= 1 && beginLine <= lineCount {
		return lineCount
	}
	return endLine
}

func patchTransactionLines(lines []string, beginLine int, endLine int, action string, replacement string) (string, error) {
	endLine = clampEndLine(beginLine, endLine, len(lines))
	if beginLine < 1 || endLine < beginLine || endLine > len(lines) {
		return "", fmt.Errorf("line range %d..%d out of bounds (file has %d lines)", beginLine, endLine, len(lines))
	}

	var newLines []string
	switch action {
	case "replace":
		if strings.TrimSpace(replacement) == "" {
			return "", fmt.Errorf("content is required for replace action")
		}
		repLines := splitNormalizedLines(strings.TrimRight(replacement, "\r\n"))
		newLines = make([]string, 0, len(lines)-(endLine-beginLine+1)+len(repLines))
		newLines = append(newLines, lines[:beginLine-1]...)
		newLines = append(newLines, repLines...)
		newLines = append(newLines, lines[endLine:]...)
	case "delete":
		endIdx := endLine
		if endIdx < len(lines) && strings.TrimSpace(lines[endIdx]) == "" {
			endIdx++
		}
		newLines = make([]string, 0, len(lines)-(endIdx-(beginLine-1)))
		newLines = append(newLines, lines[:beginLine-1]...)
		newLines = append(newLines, lines[endIdx:]...)
	default:
		return "", fmt.Errorf("unsupported action %q", action)
	}

	return strings.Join(newLines, "\n"), nil
}

// GetRawTransactionHandler returns the raw journal lines for a transaction's file and line range.
func GetRawTransactionHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		fileName := strings.TrimSpace(c.Query("file_name"))
		beginLine, err1 := strconv.Atoi(c.Query("begin_line"))
		endLine, err2 := strconv.Atoi(c.Query("end_line"))
		if fileName == "" || err1 != nil || err2 != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "file_name, begin_line, and end_line are required")
			return
		}

		filePath, err := resolveJournalSubPath(fileName)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		rawBytes, err := os.ReadFile(filePath)
		if err != nil {
			RespondError(c, http.StatusNotFound, ErrCodeInvalidRequest, "failed to read journal file")
			return
		}

		lines := splitNormalizedLines(string(rawBytes))
		endLine = clampEndLine(beginLine, endLine, len(lines))
		if beginLine < 1 || endLine < beginLine || endLine > len(lines) {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "line range out of bounds")
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"file_name":  fileName,
			"begin_line": beginLine,
			"end_line":   endLine,
			"raw":        strings.Join(lines[beginLine-1:endLine], "\n"),
		})
	}
}

// PatchTransactionHandler patches or deletes a transaction line range in a journal file,
// validates the resulting file, and saves a .backup.* snapshot before writing.
func PatchTransactionHandler(_ *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req TransactionPatchRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		filePath, err := resolveJournalSubPath(req.FileName)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		fileStat, err := os.Stat(filePath)
		if err != nil {
			RespondError(c, http.StatusNotFound, ErrCodeInvalidRequest, "journal file does not exist")
			return
		}

		existingContent, err := os.ReadFile(filePath)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to read journal file")
			return
		}

		lines := splitNormalizedLines(string(existingContent))
		newContent, err := patchTransactionLines(lines, req.BeginLine, req.EndLine, req.Action, req.Content)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		errors, _, err := validateWithTempFile(filePath, newContent)
		if err != nil {
			log.Warn("Transaction patch validation failed: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		if len(errors) > 0 {
			c.JSON(http.StatusOK, gin.H{
				"saved":   false,
				"errors":  errors,
				"message": "Validation failed",
			})
			return
		}

		perm := fileStat.Mode().Perm()
		backupPath := filePath + ".backup." + time.Now().Format("2006-01-02-15-04-05.000")
		if err := os.WriteFile(backupPath, existingContent, perm); err != nil {
			log.Warn("Failed to create backup before patching: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to create backup")
			return
		}

		if err := os.WriteFile(filePath, []byte(newContent), perm); err != nil {
			log.Warn("Failed to write patched journal file: ", err)
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "failed to write file")
			return
		}

		invalidateJournalDirtyCache()
		c.JSON(http.StatusOK, gin.H{
			"saved":  true,
			"errors": errors,
		})
	}
}
