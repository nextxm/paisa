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

func TestIngestExcelAndPDF(t *testing.T) {
	db := openTestDB(t)

	// Normalized tabular content produced from Excel or PDF
	content := `Date,Description,Debit,Credit
2026-09-15,Supermarket Groceries,45.20,
2026-09-16,Salary Deposit,,3500.00
`

	// Test Excel format request
	excelResult, err := Ingest(db, IngestionRequest{
		Format:  FormatExcel,
		Content: content,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, excelResult.TotalParsed)
	assert.Equal(t, "Supermarket Groceries", excelResult.Transactions[0].Payee)
	assert.True(t, excelResult.Transactions[0].IsDebit)
	assert.Equal(t, 45.20, excelResult.Transactions[0].Amount)
	assert.False(t, excelResult.Transactions[1].IsDebit)
	assert.Equal(t, 3500.00, excelResult.Transactions[1].Amount)

	// Test PDF format request
	pdfResult, err := Ingest(db, IngestionRequest{
		Format:  FormatPDF,
		Content: content,
	})
	require.NoError(t, err)
	assert.Equal(t, 2, pdfResult.TotalParsed)
	assert.Equal(t, "Salary Deposit", pdfResult.Transactions[1].Payee)
}

func TestIngestStatementWithMetadataHeader(t *testing.T) {
	db := openTestDB(t)

	// Sample with non-transaction top section and Dr./Cr. amounts
	statement := `,,,,,,,,,,,,,,,,,
,VIEW CURRENT STATEMENT,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,
,My Credit Card Details for,,,,XXXAFAFA 450172******0023,,,,,,,,,,,,
,Card Holder Name,,,,XXXAFAFA,,,,Card Type,,,,primary,,,,
,Statement Date,,,,12-10-2026,,,,Previous Balance,,,,INR 4083,,,,
,Transaction Details,,,,,,,,,,,,,,,,
,Transaction Date,,,,Details,,,,Amount (INR),,,,Reward Points,,,,Reference Number
,,,,,,,,,,,,,,,,,
,11-10-2026,,,,UPI-Smart Ba,,,,366.55 Dr.,,,,7,,,,14336279452
,10-10-2026,,,,AMAZON MUMBAI,,,,299 Dr.,,,,5,,,,14335604658
,26-09-2026,,,,BBPS Payment received,,,,4083 Cr.,,,,0,,,,14238609506
`

	result, err := Ingest(db, IngestionRequest{
		Format:  FormatCSV,
		Content: statement,
	})
	require.NoError(t, err)
	assert.Equal(t, 3, result.TotalParsed)

	// First tx: 366.55 Dr.
	assert.Equal(t, "2026-10-11", result.Transactions[0].Date)
	assert.Equal(t, "UPI-Smart Ba", result.Transactions[0].Payee)
	assert.Equal(t, 366.55, result.Transactions[0].Amount)
	assert.True(t, result.Transactions[0].IsDebit)

	// Second tx: 299 Dr.
	assert.Equal(t, "2026-10-10", result.Transactions[1].Date)
	assert.Equal(t, 299.0, result.Transactions[1].Amount)
	assert.True(t, result.Transactions[1].IsDebit)

	// Third tx: 4083 Cr. (Payment)
	assert.Equal(t, "2026-09-26", result.Transactions[2].Date)
	assert.Equal(t, "BBPS Payment received", result.Transactions[2].Payee)
	assert.Equal(t, 4083.0, result.Transactions[2].Amount)
	assert.False(t, result.Transactions[2].IsDebit)
}

func TestIngestSampleICICIStatement_FromDocs(t *testing.T) {
	db := openTestDB(t)

	// Ingesting the real ICICI statement as CSV content (as produced by frontend XLSX parser)
	statementText := `,VIEW CURRENT STATEMENT,,,,,,,,,,,,,,,,
,My Credit Card Details for,,,,XXXAFAFA 450172******0023,,,,,,,,,,,,
,Card Holder Name,,,,XXXAFAFA,,,,Card Type,,,,primary,,,,
,Statement Date,,,,12-10-2026,,,,Previous Balance,,,,INR 4083,,,,
,Transaction Details,,,,,,,,,,,,,,,,
,Transaction Date,,,,Details,,,,Amount (INR),,,,Reward Points,,,,Reference Number
,11-10-2026,,,,UPI-Smart Ba,,,,366.55 Dr.,,,,7,,,,14336279452
,11-10-2026,,,,UPI-ZEPTONOW,,,,427 Dr.,,,,8,,,,14333458614
,11-10-2026,,,,UPI-SWIGGY,,,,309 Dr.,,,,6,,,,14333153570
,10-10-2026,,,,AMAZON MUMBAI,,,,299 Dr.,,,,5,,,,14335604658
,09-10-2026,,,,FlipkartInternetPrivate,,,,6409 Dr.,,,,128,,,,14331602122
,26-09-2026,,,,BBPS Payment received,,,,4083 Cr.,,,,0,,,,14238609506
`

	result, err := Ingest(db, IngestionRequest{
		Format:  FormatExcel,
		Content: statementText,
	})
	require.NoError(t, err)
	assert.Equal(t, 6, result.TotalParsed)
	assert.Equal(t, 366.55, result.Transactions[0].Amount)
	assert.True(t, result.Transactions[0].IsDebit)
	assert.Equal(t, 4083.0, result.Transactions[5].Amount)
	assert.False(t, result.Transactions[5].IsDebit)
}
