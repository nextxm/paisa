package parser

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ParseResult represents the complete result of parsing a transaction from natural language text.
type ParseResult struct {
	Date        time.Time        `json:"date"`
	Amount      decimal.Decimal  `json:"amount"`
	Currency    string           `json:"currency"`
	Payee       string           `json:"payee"`
	FromAccount string           `json:"from_account"`
	ToAccount   string           `json:"to_account"`
	Direction   string           `json:"direction"` // "expense", "income", "transfer"
	Confidence  ConfidenceScores `json:"confidence"`
	Suggestions []SuggestionSet  `json:"suggestions"`
	Warnings    []string         `json:"warnings"`
}

// ConfidenceScores tracks confidence for each extracted field (0.0 to 1.0).
type ConfidenceScores struct {
	Date        float64 `json:"date"`
	Amount      float64 `json:"amount"`
	Payee       float64 `json:"payee"`
	FromAccount float64 `json:"from_account"`
	ToAccount   float64 `json:"to_account"`
	Direction   float64 `json:"direction"`
	Overall     float64 `json:"overall"`
}

// Suggestion represents a candidate account match with confidence score.
type Suggestion struct {
	Account string  `json:"account"`
	Score   float64 `json:"score"`
}

// SuggestionSet groups suggestions for a single field.
type SuggestionSet struct {
	Field       string       `json:"field"` // "from_account", "to_account", "payee"
	Suggestions []Suggestion `json:"suggestions"`
}

// KeywordMatcher holds loaded keyword configurations.
type KeywordMatcher struct {
	ExpenseMarkers      []string
	IncomeMarkers       []string
	TransferMarkers     []string
	CCPaymentMethods    []string
	DebitPaymentMethods []string
	CashMethods         []string
	CustomPayees        map[string]string // "Acme Corp" -> "Liabilities:CorporateCard:Acme"
}

// Parser holds state needed for parsing transactions.
type Parser struct {
	keywords KeywordMatcher
	db       *gorm.DB
	patterns *RegexPatterns
	accounts []string // List of known accounts for matching
}

// NewParser creates a new parser instance with loaded keywords from config.
func NewParser(keywords KeywordMatcher, db *gorm.DB, accounts []string) *Parser {
	return &Parser{
		keywords: keywords,
		db:       db,
		patterns: CompilePatterns(),
		accounts: accounts,
	}
}

// ParseTransaction is the main entry point. It parses a natural language transaction description
// and returns a structured ParseResult with confidence scores and optional suggestions.
//
// The parsing pipeline (8 steps):
// 1. normalizeText() - lowercase, trim whitespace, expand abbreviations
// 2. extractDate() - find and parse date (e.g., "20 Apr", "2026-05-10")
// 3. extractAmount() - find and parse amount (e.g., "15$", "$15.50")
// 4. extractPayee() - identify merchant/payee
// 5. extractHints() - extract account hints from text (e.g., "using amex", "from checking")
// 6. matchAccounts() - use TF-IDF to find best matching accounts
// 7. determineDirection() - classify as expense, income, or transfer
// 8. computeConfidence() - score overall confidence
//
// Returns ParseResult with all extracted fields and confidence scores.
// If confidence is low (<0.75), suggestions array is populated for interactive UI.
func (p *Parser) ParseTransaction(text string) (*ParseResult, error) {
	if text == "" {
		return nil, fmt.Errorf("input text cannot be empty")
	}

	// Step 1: Normalize
	normalized := normalizeText(text)

	// Step 2: Extract date
	date, dateConf, dateErr := p.extractDate(normalized)
	if dateErr != nil {
		// Default to today if date extraction fails
		date = time.Now()
		dateConf = 0.3 // Low confidence
	}

	// Step 3: Extract amount
	amount, currency, amountConf, amountErr := p.extractAmount(normalized)
	if amountErr != nil {
		return nil, fmt.Errorf("failed to extract amount: %w", amountErr)
	}

	// Step 4: Extract payee
	payee, payeeConf := p.extractPayee(normalized)

	// Step 5: Extract hints
	fromHint, toHint := p.extractHints(normalized)

	// Step 6: Match accounts using TF-IDF
	fromAccount, fromScore := p.matchAccounts(fromHint, "from")
	toAccount, toScore := p.matchAccounts(toHint, "to")

	// Step 7: Determine direction (expense/income/transfer)
	direction, directionConf := p.determineDirection(fromHint, toHint)

	// Step 8: Compute overall confidence
	confScores := ConfidenceScores{
		Date:        dateConf,
		Amount:      amountConf,
		Payee:       payeeConf,
		FromAccount: fromScore,
		ToAccount:   toScore,
		Direction:   directionConf,
	}
	confScores.Overall = computeConfidence(confScores)

	// Build suggestions for low-confidence fields
	suggestions := p.buildSuggestions(fromHint, toHint, confScores)

	result := &ParseResult{
		Date:        date,
		Amount:      amount,
		Currency:    currency,
		Payee:       payee,
		FromAccount: fromAccount,
		ToAccount:   toAccount,
		Direction:   direction,
		Confidence:  confScores,
		Suggestions: suggestions,
		Warnings:    []string{},
	}

	return result, nil
}

// normalizeText converts text to lowercase, trims whitespace, and expands abbreviations.
func normalizeText(text string) string {
	// Convert to lowercase and trim whitespace
	text = strings.ToLower(strings.TrimSpace(text))

	// Expand common abbreviations
	abbreviations := map[string]string{
		"cc":    "credit card",
		"debit": "debit card",
		"atm":   "cash withdrawal",
		"amt":   "amount",
		"usd":   "USD",
		"inr":   "INR",
		"eur":   "EUR",
		"gbp":   "GBP",
	}
	for abbr, expanded := range abbreviations {
		// Match word boundaries
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(abbr) + `\b`)
		text = pattern.ReplaceAllString(text, expanded)
	}

	return text
}

// extractDate finds and parses a date from the text.
// Returns the parsed date, confidence (0-1), and error.
func (p *Parser) extractDate(text string) (time.Time, float64, error) {
	now := time.Now()

	// Try ISO format first (2026-05-10, 2026/05/10)
	if match := p.patterns.DateYYYYMMDD.FindStringSubmatch(text); len(match) > 1 {
		dateStr := strings.ReplaceAll(match[1], "/", "-")
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			return date, 0.95, nil // Very high confidence for ISO format
		}
	}

	// Try month+day+year format (10 May 2026, 10-May-2026)
	if match := p.patterns.DateDDMonthYYYY.FindStringSubmatch(text); len(match) > 1 {
		if date, err := parseMonthNameDate(match[1]); err == nil {
			return date, 0.90, nil // High confidence for explicit year
		}
	}

	// Try month+day format (10 May, May 10, 10th May)
	if match := p.patterns.DateDDMonth.FindStringSubmatch(text); len(match) > 1 {
		if date, err := parseMonthNameDate(match[1] + " " + strconv.Itoa(now.Year())); err == nil {
			return date, 0.80, nil // Medium-high confidence (year assumed)
		}
	}

	// Try relative dates (today, yesterday, last friday)
	if match := p.patterns.DateRelative.FindStringSubmatch(text); len(match) > 1 {
		if date, err := parseRelativeDate(match[1]); err == nil {
			return date, 0.85, nil
		}
	}

	// No date found - return today with low confidence
	return now, 0.30, nil
}

// parseMonthNameDate parses dates like "10 May 2026" or "May 10 2026"
func parseMonthNameDate(dateStr string) (time.Time, error) {
	dateStr = strings.TrimSpace(dateStr)
	parts := strings.FieldsFunc(dateStr, func(r rune) bool {
		return r == '-' || r == ' ' || r == ','
	})

	if len(parts) < 2 {
		return time.Time{}, fmt.Errorf("invalid date format")
	}

	var day, month, year int

	// Parse components
	for _, part := range parts {
		part = strings.ToLower(strings.TrimSpace(part))
		if part == "" {
			continue
		}

		// Try parsing as month name
		if m, ok := DatePatterns.MonthNames[part]; ok {
			month = m
			continue
		}

		// Try parsing as number
		if num, err := strconv.Atoi(part); err == nil {
			if num > 31 {
				year = num
			} else if day == 0 {
				day = num
			}
		}
	}

	if month == 0 || day == 0 {
		return time.Time{}, fmt.Errorf("could not parse month or day")
	}

	if year == 0 {
		year = time.Now().Year()
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

// parseRelativeDate parses relative dates like "today", "yesterday", "last friday"
func parseRelativeDate(relStr string) (time.Time, error) {
	relStr = strings.ToLower(strings.TrimSpace(relStr))
	now := time.Now()

	switch relStr {
	case "today":
		return now, nil
	case "tomorrow":
		return now.AddDate(0, 0, 1), nil
	case "yesterday":
		return now.AddDate(0, 0, -1), nil
	default:
		// Handle "last <day>" patterns
		if strings.Contains(relStr, "last") {
			dayNames := []string{"sunday", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday"}
			for i, dayName := range dayNames {
				if strings.Contains(relStr, dayName) {
					daysBack := int(now.Weekday()) - i
					if daysBack <= 0 {
						daysBack += 7
					}
					return now.AddDate(0, 0, -daysBack), nil
				}
			}
		}
	}

	return time.Time{}, fmt.Errorf("could not parse relative date: %s", relStr)
}

// extractAmount finds and parses an amount from the text.
// Returns amount, currency, confidence, and error.
func (p *Parser) extractAmount(text string) (decimal.Decimal, string, float64, error) {
	// Try dollar prefix first ($15.50)
	if match := p.patterns.AmountDollarPrefix.FindStringSubmatch(text); len(match) > 1 {
		amount, err := decimal.NewFromString(match[1])
		if err == nil {
			return amount, "USD", 0.95, nil // High confidence
		}
	}

	// Try amount + currency suffix (15 USD, 15 INR, 15 CAD, 20 cad)
	if match := p.patterns.AmountSuffix.FindStringSubmatch(text); len(match) > 1 {
		amount, err := decimal.NewFromString(match[1])
		if err == nil {
			// Extract currency from the full match
			fullMatch := match[0]
			fullMatchLower := strings.ToLower(fullMatch)
			currency := "USD" // Default

			// Check all currency codes (case-insensitive)
			for currencyCode, mappedCurrency := range CurrencyPatterns.Codes {
				if strings.Contains(fullMatchLower, strings.ToLower(currencyCode)) {
					currency = mappedCurrency
					break
				}
			}
			return amount, currency, 0.90, nil // High confidence
		}
	}

	// Try word form (fifteen dollars, twenty rupees)
	if match := p.patterns.AmountWords.FindStringSubmatch(text); len(match) > 1 {
		if val, ok := WordToNumber[strings.ToLower(match[1])]; ok {
			amount := decimal.NewFromFloat(val)
			currency := "USD"
			if strings.Contains(match[0], "rupees") || strings.Contains(match[0], "inr") {
				currency = "INR"
			}
			return amount, currency, 0.60, nil // Lower confidence for word form
		}
	}

	// No amount found
	return decimal.NewFromInt(0), "", 0, fmt.Errorf("no valid amount found in text")
}

// extractPayee identifies the merchant or payee name from the text.
func (p *Parser) extractPayee(text string) (string, float64) {
	// Try explicit payee markers (at, from, purchased at)
	if match := p.patterns.PayeeMarker.FindStringSubmatch(text); len(match) > 1 {
		payee := strings.TrimSpace(match[1])
		// Check against custom payees first
		for customPayee := range p.keywords.CustomPayees {
			if strings.Contains(strings.ToLower(payee), strings.ToLower(customPayee)) {
				return customPayee, 0.95 // High confidence for custom match
			}
		}
		return payee, 0.80 // Good confidence for explicit marker
	}

	// Try custom payee matching
	for customPayee := range p.keywords.CustomPayees {
		if strings.Contains(strings.ToLower(text), strings.ToLower(customPayee)) {
			return customPayee, 0.85 // High confidence
		}
	}

	// Extract remaining text after removing known keywords and amounts
	cleanedText := text

	// Remove amounts (including currency that follows)
	// Matches: "20", "20.50", "$20", "20$", "20 usd", "20usd", etc.
	amountPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\s*\d+(?:\.\d{2})?`),                                  // $20
		regexp.MustCompile(`\d+(?:\.\d{2})?\s*\$`),                                  // 20$
		regexp.MustCompile(`\d+(?:\.\d{2})?\s*(?:USD|INR|EUR|GBP|CAD|AUD|JPY|CNY)`), // 20 USD (case-insensitive)
	}
	for _, pattern := range amountPatterns {
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}

	// Remove currency codes (even without amounts)
	currencyPattern := regexp.MustCompile(`(?i)\b(?:usd|inr|eur|gbp|cad|aud|jpy|cny)\b`)
	cleanedText = currencyPattern.ReplaceAllString(cleanedText, "")

	// Remove expense markers (categories like "groceries", "gas", etc.)
	for _, marker := range p.keywords.ExpenseMarkers {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(marker) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}

	// Remove income markers
	for _, marker := range p.keywords.IncomeMarkers {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(marker) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}

	// Remove payment method hints (cc, credit card, debit, etc.)
	for _, method := range p.keywords.CCPaymentMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}
	for _, method := range p.keywords.DebitPaymentMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}
	for _, method := range p.keywords.CashMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}

	// Remove common bank/card names (bmo, visa, amex, etc.) - be very aggressive
	bankKeywords := []string{"bmo", "rbc", "td", "cibc", "scotiabank", "visa", "amex", "american express", "mastercard", "discover", "diners", "jcb"}
	for _, bankName := range bankKeywords {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(bankName) + `\b`)
		cleanedText = pattern.ReplaceAllString(cleanedText, "")
	}

	// Remove dates (month names, day numbers, relative dates)
	datePattern := regexp.MustCompile(`(?i)\b(?:jan|feb|mar|apr|may|jun|jul|aug|sep|oct|nov|dec)[a-z]*\b|\b\d{1,2}(?:st|nd|rd|th)?\b|\b(?:today|tomorrow|yesterday)\b`)
	cleanedText = datePattern.ReplaceAllString(cleanedText, "")

	// Remove prepositions and connecting words
	cleanedText = regexp.MustCompile(`(?i)\b(?:from|to|at|using|via|with|for|in|on)\b`).ReplaceAllString(cleanedText, "")

	// Clean up multiple spaces, punctuation, and trim
	cleanedText = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedText, " ")
	cleanedText = regexp.MustCompile(`[,;:-]+`).ReplaceAllString(cleanedText, " ") // Remove common punctuation
	payee := strings.TrimSpace(cleanedText)

	if payee != "" {
		return payee, 0.50 // Lower confidence for inferred payee
	}

	return "", 0.20 // Very low confidence if nothing found
}

// extractHints extracts account hints (e.g., "credit card", "checking", "from amex").
func (p *Parser) extractHints(text string) (fromHint, toHint string) {
	// Find all account hints using the pattern
	if match := p.patterns.AccountHint.FindAllStringSubmatch(text, -1); len(match) > 0 {
		for _, m := range match {
			if len(m) > 1 {
				hint := strings.TrimSpace(m[1])
				// Check if this is a "from" hint or "to" hint based on preceding word
				if idx := strings.Index(text, m[0]); idx >= 0 {
					prefix := text[maxInt(0, idx-10):idx]
					if strings.Contains(strings.ToLower(prefix), "from") || strings.Contains(strings.ToLower(prefix), "using") {
						fromHint = hint
					} else if strings.Contains(strings.ToLower(prefix), "to") || strings.Contains(strings.ToLower(prefix), "into") {
						toHint = hint
					}
				}
			}
		}
	}

	// Try payment hint extraction
	if match := p.patterns.PaymentHint.FindStringSubmatch(text); len(match) > 1 {
		method := strings.TrimSpace(match[1])
		// Payment methods typically indicate "from" account
		if fromHint == "" {
			fromHint = method
		}
	}

	// Extract category/expense type as "to" hint for expense transactions
	// e.g., "groceries", "gas", "utilities" should map to Expenses:Groceries, Expenses:Gas, etc.
	if toHint == "" {
		for _, expenseMarker := range p.keywords.ExpenseMarkers {
			pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(expenseMarker) + `\b`)
			if pattern.MatchString(text) {
				toHint = expenseMarker
				break // Use first matched expense marker as category hint
			}
		}
	}

	// Enhance payment method hint with bank/card name for better matching
	// e.g., "bmo cc" → fromHint becomes "bmo credit card" to improve matching
	if fromHint != "" && strings.Contains(strings.ToLower(fromHint), "cc") {
		// Extract any bank/provider name before "cc"
		bankPattern := regexp.MustCompile(`(\b\w+)\s+(?:cc|credit\s+card|credit\s+card)\b`)
		if match := bankPattern.FindStringSubmatch(text); len(match) > 1 {
			bankName := strings.TrimSpace(match[1])
			// Expand "bmo" to "bmo credit card" for better TF-IDF matching
			if bankName != "" && bankName != "cc" {
				fromHint = bankName + " credit card"
			}
		}
	}

	return
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// matchAccounts uses TF-IDF to find the best matching account(s) for a hint.
func (p *Parser) matchAccounts(hint string, direction string) (string, float64) {
	if hint == "" {
		return "", 0.0
	}

	// Get all accounts from database
	accounts := p.accounts
	if len(accounts) == 0 {
		return "", 0.0
	}

	// Filter accounts by direction if possible
	var candidates []string
	switch direction {
	case "from":
		// "from" accounts are typically Assets or Liabilities
		for _, acc := range accounts {
			if strings.HasPrefix(acc, "Assets:") || strings.HasPrefix(acc, "Liabilities:") {
				candidates = append(candidates, acc)
			}
		}
	case "to":
		// "to" accounts are typically Expenses or Income or Assets/Liabilities (for transfers)
		for _, acc := range accounts {
			if strings.HasPrefix(acc, "Expenses:") || strings.HasPrefix(acc, "Income:") {
				candidates = append(candidates, acc)
			}
		}
	case "expense":
		for _, acc := range accounts {
			if strings.HasPrefix(acc, "Expenses:") {
				candidates = append(candidates, acc)
			}
		}
	case "income":
		for _, acc := range accounts {
			if strings.HasPrefix(acc, "Income:") {
				candidates = append(candidates, acc)
			}
		}
	case "transfer":
		for _, acc := range accounts {
			if strings.HasPrefix(acc, "Assets:") || strings.HasPrefix(acc, "Liabilities:") {
				candidates = append(candidates, acc)
			}
		}
	}

	// Fall back to all accounts if no candidates match direction
	if len(candidates) == 0 {
		candidates = accounts
	}

	// Compute similarity scores with enhanced matching
	bestScore := 0.0
	bestAccount := ""
	hintTokens := tokenizeHint(hint)
	hintLower := strings.ToLower(hint)

	for _, account := range candidates {
		accountTokens := tokenizeHint(account)
		accountLower := strings.ToLower(account)

		// Base TF-IDF similarity
		similarity := cosineSimilarity(hintTokens, accountTokens)

		// Check for exact substring matches in the account (very high boost)
		// e.g., "bmo credit card" contains "bmo" which appears in account name
		hintWords := strings.FieldsFunc(hintLower, func(r rune) bool {
			return r == ' ' || r == '-'
		})
		for _, word := range hintWords {
			if len(word) > 2 && strings.Contains(accountLower, word) {
				// Strong boost for substring match
				similarity += 0.4
				break
			}
		}

		// Boost score for payment method matching
		// Check for bank/card name matches (e.g., "bmo credit card" with "Liabilities:CreditCard:BMO")
		if direction == "from" || direction == "transfer" || (direction == "" && (strings.HasPrefix(account, "Liabilities:") || strings.HasPrefix(account, "Assets:"))) {
			// Look for known bank/card keywords that appear in both hint and account
			for _, keyword := range []string{"bmo", "visa", "amex", "mastercard", "discover", "rbc", "td", "chase", "cibc", "scotiabank"} {
				hintHasKeyword := strings.Contains(hintLower, keyword)
				accountHasKeyword := strings.Contains(accountLower, keyword)
				if hintHasKeyword && accountHasKeyword {
					// Significant boost for matching bank/card names
					similarity += 0.5
					break
				}
			}

			// Also check for "credit" + "card" or "cc" patterns
			if strings.Contains(hintLower, "credit") && (strings.Contains(accountLower, "creditcard") || strings.Contains(accountLower, "credit card") || strings.Contains(accountLower, "cc")) {
				similarity += 0.3
			}
		}

		if similarity > bestScore {
			bestScore = similarity
			bestAccount = account
		}
	}

	return bestAccount, bestScore
}

// tokenizeHint converts a string into a token frequency map for similarity computation.
func tokenizeHint(s string) map[string]float64 {
	tokens := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ' ' || r == '.' || r == '(' || r == ')' || r == '/' || r == ':' || r == '-'
	})

	tokenFreq := make(map[string]float64)
	for _, token := range tokens {
		if token != "" {
			tokenFreq[token]++
		}
	}
	return tokenFreq
}

// cosineSimilarity computes cosine similarity between two token vectors.
func cosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	// Compute dot product and magnitudes
	dotProduct := 0.0
	magnitudeA := 0.0
	magnitudeB := 0.0

	// Build set of all tokens
	allTokens := make(map[string]bool)
	for token := range a {
		allTokens[token] = true
	}
	for token := range b {
		allTokens[token] = true
	}

	for token := range allTokens {
		valA := a[token]
		valB := b[token]
		dotProduct += valA * valB
		magnitudeA += valA * valA
		magnitudeB += valB * valB
	}

	if magnitudeA == 0 || magnitudeB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(magnitudeA) * math.Sqrt(magnitudeB))
}

// determineDirection classifies the transaction as "expense", "income", or "transfer".
func (p *Parser) determineDirection(fromHint, toHint string) (string, float64) {
	// Lowercase the text for matching
	lowerFrom := strings.ToLower(fromHint)
	lowerTo := strings.ToLower(toHint)
	combinedHints := lowerFrom + " " + lowerTo

	// Get keyword matcher
	keywords := p.keywords

	// Check for transfer keywords (move between accounts)
	for _, keyword := range keywords.TransferMarkers {
		if strings.Contains(combinedHints, strings.ToLower(keyword)) {
			return "transfer", 0.85
		}
	}

	// Check for income keywords
	for _, keyword := range keywords.IncomeMarkers {
		if strings.Contains(combinedHints, strings.ToLower(keyword)) {
			return "income", 0.85
		}
	}

	// Check for expense keywords
	for _, keyword := range keywords.ExpenseMarkers {
		if strings.Contains(combinedHints, strings.ToLower(keyword)) {
			return "expense", 0.85
		}
	}

	// Fallback: assume expense if we have a payment method hint
	if fromHint != "" {
		return "expense", 0.5
	}

	// Default: expense with low confidence
	return "expense", 0.3
}

// computeConfidence calculates the overall confidence score from individual field confidences.
// Uses weighted average: date(0.1) + amount(0.3) + payee(0.2) + from(0.2) + to(0.2).
func computeConfidence(scores ConfidenceScores) float64 {
	// Weighted average:
	//  - amount: 0.30 (most critical)
	//  - from_account: 0.25
	//  - to_account: 0.25
	//  - payee: 0.15
	//  - date: 0.05 (less critical, defaults to today)
	weighted := (scores.Amount * 0.30) +
		(scores.FromAccount * 0.25) +
		(scores.ToAccount * 0.25) +
		(scores.Payee * 0.15) +
		(scores.Date * 0.05)

	// Clamp to [0.0, 1.0]
	if weighted < 0.0 {
		return 0.0
	}
	if weighted > 1.0 {
		return 1.0
	}
	return weighted
}

// buildSuggestions creates suggestion sets for fields with confidence < 0.75.
func (p *Parser) buildSuggestions(fromHint, toHint string, scores ConfidenceScores) []SuggestionSet {
	var suggestions []SuggestionSet

	// If fromAccount confidence is low, suggest top 3 matches
	if scores.FromAccount < 0.75 && fromHint != "" {
		fromSuggestions := p.getTopSuggestions(fromHint, "from", 3)
		if len(fromSuggestions) > 0 {
			suggestions = append(suggestions, SuggestionSet{
				Field:       "from_account",
				Suggestions: fromSuggestions,
			})
		}
	}

	// If toAccount confidence is low, suggest top 3 matches
	if scores.ToAccount < 0.75 && toHint != "" {
		toSuggestions := p.getTopSuggestions(toHint, "to", 3)
		if len(toSuggestions) > 0 {
			suggestions = append(suggestions, SuggestionSet{
				Field:       "to_account",
				Suggestions: toSuggestions,
			})
		}
	}

	return suggestions
}

// getTopSuggestions returns the top N matching accounts for a hint.
func (p *Parser) getTopSuggestions(hint string, _ string, limit int) []Suggestion {
	if hint == "" || len(p.accounts) == 0 {
		return []Suggestion{}
	}

	// Compute similarity scores for all accounts
	type scoreAccount struct {
		account string
		score   float64
	}
	var scores []scoreAccount

	hintTokens := tokenizeHint(hint)
	for _, account := range p.accounts {
		accountTokens := tokenizeHint(account)
		similarity := cosineSimilarity(hintTokens, accountTokens)
		if similarity > 0.0 {
			scores = append(scores, scoreAccount{account, similarity})
		}
	}

	// Sort by score descending
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[j].score > scores[i].score {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// Return top N
	var result []Suggestion
	for i := 0; i < limit && i < len(scores); i++ {
		result = append(result, Suggestion{
			Account: scores[i].account,
			Score:   scores[i].score,
		})
	}

	return result
}

// LogToTrainingDatabase logs the parsing result for ML training (Phase 3).
// Called asynchronously after user confirms the transaction.
func (p *Parser) LogToTrainingDatabase(result *ParseResult, actualFromAccount, actualToAccount string) error {
	// TODO: Implement ML training data logging
	// - Insert into parser_training_log table
	// - Store input, predictions, confidence scores
	// - Store actual user-confirmed accounts
	// - Mark if user corrected any suggestions
	// - Non-blocking (async)
	return nil
}
