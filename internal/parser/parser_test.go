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
		"Assets:Crypto:Neo",
		"Liabilities:CreditCard:BMO",
		"Liabilities:CAD:CC:BMO:CreditC",
		"Liabilities:CAD:CC:Neo",
		"Liabilities:CreditCard:Neo",
		"Liabilities:CreditCard:Visa",
		"Expenses:Groceries",
		"Expenses:Dining",
		"Expenses:Transport",
		"Expenses:Entertainment",
		"Expenses:Shopping",
		"Assets:INR:Bank:ICICI-Hyd",
		"Assets:INR:Bank:HDFC",
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

func TestTransferPhraseMatching(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("transfer 20 cad from icici hyd to hdfc")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "20", result.Amount.String())
	assert.Equal(t, "CAD", result.Currency)
	assert.Equal(t, "transfer", result.Direction)

	assert.True(t, strings.Contains(strings.ToLower(result.FromAccount), "icici"), "from account should match icici")
	assert.True(t, strings.Contains(strings.ToLower(result.ToAccount), "hdfc"), "to account should match hdfc")
	assert.NotEqual(t, result.FromAccount, result.ToAccount, "from and to should not be same account")
}

func TestXferPhraseMatching(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("xfer 20 cad from icici hyd to hdfc")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "20", result.Amount.String())
	assert.Equal(t, "CAD", result.Currency)
	assert.Equal(t, "transfer", result.Direction)
	assert.True(t, strings.Contains(strings.ToLower(result.FromAccount), "icici"), "from account should match icici")
	assert.True(t, strings.Contains(strings.ToLower(result.ToAccount), "hdfc"), "to account should match hdfc")
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
			"20 cad from bmo cc for groceries at no frills",
			"frills",
			[]string{"cad", "bmo", "groceries", "credit", "card"},
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

// TestTokenExtraction verifies that meaningful tokens are properly extracted from account names
func TestTokenExtraction(t *testing.T) {
	parser := newTestParser(DefaultKeywords())

	// Test that meaningful tokens are extracted and generic ones are filtered
	tests := []struct {
		account     string
		expected    []string // Should contain these tokens
		notExpected []string // Should NOT contain these tokens
	}{
		{
			"Liabilities:CreditCard:Neo",
			[]string{"neo"},
			[]string{"liabilities", "creditcard", "credit", "card"},
		},
		{
			"Liabilities:CAD:CC:BMO:CreditC",
			[]string{"bmo"},
			[]string{"liabilities", "creditcard", "credit", "card", "cad"},
		},
		{
			"Expenses:Groceries",
			[]string{"groceries"},
			[]string{"expenses"},
		},
	}

	for _, tt := range tests {
		tokens := parser.extractMeaningfulTokens(tt.account)
		tokenSet := make(map[string]bool)
		for _, t := range tokens {
			tokenSet[t] = true
		}

		for _, exp := range tt.expected {
			assert.True(t, tokenSet[exp],
				"Account %s should contain token '%s', got tokens: %v", tt.account, exp, tokens)
		}

		for _, notExp := range tt.notExpected {
			assert.False(t, tokenSet[notExp],
				"Account %s should NOT contain token '%s', got tokens: %v", tt.account, notExp, tokens)
		}
	}
}

// TestNeoCreditCardMatching tests that Neo credit card accounts are properly matched
// using dynamic token matching (not hardcoded bank names)
func TestNeoCreditCardMatching(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	result, err := parser.ParseTransaction("20 cad groceries at walmart on neo cc")
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify amount and currency
	assert.Equal(t, "20", result.Amount.String())
	assert.Equal(t, "CAD", result.Currency)

	// Verify from account matches Neo (should be matched dynamically, not from hardcoded list)
	assert.NotEqual(t, "", result.FromAccount, "FromAccount should not be empty - should match Neo account")
	assert.True(t, strings.Contains(strings.ToLower(result.FromAccount), "neo"),
		"FromAccount should contain Neo: "+result.FromAccount)
	assert.True(t, strings.HasPrefix(result.FromAccount, "Liabilities:"),
		"FromAccount should prefer liability card account over assets: "+result.FromAccount)

	// Verify to account is Expenses:Groceries
	assert.Equal(t, "Expenses:Groceries", result.ToAccount, "ToAccount should be Expenses:Groceries")

	// Verify payee is "walmart", not including currency/category/payment method
	payeeLower := strings.ToLower(result.Payee)
	assert.True(t, strings.Contains(payeeLower, "walmart"), "Payee should contain 'walmart'")
	assert.False(t, strings.Contains(payeeLower, "cad"), "Payee should not contain currency 'cad'")
	assert.False(t, strings.Contains(payeeLower, "neo"), "Payee should not contain payment method 'neo'")
	assert.False(t, strings.Contains(payeeLower, "groceries"), "Payee should not contain category 'groceries'")
}

func TestExtractHintsRetainsProviderAndCardTokens(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	fromHint, toHint := parser.extractHints(normalizeText("20 cad groceries at walmart on neo cc"))

	assert.True(t, strings.Contains(strings.ToLower(fromHint), "neo"), "fromHint should include provider token")
	assert.True(t, strings.Contains(strings.ToLower(fromHint), "cc"), "fromHint should include cc token")
	assert.NotEqual(t, "", toHint)
}

func TestTokenMatchingPrefersNeoCCLiability(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	account, _ := parser.matchAccounts("neo cc", "from")

	assert.True(t, strings.HasPrefix(account, "Liabilities:"), "neo cc should prefer liability account, got: "+account)
	assert.True(t, strings.Contains(strings.ToLower(account), "neo"), "matched account should contain neo")
}

// TestSpanMaskInitialization tests that span mask is properly initialized
func TestSpanMaskInitialization(t *testing.T) {
	mask := NewSpanMask("hello world 2026-05-10 $50")
	assert.NotNil(t, mask)
	assert.Equal(t, "hello world 2026-05-10 $50", mask.Source)
	assert.Equal(t, 0, len(mask.ConsumedSpans))
}

// TestSpanMaskRecordSpan tests recording consumed spans
func TestSpanMaskRecordSpan(t *testing.T) {
	mask := NewSpanMask("hello world 2026-05-10 $50")

	// Record first span (date)
	mask.RecordSpan(12, 22) // "2026-05-10"
	assert.Equal(t, 1, len(mask.ConsumedSpans))
	assert.Equal(t, 12, mask.ConsumedSpans[0].Start)
	assert.Equal(t, 22, mask.ConsumedSpans[0].End)

	// Record second span (amount)
	mask.RecordSpan(23, 26) // "$50"
	assert.Equal(t, 2, len(mask.ConsumedSpans))
}

// TestSpanMaskGetUnconsumedText tests extraction of unconsumed text
func TestSpanMaskGetUnconsumedText(t *testing.T) {
	source := "hello world 2026-05-10 $50"
	mask := NewSpanMask(source)

	// No spans consumed yet
	assert.Equal(t, source, mask.GetUnconsumedText())

	// Consume date
	mask.RecordSpan(12, 22)
	unconsumed := mask.GetUnconsumedText()
	assert.NotContains(t, unconsumed, "2026-05-10")
	assert.Contains(t, unconsumed, "hello")
	assert.Contains(t, unconsumed, "world")
	assert.Contains(t, unconsumed, "$50")

	// Consume amount
	mask.RecordSpan(23, 26)
	unconsumed = mask.GetUnconsumedText()
	assert.NotContains(t, unconsumed, "2026-05-10")
	assert.NotContains(t, unconsumed, "$50")
	assert.Contains(t, unconsumed, "hello")
	assert.Contains(t, unconsumed, "world")
}

// TestSpanMaskingPreventesDoubleCountingTokens verifies that span masking
// prevents the same tokens from being used in multiple extraction steps
func TestSpanMaskingPreventesDoubleCountingTokens(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	input := "2026-05-10 spent $50 at walmart using amex"

	result, err := parser.ParseTransaction(input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Date and amount should be extracted and recorded in spans
	assert.NotNil(t, parser.spanMask)

	// The unconsumed text should not contain the date or amount
	unconsumedText := parser.spanMask.GetUnconsumedText()
	assert.NotContains(t, unconsumedText, "2026-05-10")
	assert.NotContains(t, unconsumedText, "50") // Amount should be masked

	// But should contain the remainder
	assert.True(t, strings.Contains(strings.ToLower(unconsumedText), "walmart") ||
		strings.Contains(strings.ToLower(unconsumedText), "amex"),
		"Unconsumed text should contain payee/account hints")
}

// TestSpanMaskingWithComplexInput tests span masking on a realistic complex transaction
func TestSpanMaskingWithComplexInput(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	input := "May 15 transferred 1500 INR from savings to checking"

	result, err := parser.ParseTransaction(input)
	// Should parse successfully with the amount present
	assert.NoError(t, err, "should parse transaction with explicit amount")
	assert.NotNil(t, result)
	assert.Equal(t, "1500", result.Amount.String())

	// Span mask should have recorded consumed regions
	assert.NotNil(t, parser.spanMask)
	assert.Greater(t, len(parser.spanMask.ConsumedSpans), 0, "should have recorded at least one consumed span")
}

// TestBareAccountTokenExtraction tests extraction of account names without explicit keywords
func TestBareAccountTokenExtraction(t *testing.T) {
	parser := newTestParser(DefaultKeywords())
	input := "15 inr icici hyd for shopping clothing"

	result, err := parser.ParseTransaction(input)
	assert.NoError(t, err)
	assert.NotNil(t, result)

	// Verify amount extraction
	assert.Equal(t, "15", result.Amount.String())
	assert.Equal(t, "INR", result.Currency)

	// Verify account matching - should find ICICI-Hyd account from bare tokens "icici hyd"
	assert.NotEqual(t, "", result.FromAccount, "FromAccount should be matched from bare tokens 'icici hyd'")
	assert.True(t, strings.Contains(strings.ToUpper(result.FromAccount), "ICICI"),
		"FromAccount should contain ICICI: "+result.FromAccount)
	assert.True(t, strings.Contains(strings.ToUpper(result.FromAccount), "HYD"),
		"FromAccount should contain HYD: "+result.FromAccount)

	// Verify expense category matching - should find Expenses:Shopping from "shopping"
	assert.Equal(t, "Expenses:Shopping", result.ToAccount, "ToAccount should be Expenses:Shopping")

	// Verify direction
	assert.Equal(t, "expense", result.Direction)
}
