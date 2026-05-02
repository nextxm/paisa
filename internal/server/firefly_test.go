package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
)

func TestFireflyWebhookHandler_Ignore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/webhooks/firefly", FireflyWebhookHandler(&gorm.DB{}))

	payload := []byte(`{
		"trigger": "UPDATE_TRANSACTION",
		"content": {
			"transactions": []
		}
	}`)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/webhooks/firefly", bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), `"ignored":true`)
}

func TestMapFireflyTransaction(t *testing.T) {
	notes := "May Rent"
	ft := FireflyTransaction{
		Date:            "2026-05-01T12:52:00-04:00",
		Amount:          "2500.00",
		Description:     "Rent",
		CurrencyCode:    "CAD",
		SourceName:      "Assets:Bank",
		DestinationName: "Expenses:Housing",
		Notes:           &notes,
	}

	req := mapFireflyTransaction(ft)

	assert.Equal(t, "2026-05-01", req.Date)
	assert.Equal(t, "Rent", req.Payee)
	assert.Equal(t, "May Rent", req.Narration)
	assert.Equal(t, "Assets:Bank", req.FromAccount)
	assert.Equal(t, "Expenses:Housing", req.ToAccount)
	assert.Equal(t, "2500.00", req.Amount)
	assert.Equal(t, "CAD", req.Commodity)
}
