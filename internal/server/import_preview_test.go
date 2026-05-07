package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImportPreviewHandler_InvalidJSON(t *testing.T) {
	router := buildImportPreviewTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/import/preview", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

func TestImportPreviewHandler_UnknownTemplate(t *testing.T) {
	router := buildImportPreviewTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/import/preview", strings.NewReader(`{
		"template":"does-not-exist",
		"content":"a,b,c\n1,2,3"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "not found")
}

func TestImportPreviewHandler_Success(t *testing.T) {
	router := buildImportPreviewTestRouter()

	req := httptest.NewRequest(http.MethodPost, "/api/import/preview", strings.NewReader(`{
		"template":"Mint",
		"content":"Date,Desc,Amount\n2026-01-01,Coffee,10\n2026-01-02,MissingAmount\n,,"
	}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Template string             `json:"template"`
		DryRun   bool               `json:"dry_run"`
		Rows     []ImportPreviewRow `json:"rows"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Rows, 4)

	assert.Equal(t, "Mint", resp.Template)
	assert.True(t, resp.DryRun)

	assert.True(t, resp.Rows[0].Valid)
	assert.Equal(t, "Date", resp.Rows[0].Row["A"])
	assert.Equal(t, "0", resp.Rows[0].Row["index"])

	assert.False(t, resp.Rows[2].Valid)
	assert.Contains(t, resp.Rows[2].Errors[0], "expected 3 columns, got 2")

	assert.False(t, resp.Rows[3].Valid)
	assert.Contains(t, resp.Rows[3].Errors, "row is empty")
}

func TestColumnName(t *testing.T) {
	assert.Equal(t, "A", columnName(0))
	assert.Equal(t, "Z", columnName(25))
	assert.Equal(t, "AA", columnName(26))
	assert.Equal(t, "AB", columnName(27))
}

func buildImportPreviewTestRouter() *gin.Engine {
	router := gin.New()
	router.POST("/api/import/preview", handleImportPreview)
	return router
}
