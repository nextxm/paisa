package importer

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/import_rule"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

func TestIngestCSV_WithRulesAndDuplicates(t *testing.T) {
	db := openTestDB(t)

	// Seed existing posting in the ledger
	existingTxDate, _ := time.Parse("2006-01-02", "2026-09-15")
	require.NoError(t, db.Create(&posting.Posting{
		Date:      existingTxDate,
		Payee:     "Netflix Subscription",
		Amount:    decimal.NewFromFloat(-19.99),
		Commodity: "USD",
		Account:   "Assets:Checking",
	}).Error)

	// Seed an import rule
	require.NoError(t, import_rule.Save(db, &import_rule.Rule{
		Name:          "Starbucks Rule",
		PayeePattern:  "Starbucks",
		TargetAccount: "Expenses:Dining:Coffee",
		Tags:          []string{"#coffee"},
		Priority:      10,
	}))

	csvData := `Date,Description,Debit,Credit
2026-09-15,Netflix Subscription,19.99,
2026-09-16,Starbucks Store #123,5.75,
2026-09-20,Unknown Merchant,50.00,
`

	result, err := Ingest(db, IngestionRequest{
		Format:  FormatCSV,
		Content: csvData,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalParsed)

	// First tx should be recognized as duplicate
	tx1 := result.Transactions[0]
	assert.Equal(t, "Netflix Subscription", tx1.Payee)
	assert.True(t, tx1.IsDuplicate)
	assert.GreaterOrEqual(t, tx1.DuplicateScore, 70)

	// Second tx should be auto-categorized by rule
	tx2 := result.Transactions[1]
	assert.Equal(t, "Starbucks Store #123", tx2.Payee)
	assert.Equal(t, "Expenses:Dining:Coffee", tx2.SelectedAccount)
	assert.Equal(t, "Starbucks Rule", tx2.MatchedRuleName)
	assert.Equal(t, []string{"#coffee"}, tx2.Tags)
	assert.False(t, tx2.IsDuplicate)

	// Third tx needs classification
	tx3 := result.Transactions[2]
	assert.Equal(t, "", tx3.SelectedAccount)
	assert.False(t, tx3.IsDuplicate)
}
