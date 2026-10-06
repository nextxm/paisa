package server

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/duplicate_suppression"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makePosting is a test helper that creates a posting row with the given fields.
func makePosting(id uint, account string, amount float64, date string, payee string) posting.Posting {
	t, _ := time.Parse("2006-01-02", date)
	return posting.Posting{
		ID:        id,
		Account:   account,
		Amount:    decimal.NewFromFloat(amount),
		Quantity:  decimal.NewFromFloat(amount),
		Commodity: "INR",
		Date:      t,
		Payee:     payee,
	}
}

// ---------------------------------------------------------------------------
// Duplicate detection
// ---------------------------------------------------------------------------

func TestDetectDuplicates_SameAmountSameDateSameAccount(t *testing.T) {
	db := openTestDB(t)

	// Insert two postings with the same amount, account, and date.
	p1 := makePosting(1, "Expenses:Food", 500.0, "2024-01-10", "Restaurant A")
	p2 := makePosting(2, "Expenses:Food", 500.0, "2024-01-10", "Restaurant A")
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	pairs := DetectDuplicates(db)
	require.Len(t, pairs, 1)
	// Same date + same payee: 0.5 (amount) + 0.2 (date=0 days) + 0.2 (payee) = 0.9
	assert.GreaterOrEqual(t, pairs[0].Confidence, 0.8)
}

func TestDetectDuplicates_SameAmountWithinTwoDays(t *testing.T) {
	db := openTestDB(t)

	p1 := makePosting(1, "Expenses:Food", 500.0, "2024-01-10", "Shop")
	p2 := makePosting(2, "Expenses:Food", 500.0, "2024-01-12", "Shop")
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	pairs := DetectDuplicates(db)
	require.Len(t, pairs, 1)
	assert.Greater(t, pairs[0].Confidence, 0.0)
}

func TestDetectDuplicates_SameAmountBeyondTwoDays(t *testing.T) {
	db := openTestDB(t)

	p1 := makePosting(1, "Expenses:Food", 500.0, "2024-01-10", "Shop")
	p2 := makePosting(2, "Expenses:Food", 500.0, "2024-01-15", "Shop")
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	pairs := DetectDuplicates(db)
	assert.Empty(t, pairs)
}

func TestDetectDuplicates_DifferentAccountsNotFlagged(t *testing.T) {
	db := openTestDB(t)

	p1 := makePosting(1, "Expenses:Food", 500.0, "2024-01-10", "Shop")
	p2 := makePosting(2, "Expenses:Transport", 500.0, "2024-01-10", "Shop")
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	pairs := DetectDuplicates(db)
	assert.Empty(t, pairs)
}

func TestDetectDuplicates_SuppressedPairExcluded(t *testing.T) {
	db := openTestDB(t)

	p1 := makePosting(1, "Expenses:Food", 500.0, "2024-01-10", "Shop")
	p2 := makePosting(2, "Expenses:Food", 500.0, "2024-01-10", "Shop")
	require.NoError(t, db.Create(&p1).Error)
	require.NoError(t, db.Create(&p2).Error)

	// Suppress the pair.
	require.NoError(t, db.Create(&duplicate_suppression.DuplicateSuppression{
		PostingID1: 1,
		PostingID2: 2,
		CreatedAt:  time.Now(),
	}).Error)

	pairs := DetectDuplicates(db)
	assert.Empty(t, pairs)
}

func TestDetectDuplicates_SortedByConfidenceDescending(t *testing.T) {
	db := openTestDB(t)

	// Pair A: same date → higher confidence.
	a1 := makePosting(1, "Expenses:Food", 100.0, "2024-01-10", "X")
	a2 := makePosting(2, "Expenses:Food", 100.0, "2024-01-10", "X")
	// Pair B: 2 days apart → lower confidence.
	b1 := makePosting(3, "Expenses:Transport", 200.0, "2024-01-10", "Y")
	b2 := makePosting(4, "Expenses:Transport", 200.0, "2024-01-12", "Y")

	require.NoError(t, db.Create(&a1).Error)
	require.NoError(t, db.Create(&a2).Error)
	require.NoError(t, db.Create(&b1).Error)
	require.NoError(t, db.Create(&b2).Error)

	pairs := DetectDuplicates(db)
	require.Len(t, pairs, 2)
	assert.GreaterOrEqual(t, pairs[0].Confidence, pairs[1].Confidence)
}

// ---------------------------------------------------------------------------
// Suppress duplicate
// ---------------------------------------------------------------------------

func TestSuppressDuplicate_CreatesRecord(t *testing.T) {
	db := openTestDB(t)

	err := SuppressDuplicate(db, SuppressRequest{PostingID1: 5, PostingID2: 10})
	require.NoError(t, err)

	var count int64
	db.Model(&duplicate_suppression.DuplicateSuppression{}).
		Where("posting_id_1 = ? AND posting_id_2 = ?", 5, 10).
		Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestSuppressDuplicate_Idempotent(t *testing.T) {
	db := openTestDB(t)

	require.NoError(t, SuppressDuplicate(db, SuppressRequest{PostingID1: 3, PostingID2: 7}))
	require.NoError(t, SuppressDuplicate(db, SuppressRequest{PostingID1: 3, PostingID2: 7}))

	var count int64
	db.Model(&duplicate_suppression.DuplicateSuppression{}).Count(&count)
	assert.Equal(t, int64(1), count)
}

func TestSuppressDuplicate_NormalisesOrder(t *testing.T) {
	db := openTestDB(t)

	// Store with reversed IDs — should normalise to (low, high).
	require.NoError(t, SuppressDuplicate(db, SuppressRequest{PostingID1: 9, PostingID2: 2}))

	var s duplicate_suppression.DuplicateSuppression
	require.NoError(t, db.First(&s).Error)
	assert.Equal(t, uint(2), s.PostingID1)
	assert.Equal(t, uint(9), s.PostingID2)
}

// ---------------------------------------------------------------------------
// Outlier detection
// ---------------------------------------------------------------------------

func TestDetectOutliers_FlagsExtremValue(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// 9 normal postings of 100 and 1 extreme posting of 10000.
	for i := uint(1); i <= 9; i++ {
		p := makePosting(i, "Expenses:Food", 100.0, "2024-01-10", "Normal")
		require.NoError(t, db.Create(&p).Error)
	}
	extreme := makePosting(10, "Expenses:Food", 10000.0, "2024-01-10", "Extreme")
	require.NoError(t, db.Create(&extreme).Error)

	outliers := DetectOutliers(db)
	require.Len(t, outliers, 1)
	assert.Equal(t, uint(10), outliers[0].Posting.ID)
	assert.GreaterOrEqual(t, outliers[0].Sigma, 3.0)
}

func TestDetectOutliers_SkipsTooFewPostings(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// Only 3 postings — below the minimum of 5.
	for i := uint(1); i <= 3; i++ {
		p := makePosting(i, "Expenses:Misc", 100.0, "2024-01-10", "A")
		require.NoError(t, db.Create(&p).Error)
	}

	outliers := DetectOutliers(db)
	assert.Empty(t, outliers)
}

func TestDetectOutliers_UniformAmountsNoOutlier(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// All amounts identical — standard deviation is 0, no outliers should be reported.
	for i := uint(1); i <= 10; i++ {
		p := makePosting(i, "Expenses:Uniform", 500.0, "2024-01-10", "A")
		require.NoError(t, db.Create(&p).Error)
	}

	outliers := DetectOutliers(db)
	assert.Empty(t, outliers)
}
