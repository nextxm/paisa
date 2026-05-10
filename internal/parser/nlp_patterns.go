package parser

import (
	"regexp"
)

// RegexPatterns holds compiled regex patterns for natural language parsing.
type RegexPatterns struct {
	// Date patterns
	DateYYYYMMDD    *regexp.Regexp // 2026-05-10, 2026/05/10
	DateDDMonthYYYY *regexp.Regexp // 10 May 2026, 10-May-2026
	DateDDMonth     *regexp.Regexp // 10 May, May 10, 10th May
	DateRelative    *regexp.Regexp // today, yesterday, last friday

	// Amount patterns
	AmountDollarPrefix *regexp.Regexp // $15.50, $ 15.50
	AmountSuffix       *regexp.Regexp // 15.50$, 15 USD, 15INR
	AmountWords        *regexp.Regexp // fifteen dollars, twenty rupees

	// Account hints patterns
	AccountHint *regexp.Regexp // "from <account>", "using <account>", "to <account>"
	PaymentHint *regexp.Regexp // credit card, debit card, checking, savings

	// Payee patterns
	PayeeMarker *regexp.Regexp // "at <payee>", "from <payee>", "purchased at"

	// Direction patterns
	PrepositionFrom *regexp.Regexp // " from ", " out of "
	PrepositionTo   *regexp.Regexp // " to ", " into ", " transferred to "
}

// CompilePatterns creates and compiles all regex patterns.
// Should be called once during initialization and reused.
func CompilePatterns() *RegexPatterns {
	return &RegexPatterns{
		// Date patterns (ISO, month names, relative)
		DateYYYYMMDD:    regexp.MustCompile(`(?i)\b(\d{4}[-/](?:0?[1-9]|1[0-2])[-/](?:0?[1-9]|[12]\d|3[01]))\b`),
		DateDDMonthYYYY: regexp.MustCompile(`(?i)\b((?:0?[1-9]|[12]\d|3[01])[-\s](?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*[-\s](?:\d{4}))\b`),
		DateDDMonth:     regexp.MustCompile(`(?i)\b((?:0?[1-9]|[12]\d|3[01])(?:st|nd|rd|th)?[-\s](?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*)\b`),
		DateRelative:    regexp.MustCompile(`(?i)\b(today|tomorrow|yesterday|last\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday)|next\s+(?:monday|tuesday|wednesday|thursday|friday|saturday|sunday))\b`),

		// Amount patterns (dollar prefix, suffix with currency, word forms)
		// Handles: $15.50, 15.50$, 15 USD, 15USD, 15 cad, 15cad, etc.
		AmountDollarPrefix: regexp.MustCompile(`\$\s*(\d+(?:\.\d{2})?)`),
		AmountSuffix:       regexp.MustCompile(`(?i)(\d+(?:\.\d{2})?)\s*(?:\$|USD|INR|EUR|GBP|CAD|AUD|JPY|CNY)`),
		AmountWords:        regexp.MustCompile(`(?i)\b(zero|one|two|three|four|five|six|seven|eight|nine|ten|eleven|twelve|thirteen|fourteen|fifteen|sixteen|seventeen|eighteen|nineteen|twenty|thirty|forty|fifty|sixty|seventy|eighty|ninety|hundred|thousand|million)\s+(?:dollars?|rupees?|euros?|pounds?|cents?)\b`),

		// Account and payment method hints
		AccountHint: regexp.MustCompile(`(?i)(?:from|using|via|with|to|into)\s+(?:my\s+)?([A-Za-z\s:]+?)(?:\s+(?:account|card)|[,\.]|\s+(?:paid|spent|bought|transferred)|$)`),
		PaymentHint: regexp.MustCompile(`(?i)\b(credit\s+card|debit\s+card|debit|cc|amex|visa|mastercard|checking|chequing|savings|cash|atm)\b`),

		// Payee patterns
		PayeeMarker: regexp.MustCompile(`(?i)(?:at|from|purchased\s+(?:at|from)|bought\s+(?:at|from)|paid\s+(?:to|at)|transferred\s+(?:to|from))\s+([A-Za-z\s&]+?)(?:\s+(?:for|with)|[,\.]|$)`),

		// Prepositions for direction
		PrepositionFrom: regexp.MustCompile(`(?i)\s+(?:from|out\s+of|using)\s+`),
		PrepositionTo:   regexp.MustCompile(`(?i)\s+(?:to|into|transferred\s+to)\s+`),
	}
}

// DatePatterns holds month mappings and helpers for date extraction.
var DatePatterns = struct {
	MonthNames map[string]int
}{
	MonthNames: map[string]int{
		"jan": 1, "january": 1,
		"feb": 2, "february": 2,
		"mar": 3, "march": 3,
		"apr": 4, "april": 4,
		"may": 5,
		"jun": 6, "june": 6,
		"jul": 7, "july": 7,
		"aug": 8, "august": 8,
		"sep": 9, "september": 9,
		"oct": 10, "october": 10,
		"nov": 11, "november": 11,
		"dec": 12, "december": 12,
	},
}

// CurrencyPatterns holds currency code mappings.
var CurrencyPatterns = struct {
	Codes map[string]string
}{
	Codes: map[string]string{
		"$":    "USD",
		"usd":  "USD",
		"us$":  "USD",
		"cad":  "CAD",
		"can$": "CAD",
		"€":    "EUR",
		"eur":  "EUR",
		"£":    "GBP",
		"gbp":  "GBP",
		"inr":  "INR",
		"₹":    "INR",
		"rs":   "INR",
		"jpy":  "JPY",
		"¥jpy": "JPY",
		"aud":  "AUD",
		"a$":   "AUD",
		"cny":  "CNY",
		"¥cny": "CNY",
		"rmb":  "CNY",
	},
}

// WordToNumber maps English words to numeric values (for amount parsing).
var WordToNumber = map[string]float64{
	"zero":      0,
	"one":       1,
	"two":       2,
	"three":     3,
	"four":      4,
	"five":      5,
	"six":       6,
	"seven":     7,
	"eight":     8,
	"nine":      9,
	"ten":       10,
	"eleven":    11,
	"twelve":    12,
	"thirteen":  13,
	"fourteen":  14,
	"fifteen":   15,
	"sixteen":   16,
	"seventeen": 17,
	"eighteen":  18,
	"nineteen":  19,
	"twenty":    20,
	"thirty":    30,
	"forty":     40,
	"fifty":     50,
	"sixty":     60,
	"seventy":   70,
	"eighty":    80,
	"ninety":    90,
	"hundred":   100,
	"thousand":  1000,
	"million":   1000000,
}

// RegexMatchResult represents the result of a regex pattern match with confidence.
type RegexMatchResult struct {
	Matched    bool
	Value      string
	Confidence float64
	Group1     string // First capture group
	Group2     string // Second capture group
}

// NOTE: Regex patterns are intentionally simple to avoid over-engineering.
// The TF-IDF system handles the heavy lifting for account matching.
// These patterns focus on basic text extraction (dates, amounts, keywords).
//
// FUTURE: If pattern complexity grows significantly (>50 lines), consider:
// 1. Moving patterns to config file (YAML)
// 2. Pre-compiling patterns at startup
// 3. Using a small DSL for pattern composition
//
// But for MVP, stdlib regex is sufficient and fast.
