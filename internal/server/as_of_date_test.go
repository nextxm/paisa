package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAsOfDate_InvalidFormatReturns400(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodGet, "/api/assets/balance?as_of_date=2024/02/01", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var envelope map[string]ErrorDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	assert.Equal(t, ErrCodeInvalidRequest, envelope["error"].Code)
}

func TestAsOfDate_FutureReturns400(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)
	router := Build(db, false)

	req := httptest.NewRequest(http.MethodGet, "/api/assets/balance?as_of_date=2024-03-21", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	require.Equal(t, http.StatusBadRequest, rec.Code)

	var envelope map[string]ErrorDetail
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &envelope))
	assert.Equal(t, ErrCodeInvalidRequest, envelope["error"].Code)
}

func TestAsOfDate_FiltersAssetsBalanceAndAccountBalance(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)
	router := Build(db, false)

	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromInt(1000),
		Quantity:      decimal.NewFromInt(1000),
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t2",
		Date:          time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromInt(500),
		Quantity:      decimal.NewFromInt(500),
	}).Error)

	assetsReq := httptest.NewRequest(http.MethodGet, "/api/assets/balance?flat=true&as_of_date=2024-02-01", nil)
	assetsRec := httptest.NewRecorder()
	router.ServeHTTP(assetsRec, assetsReq)
	require.Equal(t, http.StatusOK, assetsRec.Code)

	var assetsResp struct {
		AssetBreakdowns map[string]struct {
			MarketAmount decimal.Decimal `json:"marketAmount"`
		} `json:"asset_breakdowns"`
	}
	require.NoError(t, json.Unmarshal(assetsRec.Body.Bytes(), &assetsResp))
	require.Contains(t, assetsResp.AssetBreakdowns, "Assets:Checking")
	assert.True(t, decimal.NewFromInt(1000).Equal(assetsResp.AssetBreakdowns["Assets:Checking"].MarketAmount))

	account := url.PathEscape("Assets:Checking")
	accountReq := httptest.NewRequest(http.MethodGet, "/api/account/"+account+"/balance?as_of_date=2024-02-01", nil)
	accountRec := httptest.NewRecorder()
	router.ServeHTTP(accountRec, accountReq)
	require.Equal(t, http.StatusOK, accountRec.Code)

	var accountResp struct {
		AssetBreakdown struct {
			MarketAmount decimal.Decimal `json:"marketAmount"`
		} `json:"asset_breakdown"`
	}
	require.NoError(t, json.Unmarshal(accountRec.Body.Bytes(), &accountResp))
	assert.True(t, decimal.NewFromInt(1000).Equal(accountResp.AssetBreakdown.MarketAmount))
}
