package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	modelroot "github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/parser"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const parserDateLayout = "2006-01-02"

type ParseTransactionRequest struct {
	Text string `json:"text" binding:"required"`
}

type CreateParsedTransactionRequest struct {
	Text string `json:"text" binding:"required"`

	Date         string `json:"date"`
	Payee        string `json:"payee"`
	Narration    string `json:"narration"`
	FromAccount  string `json:"from_account"`
	ToAccount    string `json:"to_account"`
	Amount       string `json:"amount"`
	Commodity    string `json:"commodity"`
	ToAmount     string `json:"to_amount"`
	ToCommodity  string `json:"to_commodity"`
	ExchangeRate string `json:"exchange_rate"`

	SuggestionUsed int `json:"suggestion_used"`
	TimeToConfirm  int `json:"time_to_confirm_ms"`
}

// ParseTransactionHandler handles parser preview requests.
func ParseTransactionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req ParseTransactionRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		parseResult, err := buildParser(db).ParseTransaction(req.Text)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		thresholds := parser.DefaultThresholds()
		c.JSON(http.StatusOK, gin.H{
			"result":                parseResult,
			"auto_create":           parseResult.Confidence.Overall >= thresholds.AutoCreate,
			"requires_confirmation": parseResult.Confidence.Overall < thresholds.AutoCreate,
		})
	}
}

// CreateParsedTransactionHandler parses user text, applies optional overrides,
// appends a transaction to add_journal_path, and triggers journal sync.
func CreateParsedTransactionHandler(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req CreateParsedTransactionRequest
		if !BindJSONOrError(c, &req) {
			return
		}

		parseResult, err := buildParser(db).ParseTransaction(req.Text)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		finalReq, err := buildFinalAddRequest(req, parseResult)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			return
		}

		entryText, errors, err := appendTransactionAndValidate(db, finalReq)
		if err != nil {
			if err.Error() == "add_journal_path is not configured" {
				RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
			} else {
				RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			}
			return
		}

		go logParserTrainingResult(db, req, parseResult, finalReq)

		c.JSON(http.StatusOK, gin.H{
			"success":            true,
			"entry":              entryText,
			"errors":             errors,
			"final_transaction":  finalReq,
			"parser_confidence":  parseResult.Confidence,
			"parser_suggestions": parseResult.Suggestions,
		})
	}
}

func buildParser(db *gorm.DB) *parser.Parser {
	cfg := config.GetConfig()
	keywords := parser.LoadKeywordsFromConfig(&cfg)
	accounts := accounting.AllAccounts(db)
	return parser.NewParser(keywords, db, accounts)
}

func buildFinalAddRequest(req CreateParsedTransactionRequest, parsed *parser.ParseResult) (AddTransactionRequest, error) {
	finalReq := AddTransactionRequest{
		Date:        parsed.Date.Format(parserDateLayout),
		Payee:       strings.TrimSpace(parsed.Payee),
		FromAccount: strings.TrimSpace(parsed.FromAccount),
		ToAccount:   strings.TrimSpace(parsed.ToAccount),
		Amount:      parsed.Amount.String(),
		Commodity:   strings.TrimSpace(parsed.Currency),
	}

	if finalReq.Commodity == "" {
		finalReq.Commodity = config.DefaultCurrency()
		if finalReq.Commodity == "" {
			finalReq.Commodity = "INR"
		}
	}

	if req.Date != "" {
		if _, err := time.Parse(parserDateLayout, req.Date); err != nil {
			return AddTransactionRequest{}, err
		}
		finalReq.Date = req.Date
	}

	if req.Payee != "" {
		finalReq.Payee = strings.TrimSpace(req.Payee)
	}
	if req.Narration != "" {
		finalReq.Narration = strings.TrimSpace(req.Narration)
	}
	if req.FromAccount != "" {
		finalReq.FromAccount = strings.TrimSpace(req.FromAccount)
	}
	if req.ToAccount != "" {
		finalReq.ToAccount = strings.TrimSpace(req.ToAccount)
	}
	if req.Amount != "" {
		if _, err := decimal.NewFromString(req.Amount); err != nil {
			return AddTransactionRequest{}, err
		}
		finalReq.Amount = req.Amount
	}
	if req.Commodity != "" {
		finalReq.Commodity = strings.TrimSpace(req.Commodity)
	}
	if req.ToAmount != "" {
		if _, err := decimal.NewFromString(req.ToAmount); err != nil {
			return AddTransactionRequest{}, err
		}
		finalReq.ToAmount = req.ToAmount
	}
	if req.ToCommodity != "" {
		finalReq.ToCommodity = strings.TrimSpace(req.ToCommodity)
	}
	if req.ExchangeRate != "" {
		if _, err := decimal.NewFromString(req.ExchangeRate); err != nil {
			return AddTransactionRequest{}, err
		}
		finalReq.ExchangeRate = req.ExchangeRate
	}

	if finalReq.FromAccount == "" || finalReq.ToAccount == "" {
		fallbackFrom, fallbackTo := defaultAccountsForDirection(parsed.Direction)
		if finalReq.FromAccount == "" {
			finalReq.FromAccount = fallbackFrom
		}
		if finalReq.ToAccount == "" {
			finalReq.ToAccount = fallbackTo
		}
	}

	return finalReq, nil
}

func defaultAccountsForDirection(direction string) (string, string) {
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "income":
		return "Income:Unknown", "Assets:Unknown"
	case "transfer":
		return "Assets:Unknown:From", "Assets:Unknown:To"
	default:
		return "Assets:Unknown", "Expenses:Unknown"
	}
}

func logParserTrainingResult(db *gorm.DB, req CreateParsedTransactionRequest, parsed *parser.ParseResult, actual AddTransactionRequest) {
	predictedDate := parsed.Date
	predictedAmount := parsed.Amount

	var actualDate *time.Time
	if t, err := time.Parse(parserDateLayout, actual.Date); err == nil {
		actualDate = &t
	}

	var actualAmount *decimal.Decimal
	if a, err := decimal.NewFromString(actual.Amount); err == nil {
		actualAmount = &a
	}

	userCorrected := false
	if actualDate != nil && actualDate.Format(parserDateLayout) != predictedDate.Format(parserDateLayout) {
		userCorrected = true
	}
	if actualAmount != nil && !actualAmount.Equal(predictedAmount) {
		userCorrected = true
	}
	if !strings.EqualFold(strings.TrimSpace(actual.Commodity), strings.TrimSpace(parsed.Currency)) {
		userCorrected = true
	}
	if !strings.EqualFold(strings.TrimSpace(actual.Payee), strings.TrimSpace(parsed.Payee)) {
		userCorrected = true
	}
	if !strings.EqualFold(strings.TrimSpace(actual.FromAccount), strings.TrimSpace(parsed.FromAccount)) {
		userCorrected = true
	}
	if !strings.EqualFold(strings.TrimSpace(actual.ToAccount), strings.TrimSpace(parsed.ToAccount)) {
		userCorrected = true
	}

	trainingLog := &modelroot.ParserTrainingLog{
		InputText:             req.Text,
		PredictedDate:         &predictedDate,
		PredictedAmount:       predictedAmount,
		PredictedCurrency:     parsed.Currency,
		PredictedPayee:        parsed.Payee,
		PredictedFromAccount:  parsed.FromAccount,
		PredictedToAccount:    parsed.ToAccount,
		PredictedDirection:    parsed.Direction,
		ConfidenceDate:        parsed.Confidence.Date,
		ConfidenceAmount:      parsed.Confidence.Amount,
		ConfidenceCurrency:    parsed.Confidence.Amount,
		ConfidencePayee:       parsed.Confidence.Payee,
		ConfidenceFromAccount: parsed.Confidence.FromAccount,
		ConfidenceToAccount:   parsed.Confidence.ToAccount,
		ConfidenceDirection:   parsed.Confidence.Direction,
		ConfidenceOverall:     parsed.Confidence.Overall,
		UserCorrected:         userCorrected,
		SuggestionsShown:      len(parsed.Suggestions),
		SuggestionUsed:        req.SuggestionUsed,
		TimeToConfirm:         req.TimeToConfirm,
	}

	if userCorrected {
		trainingLog.ActualDate = actualDate
		trainingLog.ActualAmount = actualAmount
		trainingLog.ActualCurrency = stringPtr(actual.Commodity)
		trainingLog.ActualPayee = stringPtr(actual.Payee)
		trainingLog.ActualFromAccount = stringPtr(actual.FromAccount)
		trainingLog.ActualToAccount = stringPtr(actual.ToAccount)
		trainingLog.ActualDirection = stringPtr(parsed.Direction)
	}

	if err := modelroot.CreateParserTrainingLog(db, trainingLog); err != nil {
		log.WithError(err).Warn("failed to store parser training log")
	}
}

func stringPtr(value string) *string {
	v := strings.TrimSpace(value)
	if v == "" {
		return nil
	}
	return &v
}
