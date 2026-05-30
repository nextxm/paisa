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
