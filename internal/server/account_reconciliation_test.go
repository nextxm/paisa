package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAccountReconciliation_GetDefault(t *testing.T) {
	db := openTestDB(t)
	router := buildAccountReconciliationRouter(db)

	req := httptest.NewRequest(http.MethodGet, "/api/accounts/Assets:Checking/reconciliation", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var response map[string]any
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&response))
	assert.Equal(t, "Assets:Checking", response["account"])
	assert.Equal(t, float64(30), response["frequency_days"])
	assert.Nil(t, response["last_reconciled"])
	assert.Nil(t, response["days_since"])
	assert.Equal(t, true, response["is_overdue"])
}

func TestAccountReconciliation_PatchAndGet(t *testing.T) {
	db := openTestDB(t)
	router := buildAccountReconciliationRouter(db)

	utils.SetNow("2026-05-04")
	defer utils.UnsetNow()

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/accounts/Assets:Checking/reconciliation",
		strings.NewReader(`{"mark_reconciled_now":true,"frequency_days":7}`),
	)
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	require.Equal(t, http.StatusOK, patchRec.Code)

	getReq := httptest.NewRequest(http.MethodGet, "/api/accounts/Assets:Checking/reconciliation", nil)
	getRec := httptest.NewRecorder()
	router.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)

	var response map[string]any
	require.NoError(t, json.NewDecoder(getRec.Body).Decode(&response))
	assert.Equal(t, "2026-05-04", response["last_reconciled"])
	assert.Equal(t, float64(7), response["frequency_days"])
	assert.Equal(t, float64(0), response["days_since"])
	assert.Equal(t, false, response["is_overdue"])
}

func TestAccountReconciliation_InvalidFrequency(t *testing.T) {
	db := openTestDB(t)
	router := buildAccountReconciliationRouter(db)

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/accounts/Assets:Checking/reconciliation",
		strings.NewReader(`{"frequency_days":0}`),
	)
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	require.Equal(t, http.StatusBadRequest, patchRec.Code)

	detail := decodeErrorEnvelope(t, patchRec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

func TestAccountReconciliation_List(t *testing.T) {
	db := openTestDB(t)
	router := buildAccountReconciliationRouter(db)

	patchReq := httptest.NewRequest(
		http.MethodPatch,
		"/api/accounts/Assets:Checking/reconciliation",
		strings.NewReader(`{"last_reconciled":"2026-05-01","frequency_days":30}`),
	)
	patchReq.Header.Set("Content-Type", "application/json")
	patchRec := httptest.NewRecorder()
	router.ServeHTTP(patchRec, patchReq)
	require.Equal(t, http.StatusOK, patchRec.Code)

	listReq := httptest.NewRequest(http.MethodGet, "/api/accounts/reconciliation", nil)
	listRec := httptest.NewRecorder()
	router.ServeHTTP(listRec, listReq)
	require.Equal(t, http.StatusOK, listRec.Code)
	assert.Contains(t, listRec.Body.String(), `"account":"Assets:Checking"`)
}

func buildAccountReconciliationRouter(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.GET("/api/accounts/reconciliation", func(c *gin.Context) {
		GetAllAccountReconciliations(db, c)
	})
	router.GET("/api/accounts/:account/reconciliation", func(c *gin.Context) {
		GetAccountReconciliation(db, c)
	})
	router.PATCH("/api/accounts/:account/reconciliation", func(c *gin.Context) {
		PatchAccountReconciliation(db, c)
	})
	return router
}
