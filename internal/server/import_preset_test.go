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
	"gorm.io/gorm"
)

func TestImportPreset_ListIncludesBuiltins(t *testing.T) {
	db := openTestDB(t)
	router := buildImportPresetRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/import/presets", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)
	var resp struct {
		Presets []struct {
			Name       string `json:"name"`
			PresetType string `json:"preset_type"`
		} `json:"presets"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.NotEmpty(t, resp.Presets)
	assert.Contains(t, resp.Presets, struct {
		Name       string `json:"name"`
		PresetType string `json:"preset_type"`
	}{Name: "Generic Bank CSV", PresetType: "builtin"})
}

func TestImportPreset_UpsertAndDelete(t *testing.T) {
	db := openTestDB(t)
	router := buildImportPresetRouter(db)

	upsertBody := `{
		"name":"My Preset",
		"column_mappings":{"date":"A","amount":"B"},
		"date_format":"YYYY-MM-DD",
		"default_accounts":{"asset":"Assets:Checking"},
		"delimiter":","
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/import/presets", strings.NewReader(upsertBody))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/import/presets", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	assert.Contains(t, listRec.Body.String(), `"name":"My Preset"`)
	assert.Contains(t, listRec.Body.String(), `"preset_type":"custom"`)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/import/presets", strings.NewReader(`{"name":"My Preset"}`))
	delReq.Header.Set("Content-Type", "application/json")
	delRec := httptest.NewRecorder()
	router.ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusOK, delRec.Code)
}

func TestImportPreset_InvalidJSON(t *testing.T) {
	db := openTestDB(t)
	router := buildImportPresetRouter(db)

	req := httptest.NewRequest(http.MethodPost, "/api/import/presets", strings.NewReader(`{not json}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

func buildImportPresetRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.GET("/api/import/presets", handleGetImportPresets(db))
	router.POST("/api/import/presets", handleUpsertImportPreset(db))
	router.DELETE("/api/import/presets", handleDeleteImportPreset(db))
	return router
}
