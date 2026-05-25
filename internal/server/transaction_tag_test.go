package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func buildTransactionTagRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/transactions/:id/tags", func(c *gin.Context) {
		GetTransactionTagsHandler(db, c)
	})
	r.POST("/api/transactions/:id/tags", func(c *gin.Context) {
		AddTransactionTagHandler(db, c)
	})
	r.DELETE("/api/transactions/:id/tags/:tag", func(c *gin.Context) {
		DeleteTransactionTagHandler(db, c)
	})
	r.GET("/api/tags/autocomplete", func(c *gin.Context) {
		GetTagAutocompleteHandler(db, c)
	})
	return r
}

func TestTransactionTagHandlers_CRUDAndAutocomplete(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d, Payee: "Trip", Account: "expenses:travel", Forecast: false},
		{TransactionID: "tx1", Date: d, Payee: "Trip", Account: "assets:checking", Forecast: false},
	})
	r := buildTransactionTagRouter(t, db)

	req := httptest.NewRequest(http.MethodPost, "/api/transactions/tx1/tags", bytes.NewBufferString(`{"tag":"travel"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var addResp struct {
		Tags []string `json:"tags"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&addResp))
	assert.Equal(t, []string{"travel"}, addResp.Tags)

	req = httptest.NewRequest(http.MethodPost, "/api/transactions/tx1/tags", bytes.NewBufferString(`{"tag":"travel"}`))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&addResp))
	assert.Equal(t, []string{"travel"}, addResp.Tags)

	getReq := httptest.NewRequest(http.MethodGet, "/api/transactions/tx1/tags", nil)
	getRec := httptest.NewRecorder()
	r.ServeHTTP(getRec, getReq)
	require.Equal(t, http.StatusOK, getRec.Code)
	var getResp struct {
		Tags []string `json:"tags"`
	}
	require.NoError(t, json.NewDecoder(getRec.Body).Decode(&getResp))
	assert.Equal(t, []string{"travel"}, getResp.Tags)

	autoReq := httptest.NewRequest(http.MethodGet, "/api/tags/autocomplete?q=tr", nil)
	autoRec := httptest.NewRecorder()
	r.ServeHTTP(autoRec, autoReq)
	require.Equal(t, http.StatusOK, autoRec.Code)
	var autoResp struct {
		Tags []string `json:"tags"`
	}
	require.NoError(t, json.NewDecoder(autoRec.Body).Decode(&autoResp))
	assert.Equal(t, []string{"travel"}, autoResp.Tags)

	delReq := httptest.NewRequest(http.MethodDelete, "/api/transactions/tx1/tags/travel", nil)
	delRec := httptest.NewRecorder()
	r.ServeHTTP(delRec, delReq)
	require.Equal(t, http.StatusOK, delRec.Code)
	var delResp struct {
		Tags []string `json:"tags"`
	}
	require.NoError(t, json.NewDecoder(delRec.Body).Decode(&delResp))
	assert.Empty(t, delResp.Tags)
}
