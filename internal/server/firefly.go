package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/server/assets"
	"github.com/ananthakumaran/paisa/internal/service/firefly"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
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

type ReconcileItem struct {
	FireflyAccount string          `json:"firefly_account"`
	PaisaAccount   string          `json:"paisa_account"`
	FireflyBalance decimal.Decimal `json:"firefly_balance"`
	PaisaBalance   decimal.Decimal `json:"paisa_balance"`
	Currency       string          `json:"currency"`
	Diff           decimal.Decimal `json:"diff"`
	Ignored        bool            `json:"ignored"`
}

func FireflyReconcileHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		conf := config.GetConfig()
		if !conf.Labs.FireflyReconcile {
			RespondError(c, http.StatusForbidden, ErrCodeForbidden, "Firefly reconciliation is not enabled in Labs")
			return
		}

		fireflyAccounts, err := firefly.GetAccounts(conf.Firefly.URL, conf.Firefly.Token)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}

		// Fetch Paisa balances (flat list)
		paisaBalancesH := assets.GetBalanceByMode(db, "", true)
		paisaBreakdowns, ok := paisaBalancesH["asset_breakdowns"].(map[string]assets.AssetBreakdown)
		if !ok {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, "Failed to fetch Paisa balances")
			return
		}

		var items []ReconcileItem
		ignoredSet := lo.SliceToMap(conf.Firefly.IgnoreAccounts, func(a string) (string, bool) { return a, true })

		for _, fa := range fireflyAccounts {
			if !fa.Attributes.Active || (fa.Attributes.Type != "asset" && fa.Attributes.Type != "liabilities") {
				continue
			}

			item := ReconcileItem{
				FireflyAccount: fa.Attributes.Name,
				Currency:       fa.Attributes.CurrencyCode,
				Ignored:        ignoredSet[fa.Attributes.Name],
			}

			fb, err := decimal.NewFromString(fa.Attributes.CurrentBalance)
			if err == nil {
				item.FireflyBalance = fb
			}

			// Try to find a matching Paisa account
			// Matching is case-insensitive and checks if Paisa account name contains Firefly account name or vice versa
			for name, pb := range paisaBreakdowns {
				paisaAccountName := name
				if strings.Contains(strings.ToLower(paisaAccountName), strings.ToLower(fa.Attributes.Name)) ||
					strings.Contains(strings.ToLower(fa.Attributes.Name), strings.ToLower(paisaAccountName)) {
					item.PaisaAccount = paisaAccountName
					for _, ob := range pb.OriginalBalances {
						if ob.Currency == fa.Attributes.CurrencyCode {
							item.PaisaBalance = ob.Amount
							break
						}
					}
					break
				}
			}

			item.Diff = item.FireflyBalance.Sub(item.PaisaBalance)
			items = append(items, item)
		}

		c.JSON(http.StatusOK, gin.H{"items": items})
	}
}
