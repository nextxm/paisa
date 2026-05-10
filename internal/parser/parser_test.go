package parser

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockAccounts returns a standard set of accounts for testing.
func mockAccounts() []string {
	return []string{
		"Assets:Checking",
		"Assets:Savings",
		"Assets:CAD:Checking",
		"Liabilities:CreditCard:BMO",
		"Liabilities:CAD:CC:BMO:CreditC",
		"Liabilities:CreditCard:Visa",
		"Expenses:Groceries",
		"Expenses:Dining",
		"Expenses:Transport",
		"Expenses:Entertainment",
		"Income:Salary",
		"Income:Freelance",
		"Income:Investment",
	}
}

// newTestParser creates a parser for testing with default accounts
func newTestParser(keywords KeywordMatcher) *Parser {
	return NewParser(keywords, nil, mockAccounts())
}

// TestParseTransaction_SimpleExpense tests parsing a basic expense transaction.
// Input: "20 Apr, bought 15$ groceries using bmo cc from no frills"
// Expected: Amount=15 USD, From=Liabilities:BMO:CC (or similar), To=Expenses:Groceries
func TestParseTransaction_SimpleExpense(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("20 Apr, bought 15$ groceries using bmo cc from no frills")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(15), result.Amount)
	// assert.Equal(t, "USD", result.Currency)
	// assert.True(t, strings.Contains(result.FromAccount, "BMO") || strings.Contains(result.FromAccount, "CC"))
	// assert.Equal(t, "expense", result.Direction)
	// assert.Greater(t, result.Confidence.Overall, 0.85)
}

// TestParseTransaction_IncomeDeposit tests parsing an income transaction.
// Input: "May 21, received $2500 salary from employer into checking"
// Expected: Amount=2500 USD, From=Income:Salary, To=Assets:Checking
func TestParseTransaction_IncomeDeposit(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("May 21, received $2500 salary from employer into checking")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(2500), result.Amount)
	// assert.Equal(t, "income", result.Direction)
	// assert.True(t, strings.Contains(result.FromAccount, "Salary"))
	// assert.Greater(t, result.Confidence.Overall, 0.85)
}

// TestParseTransaction_Transfer tests parsing a transfer between accounts.
// Input: "2026-05-10 moved 500 INR from checking to savings"
// Expected: Amount=500 INR, From=Assets:Checking, To=Assets:Savings
func TestParseTransaction_Transfer(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("2026-05-10 moved 500 INR from checking to savings")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(500), result.Amount)
	// assert.Equal(t, "INR", result.Currency)
	// assert.Equal(t, "transfer", result.Direction)
	// assert.Greater(t, result.Confidence.Overall, 0.85)
}

// TestParseTransaction_AmbiguousAmount tests parsing with ambiguous amount.
// Input: "bought something at amazon for about $50"
// Expected: Amount=50 USD (with lower confidence), Suggestions shown for account
func TestParseTransaction_AmbiguousAmount(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("bought something at amazon for about $50")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(50), result.Amount)
	// assert.Less(t, result.Confidence.Amount, 0.90) // Less confident due to "about"
	// assert.Greater(t, len(result.Suggestions), 0)  // Should have suggestions
}

// TestParseTransaction_MissingDate tests parsing without date (should default to today).
// Input: "paid 30$ electricity bill using debit"
// Expected: Amount=30 USD, Date=Today, Direction=expense
func TestParseTransaction_MissingDate(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("paid 30$ electricity bill using debit")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// today := time.Now().Truncate(24 * time.Hour)
	// assert.Equal(t, today, result.Date.Truncate(24*time.Hour))
	// assert.Less(t, result.Confidence.Date, 0.50) // Low confidence for defaulted date
}

// TestParseTransaction_MultiCurrency tests parsing with explicit currency.
// Input: "10 Apr spent 1500 INR at restaurant using credit card"
// Expected: Amount=1500 INR, From=Liabilities:CC (or similar), To=Expenses:Dining
func TestParseTransaction_MultiCurrency(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("10 Apr spent 1500 INR at restaurant using credit card")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(1500), result.Amount)
	// assert.Equal(t, "INR", result.Currency)
	// assert.Equal(t, "expense", result.Direction)
}

// TestParseTransaction_AmbiguousAccount tests parsing with unclear account hints.
// Input: "20 Apr transferred 100$ to my account"
// Expected: Amount=100 USD, Suggestions shown for ToAccount (low confidence)
func TestParseTransaction_AmbiguousAccount(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("20 Apr transferred 100$ to my account")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(100), result.Amount)
	// assert.Less(t, result.Confidence.ToAccount, 0.75) // Low confidence for vague account
	// assert.Greater(t, len(result.Suggestions), 0)
}

// TestParseTransaction_RefundScenario tests parsing a refund (negative direction).
// Input: "May 5, received refund of 75$ from Amazon back to credit card"
// Expected: Amount=75 USD, Direction=income, From=Assets (Refund), To=Liabilities:CC
func TestParseTransaction_RefundScenario(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("May 5, received refund of 75$ from Amazon back to credit card")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(75), result.Amount)
	// Note: Refunds need special direction handling
}

// TestParseTransaction_CashWithdrawal tests parsing a cash withdrawal.
// Input: "12 Apr withdrew 200$ cash from ATM"
// Expected: Amount=200 USD, From=Assets:Checking (or Bank), To=Assets:Cash, Direction=transfer
func TestParseTransaction_CashWithdrawal(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("12 Apr withdrew 200$ cash from ATM")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// assert.Equal(t, decimal.NewFromInt(200), result.Amount)
	// assert.Equal(t, "transfer", result.Direction)
	// assert.True(t, strings.Contains(result.ToAccount, "Cash"))
}

// TestParseTransaction_SplitTransaction tests parsing with multiple amounts (should pick largest).
// Input: "bought 2x $15 items and paid $50 for delivery"
// Expected: Should either error or pick the largest amount ($50)
func TestParseTransaction_SplitTransaction(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	_, _ = parser.ParseTransaction("bought 2x $15 items and paid $50 for delivery")
	// TODO: Implement and decide behavior
	// Either: error and request single transaction
	// Or: pick largest amount and warn
}

// TestParseTransaction_CustomPayee tests matching against user-defined custom payees.
// Input: "20 Apr bought groceries at Acme Corp using visa"
// With custom mapping: "Acme Corp" -> "Expenses:GroceriesRetailer"
// Expected: Payee recognized as custom mapping, high confidence for account
func TestParseTransaction_CustomPayee(t *testing.T) {
	keywords := DefaultKeywords()
	keywords.CustomPayees = map[string]string{
		"Acme Corp": "Expenses:GroceriesRetailer",
	}

	parser := newTestParser(keywords)
	result, err := parser.ParseTransaction("20 Apr bought $50 groceries at Acme Corp using visa")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// TODO: Implement and verify assertions
	// Custom payee should boost confidence for account matching
}

// TestConfidenceComputation tests the confidence scoring algorithm.
func TestConfidenceComputation(t *testing.T) {
	scores := ConfidenceScores{
		Date:        0.9,
		Amount:      0.95,
		Payee:       0.85,
		FromAccount: 0.80,
		ToAccount:   0.85,
		Direction:   0.90,
	}

	overall := ComputeConfidence(scores)

	// TODO: Implement and verify assertions
	// Should be weighted average: (0.95*0.30 + 0.80*0.25 + 0.85*0.25 + 0.85*0.15 + 0.9*0.05) / 1.0
	// Expected: ~0.87
	assert.Greater(t, overall, 0.80)
	assert.Less(t, overall, 0.95)
}

// TestThresholdDecisions tests the confidence threshold logic.
func TestThresholdDecisions(t *testing.T) {
	thresholds := DefaultThresholds()

	// High confidence -> auto-create
	assert.True(t, thresholds.AutoCreate >= 0.80)
	// Show suggestions for lower confidence
	assert.True(t, thresholds.ShowSuggestions < thresholds.AutoCreate)
	// Require confirmation for very low confidence
	assert.True(t, thresholds.RequireConfirmation < thresholds.ShowSuggestions)
}

// TestKeywordMatching tests keyword extraction and scoring.
// TODO: Implement KeywordScore, ExtractPaymentMethodHint, ExtractTransactionTypeHint helpers
/*
func TestKeywordMatching(t *testing.T) {
	keywords := DefaultKeywords()

	// Test expense keyword
	found, score := KeywordScore("bought groceries", keywords.ExpenseMarkers)
	assert.True(t, found)
	assert.Greater(t, score, 0.8)

	// Test payment method keyword
	method, score := keywords.ExtractPaymentMethodHint("using credit card")
	assert.Equal(t, "cc", method)
	assert.Greater(t, score, 0.8)

	// Test transaction type
	ttype, score := keywords.ExtractTransactionTypeHint("bought something")
	assert.Equal(t, "expense", ttype)
	assert.Greater(t, score, 0.8)
}
*/

// TestRegexPatterns tests NLP regex patterns.
func TestRegexPatterns(t *testing.T) {
	patterns := CompilePatterns()

	// Test date patterns
	assert.NotNil(t, patterns.DateYYYYMMDD)
	assert.NotNil(t, patterns.DateDDMonth)

	// Test amount patterns
	assert.NotNil(t, patterns.AmountDollarPrefix)
	assert.NotNil(t, patterns.AmountSuffix)

	// Test account hint patterns
	assert.NotNil(t, patterns.AccountHint)
	assert.NotNil(t, patterns.PaymentHint)
}

// BenchmarkParseTransaction measures parsing performance.
func BenchmarkParseTransaction(b *testing.B) {
	parser := newTestParser(DefaultKeywords())
	input := "20 Apr, bought 15$ groceries using bmo cc from no frills"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := parser.ParseTransaction(input)
		if err != nil {
			b.Fatalf("ParseTransaction failed: %v", err)
		}
	}
	// Target: <500ms for 1000 transactions = <0.5ms per transaction
}

// TestEdgeCase_EmptyInput tests handling of empty input.
func TestEdgeCase_EmptyInput(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	_, err := parser.ParseTransaction("")
	assert.Error(t, err)
}

// TestEdgeCase_VeryLongInput tests handling of very long input.
func TestEdgeCase_VeryLongInput(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	longInput := "bought something " // Repeat many times
	for i := 0; i < 1000; i++ {
		longInput += "and more "
	}

	result, err := parser.ParseTransaction(longInput)
	// Should either parse or error gracefully, not crash
	_ = result
	_ = err
}

// TestEdgeCase_NoAmount tests handling of transaction without amount.
func TestEdgeCase_NoAmount(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	_, err := parser.ParseTransaction("bought groceries today")
	// Should error because amount is required
	assert.Error(t, err)
}

// TestEdgeCase_SpecialCharacters tests handling of special characters.
func TestEdgeCase_SpecialCharacters(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("paid 100$ to shop@home #groceries & stuff")
	// Should handle special characters without crashing
	if err == nil {
		assert.NotNil(t, result)
	}
}

// TestCompactFormat tests parsing a compact transaction without explicit action words.
// Input: "20 cad groceries bmo cc at no frills"
// Expected: Amount=20 CAD, From=Liabilities:CAD:CC:BMO:CreditC, To=Expenses:Groceries, Direction=expense
func TestCompactFormat(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("20 cad groceries bmo cc at no frills")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify amount extraction
	assert.Equal(t, "20", result.Amount.String())
	assert.Equal(t, "CAD", result.Currency)

	// Verify account matching (should find BMO and Groceries accounts)
	assert.True(t,
		(result.FromAccount != "" && (strings.Contains(strings.ToUpper(result.FromAccount), "BMO") || strings.Contains(strings.ToUpper(result.FromAccount), "CC"))) ||
			result.Confidence.FromAccount < 0.5,
		"FromAccount should contain BMO or CC, or have low confidence with suggestions",
	)

	assert.True(t,
		(result.ToAccount != "" && strings.Contains(result.ToAccount, "Groceries")) ||
			result.Confidence.ToAccount < 0.5,
		"ToAccount should contain Groceries, or have low confidence with suggestions",
	)

	// Should classify as expense
	assert.Equal(t, "expense", result.Direction)
}

// TestBMOCreditCardMatching tests that BMO credit card accounts are properly matched
func TestBMOCreditCardMatching(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("20 cad groceries no frills, bmo cc")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify amount and currency
	assert.Equal(t, "20", result.Amount.String())
	assert.Equal(t, "CAD", result.Currency)

	// Verify from account matches BMO
	assert.NotEqual(t, "", result.FromAccount, "FromAccount should not be empty - should match BMO account")
	assert.True(t, strings.Contains(strings.ToUpper(result.FromAccount), "BMO"),
		"FromAccount should contain BMO: "+result.FromAccount)

	// Verify to account is Expenses:Groceries
	assert.Equal(t, "Expenses:Groceries", result.ToAccount, "ToAccount should be Expenses:Groceries")

	// Verify payee is "no frills", not including currency/category/method
	payeeLower := strings.ToLower(result.Payee)
	assert.True(t, strings.Contains(payeeLower, "frills"), "Payee should contain 'frills'")
	assert.False(t, strings.Contains(payeeLower, "cad"), "Payee should not contain currency 'cad'")
	assert.False(t, strings.Contains(payeeLower, "bmo"), "Payee should not contain payment method 'bmo'")
	assert.False(t, strings.Contains(payeeLower, "groceries"), "Payee should not contain category 'groceries'")
}

// TestPayeeExtraction tests that payee extraction properly excludes unwanted tokens
func TestPayeeExtraction(t *testing.T) {
	parser := newTestParser(DefaultKeywords())

	tests := []struct {
		input       string
		expected    string   // Should contain
		notExpected []string // Should NOT contain
	}{
		{
			"20 cad groceries no frills, bmo cc",
			"frills",
			[]string{"cad", "bmo", "groceries"},
		},
		{
			"100 usd coffee at starbucks",
			"starbucks",
			[]string{"usd", "coffee"},
		},
		{
			"15 eur gas from chevron",
			"chevron",
			[]string{"eur", "gas"},
		},
	}

	for _, tt := range tests {
		result, err := parser.ParseTransaction(tt.input)
		assert.NoError(t, err)

		payeeLower := strings.ToLower(result.Payee)
		assert.True(t, strings.Contains(payeeLower, strings.ToLower(tt.expected)),
			"Payee should contain '%s' for input: %s, got: %s", tt.expected, tt.input, result.Payee)

		for _, notExp := range tt.notExpected {
			assert.False(t, strings.Contains(payeeLower, strings.ToLower(notExp)),
				"Payee should NOT contain '%s' for input: %s, got: %s", notExp, tt.input, result.Payee)
		}
	}
}
