package server

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type FireflyWebhookPayload struct {
	Trigger string `json:"trigger"`
	Content struct {
		Transactions []FireflyTransaction `json:"transactions"`
	} `json:"content"`
}

type FireflyTransaction struct {
	Date            string  `json:"date"`
	Amount          string  `json:"amount"`
	Description     string  `json:"description"`
	CurrencyCode    string  `json:"currency_code"`
	SourceName      string  `json:"source_name"`
	DestinationName string  `json:"destination_name"`
	Notes           *string `json:"notes"`
}

func mapFireflyTransaction(ft FireflyTransaction) AddTransactionRequest {
	parsedDate, err := time.Parse(time.RFC3339, ft.Date)
	var dateStr string
	if err != nil {
		log.Warnf("Failed to parse Firefly transaction date: %s", ft.Date)
		if len(ft.Date) >= 10 {
			dateStr = ft.Date[:10]
		} else {
			dateStr = ft.Date
		}
	} else {
		dateStr = parsedDate.Format("2006-01-02")
	}

	narration := ""
	if ft.Notes != nil {
		narration = *ft.Notes
	}

	return AddTransactionRequest{
		Date:        dateStr,
		Payee:       ft.Description,
		Narration:   narration,
		FromAccount: ft.SourceName,
		ToAccount:   ft.DestinationName,
		Amount:      ft.Amount,
		Commodity:   ft.CurrencyCode,
	}
}

func FireflyWebhookHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var payload FireflyWebhookPayload
		if !BindJSONOrError(c, &payload) {
			return
		}

		if payload.Trigger != "STORE_TRANSACTION" {
			log.Debugf("Ignoring Firefly webhook with trigger: %s", payload.Trigger)
			c.JSON(http.StatusOK, gin.H{"success": true, "ignored": true, "message": "Only STORE_TRANSACTION is supported"})
			return
		}

		var entries []string
		for _, ft := range payload.Content.Transactions {
			req := mapFireflyTransaction(ft)

			entryText, err := appendTransactionAndSync(db, req)
			if err != nil {
				log.Errorf("Failed to append Firefly transaction: %v", err)
				RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
				return
			}
			entries = append(entries, entryText)
		}

		c.JSON(http.StatusOK, gin.H{"success": true, "entries": entries})
	}
}
