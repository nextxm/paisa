package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
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

func TestGetTransactionsHandler_LimitFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "expenses:rent", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/transaction?limit=2", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	assert.Len(t, body.Transactions, 2, "expected 2 transactions with limit=2")
}

func TestGetTransactionsHandler_OffsetFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "expenses:rent", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)

	// Transactions are sorted newest-first: Rent (d3), Salary (d2), Groceries (d1).
	// offset=1 should skip Rent and return Salary and Groceries.
	req := httptest.NewRequest(http.MethodGet, "/api/transaction?offset=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	assert.Len(t, body.Transactions, 2, "expected 2 transactions with offset=1")
	assert.Equal(t, "Salary", body.Transactions[0].Payee)
}

func TestGetTransactionsHandler_LimitAndOffsetFilter(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d1 := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2024, 1, 17, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "expenses:food", Forecast: false},
		{TransactionID: "tx1", Date: d1, Payee: "Groceries", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "income:salary", Forecast: false},
		{TransactionID: "tx2", Date: d2, Payee: "Salary", Account: "assets:checking", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "expenses:rent", Forecast: false},
		{TransactionID: "tx3", Date: d3, Payee: "Rent", Account: "assets:checking", Forecast: false},
	})

	r := buildTransactionRouter(t, db)

	// newest-first: Rent, Salary, Groceries → offset=1&limit=1 should return just Salary
	req := httptest.NewRequest(http.MethodGet, "/api/transaction?offset=1&limit=1", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	require.Len(t, body.Transactions, 1, "expected 1 transaction with offset=1&limit=1")
	assert.Equal(t, "Salary", body.Transactions[0].Payee)
}

func TestParseTransactionFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/transaction?q=Amazon&amount_min=10.5&amount_max=200&account=expenses:shopping&commodity=INR&date_from=2024-01-01&date_to=2024-12-31",
		nil,
	)

	filters := parseTransactionFilters(c)

	require.NotNil(t, filters.AmountMin)
	require.NotNil(t, filters.AmountMax)
	require.NotNil(t, filters.DateFrom)
	require.NotNil(t, filters.DateTo)
	assert.Equal(t, "Amazon", filters.Query)
	assert.Equal(t, 10.5, *filters.AmountMin)
	assert.Equal(t, 200.0, *filters.AmountMax)
	assert.Equal(t, "expenses:shopping", filters.Account)
	assert.Equal(t, "INR", filters.Commodity)
	assert.Equal(t, "2024-01-01", filters.DateFrom.Format("2006-01-02"))
	assert.Equal(t, "2024-12-31", filters.DateTo.Format("2006-01-02"))
}

func TestParseTransactionFilters_InvalidValuesIgnored(t *testing.T) {
	gin.SetMode(gin.TestMode)
	rec := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rec)
	c.Request = httptest.NewRequest(
		http.MethodGet,
		"/api/transaction?amount_min=invalid&amount_max=oops&date_from=bad&date_to=nope",
		nil,
	)

	filters := parseTransactionFilters(c)

	assert.Nil(t, filters.AmountMin)
	assert.Nil(t, filters.AmountMax)
	assert.Nil(t, filters.DateFrom)
	assert.Nil(t, filters.DateTo)
}

func TestGetTransactionsHandler_CombinedFilters(t *testing.T) {
	db := openTestDB(t)
	gin.SetMode(gin.TestMode)

	d1 := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC)
	seedTransactions(t, db, []posting.Posting{
		{
			TransactionID: "tx1", Date: d1, Payee: "Amazon Purchase", Account: "expenses:shopping",
			Commodity: "INR", Amount: decimal.NewFromInt(-15000), TransactionNote: "electronics order", Forecast: false,
		},
		{
			TransactionID: "tx1", Date: d1, Payee: "Amazon Purchase", Account: "assets:checking",
			Commodity: "INR", Amount: decimal.NewFromInt(15000), TransactionNote: "electronics order", Forecast: false,
		},
		{
			TransactionID: "tx2", Date: d2, Payee: "Amazon Pantry", Account: "expenses:groceries",
			Commodity: "INR", Amount: decimal.NewFromInt(-9000), TransactionNote: "grocery order", Forecast: false,
		},
		{
			TransactionID: "tx2", Date: d2, Payee: "Amazon Pantry", Account: "assets:checking",
			Commodity: "INR", Amount: decimal.NewFromInt(9000), TransactionNote: "grocery order", Forecast: false,
		},
		{
			TransactionID: "tx3", Date: d1, Payee: "Amazon USD Purchase", Account: "expenses:shopping",
			Commodity: "USD", Amount: decimal.NewFromInt(-16000), TransactionNote: "electronics order", Forecast: false,
		},
		{
			TransactionID: "tx3", Date: d1, Payee: "Amazon USD Purchase", Account: "assets:checking",
			Commodity: "USD", Amount: decimal.NewFromInt(16000), TransactionNote: "electronics order", Forecast: false,
		},
	})

	r := buildTransactionRouter(t, db)
	req := httptest.NewRequest(
		http.MethodGet,
		"/api/transaction?q=amazon&amount_min=10000&amount_max=20000&account=expenses:shopping&commodity=INR&date_from=2024-01-01&date_to=2024-01-31",
		nil,
	)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	body := decodeTransactionResponse(t, rec)
	require.Len(t, body.Transactions, 1, "expected only one transaction matching all filters")
	assert.Equal(t, "Amazon Purchase", body.Transactions[0].Payee)
}
