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

// Span represents a consumed region of text by byte offset (start and end indices).
// Used for span-masking to track which parts of the input have been processed.
type Span struct {
	Start int // Inclusive start byte offset
	End   int // Exclusive end byte offset
}

// SpanMask tracks consumed spans in the original text to prevent double-counting tokens.
// Immutable source text + mask prevents destructive manipulation.
type SpanMask struct {
	Source        string // Immutable original input text
	ConsumedSpans []Span // Ordered list of non-overlapping spans that have been consumed
}

// NewSpanMask creates a new span mask for tracking consumed regions during extraction.
func NewSpanMask(source string) *SpanMask {
	return &SpanMask{
		Source:        source,
		ConsumedSpans: []Span{},
	}
}

// RecordSpan adds a consumed span to the mask. Spans should not overlap.
func (m *SpanMask) RecordSpan(start, end int) {
	if start >= 0 && end > start && end <= len(m.Source) {
		m.ConsumedSpans = append(m.ConsumedSpans, Span{Start: start, End: end})
	}
}

// GetUnconsumedText returns the portions of source text that haven't been consumed yet.
// Returns the concatenation of all gaps between consumed spans.
func (m *SpanMask) GetUnconsumedText() string {
	if len(m.ConsumedSpans) == 0 {
		return m.Source
	}

	var result strings.Builder
	lastEnd := 0

	for _, span := range m.ConsumedSpans {
		if span.Start > lastEnd {
			result.WriteString(m.Source[lastEnd:span.Start])
			result.WriteString(" ")
		}
		lastEnd = span.End
	}

	if lastEnd < len(m.Source) {
		result.WriteString(m.Source[lastEnd:])
	}

	return strings.TrimSpace(result.String())
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
	keywords         KeywordMatcher
	db               *gorm.DB
	patterns         *RegexPatterns
	accounts         []string            // List of known accounts for matching
	accountTokens    map[string][]string // Pre-cached tokens for each account (for fast matching)
	historicalPayees map[string]int      // Set of payees from past transactions with frequency
	genericTokens    map[string]bool     // Cache of generic/structural tokens to filter
	spanMask         *SpanMask           // Current span mask for tracking consumed regions during parsing
}

// NewParser creates a new parser instance with loaded keywords from config.
func NewParser(keywords KeywordMatcher, db *gorm.DB, accounts []string) *Parser {
	p := &Parser{
		keywords:         keywords,
		db:               db,
		patterns:         CompilePatterns(),
		accounts:         accounts,
		accountTokens:    make(map[string][]string),
		historicalPayees: make(map[string]int),
		genericTokens: map[string]bool{
			"liabilities": true, "assets": true, "expenses": true, "income": true,
			"creditcard": true, "credit": true, "card": true, "cc": true,
			"checking": true, "chequing": true, "savings": true, "account": true,
			"bank": true, "cad": true, "usd": true, "eur": true, "gbp": true,
			"aud": true, "jpy": true, "cny": true, "inr": true,
		},
	}

	// Pre-compute token cache for all accounts
	p.cacheAccountTokens()

	// Load historical payees from database if available
	if db != nil {
		p.loadHistoricalPayees()
	}

	return p
}

// cacheAccountTokens pre-extracts meaningful tokens from all accounts for fast matching.
// Stores tokens in p.accountTokens map to avoid re-tokenization on every match call.
func (p *Parser) cacheAccountTokens() {
	for _, account := range p.accounts {
		tokens := p.extractMeaningfulTokens(account)
		p.accountTokens[account] = tokens
	}
}

// extractMeaningfulTokens extracts non-generic tokens from a string (account name or hint).
// Filters out structural tokens like "Liabilities", "creditcard", currency codes, etc.
func (p *Parser) extractMeaningfulTokens(s string) []string {
	words := strings.FieldsFunc(strings.ToLower(s), func(r rune) bool {
		return r == ' ' || r == '-' || r == ':' || r == '(' || r == ')'
	})

	var meaningful []string
	for _, token := range words {
		if len(token) > 2 && !p.genericTokens[token] {
			meaningful = append(meaningful, token)
		}
	}
	return meaningful
}

// loadHistoricalPayees loads payee names from past transactions in the database.
// Builds a frequency map of payees for suggestions and deduplication.
func (p *Parser) loadHistoricalPayees() {
	// TODO: Query database for distinct payees from past transactions
	// Example: SELECT DISTINCT payee FROM postings WHERE payee IS NOT NULL
	// Store in p.historicalPayees with frequency count
	// This helps identify common merchants for future transactions
	if p.db == nil {
		return
	}

	// Placeholder: database query would go here
	// For now, initialize as empty map
}

// ParseTransaction is the main entry point. It parses a natural language transaction description
// and returns a structured ParseResult with confidence scores and optional suggestions.
//
// The parsing pipeline uses ordered extraction with span-masking to prevent double-counting:
// 1. normalizeText() - lowercase, trim whitespace, expand abbreviations
// 2. extractDate() - find and parse date, record consumed span
// 3. extractAmount() - find and parse amount, record consumed span
// 4. extractHints() - extract from/to account hints from unconsumed text
// 5. matchAccounts() - use TF-IDF to find best matching accounts
// 6. extractPayee() - identify merchant from remaining text
// 7. determineDirection() - classify as expense, income, or transfer
// 8. computeConfidence() - score overall confidence
//
// Span-masking preserves the original text and tracks consumed regions,
// allowing downstream steps to access full text if needed for context.
// This reduces bugs from destructive text manipulation.
//
// Returns ParseResult with all extracted fields and confidence scores.
// If confidence is low (<0.75), suggestions array is populated for interactive UI.
func (p *Parser) ParseTransaction(text string) (*ParseResult, error) {
	if text == "" {
		return nil, fmt.Errorf("input text cannot be empty")
	}

	// Step 1: Normalize
	normalized := normalizeText(text)

	// Initialize span mask to track consumed regions during extraction
	p.spanMask = NewSpanMask(normalized)

	// Step 2: Extract date (with span tracking)
	date, dateConf, dateErr := p.extractDate(normalized)
	if dateErr != nil {
		// Default to today if date extraction fails
		date = time.Now()
		dateConf = 0.3 // Low confidence
	}

	// Step 3: Extract amount (with span tracking)
	amount, currency, amountConf, amountErr := p.extractAmount(normalized)
	if amountErr != nil {
		return nil, fmt.Errorf("failed to extract amount: %w", amountErr)
	}

	// Get unconsumed text for next steps
	unconsumedText := p.spanMask.GetUnconsumedText()

	// Step 4: Extract hints from unconsumed text
	fromHint, toHint := p.extractHints(unconsumedText)

	// Step 5: Determine direction (expense/income/transfer)
	direction, directionConf := p.determineDirection(unconsumedText, fromHint, toHint)

	// Step 6: Match accounts using joint role-aware scoring
	fromAccount, toAccount, fromScore, toScore := p.matchAccountPair(fromHint, toHint, unconsumedText, direction)

	// Step 7: Extract payee from unconsumed text
	payee, payeeConf := p.extractPayee(unconsumedText)

	// Step 8: Compute overall confidence

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
		"xfer":  "transfer",
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
// Records the consumed span in the span mask for tracking.
// Returns the parsed date, confidence (0-1), and error.
func (p *Parser) extractDate(text string) (time.Time, float64, error) {
	now := time.Now()

	// Try ISO format first (2026-05-10, 2026/05/10)
	if indices := p.patterns.DateYYYYMMDD.FindStringSubmatchIndex(text); len(indices) >= 4 {
		match := text[indices[2]:indices[3]]
		dateStr := strings.ReplaceAll(match, "/", "-")
		if date, err := time.Parse("2006-01-02", dateStr); err == nil {
			// Record consumed span (full match is indices[0]:indices[1])
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return date, 0.95, nil // Very high confidence for ISO format
		}
	}

	// Try month+day+year format (10 May 2026, 10-May-2026)
	if indices := p.patterns.DateDDMonthYYYY.FindStringSubmatchIndex(text); len(indices) >= 4 {
		match := text[indices[2]:indices[3]]
		if date, err := parseMonthNameDate(match); err == nil {
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return date, 0.90, nil // High confidence for explicit year
		}
	}

	// Try month+day format (10 May, May 10, 10th May)
	if indices := p.patterns.DateDDMonth.FindStringSubmatchIndex(text); len(indices) >= 4 {
		match := text[indices[2]:indices[3]]
		if date, err := parseMonthNameDate(match + " " + strconv.Itoa(now.Year())); err == nil {
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return date, 0.80, nil // Medium-high confidence (year assumed)
		}
	}

	// Try relative dates (today, yesterday, last friday)
	if indices := p.patterns.DateRelative.FindStringSubmatchIndex(text); len(indices) >= 4 {
		match := text[indices[2]:indices[3]]
		if date, err := parseRelativeDate(match); err == nil {
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return date, 0.85, nil
		}
	}

	// No date found - return today with low confidence (no span recorded)
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
// Records the consumed span in the span mask for tracking.
// Returns amount, currency, confidence, and error.
func (p *Parser) extractAmount(text string) (decimal.Decimal, string, float64, error) {
	// Try dollar prefix first ($15.50)
	if indices := p.patterns.AmountDollarPrefix.FindStringSubmatchIndex(text); len(indices) >= 4 {
		amountStr := text[indices[2]:indices[3]]
		amount, err := decimal.NewFromString(amountStr)
		if err == nil {
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return amount, "USD", 0.95, nil // High confidence
		}
	}

	// Try amount + currency suffix (15 USD, 15 INR, 15 CAD, 20 cad)
	if indices := p.patterns.AmountSuffix.FindStringSubmatchIndex(text); len(indices) >= 4 {
		amountStr := text[indices[2]:indices[3]]
		amount, err := decimal.NewFromString(amountStr)
		if err == nil {
			// Extract currency from the full match
			fullMatch := text[indices[0]:indices[1]]
			fullMatchLower := strings.ToLower(fullMatch)
			currency := "USD" // Default

			// Check all currency codes (case-insensitive)
			for currencyCode, mappedCurrency := range CurrencyPatterns.Codes {
				if strings.Contains(fullMatchLower, strings.ToLower(currencyCode)) {
					currency = mappedCurrency
					break
				}
			}
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return amount, currency, 0.90, nil // High confidence
		}
	}

	// Try word form (fifteen dollars, twenty rupees)
	if indices := p.patterns.AmountWords.FindStringSubmatchIndex(text); len(indices) >= 4 {
		amountStr := text[indices[2]:indices[3]]
		if val, ok := WordToNumber[strings.ToLower(amountStr)]; ok {
			amount := decimal.NewFromFloat(val)
			currency := "USD"
			fullMatch := text[indices[0]:indices[1]]
			if strings.Contains(fullMatch, "rupees") || strings.Contains(fullMatch, "inr") {
				currency = "INR"
			}
			if p.spanMask != nil {
				p.spanMask.RecordSpan(indices[0], indices[1])
			}
			return amount, currency, 0.60, nil // Lower confidence for word form
		}
	}

	// No amount found
	return decimal.NewFromInt(0), "", 0, fmt.Errorf("no valid amount found in text")
}

// extractPayee identifies the merchant or payee name from the text.
func (p *Parser) extractPayee(text string) (string, float64) {
	var payee string
	var confidence float64

	// Try explicit payee markers (at, from, purchased at)
	if match := p.patterns.PayeeMarker.FindStringSubmatch(text); len(match) > 1 {
		payee = strings.TrimSpace(match[1])
		confidence = 0.80 // Good confidence for explicit marker

		// Check against custom payees first
		for customPayee := range p.keywords.CustomPayees {
			if strings.Contains(strings.ToLower(payee), strings.ToLower(customPayee)) {
				return customPayee, 0.95 // High confidence for custom match
			}
		}

		// Continue to filter the extracted payee (don't return immediately)
	} else {
		// Try custom payee matching if no explicit marker
		for customPayee := range p.keywords.CustomPayees {
			if strings.Contains(strings.ToLower(text), strings.ToLower(customPayee)) {
				return customPayee, 0.85 // High confidence
			}
		}

		payee = text
		confidence = 0.50 // Lower confidence for inferred payee
	}

	// Clean the payee by removing amounts, currency, and payment methods
	cleanedPayee := payee

	// Remove amounts (including currency that follows)
	// Matches: "20", "20.50", "$20", "20$", "20 usd", "20usd", etc.
	amountPatterns := []*regexp.Regexp{
		regexp.MustCompile(`\$\s*\d+(?:\.\d{2})?`),                                  // $20
		regexp.MustCompile(`\d+(?:\.\d{2})?\s*\$`),                                  // 20$
		regexp.MustCompile(`\d+(?:\.\d{2})?\s*(?:USD|INR|EUR|GBP|CAD|AUD|JPY|CNY)`), // 20 USD (case-insensitive)
	}
	for _, pattern := range amountPatterns {
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}

	// Remove currency codes (even without amounts)
	currencyPattern := regexp.MustCompile(`(?i)\b(?:usd|inr|eur|gbp|cad|aud|jpy|cny)\b`)
	cleanedPayee = currencyPattern.ReplaceAllString(cleanedPayee, "")

	// Remove expense markers (categories like "groceries", "gas", etc.)
	for _, marker := range p.keywords.ExpenseMarkers {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(marker) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}

	// Remove income markers
	for _, marker := range p.keywords.IncomeMarkers {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(marker) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}
	for _, method := range p.keywords.CCPaymentMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}
	for _, method := range p.keywords.DebitPaymentMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}
	for _, method := range p.keywords.CashMethods {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(method) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}

	// Remove common bank/card names - extracted dynamically from accounts for better coverage
	bankKeywords := []string{
		"bmo", "rbc", "td", "cibc", "scotiabank",
		"visa", "amex", "american express", "mastercard", "discover", "diners", "jcb",
		"neo", "tangerine", "wealthsimple", "questrade", "interactive brokers",
	}

	// Dynamically add any meaningful account name tokens to filter from payee
	seenKeywords := make(map[string]bool)
	for _, kw := range bankKeywords {
		seenKeywords[strings.ToLower(kw)] = true
	}
	for _, account := range p.accounts {
		tokens := p.extractMeaningfulTokens(account)
		for _, token := range tokens {
			if len(token) > 2 && !seenKeywords[token] {
				bankKeywords = append(bankKeywords, token)
				seenKeywords[token] = true
			}
		}
	}

	for _, bankName := range bankKeywords {
		pattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(bankName) + `\b`)
		cleanedPayee = pattern.ReplaceAllString(cleanedPayee, "")
	}

	// Remove prepositions and connecting words
	cleanedPayee = regexp.MustCompile(`(?i)\b(?:from|to|at|using|via|with|for|in|on)\b`).ReplaceAllString(cleanedPayee, "")

	// Clean up multiple spaces, punctuation, and trim
	cleanedPayee = regexp.MustCompile(`\s+`).ReplaceAllString(cleanedPayee, " ")
	cleanedPayee = regexp.MustCompile(`[,;:-]+`).ReplaceAllString(cleanedPayee, " ") // Remove common punctuation
	cleanedPayee = strings.TrimSpace(cleanedPayee)

	if cleanedPayee != "" {
		return cleanedPayee, confidence
	}

	// If initial extraction collapses to empty (e.g., "from bmo credit card for ... at no frills"),
	// prefer a merchant that appears in an explicit "at <payee>" segment.
	if atMatch := regexp.MustCompile(`(?i)\bat\s+([A-Za-z\s&]+?)(?:\s+(?:for|with|using|via|on|from)|[,\.]|$)`).FindStringSubmatch(text); len(atMatch) > 1 {
		merchant := strings.TrimSpace(atMatch[1])
		if merchant != "" {
			return merchant, 0.70
		}
	}

	// If cleaning removed everything, return the original payee trimmed
	return strings.TrimSpace(payee), 0.30 // Very low confidence if cleaning removed everything
}

// extractHints extracts account hints (e.g., "credit card", "checking", "from amex").
func (p *Parser) extractHints(text string) (fromHint, toHint string) {
	// Prefer explicit transfer form first: "from <account> to <account>"
	transferPattern := regexp.MustCompile(`(?i)\bfrom\s+([A-Za-z0-9\s:-]+?)\s+to\s+([A-Za-z0-9\s:-]+?)(?:[,\.]|$)`)
	if match := transferPattern.FindStringSubmatch(text); len(match) > 2 {
		fromHint = strings.TrimSpace(match[1])
		toHint = strings.TrimSpace(match[2])
	}

	// Find all account hints using the pattern
	if match := p.patterns.AccountHint.FindAllStringSubmatch(text, -1); len(match) > 0 {
		for _, m := range match {
			if len(m) > 1 {
				hint := strings.TrimSpace(m[1])
				if hint == "" {
					continue
				}

				if strings.HasPrefix(strings.ToLower(m[0]), "from ") || strings.HasPrefix(strings.ToLower(m[0]), "using ") ||
					strings.HasPrefix(strings.ToLower(m[0]), "with ") || strings.HasPrefix(strings.ToLower(m[0]), "on ") {
					if fromHint == "" {
						fromHint = hint
					}
					continue
				}
				if strings.HasPrefix(strings.ToLower(m[0]), "to ") || strings.HasPrefix(strings.ToLower(m[0]), "into ") {
					if toHint == "" {
						toHint = hint
					}
					continue
				}

				// Check if this is a "from" hint or "to" hint based on preceding word
				if idx := strings.Index(text, m[0]); idx >= 0 {
					prefix := text[maxInt(0, idx-10):idx]
					prefixLower := strings.ToLower(prefix)
					if fromHint == "" && (strings.Contains(prefixLower, "from") || strings.Contains(prefixLower, "using") ||
						strings.Contains(prefixLower, "on") || strings.Contains(prefixLower, "with")) {
						fromHint = hint
					} else if toHint == "" && (strings.Contains(prefixLower, "to") || strings.Contains(prefixLower, "into")) {
						toHint = hint
					}
				}
			}
		}
	}

	// Try payment hint extraction
	if match := p.patterns.PaymentHint.FindStringSubmatch(text); len(match) > 1 {
		method := strings.TrimSpace(match[1])
		// Payment methods typically indicate "from" account.
		// If provider hint already exists (e.g., "neo"), keep both tokens ("neo credit card").
		if fromHint == "" {
			fromHint = method
		} else if !strings.Contains(strings.ToLower(fromHint), strings.ToLower(method)) {
			fromHint = strings.TrimSpace(fromHint + " " + method)
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

	// Fallback: Extract bare account-name tokens (without explicit keywords)
	// This handles cases like "15 inr icici hyd for shopping clothing"
	// where "icici hyd" appears without "from" keyword.
	// Strategy: find tokens from Assets/Liabilities accounts for the "from" hint.
	if fromHint == "" {
		bareAccountTokens := p.findBareAccountTokensByPrefix(text, "Assets:", "Liabilities:")
		if len(bareAccountTokens) > 0 {
			fromHint = strings.Join(bareAccountTokens, " ")
		}
	}

	// Secondary fallback: infer category/account tokens for "to" from Expenses/Income accounts.
	// This captures compact phrases like "... for shopping clothing" even without explicit "to".
	if toHint == "" {
		bareCategoryTokens := p.findBareAccountTokensByPrefix(text, "Expenses:", "Income:")
		if len(bareCategoryTokens) > 0 {
			toHint = strings.Join(bareCategoryTokens, " ")
		}
	}

	// Enhance payment method hint with bank/card name for better matching
	// e.g., "bmo cc" → becomes "bmo credit card" after normalization
	// We need to extract the bank name from hints like "bmo credit card"
	if fromHint != "" {
		fromHintLower := strings.ToLower(fromHint)
		// Check if this is a credit card hint (contains "credit", "card", or "cc")
		if strings.Contains(fromHintLower, "credit") || strings.Contains(fromHintLower, "card") || strings.Contains(fromHintLower, "cc") {
			// Extract bank/provider name (usually the first meaningful token before "credit card")
			// e.g., "bmo credit card" → "bmo"
			bankPattern := regexp.MustCompile(`(\b\w+)\s+(?:credit\s+card|cc)\b`)
			if match := bankPattern.FindStringSubmatch(text); len(match) > 1 {
				bankName := strings.TrimSpace(match[1])
				// Keep "cc" token in hint so accounts matching both bank token and cc token rank higher.
				if bankName != "" && bankName != "credit" && bankName != "card" && bankName != "cc" {
					fromHint = bankName + " cc"
				}
			}
		}
	}

	return
}

// findBareAccountTokensByPrefix extracts tokens that appear in known account names
// from the input text, without requiring explicit keywords like "from" or "using".
// It only considers accounts whose name starts with one of the provided prefixes.
// Returns tokens in order of appearance that match account names.
func (p *Parser) findBareAccountTokensByPrefix(text string, prefixes ...string) []string {
	// Build a set of meaningful tokens from matching account roots only.
	accountTokenMap := make(map[string]bool)
	for _, account := range p.accounts {
		if len(prefixes) > 0 {
			matchedPrefix := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(account, prefix) {
					matchedPrefix = true
					break
				}
			}
			if !matchedPrefix {
				continue
			}
		}

		tokens := p.extractMeaningfulTokens(account)
		for _, token := range tokens {
			accountTokenMap[token] = true
		}
	}

	// Tokenize the input text
	words := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return r == ' ' || r == ',' || r == ';' || r == '.' || r == ':' || r == '-'
	})

	// Collect tokens that appear in both input and account names, preserving order
	var result []string
	seen := make(map[string]bool)
	for _, word := range words {
		if len(word) > 2 && accountTokenMap[word] && !seen[word] {
			result = append(result, word)
			seen[word] = true
		}
	}

	return result
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// candidateAccountsForRole returns account candidates for from/to role based on direction.
func (p *Parser) candidateAccountsForRole(role, direction string) []string {
	var candidates []string

	switch role {
	case "from":
		if direction == "income" {
			for _, acc := range p.accounts {
				if strings.HasPrefix(acc, "Income:") {
					candidates = append(candidates, acc)
				}
			}
		} else {
			for _, acc := range p.accounts {
				if strings.HasPrefix(acc, "Assets:") || strings.HasPrefix(acc, "Liabilities:") {
					candidates = append(candidates, acc)
				}
			}
		}
	case "to":
		switch direction {
		case "transfer":
			for _, acc := range p.accounts {
				if strings.HasPrefix(acc, "Assets:") || strings.HasPrefix(acc, "Liabilities:") {
					candidates = append(candidates, acc)
				}
			}
		case "income":
			for _, acc := range p.accounts {
				if strings.HasPrefix(acc, "Assets:") || strings.HasPrefix(acc, "Liabilities:") {
					candidates = append(candidates, acc)
				}
			}
		default:
			for _, acc := range p.accounts {
				if strings.HasPrefix(acc, "Expenses:") || strings.HasPrefix(acc, "Income:") {
					candidates = append(candidates, acc)
				}
			}
		}
	}

	if len(candidates) == 0 {
		return p.accounts
	}

	return candidates
}

// scoreAccountAgainstHint computes a similarity score using token overlap + TF-IDF style cosine.
func (p *Parser) scoreAccountAgainstHint(hint, account string) float64 {
	if hint == "" || account == "" {
		return 0.0
	}

	hintTokens := normalizeTokensForMatching(tokenizeHint(hint))
	accountTokens := normalizeTokensForMatching(tokenizeHint(account))
	meaningfulHintTokens := p.extractMeaningfulTokens(hint)
	meaningfulAccountTokens := p.accountTokens[account]

	similarity := cosineSimilarity(hintTokens, accountTokens)

	sharedTokens := 0.0
	for token := range hintTokens {
		if _, ok := accountTokens[token]; ok {
			sharedTokens++
		}
	}
	similarity += 0.2 * sharedTokens

	for _, hintToken := range meaningfulHintTokens {
		for _, accountToken := range meaningfulAccountTokens {
			if hintToken == accountToken {
				similarity += 0.6
			} else if strings.Contains(accountToken, hintToken) || strings.Contains(hintToken, accountToken) {
				similarity += 0.3
			}
		}
	}

	return similarity
}

// pickBestAccount selects the highest-scoring account for a hint from candidate accounts.
func (p *Parser) pickBestAccount(hint string, candidates []string, excluded string) (string, float64) {
	bestScore := 0.0
	bestAccount := ""

	for _, account := range candidates {
		if excluded != "" && account == excluded {
			continue
		}

		score := p.scoreAccountAgainstHint(hint, account)
		if score > bestScore {
			bestScore = score
			bestAccount = account
		}
	}

	return bestAccount, bestScore
}

// matchAccountPair scores from/to accounts jointly using full text plus role-specific hints.
// This supports compact phrases where explicit "from/to" markers may be missing.
func (p *Parser) matchAccountPair(fromHint, toHint, fullText, direction string) (string, string, float64, float64) {
	combinedHint := strings.TrimSpace(fullText)

	fromCandidates := p.candidateAccountsForRole("from", direction)
	toCandidates := p.candidateAccountsForRole("to", direction)

	fromPrimaryHint := fromHint
	if fromPrimaryHint == "" {
		fromPrimaryHint = combinedHint
	}

	toPrimaryHint := toHint
	if toPrimaryHint == "" {
		toPrimaryHint = combinedHint
	}

	fromAccount, fromPrimaryScore := p.pickBestAccount(fromPrimaryHint, fromCandidates, "")
	toAccount, toPrimaryScore := p.pickBestAccount(toPrimaryHint, toCandidates, "")

	fromScore := fromPrimaryScore
	toScore := toPrimaryScore

	if fromAccount != "" && combinedHint != "" && fromPrimaryHint != combinedHint {
		fullTextScore := p.scoreAccountAgainstHint(combinedHint, fromAccount)
		fromScore = (0.7 * fromPrimaryScore) + (0.3 * fullTextScore)
	}

	if toAccount != "" && combinedHint != "" && toPrimaryHint != combinedHint {
		fullTextScore := p.scoreAccountAgainstHint(combinedHint, toAccount)
		toScore = (0.7 * toPrimaryScore) + (0.3 * fullTextScore)
	}

	// For transfer-like flows where candidate pools overlap, avoid selecting same account on both sides.
	if fromAccount != "" && toAccount == fromAccount {
		altToAccount, altToScore := p.pickBestAccount(toPrimaryHint, toCandidates, fromAccount)
		if altToAccount != "" {
			toAccount = altToAccount
			toScore = altToScore
		}
	}

	return fromAccount, toAccount, fromScore, toScore
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
	meaningfulHintTokens := p.extractMeaningfulTokens(hint)
	hintTokens := normalizeTokensForMatching(tokenizeHint(hint))

	for _, account := range candidates {
		accountTokens := normalizeTokensForMatching(tokenizeHint(account))

		// Base TF-IDF similarity using tokenized forms
		similarity := cosineSimilarity(hintTokens, accountTokens)

		// Prefer accounts that match more hint tokens (e.g., "neo cc" should beat "neo" only).
		sharedTokens := 0.0
		for token := range hintTokens {
			if _, ok := accountTokens[token]; ok {
				sharedTokens++
			}
		}
		similarity += 0.2 * sharedTokens

		// Use cached meaningful tokens for account
		meaningfulAccountTokens := p.accountTokens[account]

		// Check if any meaningful hint token matches any account token
		for _, hintToken := range meaningfulHintTokens {
			for _, accountToken := range meaningfulAccountTokens {
				if hintToken == accountToken {
					// Exact token match - strong boost
					similarity += 0.6
				} else if strings.Contains(accountToken, hintToken) || strings.Contains(hintToken, accountToken) {
					// Substring match - moderate boost
					similarity += 0.3
				}
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

// normalizeTokensForMatching adds alias tokens so equivalent terms match naturally.
// Example: "credit card" and "creditcard" both add an implicit "cc" token.
func normalizeTokensForMatching(tokens map[string]float64) map[string]float64 {
	normalized := make(map[string]float64, len(tokens)+2)
	for k, v := range tokens {
		normalized[k] = v
	}

	if normalized["creditcard"] > 0 || normalized["creditc"] > 0 ||
		normalized["cc"] > 0 || (normalized["credit"] > 0 && normalized["card"] > 0) {
		normalized["cc"] += 1
	}

	return normalized
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
func (p *Parser) determineDirection(text, fromHint, toHint string) (string, float64) {
	// Lowercase the text for matching
	lowerText := strings.ToLower(text)
	lowerFrom := strings.ToLower(fromHint)
	lowerTo := strings.ToLower(toHint)
	combinedHints := strings.TrimSpace(lowerFrom + " " + lowerTo)

	// Get keyword matcher
	keywords := p.keywords

	// Check for transfer keywords (move between accounts)
	for _, keyword := range keywords.TransferMarkers {
		lowerKeyword := strings.ToLower(keyword)
		if strings.Contains(lowerText, lowerKeyword) || strings.Contains(combinedHints, lowerKeyword) {
			return "transfer", 0.85
		}
	}

	// Strong transfer cue: explicit from ... to ... structure with both hints present.
	if fromHint != "" && toHint != "" && (strings.Contains(lowerText, " from ") && strings.Contains(lowerText, " to ")) {
		return "transfer", 0.8
	}

	// Check for income keywords
	for _, keyword := range keywords.IncomeMarkers {
		lowerKeyword := strings.ToLower(keyword)
		if strings.Contains(lowerText, lowerKeyword) || strings.Contains(combinedHints, lowerKeyword) {
			return "income", 0.85
		}
	}

	// Check for expense keywords
	for _, keyword := range keywords.ExpenseMarkers {
		lowerKeyword := strings.ToLower(keyword)
		if strings.Contains(lowerText, lowerKeyword) || strings.Contains(combinedHints, lowerKeyword) {
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
