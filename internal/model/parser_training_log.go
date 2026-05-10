package model

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ParserTrainingLog represents a single parsing result logged for ML training.
// Each successful transaction creation creates one log entry containing:
// - The original user input
// - What the parser predicted
// - The confidence scores for each field
// - What the user actually confirmed (if it differs from prediction)
// - Whether the user corrected any suggestions
//
// This data is collected non-invasively and used in Phase 3 to train ML models
// to improve prediction accuracy over time.
//
// Privacy: Stored locally in SQLite, never uploaded, user can delete anytime.
type ParserTrainingLog struct {
	ID        uint      `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	// Input: Raw user text
	InputText string

	// Predictions: What the parser extracted
	PredictedDate        *time.Time
	PredictedAmount      decimal.Decimal
	PredictedCurrency    string
	PredictedPayee       string
	PredictedFromAccount string
	PredictedToAccount   string
	PredictedDirection   string // "expense", "income", "transfer"

	// Confidence scores (0.0 to 1.0) for each field
	ConfidenceDate        float64
	ConfidenceAmount      float64
	ConfidenceCurrency    float64
	ConfidencePayee       float64
	ConfidenceFromAccount float64
	ConfidenceToAccount   float64
	ConfidenceDirection   float64
	ConfidenceOverall     float64

	// Actual user confirmation: What the user ended up using
	// If null, user accepted parser prediction
	ActualDate        *time.Time
	ActualAmount      *decimal.Decimal
	ActualCurrency    *string
	ActualPayee       *string
	ActualFromAccount *string
	ActualToAccount   *string
	ActualDirection   *string

	// User correction feedback
	UserCorrected   bool   // true if ActualX differs from PredictedX
	CorrectionNotes string `gorm:"type:text"` // Optional user notes about why they corrected

	// Engagement metrics (Phase 3)
	SuggestionsShown int // How many suggestions were displayed to user
	SuggestionUsed   int // Which suggestion index user selected (0-based, -1 if custom)
	TimeToConfirm    int // Milliseconds from parse to confirmation
}

// TableName specifies the database table name.
func (ParserTrainingLog) TableName() string {
	return "parser_training_log"
}

// CreateParserTrainingLog inserts a new training log record.
func CreateParserTrainingLog(db *gorm.DB, log *ParserTrainingLog) error {
	return db.Create(log).Error
}

// GetParserTrainingLogs retrieves training logs for analysis.
// Filters by date range and returns paginated results.
func GetParserTrainingLogs(db *gorm.DB, startDate, endDate time.Time, limit, offset int) ([]ParserTrainingLog, error) {
	var logs []ParserTrainingLog
	err := db.Where("created_at BETWEEN ? AND ?", startDate, endDate).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&logs).Error
	return logs, err
}

// GetCorrectedLogs retrieves only logs where user made corrections.
// Used for analyzing which predictions were wrong.
func GetCorrectedLogs(db *gorm.DB, startDate, endDate time.Time) ([]ParserTrainingLog, error) {
	var logs []ParserTrainingLog
	err := db.Where("user_corrected = true AND created_at BETWEEN ? AND ?", startDate, endDate).
		Order("created_at DESC").
		Find(&logs).Error
	return logs, err
}

// AnalyzeConfidenceAccuracy computes how well confidence scores correlate with actual accuracy.
// Returns a structure with statistics about prediction quality.
type ConfidenceAnalysis struct {
	TotalLogs               int
	TotalCorrected          int
	CorrectionRate          float64            // Percentage of logs user corrected
	AvgConfidenceByAccuracy map[string]float64 // Avg confidence for correct vs incorrect predictions
	FieldAccuracies         map[string]float64 // Per-field accuracy (amount, payee, accounts, etc)
}

// AnalyzeConfidence analyzes the relationship between confidence scores and prediction accuracy.
// Used to validate confidence scoring and detect systematic bias.
func AnalyzeConfidence(db *gorm.DB, startDate, endDate time.Time) (*ConfidenceAnalysis, error) {
	logs, err := GetCorrectedLogs(db, startDate, endDate)
	if err != nil {
		return nil, err
	}

	// TODO: Implement analysis
	// - Count correct vs corrected predictions
	// - Compare confidence scores to accuracy
	// - Identify which fields are over/under-confident
	// - Output statistics for Phase 3 model training

	return &ConfidenceAnalysis{
		TotalLogs:       len(logs),
		TotalCorrected:  0, // Count from logs
		CorrectionRate:  0, // Calculated
		FieldAccuracies: make(map[string]float64),
	}, nil
}

// PruneOldLogs deletes training logs older than retentionDays.
// Use this to manage database growth if training log table gets large.
// Phase 3+: Implement archival strategy (e.g., export before deleting).
func PruneOldLogs(db *gorm.DB, retentionDays int) error {
	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	return db.Where("created_at < ?", cutoff).Delete(&ParserTrainingLog{}).Error
}

// ExportTrainingData exports logs in a format suitable for ML training.
// Used for Phase 3 when building ML models.
// Format: CSV or JSON with normalized fields.
func ExportTrainingData(db *gorm.DB, startDate, endDate time.Time) (string, error) {
	// TODO: Implement export
	// Export format options:
	// 1. CSV: input_text, predicted_amount, actual_amount, confidence_amount, user_corrected
	// 2. JSON: Array of {input, predictions, actuals, confidences}
	// Used by data scientists for model training in Phase 3
	return "", nil
}

// HidePersonalData scrubs sensitive data from training logs for sharing with data scientists.
// Removes payee names, amounts, account names - keeps only structural patterns.
func HidePersonalData(log *ParserTrainingLog) *ParserTrainingLog {
	// TODO: Implement data anonymization
	// Keep: confidence scores, pattern types, field types
	// Remove: specific amounts, account names, payee names, dates
	// This allows sharing for model improvement without privacy concerns
	return log
}
