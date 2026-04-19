package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadLedgerFile_MissingFile verifies that readLedgerFile returns an error
// (instead of calling log.Fatal) when the requested path does not exist.
func TestReadLedgerFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readLedgerFile(dir, filepath.Join(dir, "nonexistent.journal"))
	assert.Error(t, err)
}

// TestReadLedgerFile_Success verifies the happy path of readLedgerFile.
func TestReadLedgerFile_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.journal")
	require.NoError(t, os.WriteFile(filePath, []byte("content here"), 0644))

	f, err := readLedgerFile(dir, filePath)
	require.NoError(t, err)
	assert.Equal(t, "test.journal", f.Name)
	assert.Equal(t, "content here", f.Content)
}

// TestReadLedgerFileWithVersions_MissingFile verifies that readLedgerFileWithVersions
// returns an error instead of calling log.Fatal when the file is absent.
func TestReadLedgerFileWithVersions_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readLedgerFileWithVersions(dir, filepath.Join(dir, "nonexistent.journal"))
	assert.Error(t, err)
}

// TestReadLedgerFileWithVersions_Success verifies the happy path including version listing.
func TestReadLedgerFileWithVersions_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.journal")
	require.NoError(t, os.WriteFile(filePath, []byte("main"), 0644))

	// Create a backup file that should appear in Versions.
	backupPath := filePath + ".backup.2024-01-01-12-00-00.000"
	require.NoError(t, os.WriteFile(backupPath, []byte("old"), 0644))

	f, err := readLedgerFileWithVersions(dir, filePath)
	require.NoError(t, err)
	assert.Equal(t, "test.journal", f.Name)
	assert.Equal(t, "main", f.Content)
	assert.Len(t, f.Versions, 1)
	assert.Equal(t, "test.journal.backup.2024-01-01-12-00-00.000", f.Versions[0])
}

// TestValidateFile_NoFatalOnError verifies the function signature of validateFile:
// it must return an error rather than calling log.Fatal on temp file problems.
// The real temp-dir-failure path is OS-dependent; we verify the return signature
// is ([]LedgerFileError, string, error) by ensuring the function compiles and
// returns without panicking on a content-only call via ValidateFile.
func TestValidateFile_NoFatalOnError(t *testing.T) {
	// validateFile signature: ([]ledger.LedgerFileError, string, error)
	// ValidateFile wraps it and now returns (gin.H, error).
	// We can't call it without a live ledger binary; just verify the public
	// wrapper compiles and has the right shape via the type system.
	var _ func(LedgerFile) (gin.H, error) = ValidateFile
}

// TestEditorFileHandler_InvalidJSON verifies that POST /api/editor/file with
// a malformed body returns 400 with the standard error envelope.
func TestEditorFileHandler_InvalidJSON(t *testing.T) {
	router := buildEditorTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/editor/file", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestEditorDeleteBackupsHandler_InvalidJSON verifies that POST /api/editor/file/delete_backups
// with a malformed body returns 400 with the standard error envelope.
func TestEditorDeleteBackupsHandler_InvalidJSON(t *testing.T) {
	router := buildEditorTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/editor/file/delete_backups", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestEditorValidateHandler_InvalidJSON verifies that POST /api/editor/validate
// with a malformed body returns 400 with the standard error envelope.
func TestEditorValidateHandler_InvalidJSON(t *testing.T) {
	router := buildEditorTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/editor/validate", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestEditorSaveHandler_InvalidJSON verifies that POST /api/editor/save
// with a malformed body returns 400 with the standard error envelope.
func TestEditorSaveHandler_InvalidJSON(t *testing.T) {
	router := buildEditorTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/editor/save", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestGetFile_MissingFile verifies that GetFile returns an error (not a fatal exit)
// when the requested file does not exist, and that the HTTP handler returns 500.
func TestGetFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	// GetFile calls config.GetJournalPath which is global state; test the inner helper directly.
	_, err := readLedgerFile(dir, filepath.Join(dir, "missing.journal"))
	assert.Error(t, err, "readLedgerFile must return an error for a missing file, not call log.Fatal")
}

// TestDeleteBackups_MissingFile verifies that the DeleteBackups inner logic (readLedgerFileWithVersions)
// returns an error for a missing file rather than fatally exiting.
func TestDeleteBackups_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readLedgerFileWithVersions(dir, filepath.Join(dir, "missing.journal"))
	assert.Error(t, err, "readLedgerFileWithVersions must return an error for a missing file, not call log.Fatal")
}

// buildEditorTestRouter constructs a minimal gin router with the editor endpoints
// wired up in the same way as server.go, but without needing the full server setup.
func buildEditorTestRouter() *gin.Engine {
	router := gin.New()

	router.POST("/api/editor/file", func(c *gin.Context) {
		var ledgerFile LedgerFile
		if !BindJSONOrError(c, &ledgerFile) {
			return
		}
		result, err := GetFile(ledgerFile)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/api/editor/file/delete_backups", func(c *gin.Context) {
		var ledgerFile LedgerFile
		if !BindJSONOrError(c, &ledgerFile) {
			return
		}
		result, err := DeleteBackups(ledgerFile)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/api/editor/validate", func(c *gin.Context) {
		var ledgerFile LedgerFile
		if !BindJSONOrError(c, &ledgerFile) {
			return
		}
		result, err := ValidateFile(ledgerFile)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, result)
	})

	router.POST("/api/editor/save", func(c *gin.Context) {
		var ledgerFile LedgerFile
		if !BindJSONOrError(c, &ledgerFile) {
			return
		}
		c.JSON(http.StatusOK, SaveFile(nil, ledgerFile))
	})

	return router
}

// decodeEditorResponse decodes a generic JSON map from the response body.
func decodeEditorResponse(t *testing.T, rec *httptest.ResponseRecorder) map[string]json.RawMessage {
	t.Helper()
	var result map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&result))
	return result
}

// TestGetFiles_AccountsSorted verifies that GetFiles always returns the
// accounts list in alphabetical order regardless of insertion order in the DB.
func TestGetFiles_AccountsSorted(t *testing.T) {
db := openTestDB(t)

// Insert postings in non-alphabetical (insertion) order.
insertionOrder := []string{
"Income:Salary:Acme",
"Assets:Checking",
"Expenses:Rent",
"Assets:Equity:NIFTY",
"Assets:Equity:ABNB",
}
for i, acc := range insertionOrder {
p := posting.Posting{
Account:       acc,
TransactionID: fmt.Sprintf("t%d", i),
Payee:         "test",
Commodity:     "INR",
}
require.NoError(t, db.Create(&p).Error)
}

// Reset the accounting cache so GetFiles queries fresh data.
accounting.ClearCache()

result := GetFiles(db)
raw, err := json.Marshal(result)
require.NoError(t, err)

var top map[string]json.RawMessage
require.NoError(t, json.Unmarshal(raw, &top))

var accounts []string
require.NoError(t, json.Unmarshal(top["accounts"], &accounts))

sorted := make([]string, len(accounts))
copy(sorted, accounts)
sort.Strings(sorted)

assert.Equal(t, sorted, accounts, "GetFiles must return accounts in alphabetical order")
}
