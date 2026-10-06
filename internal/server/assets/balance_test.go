package assets

import (
	"fmt"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// openTestDB opens an in-memory SQLite database and runs migrations.
func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

// loadTestConfig sets up a minimal config and restores the original on cleanup.
func loadTestConfig(t *testing.T) {
	t.Helper()
	orig := config.GetConfig()
	yaml := "journal_path: main.ledger\ndb_path: paisa.db\ntime_zone: UTC\nchecking_accounts:\n  - Assets:Checking"
	require.NoError(t, config.LoadConfig([]byte(yaml), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

// ---------------------------------------------------------------------------
// GetCheckingBalance integration tests
// ---------------------------------------------------------------------------

// TestGetCheckingBalance_EmptyDB verifies that an empty database returns an
// empty asset_breakdowns map.
func TestGetCheckingBalance_EmptyDB(t *testing.T) {
	loadTestConfig(t)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)

	result := GetCheckingBalance(db, "")
	breakdowns, ok := result["asset_breakdowns"].(map[string]AssetBreakdown)
	require.True(t, ok)
	assert.Empty(t, breakdowns)
}

// TestGetCheckingBalance_SingleAccount verifies that multiple checking account
// postings are aggregated into a single breakdown entry with the correct net
// balance.
func TestGetCheckingBalance_SingleAccount(t *testing.T) {
	loadTestConfig(t)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)

	// deposit 1000, deposit 500, withdraw 200 → net 1300
	for i, amount := range []float64{1000, 500, -200} {
		require.NoError(t, db.Create(&posting.Posting{
			TransactionID: fmt.Sprintf("t%d", i+1),
			Date:          time.Date(2024, 1, 10+i, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Checking",
			Commodity:     "INR",
			Amount:        decimal.NewFromFloat(amount),
			Quantity:      decimal.NewFromFloat(amount),
		}).Error)
	}

	result := GetCheckingBalance(db, "")
	breakdowns, ok := result["asset_breakdowns"].(map[string]AssetBreakdown)
	require.True(t, ok)
	require.Len(t, breakdowns, 1)

	bd, ok := breakdowns["Assets:Checking"]
	require.True(t, ok)
	assert.True(t, decimal.NewFromFloat(1300).Equal(bd.MarketAmount),
		"expected MarketAmount=1300; got %s", bd.MarketAmount)
	// InvestmentAmount and WithdrawalAmount are zero for checking accounts (by
	// definition in ComputeBreakdown – utils.IsCheckingAccount skips deposits
	// and withdrawals).
	assert.True(t, bd.InvestmentAmount.IsZero(), "InvestmentAmount must be zero for checking accounts")
	assert.True(t, bd.WithdrawalAmount.IsZero(), "WithdrawalAmount must be zero for checking accounts")
}

// TestGetCheckingBalance_OriginalBalances verifies that OriginalBalances is
// populated with the correct per-currency totals.
func TestGetCheckingBalance_OriginalBalances(t *testing.T) {
	loadTestConfig(t)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)

	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "t1",
		Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
		Account:       "Assets:Checking",
		Commodity:     "INR",
		Amount:        decimal.NewFromFloat(5000),
		Quantity:      decimal.NewFromFloat(5000),
	}).Error)

	result := GetCheckingBalance(db, "")
	breakdowns, ok := result["asset_breakdowns"].(map[string]AssetBreakdown)
	require.True(t, ok)
	bd, ok := breakdowns["Assets:Checking"]
	require.True(t, ok)

	require.Len(t, bd.OriginalBalances, 1)
	assert.Equal(t, "INR", bd.OriginalBalances[0].Currency)
	assert.True(t, decimal.NewFromFloat(5000).Equal(bd.OriginalBalances[0].Amount),
		"expected OriginalBalance=5000 for INR; got %s", bd.OriginalBalances[0].Amount)
}

// ---------------------------------------------------------------------------
// ComputeBreakdowns algorithm correctness tests
// ---------------------------------------------------------------------------

// TestComputeBreakdowns_PreGroupingConsistency verifies that the refactored
// O(A×C) algorithm produces the correct market amounts for a multi-account
// dataset with pre-populated MarketAmount fields.
func TestComputeBreakdowns_PreGroupingConsistency(t *testing.T) {
	loadTestConfig(t)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)

	type row struct {
		account, commodity string
		amount             float64
	}
	rows := []row{
		{"Assets:Checking", "INR", 1000},
		{"Assets:Checking", "INR", -200},
		{"Assets:Savings", "INR", 5000},
		{"Assets:Savings", "INR", 1000},
	}
	// Build postings in-memory with MarketAmount pre-set (simulating what
	// doGetBalance does via service.PopulateMarketPrice for INR postings).
	postings := make([]posting.Posting, 0, len(rows))
	for i, r := range rows {
		amt := decimal.NewFromFloat(r.amount)
		postings = append(postings, posting.Posting{
			TransactionID: fmt.Sprintf("tx%d", i+1),
			Date:          time.Date(2024, 1, 10+i, 0, 0, 0, 0, time.UTC),
			Account:       r.account,
			Commodity:     r.commodity,
			Amount:        amt,
			Quantity:      amt,
			MarketAmount:  amt,
		})
	}

	breakdowns := ComputeBreakdowns(db, postings, false)

	checkingBD, ok := breakdowns["Assets:Checking"]
	require.True(t, ok, "expected Assets:Checking in breakdowns")
	assert.True(t, decimal.NewFromFloat(800).Equal(checkingBD.MarketAmount),
		"Assets:Checking marketAmount: expected 800, got %s", checkingBD.MarketAmount)

	savingsBD, ok := breakdowns["Assets:Savings"]
	require.True(t, ok, "expected Assets:Savings in breakdowns")
	assert.True(t, decimal.NewFromFloat(6000).Equal(savingsBD.MarketAmount),
		"Assets:Savings marketAmount: expected 6000, got %s", savingsBD.MarketAmount)
}

// TestComputeBreakdowns_Rollup verifies that rollup=true produces parent-group
// entries that aggregate all children.
func TestComputeBreakdowns_Rollup(t *testing.T) {
	loadTestConfig(t)
	utils.SetNow("2024-03-20")
	defer utils.UnsetNow()
	db := openTestDB(t)

	mkAmt := func(a float64) decimal.Decimal { return decimal.NewFromFloat(a) }
	postings := []posting.Posting{
		{
			TransactionID: "t1",
			Date:          time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Checking:HDFC",
			Commodity:     "INR",
			Amount:        mkAmt(2000),
			Quantity:      mkAmt(2000),
			MarketAmount:  mkAmt(2000),
		},
		{
			TransactionID: "t2",
			Date:          time.Date(2024, 1, 11, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Checking:SBI",
			Commodity:     "INR",
			Amount:        mkAmt(3000),
			Quantity:      mkAmt(3000),
			MarketAmount:  mkAmt(3000),
		},
	}

	breakdowns := ComputeBreakdowns(db, postings, true)

	assets, ok := breakdowns["Assets"]
	require.True(t, ok, "expected Assets rollup entry")
	assert.True(t, decimal.NewFromFloat(5000).Equal(assets.MarketAmount),
		"Assets rollup: expected 5000, got %s", assets.MarketAmount)

	checking, ok := breakdowns["Assets:Checking"]
	require.True(t, ok, "expected Assets:Checking rollup entry")
	assert.True(t, decimal.NewFromFloat(5000).Equal(checking.MarketAmount),
		"Assets:Checking rollup: expected 5000, got %s", checking.MarketAmount)
}
