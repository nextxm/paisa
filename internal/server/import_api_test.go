package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/import_rule"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func buildImportTestRouter(db *gorm.DB) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/import/statement", HandleParseStatement(db))
	r.POST("/api/import/commit", HandleCommitImport(db))
	r.GET("/api/rules", HandleGetRules(db))
	r.POST("/api/rules", HandleSaveRule(db))
	r.DELETE("/api/rules/:id", HandleDeleteRule(db))
	return r
}

func TestImportRules_API(t *testing.T) {
	db := openTestDB(t)
	router := buildImportTestRouter(db)

	// 1. Create a rule
	rulePayload := `{
		"name": "Groceries Rule",
		"payee_pattern": "Whole Foods",
		"target_account": "Expenses:Groceries",
		"priority": 10
	}`
	req := httptest.NewRequest(http.MethodPost, "/api/rules", bytes.NewBufferString(rulePayload))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var saveResp struct {
		Rule  import_rule.Rule `json:"rule"`
		Saved bool             `json:"saved"`
	}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&saveResp))
	assert.True(t, saveResp.Saved)
	assert.NotZero(t, saveResp.Rule.ID)

	// 2. List rules
	reqList := httptest.NewRequest(http.MethodGet, "/api/rules", nil)
	recList := httptest.NewRecorder()
	router.ServeHTTP(recList, reqList)
	require.Equal(t, http.StatusOK, recList.Code)

	var listResp struct {
		Rules []import_rule.Rule `json:"rules"`
	}
	require.NoError(t, json.NewDecoder(recList.Body).Decode(&listResp))
	require.Len(t, listResp.Rules, 1)
	assert.Equal(t, "Groceries Rule", listResp.Rules[0].Name)

	// 3. Test Statement Parsing with the saved rule
	stmtPayload := `{
		"format": "csv",
		"content": "Date,Description,Debit,Credit\n2026-10-01,Whole Foods Market,45.20,\n"
	}`
	reqParse := httptest.NewRequest(http.MethodPost, "/api/import/statement", bytes.NewBufferString(stmtPayload))
	reqParse.Header.Set("Content-Type", "application/json")
	recParse := httptest.NewRecorder()
	router.ServeHTTP(recParse, reqParse)
	require.Equal(t, http.StatusOK, recParse.Code)

	var parseResp struct {
		TotalParsed  int `json:"total_parsed"`
		Transactions []struct {
			Payee           string `json:"payee"`
			SelectedAccount string `json:"selected_account"`
			MatchedRuleName string `json:"matched_rule_name"`
		} `json:"transactions"`
	}
	require.NoError(t, json.NewDecoder(recParse.Body).Decode(&parseResp))
	assert.Equal(t, 1, parseResp.TotalParsed)
	assert.Equal(t, "Expenses:Groceries", parseResp.Transactions[0].SelectedAccount)
	assert.Equal(t, "Groceries Rule", parseResp.Transactions[0].MatchedRuleName)
}
