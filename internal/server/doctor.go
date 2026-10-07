package server

import (
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/duplicate_suppression"
	"github.com/ananthakumaran/paisa/internal/model/finding_dismissal"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Level string

const (
	WARN  Level = "warning"
	ERROR Level = "danger"
)

type Issue struct {
	Level       Level  `json:"level"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Details     string `json:"details"`
}

type Rule struct {
	Issue     Issue
	Predicate func(db *gorm.DB) []error
}

const DATE_FORMAT string = "02 Jan 2006"

var rules []Rule

func init() {
	rules = []Rule{
		{
			Issue: Issue{
				Level:       ERROR,
				Summary:     "Negative Balance",
				Description: "The running balance of an <b>asset</b> account is not supposed to go negative at any time. This issue typically happens due to incorrect transaction entries."},
			Predicate: ruleAssetRegisterNonNegative},
		{
			Issue: Issue{
				Level:       ERROR,
				Summary:     "Credit Entry",
				Description: "Income account should never have credit entry."},
			Predicate: ruleNonCreditAccount},
		{
			Issue: Issue{
				Level:       ERROR,
				Summary:     "Debit Entry",
				Description: "Expense Account should never have debit entry."},
			Predicate: ruleNonDebitAccount},
		{
			Issue: Issue{
				Level:       ERROR,
				Summary:     "Exchange Price Missing",
				Description: "Exchange price is missing for the commodity."},
			Predicate: ruleExchangePriceMissing},
		{
			Issue: Issue{
				Level:       WARN,
				Summary:     "Unit Price Mismatch",
				Description: "Unit price used in the journal doesn't match the price fetched from external system."},
			Predicate: ruleJournalPriceMismatch},
		{
			Issue: Issue{
				Level:       WARN,
				Summary:     "Asset Accounts missing from Allocation Target",
				Description: "Asset accounts are not part of any allocation target."},
			Predicate: ruleAllocationTargetMissingAssetAccounts}}
}

func GetDiagnosis(db *gorm.DB) gin.H {
	issues := make([]Issue, 0)
	for _, rule := range rules {
		for _, error := range rule.Predicate(db) {
			issue := rule.Issue
			issue.Details = error.Error()
			issues = append(issues, issue)
		}
	}
	return gin.H{"issues": issues}
}

func ruleAssetRegisterNonNegative(db *gorm.DB) []error {
	errs := make([]error, 0)
	ruleConfig := config.GetConfig().Doctor.NegativeBalance
	if ruleConfig.Enabled == config.No {
		return errs
	}
	assets := query.Init(db).Like(ruleConfig.Pattern...).All()
	for account, ps := range lo.GroupBy(assets, func(posting posting.Posting) string { return posting.Account }) {
		for _, balance := range accounting.Register(ps) {
			if balance.Quantity.LessThan(decimal.NewFromFloat(0.01).Neg()) {
				errs = append(errs, fmt.Errorf("<b>%s</b> account went negative (%.2f) on %s", account, balance.Quantity.InexactFloat64(), balance.Date.Format(DATE_FORMAT)))
				break
			}
		}
	}
	return errs
}

func ruleNonCreditAccount(db *gorm.DB) []error {
	errs := make([]error, 0)
	ruleConfig := config.GetConfig().Doctor.NonCreditAccount
	if ruleConfig.Enabled == config.No {
		return errs
	}
	// Income:CapitalGains accounts are excluded unconditionally: capital-gain
	// postings are expected to carry positive amounts, so flagging them here
	// would always produce false positives regardless of the user's pattern.
	incomes := query.Init(db).Like(ruleConfig.Pattern...).NotLike("Income:CapitalGains:%").All()
	for _, p := range incomes {
		if p.Amount.GreaterThan(decimal.NewFromFloat(0.01)) {
			errs = append(errs, fmt.Errorf("<b>%.4f</b> got credited to <b>%s</b> on %s", p.Amount.InexactFloat64(), p.Account, p.Date.Format(DATE_FORMAT)))
		}
	}
	return errs
}

func ruleNonDebitAccount(db *gorm.DB) []error {
	errs := make([]error, 0)
	ruleConfig := config.GetConfig().Doctor.NonDebitAccount
	if ruleConfig.Enabled == config.No {
		return errs
	}
	expenses := query.Init(db).Like(ruleConfig.Pattern...).All()
	for _, p := range expenses {
		if p.Amount.LessThan(decimal.NewFromFloat(0.01).Neg()) {
			errs = append(errs, fmt.Errorf("<b>%.4f</b> got debited from <b>%s</b> on %s", p.Amount.InexactFloat64(), p.Account, p.Date.Format(DATE_FORMAT)))
		}
	}
	return errs
}

func ruleExchangePriceMissing(db *gorm.DB) []error {
	errs := make([]error, 0)
	if config.GetConfig().Doctor.ExchangePriceMissing.Enabled == config.No {
		return errs
	}
	postings := query.Init(db).Desc().All()

	for _, p := range postings {
		if !utils.IsCurrency(p.Commodity) {
			externalPrice := service.GetUnitPrice(db, p.Commodity, p.Date)
			if externalPrice.CommodityName != "" && externalPrice.CommodityName != p.Commodity {
				errs = append(errs, fmt.Errorf("Exchange price from <b>%s</b> to your default currency <b>%s</b> is not specified for posting %s", p.Commodity, config.DefaultCurrency(), formatPosting(p)))
			}
		}
	}
	return errs
}

func ruleJournalPriceMismatch(db *gorm.DB) []error {
	errs := make([]error, 0)
	if config.GetConfig().Doctor.UnitPriceMismatch.Enabled == config.No {
		return errs
	}
	postings := query.Init(db).Desc().All()
	for _, p := range postings {
		if !utils.IsCurrency(p.Commodity) {
			externalPrice := service.GetUnitPrice(db, p.Commodity, p.Date)
			diff := externalPrice.Value.Sub(p.Price()).Abs()
			if externalPrice.CommodityName == p.Commodity &&
				externalPrice.CommodityType != config.Unknown &&
				!service.IsSellWithCapitalGains(db, p) &&
				diff.GreaterThanOrEqual(decimal.NewFromFloat(0.0001)) {
				errs = append(errs, fmt.Errorf("The price specified in your posting %s doesn't match the price <b>%.4f</b> (%s) fetched from external system", formatPosting(p), externalPrice.Value.InexactFloat64(), externalPrice.Date.Format(DATE_FORMAT)))
			}
		}
	}
	return errs
}

func formatPosting(p posting.Posting) string {
	var price string
	if p.Quantity.Equal(p.Amount) {
		price = fmt.Sprintf("%.4f %s", p.Quantity.InexactFloat64(), p.Commodity)
	} else {
		price = fmt.Sprintf("%.4f %s @ %.4f %s", p.Quantity.InexactFloat64(), p.Commodity, p.Price().InexactFloat64(), config.DefaultCurrency())
	}

	postingUrl := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(p.FileName), p.TransactionBeginLine)
	return fmt.Sprintf("<a href=\"%s\"> %s\t%s\t%s</a>", postingUrl, p.Date.Format(DATE_FORMAT), p.Account, price)
}

func ruleAllocationTargetMissingAssetAccounts(db *gorm.DB) []error {
	errs := make([]error, 0)

	ruleConfig := config.GetConfig().Doctor.AssetAllocationMissing
	if ruleConfig.Enabled == config.No {
		return errs
	}

	if len(config.GetConfig().AllocationTargets) == 0 {
		return errs
	}

	var accounts []string
	conditions := make([]string, len(ruleConfig.Pattern))
	args := make([]interface{}, len(ruleConfig.Pattern))
	for i, p := range ruleConfig.Pattern {
		conditions[i] = "account like ?"
		args[i] = p
	}
	db.Model(&posting.Posting{}).Where(strings.Join(conditions, " or "), args...).Distinct().Pluck("Account", &accounts)

	ignoredAccounts := make([]string, 0)
	for _, account := range accounts {
		found := false
		for _, target := range config.GetConfig().AllocationTargets {
			for _, targetAccount := range target.Accounts {
				match, _ := filepath.Match(targetAccount, account)
				if match {
					found = true
					break
				}
			}

			if found {
				break
			}
		}

		if !found {
			ignoredAccounts = append(ignoredAccounts, account)
		}
	}

	if len(ignoredAccounts) > 0 {
		errs = append(errs, fmt.Errorf("The following asset accounts are not part of any asset allocation target: <b>%s</b>", strings.Join(ignoredAccounts, ", ")))
	}

	return errs
}

// DuplicatePair represents two postings that are potential duplicates of each other.
type DuplicatePair struct {
	Posting1   posting.Posting `json:"posting1"`
	Posting2   posting.Posting `json:"posting2"`
	Confidence float64         `json:"confidence"`
	Reason     string          `json:"reason"`
}

// OutlierTransaction represents a posting that is a statistical outlier within
// its account category (more than 3 standard deviations from the mean).
type OutlierTransaction struct {
	Posting    posting.Posting `json:"posting"`
	Mean       float64         `json:"mean"`
	StdDev     float64         `json:"std_dev"`
	Sigma      float64         `json:"sigma"`
	Confidence float64         `json:"confidence"`
}

// SuppressRequest is the request body for suppressing a duplicate pair.
type SuppressRequest struct {
	PostingID1 uint `json:"posting_id_1"`
	PostingID2 uint `json:"posting_id_2"`
}

// DetectDuplicates finds pairs of postings with the same amount and account
// within a 2-day window, excluding any pairs that have been suppressed by the user.
// Each pair is assigned a confidence score (0–1) based on how closely the two
// postings match in date, amount, account, and payee.
func DetectDuplicates(db *gorm.DB) []DuplicatePair {
	// Load all suppressed pair IDs for fast lookups.
	var suppressions []duplicate_suppression.DuplicateSuppression
	db.Find(&suppressions)
	suppressed := make(map[[2]uint]bool, len(suppressions))
	for _, s := range suppressions {
		// Store both orderings so the lookup is symmetric.
		a, b := s.PostingID1, s.PostingID2
		if a > b {
			a, b = b, a
		}
		suppressed[[2]uint{a, b}] = true
	}

	isSuppressed := func(id1, id2 uint) bool {
		a, b := id1, id2
		if a > b {
			a, b = b, a
		}
		return suppressed[[2]uint{a, b}]
	}

	// Load all non-forecast postings.
	postings := query.Init(db).All()

	// Group by account.
	byAccount := lo.GroupBy(postings, func(p posting.Posting) string { return p.Account })

	var pairs []DuplicatePair

	for _, accountPostings := range byAccount {
		// Within each account group, group by rounded amount.
		byAmount := lo.GroupBy(accountPostings, func(p posting.Posting) string {
			return p.Amount.StringFixed(2)
		})

		for _, sameAmountPostings := range byAmount {
			if len(sameAmountPostings) < 2 {
				continue
			}
			// Sort by date for efficient window checking.
			sort.Slice(sameAmountPostings, func(i, j int) bool {
				return sameAmountPostings[i].Date.Before(sameAmountPostings[j].Date)
			})

			// Check each pair within a 2-day window.
			for i := 0; i < len(sameAmountPostings); i++ {
				for j := i + 1; j < len(sameAmountPostings); j++ {
					p1 := sameAmountPostings[i]
					p2 := sameAmountPostings[j]

					diff := p2.Date.Sub(p1.Date)
					if diff > 48*time.Hour {
						// Too far apart; advance the outer pointer.
						break
					}

					if isSuppressed(p1.ID, p2.ID) {
						continue
					}

					confidence := duplicateConfidence(p1, p2)
					reason := buildDuplicateReason(p1, p2)
					pairs = append(pairs, DuplicatePair{
						Posting1:   p1,
						Posting2:   p2,
						Confidence: confidence,
						Reason:     reason,
					})
				}
			}
		}
	}

	// Sort descending by confidence.
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Confidence > pairs[j].Confidence
	})
	return pairs
}

// duplicateConfidence returns a score in [0, 1] representing how likely two
// postings are to be duplicates. A score of 1.0 means near-certain duplicate.
func duplicateConfidence(p1, p2 posting.Posting) float64 {
	score := 0.0

	// Same amount is the primary signal (already guaranteed at this point).
	score += 0.5

	// Closer dates → higher score (max 0.2 at 0 days apart, 0 at 2 days apart).
	diffHours := math.Abs(p2.Date.Sub(p1.Date).Hours())
	score += 0.2 * (1.0 - diffHours/48.0)

	// Same payee adds 0.2.
	if strings.EqualFold(p1.Payee, p2.Payee) && p1.Payee != "" {
		score += 0.2
	}

	// Same transaction note adds 0.1.
	if p1.TransactionNote != "" && p1.TransactionNote == p2.TransactionNote {
		score += 0.1
	}

	if score > 1.0 {
		score = 1.0
	}
	return math.Round(score*100) / 100
}

// buildDuplicateReason returns a human-readable explanation for why two postings
// were flagged as potential duplicates.
func buildDuplicateReason(p1, p2 posting.Posting) string {
	diffDays := int(math.Round(math.Abs(p2.Date.Sub(p1.Date).Hours()) / 24))
	switch diffDays {
	case 0:
		return fmt.Sprintf("Same amount <b>%s</b> on the same date in account <b>%s</b>", p1.Amount.StringFixed(2), p1.Account)
	default:
		return fmt.Sprintf("Same amount <b>%s</b> in account <b>%s</b>, %d day(s) apart", p1.Amount.StringFixed(2), p1.Account, diffDays)
	}
}

// DetectOutliers identifies postings whose absolute amount is more than 3
// standard deviations above the per-account mean (using only postings with the
// default currency commodity so amounts are comparable).
func DetectOutliers(db *gorm.DB) []OutlierTransaction {
	defaultCurrency := config.DefaultCurrency()

	// Load non-forecast postings and filter to default currency in Go.
	all := query.Init(db).All()
	var postings []posting.Posting
	for _, p := range all {
		if p.Commodity == defaultCurrency {
			postings = append(postings, p)
		}
	}

	byAccount := lo.GroupBy(postings, func(p posting.Posting) string { return p.Account })

	var outliers []OutlierTransaction

	for _, accountPostings := range byAccount {
		if len(accountPostings) < 5 {
			// Too few postings to compute meaningful statistics.
			continue
		}

		// Compute mean of absolute amounts.
		var sum float64
		for _, p := range accountPostings {
			sum += math.Abs(p.Amount.InexactFloat64())
		}
		mean := sum / float64(len(accountPostings))

		// Compute standard deviation.
		var variance float64
		for _, p := range accountPostings {
			diff := math.Abs(p.Amount.InexactFloat64()) - mean
			variance += diff * diff
		}
		stdDev := math.Sqrt(variance / float64(len(accountPostings)))

		if stdDev < 1e-9 {
			// All values identical – no meaningful outlier detection possible.
			continue
		}

		for _, p := range accountPostings {
			absAmt := math.Abs(p.Amount.InexactFloat64())
			sigma := (absAmt - mean) / stdDev
			if sigma >= 3.0 {
				// confidence approaches 1.0 as sigma grows far beyond 3.
				confidence := math.Min(1.0, (sigma-3.0)/3.0+0.5)
				confidence = math.Round(confidence*100) / 100
				outliers = append(outliers, OutlierTransaction{
					Posting:    p,
					Mean:       math.Round(mean*100) / 100,
					StdDev:     math.Round(stdDev*100) / 100,
					Sigma:      math.Round(sigma*100) / 100,
					Confidence: confidence,
				})
			}
		}
	}

	// Sort descending by sigma (most extreme first).
	sort.Slice(outliers, func(i, j int) bool {
		return outliers[i].Sigma > outliers[j].Sigma
	})
	return outliers
}

// GetDuplicatesAndOutliers returns all detected duplicate pairs and outlier
// transactions, excluding user-suppressed pairs.
func GetDuplicatesAndOutliers(db *gorm.DB) gin.H {
	return gin.H{
		"duplicates": DetectDuplicates(db),
		"outliers":   DetectOutliers(db),
	}
}

// SuppressDuplicate records a pair of posting IDs as a user-confirmed
// false-positive so that it no longer appears in future duplicate results.
func SuppressDuplicate(db *gorm.DB, req SuppressRequest) error {
	// Normalise the pair order so the unique index works regardless of call order.
	id1, id2 := req.PostingID1, req.PostingID2
	if id1 > id2 {
		id1, id2 = id2, id1
	}

	// Upsert: skip if a record already exists for this pair.
	var count int64
	db.Model(&duplicate_suppression.DuplicateSuppression{}).
		Where("posting_id_1 = ? AND posting_id_2 = ?", id1, id2).
		Count(&count)
	if count > 0 {
		return nil
	}

	return db.Create(&duplicate_suppression.DuplicateSuppression{
		PostingID1: id1,
		PostingID2: id2,
		CreatedAt:  time.Now(),
	}).Error
}

// ---------------------------------------------------------------------------
// Unified Doctor Findings (Phase B)
// ---------------------------------------------------------------------------

type FindingKind string

const (
	KindRule      FindingKind = "rule"
	KindDuplicate FindingKind = "duplicate"
	KindOutlier   FindingKind = "outlier"
)

type FindingSeverity string

const (
	SeverityFix    FindingSeverity = "fix"
	SeverityReview FindingSeverity = "review"
	SeverityInfo   FindingSeverity = "info"
)

type FindingEvidence struct {
	PostingID  uint    `json:"posting_id,omitempty"`
	PostingID2 uint    `json:"posting_id_2,omitempty"`
	FileName   string  `json:"file_name,omitempty"`
	LineNumber uint64  `json:"line_number,omitempty"`
	Account    string  `json:"account,omitempty"`
	Date       string  `json:"date,omitempty"`
	Amount     float64 `json:"amount,omitempty"`
	Commodity  string  `json:"commodity,omitempty"`
	Payee      string  `json:"payee,omitempty"`
	RawMessage string  `json:"raw_message,omitempty"`
	TargetURL  string  `json:"target_url,omitempty"`
	Sigma      float64 `json:"sigma,omitempty"`
	Mean       float64 `json:"mean,omitempty"`
	StdDev     float64 `json:"std_dev,omitempty"`
}

type FindingAction struct {
	Type   string            `json:"type"`
	Label  string            `json:"label"`
	URL    string            `json:"url,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

type DoctorFinding struct {
	ID           string            `json:"id"`
	RuleID       string            `json:"rule_id"`
	Kind         FindingKind       `json:"kind"`
	Title        string            `json:"title"`
	Summary      string            `json:"summary"`
	Description  string            `json:"description"`
	WhyItMatters string            `json:"why_it_matters"`
	HowToFix     string            `json:"how_to_fix"`
	Severity     FindingSeverity   `json:"severity"`
	Details      string            `json:"details"`
	Evidence     []FindingEvidence `json:"evidence"`
	Actions      []FindingAction   `json:"actions"`
	Confidence   float64           `json:"confidence,omitempty"`
	Dismissed    bool              `json:"dismissed"`
	DismissNote  string            `json:"dismiss_note,omitempty"`
}

type DoctorSummary struct {
	Total          int `json:"total"`
	FixCount       int `json:"fix_count"`
	ReviewCount    int `json:"review_count"`
	InfoCount      int `json:"info_count"`
	DismissedCount int `json:"dismissed_count"`
}

type UnifiedFindingsResponse struct {
	Findings []DoctorFinding `json:"findings"`
	Summary  DoctorSummary   `json:"summary"`
}

type DismissRequest struct {
	Fingerprint string `json:"fingerprint"`
	RuleID      string `json:"rule_id"`
	Note        string `json:"note"`
}

type UndismissRequest struct {
	Fingerprint string `json:"fingerprint"`
}

func evaluateAssetRegisterNonNegative(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	ruleConfig := config.GetConfig().Doctor.NegativeBalance
	if ruleConfig.Enabled == config.No {
		return findings
	}
	assets := query.Init(db).Like(ruleConfig.Pattern...).All()
	for account, ps := range lo.GroupBy(assets, func(posting posting.Posting) string { return posting.Account }) {
		for _, balance := range accounting.Register(ps) {
			if balance.Quantity.LessThan(decimal.NewFromFloat(0.01).Neg()) {
				fingerprint := fmt.Sprintf("negative_balance:%s", account)
				details := fmt.Sprintf("%s account went negative (%.2f) on %s", account, balance.Quantity.InexactFloat64(), balance.Date.Format(DATE_FORMAT))
				findings = append(findings, DoctorFinding{
					ID:           fingerprint,
					RuleID:       "negative_balance",
					Kind:         KindRule,
					Title:        "Negative Account Balance",
					Summary:      "Negative Balance",
					Description:  "The running balance of an asset account is not supposed to go negative at any time.",
					WhyItMatters: "Asset accounts like cash or bank balances should never drop below zero in double-entry accounting. A negative balance usually indicates missing income transactions or entries in the wrong order.",
					HowToFix:     "Review transactions for this account around the date it went negative and insert missing deposits or correct transaction dates.",
					Severity:     SeverityFix,
					Details:      details,
					Evidence: []FindingEvidence{
						{
							Account:    account,
							Amount:     balance.Quantity.InexactFloat64(),
							Date:       balance.Date.Format("2006-01-02"),
							RawMessage: details,
							TargetURL:  "/ledger/editor",
						},
					},
					Actions: []FindingAction{
						{Type: "open_url", Label: "Open Ledger Editor", URL: "/ledger/editor"},
					},
				})
				break
			}
		}
	}
	return findings
}

func evaluateNonCreditAccount(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	ruleConfig := config.GetConfig().Doctor.NonCreditAccount
	if ruleConfig.Enabled == config.No {
		return findings
	}
	incomes := query.Init(db).Like(ruleConfig.Pattern...).NotLike("Income:CapitalGains:%").All()
	for _, p := range incomes {
		if p.Amount.GreaterThan(decimal.NewFromFloat(0.01)) {
			fingerprint := fmt.Sprintf("credit_entry:%d:%s:%s", p.ID, p.Account, p.Date.Format("2006-01-02"))
			editorURL := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(p.FileName), p.TransactionBeginLine)
			details := fmt.Sprintf("%.4f got credited to %s on %s", p.Amount.InexactFloat64(), p.Account, p.Date.Format(DATE_FORMAT))
			findings = append(findings, DoctorFinding{
				ID:           fingerprint,
				RuleID:       "credit_entry",
				Kind:         KindRule,
				Title:        "Credit Entry in Income Account",
				Summary:      "Credit Entry",
				Description:  "Income account should never have credit entry.",
				WhyItMatters: "Income accounts receive credits (negative values in ledger convention). A positive amount distorts total income calculations.",
				HowToFix:     "Check if the transaction signs are flipped or if a refund/expense was incorrectly posted as positive income.",
				Severity:     SeverityFix,
				Details:      details,
				Evidence: []FindingEvidence{
					{
						PostingID:  p.ID,
						FileName:   p.FileName,
						LineNumber: p.TransactionBeginLine,
						Account:    p.Account,
						Date:       p.Date.Format("2006-01-02"),
						Amount:     p.Amount.InexactFloat64(),
						Commodity:  p.Commodity,
						Payee:      p.Payee,
						RawMessage: details,
						TargetURL:  editorURL,
					},
				},
				Actions: []FindingAction{
					{Type: "open_editor", Label: "Edit Transaction", URL: editorURL},
				},
			})
		}
	}
	return findings
}

func evaluateNonDebitAccount(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	ruleConfig := config.GetConfig().Doctor.NonDebitAccount
	if ruleConfig.Enabled == config.No {
		return findings
	}
	expenses := query.Init(db).Like(ruleConfig.Pattern...).All()
	for _, p := range expenses {
		if p.Amount.LessThan(decimal.NewFromFloat(0.01).Neg()) {
			fingerprint := fmt.Sprintf("debit_entry:%d:%s:%s", p.ID, p.Account, p.Date.Format("2006-01-02"))
			editorURL := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(p.FileName), p.TransactionBeginLine)
			details := fmt.Sprintf("%.4f got debited from %s on %s", p.Amount.InexactFloat64(), p.Account, p.Date.Format(DATE_FORMAT))
			findings = append(findings, DoctorFinding{
				ID:           fingerprint,
				RuleID:       "debit_entry",
				Kind:         KindRule,
				Title:        "Debit Entry in Expense Account",
				Summary:      "Debit Entry",
				Description:  "Expense Account should never have debit entry.",
				WhyItMatters: "Expense accounts receive debits (positive values in ledger convention). A negative entry distorts expense reporting.",
				HowToFix:     "Check if the transaction sign was inverted or if a refund should be handled under a dedicated category.",
				Severity:     SeverityFix,
				Details:      details,
				Evidence: []FindingEvidence{
					{
						PostingID:  p.ID,
						FileName:   p.FileName,
						LineNumber: p.TransactionBeginLine,
						Account:    p.Account,
						Date:       p.Date.Format("2006-01-02"),
						Amount:     p.Amount.InexactFloat64(),
						Commodity:  p.Commodity,
						Payee:      p.Payee,
						RawMessage: details,
						TargetURL:  editorURL,
					},
				},
				Actions: []FindingAction{
					{Type: "open_editor", Label: "Edit Transaction", URL: editorURL},
				},
			})
		}
	}
	return findings
}

func evaluateExchangePriceMissing(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	if config.GetConfig().Doctor.ExchangePriceMissing.Enabled == config.No {
		return findings
	}
	postings := query.Init(db).Desc().All()

	seenCommodities := make(map[string]bool)
	for _, p := range postings {
		if !utils.IsCurrency(p.Commodity) && !seenCommodities[p.Commodity] {
			externalPrice := service.GetUnitPrice(db, p.Commodity, p.Date)
			if externalPrice.CommodityName != "" && externalPrice.CommodityName != p.Commodity {
				seenCommodities[p.Commodity] = true
				fingerprint := fmt.Sprintf("exchange_price_missing:%s", p.Commodity)
				details := fmt.Sprintf("Exchange price from %s to your default currency %s is not specified", p.Commodity, config.DefaultCurrency())
				findings = append(findings, DoctorFinding{
					ID:           fingerprint,
					RuleID:       "exchange_price_missing",
					Kind:         KindRule,
					Title:        "Missing Commodity Price",
					Summary:      "Exchange Price Missing",
					Description:  "Exchange price is missing for the commodity.",
					WhyItMatters: "Paisa requires commodity exchange prices to calculate total net worth and valuations accurately.",
					HowToFix:     "Add a price entry for this commodity in the Price section or configure external price providers.",
					Severity:     SeverityReview,
					Details:      details,
					Evidence: []FindingEvidence{
						{
							Commodity:  p.Commodity,
							RawMessage: details,
							TargetURL:  "/price",
						},
					},
					Actions: []FindingAction{
						{Type: "open_url", Label: "Open Price Manager", URL: "/price"},
					},
				})
			}
		}
	}
	return findings
}

func evaluateJournalPriceMismatch(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	if config.GetConfig().Doctor.UnitPriceMismatch.Enabled == config.No {
		return findings
	}
	postings := query.Init(db).Desc().All()
	for _, p := range postings {
		if !utils.IsCurrency(p.Commodity) {
			externalPrice := service.GetUnitPrice(db, p.Commodity, p.Date)
			diff := externalPrice.Value.Sub(p.Price()).Abs()
			if externalPrice.CommodityName == p.Commodity &&
				externalPrice.CommodityType != config.Unknown &&
				!service.IsSellWithCapitalGains(db, p) &&
				diff.GreaterThanOrEqual(decimal.NewFromFloat(0.0001)) {
				fingerprint := fmt.Sprintf("unit_price_mismatch:%d:%s", p.ID, p.Commodity)
				editorURL := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(p.FileName), p.TransactionBeginLine)
				details := fmt.Sprintf("The price specified in posting doesn't match the price %.4f (%s) fetched from external system", externalPrice.Value.InexactFloat64(), externalPrice.Date.Format(DATE_FORMAT))
				findings = append(findings, DoctorFinding{
					ID:           fingerprint,
					RuleID:       "unit_price_mismatch",
					Kind:         KindRule,
					Title:        "Journal Unit Price Mismatch",
					Summary:      "Unit Price Mismatch",
					Description:  "Unit price used in the journal doesn't match the price fetched from external system.",
					WhyItMatters: "The unit price recorded in your ledger posting differs significantly from market prices.",
					HowToFix:     "Verify the transaction price in your ledger entry or adjust the external price feed.",
					Severity:     SeverityReview,
					Details:      details,
					Evidence: []FindingEvidence{
						{
							PostingID:  p.ID,
							FileName:   p.FileName,
							LineNumber: p.TransactionBeginLine,
							Account:    p.Account,
							Date:       p.Date.Format("2006-01-02"),
							Amount:     p.Price().InexactFloat64(),
							Commodity:  p.Commodity,
							RawMessage: details,
							TargetURL:  editorURL,
						},
					},
					Actions: []FindingAction{
						{Type: "open_editor", Label: "Edit Transaction", URL: editorURL},
					},
				})
			}
		}
	}
	return findings
}

func evaluateAllocationTargetMissingAssetAccounts(db *gorm.DB) []DoctorFinding {
	findings := make([]DoctorFinding, 0)
	ruleConfig := config.GetConfig().Doctor.AssetAllocationMissing
	if ruleConfig.Enabled == config.No {
		return findings
	}
	if len(config.GetConfig().AllocationTargets) == 0 {
		return findings
	}
	var accounts []string
	conditions := make([]string, len(ruleConfig.Pattern))
	args := make([]interface{}, len(ruleConfig.Pattern))
	for i, p := range ruleConfig.Pattern {
		conditions[i] = "account like ?"
		args[i] = p
	}
	db.Model(&posting.Posting{}).Where(strings.Join(conditions, " or "), args...).Distinct().Pluck("Account", &accounts)

	ignoredAccounts := make([]string, 0)
	for _, account := range accounts {
		found := false
		for _, target := range config.GetConfig().AllocationTargets {
			for _, targetAccount := range target.Accounts {
				match, _ := filepath.Match(targetAccount, account)
				if match {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			ignoredAccounts = append(ignoredAccounts, account)
		}
	}

	if len(ignoredAccounts) > 0 {
		fingerprint := "asset_allocation_missing:unallocated"
		details := fmt.Sprintf("The following asset accounts are not part of any asset allocation target: %s", strings.Join(ignoredAccounts, ", "))
		findings = append(findings, DoctorFinding{
			ID:           fingerprint,
			RuleID:       "asset_allocation_missing",
			Kind:         KindRule,
			Title:        "Asset Accounts Missing Allocation Target",
			Summary:      "Asset Accounts missing from Allocation Target",
			Description:  "Asset accounts are not part of any allocation target.",
			WhyItMatters: "Asset accounts not assigned to any allocation target will be excluded from target portfolio rebalancing.",
			HowToFix:     "Configure your allocation targets to assign these asset accounts.",
			Severity:     SeverityInfo,
			Details:      details,
			Evidence: []FindingEvidence{
				{
					RawMessage: strings.Join(ignoredAccounts, ", "),
					TargetURL:  "/allocation",
				},
			},
			Actions: []FindingAction{
				{Type: "open_url", Label: "Configure Allocation", URL: "/allocation"},
			},
		})
	}
	return findings
}

func evaluateDuplicates(db *gorm.DB) []DoctorFinding {
	pairs := DetectDuplicates(db)
	findings := make([]DoctorFinding, 0, len(pairs))
	for _, pair := range pairs {
		id1, id2 := pair.Posting1.ID, pair.Posting2.ID
		if id1 > id2 {
			id1, id2 = id2, id1
		}
		fingerprint := fmt.Sprintf("duplicate:%d:%d", id1, id2)
		url1 := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(pair.Posting1.FileName), pair.Posting1.TransactionBeginLine)
		url2 := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(pair.Posting2.FileName), pair.Posting2.TransactionBeginLine)

		findings = append(findings, DoctorFinding{
			ID:           fingerprint,
			RuleID:       "duplicate_transaction",
			Kind:         KindDuplicate,
			Title:        "Potential Duplicate Transaction",
			Summary:      "Potential Duplicate",
			Description:  "Two postings with identical amounts and accounts were recorded within 2 days.",
			WhyItMatters: "Duplicate entries inflate expenses or income figures and distort net worth.",
			HowToFix:     "Inspect both postings. If it is a true duplicate, remove one entry in the editor. Otherwise, dismiss this finding.",
			Severity:     SeverityReview,
			Details:      pair.Reason,
			Confidence:   pair.Confidence,
			Evidence: []FindingEvidence{
				{
					PostingID:  pair.Posting1.ID,
					FileName:   pair.Posting1.FileName,
					LineNumber: pair.Posting1.TransactionBeginLine,
					Account:    pair.Posting1.Account,
					Date:       pair.Posting1.Date.Format("2006-01-02"),
					Amount:     pair.Posting1.Amount.InexactFloat64(),
					Commodity:  pair.Posting1.Commodity,
					Payee:      pair.Posting1.Payee,
					TargetURL:  url1,
				},
				{
					PostingID:  pair.Posting2.ID,
					FileName:   pair.Posting2.FileName,
					LineNumber: pair.Posting2.TransactionBeginLine,
					Account:    pair.Posting2.Account,
					Date:       pair.Posting2.Date.Format("2006-01-02"),
					Amount:     pair.Posting2.Amount.InexactFloat64(),
					Commodity:  pair.Posting2.Commodity,
					Payee:      pair.Posting2.Payee,
					TargetURL:  url2,
				},
			},
			Actions: []FindingAction{
				{Type: "open_editor", Label: "Open Posting 1", URL: url1},
				{Type: "open_editor", Label: "Open Posting 2", URL: url2},
				{Type: "dismiss", Label: "Dismiss"},
			},
		})
	}
	return findings
}

func evaluateOutliers(db *gorm.DB) []DoctorFinding {
	outliers := DetectOutliers(db)
	findings := make([]DoctorFinding, 0, len(outliers))
	for _, out := range outliers {
		fingerprint := fmt.Sprintf("outlier:%d:%s", out.Posting.ID, out.Posting.Account)
		editorURL := fmt.Sprintf("/ledger/editor/%s#%d", url.PathEscape(out.Posting.FileName), out.Posting.TransactionBeginLine)
		details := fmt.Sprintf("%.1fσ above mean (mean: %.2f, σ: %.2f)", out.Sigma, out.Mean, out.StdDev)

		findings = append(findings, DoctorFinding{
			ID:           fingerprint,
			RuleID:       "outlier_transaction",
			Kind:         KindOutlier,
			Title:        "Statistical Outlier Transaction",
			Summary:      "Statistical Outlier",
			Description:  "Posting amount is significantly higher than historical average for this account.",
			WhyItMatters: "Unusually large amounts may indicate a typo (such as misplaced decimal points) or a misplaced zero.",
			HowToFix:     "Verify the transaction amount in the ledger editor. If correct, dismiss this alert.",
			Severity:     SeverityReview,
			Details:      details,
			Confidence:   out.Confidence,
			Evidence: []FindingEvidence{
				{
					PostingID:  out.Posting.ID,
					FileName:   out.Posting.FileName,
					LineNumber: out.Posting.TransactionBeginLine,
					Account:    out.Posting.Account,
					Date:       out.Posting.Date.Format("2006-01-02"),
					Amount:     out.Posting.Amount.InexactFloat64(),
					Commodity:  out.Posting.Commodity,
					Payee:      out.Posting.Payee,
					Sigma:      out.Sigma,
					Mean:       out.Mean,
					StdDev:     out.StdDev,
					TargetURL:  editorURL,
				},
			},
			Actions: []FindingAction{
				{Type: "open_editor", Label: "Edit Transaction", URL: editorURL},
				{Type: "dismiss", Label: "Dismiss"},
			},
		})
	}
	return findings
}

func GetUnifiedFindings(db *gorm.DB) UnifiedFindingsResponse {
	var dismissals []finding_dismissal.FindingDismissal
	db.Find(&dismissals)
	dismissedMap := make(map[string]string, len(dismissals))
	for _, d := range dismissals {
		dismissedMap[d.Fingerprint] = d.Note
	}

	var rawFindings []DoctorFinding
	rawFindings = append(rawFindings, evaluateAssetRegisterNonNegative(db)...)
	rawFindings = append(rawFindings, evaluateNonCreditAccount(db)...)
	rawFindings = append(rawFindings, evaluateNonDebitAccount(db)...)
	rawFindings = append(rawFindings, evaluateExchangePriceMissing(db)...)
	rawFindings = append(rawFindings, evaluateJournalPriceMismatch(db)...)
	rawFindings = append(rawFindings, evaluateAllocationTargetMissingAssetAccounts(db)...)
	rawFindings = append(rawFindings, evaluateDuplicates(db)...)
	rawFindings = append(rawFindings, evaluateOutliers(db)...)

	findings := make([]DoctorFinding, 0, len(rawFindings))
	var summary DoctorSummary

	for _, f := range rawFindings {
		note, isDismissed := dismissedMap[f.ID]
		if isDismissed {
			f.Dismissed = true
			f.DismissNote = note
			summary.DismissedCount++
		} else {
			switch f.Severity {
			case SeverityFix:
				summary.FixCount++
			case SeverityReview:
				summary.ReviewCount++
			case SeverityInfo:
				summary.InfoCount++
			}
		}
		summary.Total++
		findings = append(findings, f)
	}

	return UnifiedFindingsResponse{
		Findings: findings,
		Summary:  summary,
	}
}

func GetDismissedFindings(db *gorm.DB) UnifiedFindingsResponse {
	all := GetUnifiedFindings(db)
	dismissed := make([]DoctorFinding, 0)
	for _, f := range all.Findings {
		if f.Dismissed {
			dismissed = append(dismissed, f)
		}
	}
	return UnifiedFindingsResponse{
		Findings: dismissed,
		Summary:  all.Summary,
	}
}

func DismissFinding(db *gorm.DB, req DismissRequest) error {
	if req.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	var count int64
	db.Model(&finding_dismissal.FindingDismissal{}).
		Where("fingerprint = ?", req.Fingerprint).
		Count(&count)
	if count == 0 {
		err := db.Create(&finding_dismissal.FindingDismissal{
			Fingerprint: req.Fingerprint,
			RuleID:      req.RuleID,
			Note:        req.Note,
			CreatedAt:   time.Now(),
		}).Error
		if err != nil {
			return err
		}
	} else if req.Note != "" {
		db.Model(&finding_dismissal.FindingDismissal{}).
			Where("fingerprint = ?", req.Fingerprint).
			Update("note", req.Note)
	}

	if strings.HasPrefix(req.Fingerprint, "duplicate:") {
		parts := strings.Split(req.Fingerprint, ":")
		if len(parts) == 3 {
			var id1, id2 uint
			fmt.Sscanf(parts[1], "%d", &id1)
			fmt.Sscanf(parts[2], "%d", &id2)
			if id1 > 0 && id2 > 0 {
				_ = SuppressDuplicate(db, SuppressRequest{PostingID1: id1, PostingID2: id2})
			}
		}
	}

	return nil
}

func UndismissFinding(db *gorm.DB, req UndismissRequest) error {
	if req.Fingerprint == "" {
		return fmt.Errorf("fingerprint is required")
	}

	if err := db.Where("fingerprint = ?", req.Fingerprint).Delete(&finding_dismissal.FindingDismissal{}).Error; err != nil {
		return err
	}

	if strings.HasPrefix(req.Fingerprint, "duplicate:") {
		parts := strings.Split(req.Fingerprint, ":")
		if len(parts) == 3 {
			var id1, id2 uint
			fmt.Sscanf(parts[1], "%d", &id1)
			fmt.Sscanf(parts[2], "%d", &id2)
			if id1 > 0 && id2 > 0 {
				a, b := id1, id2
				if a > b {
					a, b = b, a
				}
				db.Where("posting_id_1 = ? AND posting_id_2 = ?", a, b).Delete(&duplicate_suppression.DuplicateSuppression{})
			}
		}
	}

	return nil
}
