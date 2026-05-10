package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/parser"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTransactionHandler_ReturnsParsedResult(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)

	body := ParseTransactionRequest{Text: "20 Apr spent $15 at grocery store"}
	payload, err := json.Marshal(body)
	require.NoError(t, err)

	recorder := httptest.NewRecorder()
	ctx, engine := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/parser/parse", bytes.NewReader(payload))
	ctx.Request.Header.Set("Content-Type", "application/json")

	engine.POST("/api/parser/parse", ParseTransactionHandler(db))
	engine.ServeHTTP(recorder, ctx.Request)

	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "result")
	assert.Contains(t, recorder.Body.String(), "auto_create")
}

func TestBuildFinalAddRequest_UsesDirectionFallbackAccounts(t *testing.T) {
	parsed := &parser.ParseResult{
		Date:      time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		Currency:  "USD",
		Direction: "income",
	}

	finalReq, err := buildFinalAddRequest(CreateParsedTransactionRequest{Text: "received $100"}, parsed)
	require.NoError(t, err)
	assert.Equal(t, "Income:Unknown", finalReq.FromAccount)
	assert.Equal(t, "Assets:Unknown", finalReq.ToAccount)
	assert.Equal(t, "2026-05-10", finalReq.Date)
}

func TestBuildFinalAddRequest_AppliesOverrides(t *testing.T) {
	parsed := &parser.ParseResult{
		Date:        time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC),
		Amount:      mustDecimal("15"),
		Currency:    "USD",
		Payee:       "Store",
		FromAccount: "Assets:Checking",
		ToAccount:   "Expenses:Groceries",
		Direction:   "expense",
	}

	req := CreateParsedTransactionRequest{
		Text:        "spent $15",
		Date:        "2026-05-09",
		Amount:      "17.5",
		Commodity:   "EUR",
		FromAccount: "Assets:Savings",
		ToAccount:   "Expenses:Dining",
	}

	finalReq, err := buildFinalAddRequest(req, parsed)
	require.NoError(t, err)
	assert.Equal(t, "2026-05-09", finalReq.Date)
	assert.Equal(t, "17.5", finalReq.Amount)
	assert.Equal(t, "EUR", finalReq.Commodity)
	assert.Equal(t, "Assets:Savings", finalReq.FromAccount)
	assert.Equal(t, "Expenses:Dining", finalReq.ToAccount)
}

func mustDecimal(v string) decimal.Decimal {
	d, err := decimal.NewFromString(v)
	if err != nil {
		panic(err)
	}
	return d
}
