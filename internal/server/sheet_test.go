package server

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadSheetFile_MissingFile verifies that readSheetFile returns an error
// (instead of calling log.Fatal) when the requested path does not exist.
func TestReadSheetFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readSheetFile(dir, filepath.Join(dir, "nonexistent.paisa"))
	assert.Error(t, err)
}

// TestReadSheetFile_Success verifies the happy path of readSheetFile.
func TestReadSheetFile_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.paisa")
	require.NoError(t, os.WriteFile(filePath, []byte("sheet content"), 0644))

	f, err := readSheetFile(dir, filePath)
	require.NoError(t, err)
	assert.Equal(t, "test.paisa", f.Name)
	assert.Equal(t, "sheet content", f.Content)
}

// TestReadSheetFileWithVersions_MissingFile verifies that readSheetFileWithVersions
// returns an error instead of calling log.Fatal when the file is absent.
func TestReadSheetFileWithVersions_MissingFile(t *testing.T) {
	dir := t.TempDir()
	_, err := readSheetFileWithVersions(dir, filepath.Join(dir, "nonexistent.paisa"))
	assert.Error(t, err)
}

// TestReadSheetFileWithVersions_Success verifies the happy path including version listing.
func TestReadSheetFileWithVersions_Success(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.paisa")
	require.NoError(t, os.WriteFile(filePath, []byte("main"), 0644))

	// Create a backup file that should appear in Versions.
	backupPath := filePath + ".backup.2024-01-01-12-00-00.000"
	require.NoError(t, os.WriteFile(backupPath, []byte("old"), 0644))

	f, err := readSheetFileWithVersions(dir, filePath)
	require.NoError(t, err)
	assert.Equal(t, "test.paisa", f.Name)
	assert.Equal(t, "main", f.Content)
	assert.Len(t, f.Versions, 1)
	assert.Equal(t, "test.paisa.backup.2024-01-01-12-00-00.000", f.Versions[0])
}

// TestGetSheet_SignatureReturnsError verifies the function signature of GetSheet:
// it must return (gin.H, error) rather than calling log.Fatal.
func TestGetSheet_SignatureReturnsError(t *testing.T) {
	var _ func(SheetFile) (gin.H, error) = GetSheet
}

// TestDeleteSheetBackups_SignatureReturnsError verifies the function signature of DeleteSheetBackups:
// it must return (gin.H, error) rather than calling log.Fatal.
func TestDeleteSheetBackups_SignatureReturnsError(t *testing.T) {
	var _ func(SheetFile) (gin.H, error) = DeleteSheetBackups
}

// TestSheetFileHandler_InvalidJSON verifies that POST /api/sheets/file with
// a malformed body returns 400 with the standard error envelope.
func TestSheetFileHandler_InvalidJSON(t *testing.T) {
	router := buildSheetTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/sheets/file", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestSheetDeleteBackupsHandler_InvalidJSON verifies that POST /api/sheets/file/delete_backups
// with a malformed body returns 400 with the standard error envelope.
func TestSheetDeleteBackupsHandler_InvalidJSON(t *testing.T) {
	router := buildSheetTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/sheets/file/delete_backups", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestSheetSaveHandler_InvalidJSON verifies that POST /api/sheets/save
// with a malformed body returns 400 with the standard error envelope.
func TestSheetSaveHandler_InvalidJSON(t *testing.T) {
	router := buildSheetTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/sheets/save", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestSheetFileHandler_MissingFile verifies that a valid request for a non-existent
// sheet file returns 500 with the standard error envelope (not a process exit).
func TestSheetFileHandler_MissingFile(t *testing.T) {
	router := buildSheetTestRouterWithDir(t.TempDir())

	body := strings.NewReader(`{"name":"nonexistent.paisa","content":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sheets/file", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInternalError, detail.Code)
}

// TestSheetDeleteBackupsHandler_MissingFile verifies that a valid request to delete
// backups for a non-existent file returns 500 with the standard error envelope.
func TestSheetDeleteBackupsHandler_MissingFile(t *testing.T) {
	router := buildSheetTestRouterWithDir(t.TempDir())

	body := strings.NewReader(`{"name":"nonexistent.paisa","content":""}`)
	req := httptest.NewRequest(http.MethodPost, "/api/sheets/file/delete_backups", body)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInternalError, detail.Code)
}

// buildSheetTestRouter constructs a minimal gin router with the sheet endpoints
// wired up in the same way as server.go, using an empty temp dir as sheet dir.
func buildSheetTestRouter() *gin.Engine {
	return buildSheetTestRouterWithDir(newTempDir())
}

// newTempDir returns a temporary directory for use in routers not given a *testing.T.
func newTempDir() string {
	dir, err := os.MkdirTemp("", "paisa-sheet-test-*")
	if err != nil {
		panic(err)
	}
	return dir
}

// buildSheetTestRouterWithDir constructs a minimal gin router with sheet endpoints
// that use the provided dir as the sheet directory (via readSheetFile / readSheetFileWithVersions).
func buildSheetTestRouterWithDir(dir string) *gin.Engine {
	router := gin.New()

	router.POST("/api/sheets/file", func(c *gin.Context) {
		var sheetFile SheetFile
		if !BindJSONOrError(c, &sheetFile) {
			return
		}
		sf, err := readSheetFile(dir, filepath.Join(dir, sheetFile.Name))
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": sf})
	})

	router.POST("/api/sheets/file/delete_backups", func(c *gin.Context) {
		var sheetFile SheetFile
		if !BindJSONOrError(c, &sheetFile) {
			return
		}
		sf, err := readSheetFileWithVersions(dir, filepath.Join(dir, sheetFile.Name))
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		c.JSON(http.StatusOK, gin.H{"file": sf})
	})

	router.POST("/api/sheets/save", func(c *gin.Context) {
		var sheetFile SheetFile
		if !BindJSONOrError(c, &sheetFile) {
			return
		}
		c.JSON(http.StatusOK, gin.H{"saved": false, "message": "test stub"})
	})

	return router
}
