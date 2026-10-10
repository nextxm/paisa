package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
)

func TestComputeAllocationTarget_DriftAndStatus(t *testing.T) {
	db := openTestDB(t)
	now := time.Now()

	p1 := posting.Posting{
		Account:      "Assets:Equity:Stock",
		Amount:       decimal.NewFromInt(6000),
		Quantity:     decimal.NewFromInt(6000),
		MarketAmount: decimal.NewFromInt(6000),
		Commodity:    "INR",
		Date:         now,
	}
	p2 := posting.Posting{
		Account:      "Assets:Debt:FixedDeposit",
		Amount:       decimal.NewFromInt(4000),
		Quantity:     decimal.NewFromInt(4000),
		MarketAmount: decimal.NewFromInt(4000),
		Commodity:    "INR",
		Date:         now,
	}
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	postings := []posting.Posting{p1, p2}
	total := decimal.NewFromInt(10000)

	equityCfg := config.AllocationTarget{
		Name:     "Equity",
		Target:   50,
		Drift:    5,
		Accounts: []string{"Assets:Equity:*"},
	}
	debtCfg := config.AllocationTarget{
		Name:     "Debt",
		Target:   50,
		Drift:    5,
		Accounts: []string{"Assets:Debt:*"},
	}

	equityTarget := computeAllocationTarget(db, postings, equityCfg, total)
	debtTarget := computeAllocationTarget(db, postings, debtCfg, total)

	assert.Equal(t, "Equity", equityTarget.Name)
	assert.True(t, equityTarget.Current.Equal(decimal.NewFromInt(60)))
	assert.True(t, equityTarget.Drift.Equal(decimal.NewFromInt(10)))
	assert.Equal(t, "overweight", equityTarget.Status)

	assert.Equal(t, "Debt", debtTarget.Name)
	assert.True(t, debtTarget.Current.Equal(decimal.NewFromInt(40)))
	assert.True(t, debtTarget.Drift.Equal(decimal.NewFromInt(-10)))
	assert.Equal(t, "underweight", debtTarget.Status)
}

func TestComputeRebalancePlan_FullRebalance(t *testing.T) {
	targets := []AllocationTarget{
		{
			Name:         "Equity",
			Target:       decimal.NewFromInt(50),
			Current:      decimal.NewFromInt(60),
			MarketAmount: decimal.NewFromInt(6000),
		},
		{
			Name:         "Debt",
			Target:       decimal.NewFromInt(50),
			Current:      decimal.NewFromInt(40),
			MarketAmount: decimal.NewFromInt(4000),
		},
	}

	plan := ComputeRebalancePlan(targets, decimal.Zero, "full_rebalance")

	assert.Equal(t, "full_rebalance", plan.Mode)
	assert.True(t, plan.TotalBefore.Equal(decimal.NewFromInt(10000)))
	assert.True(t, plan.TotalAfter.Equal(decimal.NewFromInt(10000)))
	require.Len(t, plan.Items, 2)

	// Equity should sell 1000
	assert.Equal(t, "Equity", plan.Items[0].Name)
	assert.Equal(t, "SELL", plan.Items[0].Action)
	assert.True(t, plan.Items[0].Amount.Equal(decimal.NewFromInt(1000)))
	assert.True(t, plan.Items[0].TargetAmount.Equal(decimal.NewFromInt(5000)))

	// Debt should buy 1000
	assert.Equal(t, "Debt", plan.Items[1].Name)
	assert.Equal(t, "BUY", plan.Items[1].Action)
	assert.True(t, plan.Items[1].Amount.Equal(decimal.NewFromInt(1000)))
	assert.True(t, plan.Items[1].TargetAmount.Equal(decimal.NewFromInt(5000)))
}

func TestComputeRebalancePlan_CashInjection(t *testing.T) {
	targets := []AllocationTarget{
		{
			Name:         "Equity",
			Target:       decimal.NewFromInt(50),
			Current:      decimal.NewFromInt(60),
			MarketAmount: decimal.NewFromInt(6000),
		},
		{
			Name:         "Debt",
			Target:       decimal.NewFromInt(50),
			Current:      decimal.NewFromInt(40),
			MarketAmount: decimal.NewFromInt(4000),
		},
	}

	// Inject 2000 fresh cash. Total after: 12000.
	// Ideal Equity: 6000 (deficit: 0). Ideal Debt: 6000 (deficit: 2000).
	// All 2000 should go to Debt.
	plan := ComputeRebalancePlan(targets, decimal.NewFromInt(2000), "cash_injection")

	assert.Equal(t, "cash_injection", plan.Mode)
	assert.True(t, plan.TotalBefore.Equal(decimal.NewFromInt(10000)))
	assert.True(t, plan.TotalAfter.Equal(decimal.NewFromInt(12000)))
	require.Len(t, plan.Items, 2)

	assert.Equal(t, "Equity", plan.Items[0].Name)
	assert.Equal(t, "HOLD", plan.Items[0].Action)
	assert.True(t, plan.Items[0].Amount.Equal(decimal.Zero))
	assert.True(t, plan.Items[0].NewPercent.Equal(decimal.NewFromInt(50)))

	assert.Equal(t, "Debt", plan.Items[1].Name)
	assert.Equal(t, "BUY", plan.Items[1].Action)
	assert.True(t, plan.Items[1].Amount.Equal(decimal.NewFromInt(2000)))
	assert.True(t, plan.Items[1].NewPercent.Equal(decimal.NewFromInt(50)))
}

func TestRebalanceHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := openTestDB(t)

	router := gin.New()
	router.POST("/api/allocation/rebalance", RebalanceHandler(db))

	body := RebalanceRequest{
		Cash: 1000,
		Mode: "cash_injection",
	}
	payload, _ := json.Marshal(body)

	req, _ := http.NewRequest(http.MethodPost, "/api/allocation/rebalance", bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
