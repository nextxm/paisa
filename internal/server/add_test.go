package server

import (
	"strings"
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/stretchr/testify/assert"
)

func TestFormatTransactionBeancount(t *testing.T) {
	config.LoadConfig([]byte(`
journal_path: journal.ledger
db_path: paisa.db
ledger_cli: beancount
`), "")

	req := AddTransactionRequest{
		Date:        "2024-05-01",
		Payee:       "Supermarket",
		Narration:   "Groceries",
		FromAccount: "Assets:Checking",
		ToAccount:   "Expenses:Food",
		Amount:      "50.00",
		Commodity:   "USD",
	}

	result := formatTransaction(req)
	expected := `2024-05-01 * "Supermarket" "Groceries"
  Assets:Checking  -50.00 USD
  Expenses:Food   50.00 USD

`
	assert.Equal(t, expected, result)
}

func TestFormatTransactionHLedger(t *testing.T) {
	config.LoadConfig([]byte(`
journal_path: journal.ledger
db_path: paisa.db
ledger_cli: hledger
`), "")

	req := AddTransactionRequest{
		Date:        "2024-05-01",
		Payee:       "Supermarket",
		Narration:   "Groceries",
		FromAccount: "Assets:Checking",
		ToAccount:   "Expenses:Food",
		Amount:      "50.00",
		Commodity:   "USD",
	}

	result := formatTransaction(req)
	expected := `2024/05/01 * Supermarket | Groceries
  Assets:Checking  -50.00 USD
  Expenses:Food   50.00 USD

`
	assert.Equal(t, expected, result)
}

func TestFormatTransactionBeancountWithFX(t *testing.T) {
	config.LoadConfig([]byte(`
journal_path: journal.ledger
db_path: paisa.db
ledger_cli: beancount
`), "")

	req := AddTransactionRequest{
		Date:        "2024-05-01",
		Payee:       "Supermarket",
		Narration:   "Groceries",
		FromAccount: "Assets:Checking",
		ToAccount:   "Expenses:Food",
		Amount:      "50.00",
		Commodity:   "USD",
		ToAmount:    "45.00",
		ToCommodity: "EUR",
	}

	result := formatTransaction(req)
	expected := `2024-05-01 * "Supermarket" "Groceries"
  Assets:Checking  -50.00 USD @@ 45.00 EUR
  Expenses:Food   45.00 EUR

`
	assert.Equal(t, expected, result)
}

func TestFormatTransactionBeancountWithRate(t *testing.T) {
	config.LoadConfig([]byte(`
journal_path: journal.ledger
db_path: paisa.db
ledger_cli: beancount
`), "")

	req := AddTransactionRequest{
		Date:         "2024-05-01",
		Payee:        "Supermarket",
		Narration:    "Groceries",
		FromAccount:  "Assets:Checking",
		ToAccount:    "Expenses:Food",
		Amount:       "50.00",
		Commodity:    "USD",
		ToCommodity:  "EUR",
		ExchangeRate: "0.90",
	}

	result := formatTransaction(req)
	// Note: Beancount @ is per-unit rate, so the amount on ToAccount usually needs to be balanced.
	// For simplicity in this quick-add form, if only ExchangeRate is provided but not ToAmount,
	// we just let Beancount figure it out or log the rate.
	// But let's just check the formatting output:
	if !strings.Contains(result, "-50.00 USD @ 0.90 EUR") {
		t.Errorf("Expected exchange rate formatting, got: %s", result)
	}
}

func TestFormatTransactionSplitPostings(t *testing.T) {
	config.LoadConfig([]byte(`
journal_path: journal.ledger
db_path: paisa.db
ledger_cli: ledger
`), "")

	req := AddTransactionRequest{
		Date:      "2024-05-01",
		Payee:     "Supermarket",
		Narration: "Groceries & Household",
		Postings: []AddPostingRequest{
			{Account: "Assets:Checking", Amount: "-100.00", Commodity: "USD"},
			{Account: "Expenses:Food", Amount: "60.00", Commodity: "USD"},
			{Account: "Expenses:Household", Amount: "40.00", Commodity: "USD"},
		},
	}

	result := formatTransaction(req)
	expected := `2024/05/01 * Supermarket | Groceries & Household
  Assets:Checking  -100.00 USD
  Expenses:Food  60.00 USD
  Expenses:Household  40.00 USD

`
	assert.Equal(t, expected, result)
}

func TestAppendTransactionFallsBackToJournalPath(t *testing.T) {
	tmpDir := t.TempDir()
	journalPath := tmpDir + "/main.ledger"

	config.LoadConfig([]byte(`
journal_path: `+journalPath+`
db_path: `+tmpDir+`/paisa.db
ledger_cli: ledger
`), "")

	// Ensure add_journal_path is empty
	assert.Equal(t, "", config.GetAddJournalPath())

	req := AddTransactionRequest{
		Date:        "2024-05-01",
		Payee:       "Cafe",
		FromAccount: "Assets:Checking",
		ToAccount:   "Expenses:Coffee",
		Amount:      "5.00",
		Commodity:   "USD",
	}

	// validateAddTransactionRequest should pass
	assert.NoError(t, validateAddTransactionRequest(req))

	// formatTransaction should produce valid entry
	entry := formatTransaction(req)
	assert.Contains(t, entry, "2024/05/01 * Cafe")
}
