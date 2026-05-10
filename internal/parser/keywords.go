package parser

import (
	"strings"

	"github.com/ananthakumaran/paisa/internal/config"
)

// DefaultKeywords returns the built-in default keyword set.
func DefaultKeywords() KeywordMatcher {
	return KeywordMatcher{
		ExpenseMarkers: []string{
			"bought", "purchase", "purchased", "paid", "spent", "withdrew", "withdraw",
			"charged", "cost", "expense", "groceries", "fuel", "gas", "utility",
			"bill", "subscription", "rent", "fee", "commission",
		},
		IncomeMarkers: []string{
			"received", "transferred in", "deposit", "earned", "salary", "bonus",
			"sold", "income", "refund", "reimbursement", "payment", "income", "dividends",
		},
		TransferMarkers: []string{
			"moved", "transferred", "transfer", "sent", "sent to", "move", "moved to",
			"from", "to", "into", "between",
		},
		CCPaymentMethods: []string{
			"credit card", "cc", "amex", "american express", "visa", "mastercard",
			"discover", "diners", "jcb", "carte", "tarjeta",
		},
		DebitPaymentMethods: []string{
			"debit card", "debit", "chequing", "checking", "savings account", "bank account",
			"account", "atm", "withdrawal",
		},
		CashMethods: []string{
			"cash", "handed", "hand", "paid cash", "withdrew cash",
		},
		CustomPayees: map[string]string{
			// User can configure these in paisa.yaml
		},
	}
}

// LoadKeywordsFromConfig loads keyword configuration from paisa.yaml.
// Merges defaults with user-provided customizations.
func LoadKeywordsFromConfig(cfg *config.Config) KeywordMatcher {
	keywords := DefaultKeywords()

	// TODO: Load user customizations from cfg if parser.keywords section exists
	// Example structure in paisa.yaml:
	// parser:
	//   keywords:
	//     transaction_markers:
	//       expense: ["bought", "paid", "spent"]
	//     payment_methods:
	//       cc: ["credit card", "amex"]
	//     custom_payees:
	//       "Acme Corp": "Liabilities:CorporateCard"

	return keywords
}

// KeywordScore checks if a keyword list contains a word and returns a score.
// Used to detect transaction patterns from text.
func KeywordScore(text string, keywords []string) (bool, float64) {
	text = strings.ToLower(strings.TrimSpace(text))

	// TODO: Implement keyword scoring
	// - Check if any keyword appears in text (substring or word boundary)
	// - Return (found bool, score float64)
	// - Score higher for exact matches, lower for partial matches
	// - Examples:
	//   - "bought groceries" contains "bought" -> (true, 0.95)
	//   - "I bought milk" contains "bought" -> (true, 0.90)
	//   - "rebought" contains "bought" substring but not word -> (false, 0)

	return false, 0.0
}

// ExtractPaymentMethodHint identifies payment method from text.
// Returns the detected method (cc, debit, cash, unknown) and confidence.
func (km *KeywordMatcher) ExtractPaymentMethodHint(text string) (string, float64) {
	// TODO: Implement payment method extraction
	// - Check CC keywords: "credit card", "amex", "visa", "mastercard"
	// - Check debit keywords: "debit", "checking", "chequing", "savings"
	// - Check cash keywords: "cash", "handed", "withdrawn"
	// - Return method + confidence (0-1)
	// - If multiple methods found, return highest confidence

	text = strings.ToLower(text)

	if found, score := KeywordScore(text, km.CCPaymentMethods); found {
		return "cc", score
	}
	if found, score := KeywordScore(text, km.DebitPaymentMethods); found {
		return "debit", score
	}
	if found, score := KeywordScore(text, km.CashMethods); found {
		return "cash", score
	}

	return "unknown", 0.0
}

// ExtractTransactionTypeHint identifies transaction type from text.
// Returns the detected type (expense, income, transfer) and confidence.
func (km *KeywordMatcher) ExtractTransactionTypeHint(text string) (string, float64) {
	// TODO: Implement transaction type extraction
	// - Check expense keywords: "bought", "paid", "spent", "withdrew"
	// - Check income keywords: "received", "earned", "sold", "refund"
	// - Check transfer keywords: "moved", "transferred", "sent", "from...to"
	// - Return type + confidence (0-1)
	// - If multiple types found, return highest confidence

	text = strings.ToLower(text)

	if found, score := KeywordScore(text, km.ExpenseMarkers); found {
		return "expense", score
	}
	if found, score := KeywordScore(text, km.IncomeMarkers); found {
		return "income", score
	}
	if found, score := KeywordScore(text, km.TransferMarkers); found {
		return "transfer", score
	}

	return "unknown", 0.0
}

// FindCustomPayee checks if the text contains a known custom payee.
// Returns the mapped account and a confidence score.
func (km *KeywordMatcher) FindCustomPayee(text string) (string, float64) {
	// TODO: Implement custom payee matching
	// - Check each custom payee in km.CustomPayees
	// - Find best substring match in text
	// - Return mapped account + confidence (exact: 1.0, partial: 0.8, none: 0.0)

	text = strings.ToLower(text)
	bestMatch := ""
	bestScore := 0.0

	for payee, account := range km.CustomPayees {
		if strings.Contains(text, strings.ToLower(payee)) {
			if len(payee) > len(bestMatch) {
				bestMatch = account
				bestScore = 0.95 // High confidence for custom matches
			}
		}
	}

	if bestScore > 0 {
		return bestMatch, bestScore
	}
	return "", 0.0
}
