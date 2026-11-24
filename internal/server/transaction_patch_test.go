package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPatchTransactionLines(t *testing.T) {
	lines := []string{
		"2024/05/01 * Coffee Shop",
		"    Assets:Cash  -5.00 USD",
		"    Expenses:Food  5.00 USD",
		"",
		"2024/05/02 * Grocery Store",
		"    Assets:Cash  -20.00 USD",
		"    Expenses:Groceries  20.00 USD",
	}

	t.Run("replace transaction", func(t *testing.T) {
		out, err := patchTransactionLines(lines, 1, 3, "replace", "2024/05/01 * Coffee Shop\n    Assets:Cash  -5.00 USD\n    Expenses:Dining  5.00 USD")
		require.NoError(t, err)
		assert.Contains(t, out, "Expenses:Dining  5.00 USD")
		assert.Contains(t, out, "2024/05/02 * Grocery Store")
	})

	t.Run("delete transaction consumes trailing blank line", func(t *testing.T) {
		out, err := patchTransactionLines(lines, 1, 3, "delete", "")
		require.NoError(t, err)
		assert.Equal(t, "2024/05/02 * Grocery Store\n    Assets:Cash  -20.00 USD\n    Expenses:Groceries  20.00 USD", out)
	})

	t.Run("clamps endLine at EOF when lastPostingLine+1 overshoots by one", func(t *testing.T) {
		out, err := patchTransactionLines(lines, 5, 8, "replace", "2024/05/02 * Grocery Store\n    Assets:Cash  -25.00 USD\n    Expenses:Groceries  25.00 USD")
		require.NoError(t, err)
		assert.Contains(t, out, "Expenses:Groceries  25.00 USD")
	})

	t.Run("out of bounds rejected", func(t *testing.T) {
		_, err := patchTransactionLines(lines, 1, 99, "delete", "")
		assert.Error(t, err)
	})
}

func TestGetRawTransactionHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	journalPath := filepath.Join(tmpDir, "main.ledger")
	err := os.WriteFile(journalPath, []byte("2024/05/01 * Coffee\n    Assets:Cash  -5.00 USD\n    Expenses:Food  5.00 USD\n"), 0644)
	require.NoError(t, err)

	config.LoadConfig([]byte("journal_path: "+journalPath+"\ndb_path: "+filepath.Join(tmpDir, "paisa.db")+"\nledger_cli: ledger\n"), "")

	r := gin.New()
	r.GET("/api/transaction/raw", GetRawTransactionHandler())

	t.Run("returns raw lines", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/transaction/raw?file_name=main.ledger&begin_line=1&end_line=3", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "2024/05/01 * Coffee\n    Assets:Cash  -5.00 USD\n    Expenses:Food  5.00 USD", resp["raw"])
	})

	t.Run("clamps end_line at EOF without trailing newline", func(t *testing.T) {
		noNewlinePath := filepath.Join(tmpDir, "no_newline.beancount")
		err := os.WriteFile(noNewlinePath, []byte("2026-06-01 * \"Test\"\n  Expenses:Groceries  12 CAD\n  Assets:CAD:Cash"), 0644)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/api/transaction/raw?file_name=no_newline.beancount&begin_line=1&end_line=4", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		var resp map[string]any
		require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
		assert.Equal(t, "2026-06-01 * \"Test\"\n  Expenses:Groceries  12 CAD\n  Assets:CAD:Cash", resp["raw"])
	})

	t.Run("rejects path traversal", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/transaction/raw?file_name=../secret.ledger&begin_line=1&end_line=1", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestPatchTransactionHandler_RejectsTraversalAndMissingAction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmpDir := t.TempDir()
	journalPath := filepath.Join(tmpDir, "main.ledger")
	err := os.WriteFile(journalPath, []byte("2024/05/01 * Coffee\n    Assets:Cash  -5.00 USD\n    Expenses:Food  5.00 USD\n"), 0644)
	require.NoError(t, err)

	config.LoadConfig([]byte("journal_path: "+journalPath+"\ndb_path: "+filepath.Join(tmpDir, "paisa.db")+"\nledger_cli: ledger\n"), "")

	r := gin.New()
	r.POST("/api/transaction/patch", PatchTransactionHandler(nil))

	body, _ := json.Marshal(TransactionPatchRequest{
		FileName:  "../evil.ledger",
		BeginLine: 1,
		EndLine:   3,
		Action:    "delete",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/transaction/patch", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}
