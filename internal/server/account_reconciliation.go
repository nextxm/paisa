package server

import (
	"net/http"
	"slices"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/account_reconciliation"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type accountReconciliationRequest struct {
	LastReconciled    *string `json:"last_reconciled"`
	FrequencyDays     *int    `json:"frequency_days"`
	MarkReconciledNow bool    `json:"mark_reconciled_now"`
}

type accountReconciliationResponse struct {
	Account        string  `json:"account"`
	LastReconciled *string `json:"last_reconciled"`
	FrequencyDays  int     `json:"frequency_days"`
	DaysSince      *int    `json:"days_since"`
	IsOverdue      bool    `json:"is_overdue"`
}

func normalizeDate(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
}

func toAccountReconciliationResponse(account string, reconciliation *account_reconciliation.AccountReconciliation) accountReconciliationResponse {
	frequencyDays := account_reconciliation.DefaultFrequencyDays
	if reconciliation != nil && reconciliation.FrequencyDays > 0 {
		frequencyDays = reconciliation.FrequencyDays
	}

	response := accountReconciliationResponse{
		Account:       account,
		FrequencyDays: frequencyDays,
		IsOverdue:     true,
	}

	if reconciliation == nil || reconciliation.LastReconciledDate == nil {
		return response
	}

	lastReconciledDate := normalizeDate(*reconciliation.LastReconciledDate)
	now := normalizeDate(utils.Now())
	daysSince := int(now.Sub(lastReconciledDate) / (24 * time.Hour))
	if daysSince < 0 {
		daysSince = 0
	}
	lastReconciled := lastReconciledDate.Format("2006-01-02")
	response.LastReconciled = &lastReconciled
	response.DaysSince = &daysSince
	response.IsOverdue = daysSince > frequencyDays
	return response
}

func GetAccountReconciliation(db *gorm.DB, c *gin.Context) {
	account := c.Param("account")
	reconciliation, err := account_reconciliation.Get(db, account)
	if err != nil && err != gorm.ErrRecordNotFound {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}
	if err == gorm.ErrRecordNotFound {
		reconciliation = nil
	}

	c.JSON(http.StatusOK, toAccountReconciliationResponse(account, reconciliation))
}

func PatchAccountReconciliation(db *gorm.DB, c *gin.Context) {
	account := c.Param("account")

	var request accountReconciliationRequest
	if !BindJSONOrError(c, &request) {
		return
	}

	reconciliation, err := account_reconciliation.Get(db, account)
	if err != nil && err != gorm.ErrRecordNotFound {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	lastReconciledDate := (*time.Time)(nil)
	frequencyDays := account_reconciliation.DefaultFrequencyDays
	if err == nil {
		lastReconciledDate = reconciliation.LastReconciledDate
		if reconciliation.FrequencyDays > 0 {
			frequencyDays = reconciliation.FrequencyDays
		}
	}

	if request.FrequencyDays != nil {
		if *request.FrequencyDays <= 0 {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "frequency_days must be greater than 0")
			return
		}
		frequencyDays = *request.FrequencyDays
	}

	if request.LastReconciled != nil {
		parsedDate, parseError := time.Parse("2006-01-02", *request.LastReconciled)
		if parseError != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, "last_reconciled must be YYYY-MM-DD")
			return
		}
		normalized := normalizeDate(parsedDate)
		lastReconciledDate = &normalized
	}

	if request.MarkReconciledNow {
		now := normalizeDate(utils.Now())
		lastReconciledDate = &now
	}

	updated, err := account_reconciliation.Upsert(db, account, lastReconciledDate, frequencyDays)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, toAccountReconciliationResponse(account, updated))
}

func GetAllAccountReconciliations(db *gorm.DB, c *gin.Context) {
	reconciliations, err := account_reconciliation.GetAll(db)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	reconciliationsByAccount := map[string]account_reconciliation.AccountReconciliation{}
	for _, reconciliation := range reconciliations {
		reconciliationsByAccount[reconciliation.Account] = reconciliation
	}

	accounts := accounting.AllAccounts(db)
	allAccounts := append([]string{}, accounts...)
	for _, reconciliation := range reconciliations {
		if !slices.Contains(allAccounts, reconciliation.Account) {
			allAccounts = append(allAccounts, reconciliation.Account)
		}
	}
	slices.Sort(allAccounts)

	response := make([]accountReconciliationResponse, 0, len(allAccounts))
	for _, account := range allAccounts {
		reconciliation, ok := reconciliationsByAccount[account]
		if !ok {
			response = append(response, toAccountReconciliationResponse(account, nil))
			continue
		}
		item := reconciliation
		response = append(response, toAccountReconciliationResponse(account, &item))
	}

	c.JSON(http.StatusOK, gin.H{"reconciliations": response})
}
