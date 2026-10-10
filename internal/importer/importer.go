package importer

import (
	"encoding/csv"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/importer/ofx"
	"github.com/ananthakumaran/paisa/internal/model/import_rule"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"gorm.io/gorm"
)

// IngestionFormat denotes the input file type.
type IngestionFormat string

const (
	FormatCSV IngestionFormat = "csv"
	FormatOFX IngestionFormat = "ofx"
	FormatQFX IngestionFormat = "qfx"
	FormatCAS IngestionFormat = "cas"
)

// StagedTransaction represents an ingested row enriched with classification and duplicate analysis.
type StagedTransaction struct {
	Index            int      `json:"index"`
	Date             string   `json:"date"`
	Payee            string   `json:"payee"`
	Memo             string   `json:"memo"`
	Amount           float64  `json:"amount"` // Absolute amount
	IsDebit          bool     `json:"is_debit"`
	SelectedAccount  string   `json:"selected_account"`
	SuggestedAccount string   `json:"suggested_account,omitempty"`
	Confidence       string   `json:"confidence,omitempty"` // "rule", "preset", "manual"
	MatchedRuleName  string   `json:"matched_rule_name,omitempty"`
	MatchedRuleID    uint     `json:"matched_rule_id,omitempty"`
	Tags             []string `json:"tags,omitempty"`
	FlagForReview    bool     `json:"flag_for_review"`

	// Duplicate Detection
	IsDuplicate      bool   `json:"is_duplicate"`
	DuplicateScore   int    `json:"duplicate_score"` // 0-100
	DuplicateMatchID uint   `json:"duplicate_match_id,omitempty"`
	DuplicateReason  string `json:"duplicate_reason,omitempty"`
}

// IngestionRequest specifies payload for statement parsing.
type IngestionRequest struct {
	Format        IngestionFormat `json:"format"`
	Content       string          `json:"content"`
	Delimiter     string          `json:"delimiter,omitempty"`
	PresetName    string          `json:"preset_name,omitempty"`
	BaseAccount   string          `json:"base_account,omitempty"` // e.g., Assets:Checking:Chase
	SourceAccount string          `json:"source_account,omitempty"`
}

// IngestionResult returns the staged candidate list.
type IngestionResult struct {
	TotalParsed        int                 `json:"total_parsed"`
	AutoCategorized    int                 `json:"auto_categorized"`
	PossibleDuplicates int                 `json:"possible_duplicates"`
	Transactions       []StagedTransaction `json:"transactions"`
}

// Ingest parses input and enriches every transaction with rules and duplicate checks.
func Ingest(db *gorm.DB, req IngestionRequest) (*IngestionResult, error) {
	var candidates []import_rule.CandidateTransaction

	switch strings.ToLower(string(req.Format)) {
	case string(FormatOFX), string(FormatQFX):
		stmt, err := ofx.Parse(req.Content)
		if err != nil {
			return nil, fmt.Errorf("OFX parse error: %w", err)
		}
		for _, tx := range stmt.Transactions {
			candidates = append(candidates, import_rule.CandidateTransaction{
				Date:    tx.Date.Format("2006-01-02"),
				Payee:   tx.Payee,
				Memo:    tx.Memo,
				Amount:  tx.AbsAmount(),
				IsDebit: tx.IsDebit(),
			})
		}

	case string(FormatCSV), "":
		parsed, err := parseCSVRows(req.Content, req.Delimiter, req.PresetName, db)
		if err != nil {
			return nil, err
		}
		candidates = parsed

	default:
		return nil, fmt.Errorf("unsupported format %q", req.Format)
	}

	// Fetch active user rules
	rules, _ := import_rule.ActiveRules(db)

	// Fetch existing ledger postings for duplicate detection
	var existingPostings []posting.Posting
	db.Select("id, transaction_id, date, payee, account, amount, original_amount, note").
		Find(&existingPostings)

	staged := make([]StagedTransaction, 0, len(candidates))
	autoCatCount := 0
	duplicateCount := 0

	for i, c := range candidates {
		item := StagedTransaction{
			Index:   i,
			Date:    c.Date,
			Payee:   c.Payee,
			Memo:    c.Memo,
			Amount:  c.Amount,
			IsDebit: c.IsDebit,
			Tags:    []string{},
		}

		// 1. Evaluate Rule Match
		match := import_rule.MatchCandidate(rules, c)
		if match.Matched {
			item.SuggestedAccount = match.TargetAccount
			item.SelectedAccount = match.TargetAccount
			item.MatchedRuleName = match.RuleName
			item.MatchedRuleID = match.RuleID
			item.Tags = match.Tags
			item.FlagForReview = match.FlagForReview
			item.Confidence = "rule"
			autoCatCount++
		}

		// 2. Duplicate Detection Scoring
		dupMatch, score, reason := findBestDuplicateMatch(c, existingPostings)
		if score >= 70 {
			item.IsDuplicate = true
			item.DuplicateScore = score
			item.DuplicateReason = reason
			if dupMatch != nil {
				item.DuplicateMatchID = dupMatch.ID
			}
			duplicateCount++
		}

		staged = append(staged, item)
	}

	return &IngestionResult{
		TotalParsed:        len(staged),
		AutoCategorized:    autoCatCount,
		PossibleDuplicates: duplicateCount,
		Transactions:       staged,
	}, nil
}

func parseCSVRows(content, delimiter, presetName string, db *gorm.DB) ([]import_rule.CandidateTransaction, error) {
	reader := csv.NewReader(strings.NewReader(content))
	reader.FieldsPerRecord = -1
	if delimiter != "" {
		runes := []rune(delimiter)
		if len(runes) == 1 {
			reader.Comma = runes[0]
		}
	}

	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to parse CSV: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("CSV statement must have at least header and one data row")
	}

	header := records[0]
	// Attempt column indices
	dateIdx := findColumnIndex(header, []string{"date", "txn date", "value date", "trans date"})
	descIdx := findColumnIndex(header, []string{"narration", "description", "particulars", "memo", "payee", "name"})
	debitIdx := findColumnIndex(header, []string{"debit", "withdrawal", "dr", "debit amount"})
	creditIdx := findColumnIndex(header, []string{"credit", "deposit", "cr", "credit amount"})
	amountIdx := findColumnIndex(header, []string{"amount", "net amount", "trans amount"})

	var candidates []import_rule.CandidateTransaction

	for _, row := range records[1:] {
		if len(row) == 0 || allEmpty(row) {
			continue
		}

		dateStr := ""
		if dateIdx >= 0 && dateIdx < len(row) {
			dateStr = normalizeDateString(row[dateIdx])
		}
		if dateStr == "" {
			dateStr = time.Now().Format("2006-01-02")
		}

		desc := ""
		if descIdx >= 0 && descIdx < len(row) {
			desc = strings.TrimSpace(row[descIdx])
		}

		var amt float64
		var isDebit bool

		if debitIdx >= 0 && debitIdx < len(row) && strings.TrimSpace(row[debitIdx]) != "" {
			val, _ := parseAmountFloat(row[debitIdx])
			if val > 0 {
				amt = val
				isDebit = true
			}
		}

		if amt == 0 && creditIdx >= 0 && creditIdx < len(row) && strings.TrimSpace(row[creditIdx]) != "" {
			val, _ := parseAmountFloat(row[creditIdx])
			if val > 0 {
				amt = val
				isDebit = false
			}
		}

		if amt == 0 && amountIdx >= 0 && amountIdx < len(row) {
			val, _ := parseAmountFloat(row[amountIdx])
			if val < 0 {
				amt = math.Abs(val)
				isDebit = true
			} else {
				amt = val
				isDebit = false
			}
		}

		if amt > 0 {
			candidates = append(candidates, import_rule.CandidateTransaction{
				Date:    dateStr,
				Payee:   desc,
				Memo:    desc,
				Amount:  amt,
				IsDebit: isDebit,
			})
		}
	}

	return candidates, nil
}

func findColumnIndex(header []string, targets []string) int {
	for i, h := range header {
		clean := strings.ToLower(strings.TrimSpace(h))
		for _, target := range targets {
			if strings.Contains(clean, target) {
				return i
			}
		}
	}
	return -1
}

func allEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}

func parseAmountFloat(val string) (float64, error) {
	cleaned := strings.ReplaceAll(val, ",", "")
	cleaned = strings.ReplaceAll(cleaned, "$", "")
	cleaned = strings.ReplaceAll(cleaned, "₹", "")
	cleaned = strings.TrimSpace(cleaned)
	return strconv.ParseFloat(cleaned, 64)
}

func normalizeDateString(raw string) string {
	raw = strings.TrimSpace(raw)
	formats := []string{
		"2006-01-02", "02/01/2006", "01/02/2006", "02-01-2006",
		"02-Jan-2006", "02 Jan 2006", "2006/01/02",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, raw); err == nil {
			return t.Format("2006-01-02")
		}
	}
	return raw
}

func findBestDuplicateMatch(c import_rule.CandidateTransaction, existing []posting.Posting) (*posting.Posting, int, string) {
	cTime, err := time.Parse("2006-01-02", c.Date)
	if err != nil {
		cTime = time.Now()
	}

	var bestMatch *posting.Posting
	highestScore := 0
	bestReason := ""

	for i := range existing {
		p := &existing[i]
		score := 0
		reasons := []string{}

		// 1. Amount match
		pAmt := math.Abs(p.Amount.InexactFloat64())
		if pAmt == 0 {
			pAmt = math.Abs(p.OriginalAmount.InexactFloat64())
		}

		diff := math.Abs(pAmt - c.Amount)
		if diff < 0.01 {
			score += 50
			reasons = append(reasons, "exact amount match")
		} else if diff <= 0.05 {
			score += 30
		} else {
			continue // Non-matching amounts are unlikely duplicates
		}

		// 2. Date proximity
		daysDiff := math.Abs(p.Date.Sub(cTime).Hours() / 24.0)
		if daysDiff == 0 {
			score += 30
			reasons = append(reasons, "same day")
		} else if daysDiff <= 3 {
			score += 20
			reasons = append(reasons, fmt.Sprintf("within %.0f days", daysDiff))
		} else if daysDiff <= 7 {
			score += 10
		} else {
			continue // More than 7 days apart is usually distinct
		}

		// 3. Payee / Note similarity
		pPayee := strings.ToLower(p.Payee + " " + p.Note)
		cPayee := strings.ToLower(c.Payee + " " + c.Memo)
		if strings.Contains(pPayee, cPayee) || strings.Contains(cPayee, pPayee) {
			score += 20
			reasons = append(reasons, "payee text match")
		}

		if score > highestScore {
			highestScore = score
			bestMatch = p
			bestReason = strings.Join(reasons, ", ")
		}
	}

	return bestMatch, highestScore, bestReason
}
