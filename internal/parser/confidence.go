package parser

// ConfidenceThresholds defines the thresholds for confidence-based decisions.
type ConfidenceThresholds struct {
	// AutoCreate: if overall confidence >= this, automatically create transaction (default 0.85)
	AutoCreate float64

	// ShowSuggestions: if field confidence < this, show interactive suggestions (default 0.75)
	ShowSuggestions float64

	// RequireConfirmation: if field confidence < this, always require user selection (default 0.60)
	RequireConfirmation float64

	// MinimumAcceptable: if field confidence < this, warn user (default 0.50)
	MinimumAcceptable float64
}

// DefaultThresholds returns the recommended confidence thresholds.
func DefaultThresholds() ConfidenceThresholds {
	return ConfidenceThresholds{
		AutoCreate:          0.85,
		ShowSuggestions:     0.75,
		RequireConfirmation: 0.60,
		MinimumAcceptable:   0.50,
	}
}

// ScoreRange represents a range of confidence scores for a field with weights.
type ScoreRange struct {
	Min       float64
	Max       float64
	Weight    float64
	FieldName string
}

// confidenceWeights defines the relative importance of each field in overall confidence.
// These weights are applied when computing the overall confidence score.
var confidenceWeights = map[string]float64{
	"amount":       0.30, // Most critical - wrong amount invalidates transaction
	"from_account": 0.25, // Critical - wrong source account breaks history
	"to_account":   0.25, // Critical - wrong destination account breaks reporting
	"payee":        0.15, // Important for tracking/categorization
	"date":         0.05, // Less critical - defaults to today if missing
	"direction":    0.00, // Derived from from/to accounts
}

// ComputeConfidence calculates the weighted average confidence score across all fields.
// Formula: Σ(field_confidence × field_weight) / Σ(weights)
//
// Weights:
// - amount: 0.30 (most critical)
// - from_account: 0.25
// - to_account: 0.25
// - payee: 0.15
// - date: 0.05
//
// Range: 0.0 (completely uncertain) to 1.0 (completely certain)
func ComputeConfidence(scores ConfidenceScores) float64 {
	// TODO: Implement weighted average
	// Formula:
	//   overall = (
	//     amount × 0.30 +
	//     from × 0.25 +
	//     to × 0.25 +
	//     payee × 0.15 +
	//     date × 0.05
	//   ) / 1.0
	//
	// Constraints:
	// - Clamp all individual scores to [0.0, 1.0]
	// - Return value must be in [0.0, 1.0]
	// - If any critical field (amount, accounts) is 0, return lower overall score

	weights := confidenceWeights
	totalWeight := 0.0
	weightedSum := 0.0

	// Amount (0.30)
	if s := clampScore(scores.Amount); s > 0 {
		weightedSum += s * weights["amount"]
		totalWeight += weights["amount"]
	}

	// From Account (0.25)
	if s := clampScore(scores.FromAccount); s > 0 {
		weightedSum += s * weights["from_account"]
		totalWeight += weights["from_account"]
	}

	// To Account (0.25)
	if s := clampScore(scores.ToAccount); s > 0 {
		weightedSum += s * weights["to_account"]
		totalWeight += weights["to_account"]
	}

	// Payee (0.15)
	if s := clampScore(scores.Payee); s > 0 {
		weightedSum += s * weights["payee"]
		totalWeight += weights["payee"]
	}

	// Date (0.05)
	if s := clampScore(scores.Date); s > 0 {
		weightedSum += s * weights["date"]
		totalWeight += weights["date"]
	}

	if totalWeight == 0 {
		return 0.0
	}

	return clampScore(weightedSum / totalWeight)
}

// clampScore ensures a confidence score is in the valid range [0.0, 1.0].
func clampScore(score float64) float64 {
	if score < 0.0 {
		return 0.0
	}
	if score > 1.0 {
		return 1.0
	}
	return score
}

// ScoreBoost increases a confidence score based on supporting evidence.
// Used to boost scores when multiple signals align (e.g., keyword + TF-IDF match).
func ScoreBoost(baseScore float64, boosts ...float64) float64 {
	result := baseScore
	for _, boost := range boosts {
		// Additive boost, clamped to [0, 1]
		result = clampScore(result + boost)
	}
	return result
}

// ScorePenalty decreases a confidence score based on conflicting signals.
// Used to penalize scores when signals disagree (e.g., amount ambiguous + hint weak).
func ScorePenalty(baseScore float64, penalties ...float64) float64 {
	result := baseScore
	for _, penalty := range penalties {
		// Subtractive penalty, clamped to [0, 1]
		result = clampScore(result - penalty)
	}
	return result
}

// PatternConfidence returns a score based on how clearly a pattern matched.
// Examples:
// - Exact match: 0.95
// - Regex match: 0.80
// - Fuzzy match: 0.60
// - No match: 0.0
type PatternConfidence struct {
	PatternType string
	Score       float64
	Evidence    string // Human-readable explanation
}

// PatternConfidences provides confidence scores for common pattern types.
var PatternConfidences = map[string]float64{
	"exact_regex":      0.95, // Pattern matched exactly as expected
	"fuzzy_regex":      0.80, // Pattern matched but with some variation
	"keyword_match":    0.75, // Keyword found but not in exact position
	"tfidf_match":      0.70, // TF-IDF cosine similarity with account
	"default_fallback": 0.40, // Using a configuration default
	"no_match":         0.00, // No pattern matched
}

// DateConfidenceFor returns the confidence score for a date extraction based on pattern type.
func DateConfidenceFor(patternType string) float64 {
	// Exact date matches (2026-05-10, 10/05/2026) -> 0.95
	// Month+day matches (May 10, 10 May) -> 0.90
	// Month+year matches (May 2026) -> 0.70
	// Relative dates (today, yesterday) -> 0.85
	// No date found, using default -> 0.30

	conf, ok := PatternConfidences[patternType]
	if !ok {
		return 0.5
	}
	return conf
}

// AmountConfidenceFor returns the confidence score for an amount extraction.
func AmountConfidenceFor(amountClarity string, currencyClarity string) float64 {
	// High clarity: "15$", "$15.50" -> 0.95
	// Medium clarity: "15 USD" -> 0.85
	// Low clarity: "fifteen dollars" -> 0.60
	// Unknown currency -> -0.10 penalty
	// No amount found -> 0.0

	baseConf := PatternConfidences["exact_regex"] // 0.95

	if amountClarity == "ambiguous" || amountClarity == "fuzzy" {
		baseConf = 0.70
	}

	if currencyClarity == "unknown" || currencyClarity == "missing" {
		baseConf = ScorePenalty(baseConf, 0.10)
	}

	return baseConf
}

// AccountMatchConfidenceFor returns confidence based on TF-IDF cosine similarity.
func AccountMatchConfidenceFor(cosineSimilarity float64) float64 {
	// Cosine similarity is in [0, 1]
	// 0.95+ -> near perfect match -> 0.95
	// 0.80-0.95 -> strong match -> 0.85
	// 0.60-0.80 -> moderate match -> 0.70
	// 0.40-0.60 -> weak match -> 0.50
	// <0.40 -> poor match -> 0.30

	if cosineSimilarity >= 0.95 {
		return 0.95
	}
	if cosineSimilarity >= 0.80 {
		return 0.85
	}
	if cosineSimilarity >= 0.60 {
		return 0.70
	}
	if cosineSimilarity >= 0.40 {
		return 0.50
	}
	return 0.30
}

// DirectionConfidenceFor returns confidence based on transaction direction clarity.
func DirectionConfidenceFor(hasExpenseMarkers, hasIncomeMarkers, hasTransferMarkers bool) float64 {
	// If only one type is clearly marked -> 0.90
	// If two types are marked (ambiguous) -> 0.50
	// If all three types are marked (very ambiguous) -> 0.30
	// If no type is marked -> 0.40 (default to expense)

	count := 0
	if hasExpenseMarkers {
		count++
	}
	if hasIncomeMarkers {
		count++
	}
	if hasTransferMarkers {
		count++
	}

	switch count {
	case 1:
		return 0.90 // Clear direction
	case 2:
		return 0.50 // Ambiguous
	case 3:
		return 0.30 // Very ambiguous
	default:
		return 0.40 // No markers, default
	}
}

// ShouldShowSuggestions returns true if any field confidence is below the threshold.
func ShouldShowSuggestions(scores ConfidenceScores, threshold float64) bool {
	return scores.FromAccount < threshold || scores.ToAccount < threshold ||
		scores.Amount < threshold || scores.Payee < threshold
}

// WhichFieldsNeedSuggestions returns a list of field names that need suggestions.
func WhichFieldsNeedSuggestions(scores ConfidenceScores, threshold float64) []string {
	var fields []string
	if scores.FromAccount < threshold {
		fields = append(fields, "from_account")
	}
	if scores.ToAccount < threshold {
		fields = append(fields, "to_account")
	}
	if scores.Amount < threshold {
		fields = append(fields, "amount")
	}
	if scores.Payee < threshold {
		fields = append(fields, "payee")
	}
	return fields
}
