package server

import (
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

type transactionListResponse struct {
	Transactions []struct {
		ID       string `json:"id"`
		Payee    string `json:"payee"`
		Postings []struct {
			Account string `json:"account"`
		} `json:"postings"`
	} `json:"transactions"`
}

func buildTransactionRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/transaction", func(c *gin.Context) {
		GetTransactionsHandler(db, c)
	})
	return r
}

func seedTransactions(t *testing.T, db *gorm.DB, postings []posting.Posting) {
	t.Helper()
	for i := range postings {
		require.NoError(t, db.Create(&postings[i]).Error)
	}
}

func decodeTransactionResponse(t *testing.T, rec *httptest.ResponseRecorder) transactionListResponse {
	t.Helper()
	var body transactionListResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	return body
}

func TestGetTransactionsHandler_NoFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)
	req := httptest.NewRequest(http.MethodGet, "/api/transaction", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	assert.Len(t, body.Transactions, 2, "expected 2 transactions without filter")
}

func TestGetTransactionsHandler_AccountFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx3", Date: d, Payee: "Investment", Account: "assets:savings", Forecast: false},
		{TransactionID: "tx3", Date: d, Payee: "Investment", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)

	// Filter to "assets:savings" — should only get tx3
	req := httptest.NewRequest(http.MethodGet, "/api/transaction?account=assets:savings", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	assert.Len(t, body.Transactions, 1, "expected 1 transaction for assets:savings")
	assert.Equal(t, "Investment", body.Transactions[0].Payee)
}

func TestGetTransactionsHandler_AccountPrefixFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx1", Date: d, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx2", Date: d, Payee: "Groceries", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)

	// Filter to "income" prefix — should only get tx1
	req := httptest.NewRequest(http.MethodGet, "/api/transaction?account=income", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	assert.Len(t, body.Transactions, 1, "expected 1 transaction for income prefix")
	assert.Equal(t, "Salary", body.Transactions[0].Payee)
}
