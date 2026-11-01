package dna

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func loadTestConfig(t *testing.T) {
	t.Helper()
	orig := config.GetConfig()
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\n"), ""))
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
}

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	service.ClearPriceCache()
	service.ClearRateCache()
	t.Cleanup(func() {
		service.ClearPriceCache()
		service.ClearRateCache()
	})
	return db
}

func date(y int, m time.Month, d int) time.Time {
	return utils.ToDate(time.Date(y, m, d, 0, 0, 0, 0, time.UTC))
}

var txnSeq int

func seedPosting(t *testing.T, db *gorm.DB, d time.Time, account, commodity string, quantity, amount float64) {
	t.Helper()
	txnSeq++
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: fmt.Sprintf("txn-%d", txnSeq),
		Date:          d,
		Payee:         "test",
		Account:       account,
		Commodity:     commodity,
		Quantity:      decimal.NewFromFloat(quantity),
		Amount:        decimal.NewFromFloat(amount),
	}).Error)
}

func seedFundPrice(t *testing.T, db *gorm.DB, commodity string, d time.Time, value float64) {
	t.Helper()
	require.NoError(t, db.Create(&price.Price{
		Date:           d,
		CommodityType:  config.MutualFund,
		CommodityID:    commodity,
		CommodityName:  commodity,
		QuoteCommodity: config.DefaultCurrency(),
		Value:          decimal.NewFromFloat(value),
		Source:         "provider",
	}).Error)
}

// legacyComputeReturnMetrics is a copy of the pre-optimization implementation
// (three queries and two market-price passes per month). It is used as the
// reference oracle to guarantee the optimized version produces identical
// results.
//
// The only deviation from the original is the cumulative lower bound: the
// original used Where("date between ? AND ?", time.Time{}, d), which the sqlc
// query path maps to "date >= NULL" and therefore matched no rows (making the
// metrics always zero). UntilDate(d) is the intended "date <= d" filter.
func legacyComputeReturnMetrics(db *gorm.DB, now time.Time) (annualReturn, volatility decimal.Decimal, monthsCovered int) {
	allPrices, err := price.FindFiltered(db, price.PriceFilter{})
	if err != nil || len(allPrices) == 0 {
		return decimal.Zero, decimal.Zero, 0
	}

	twoYearsAgo := utils.BeginningOfMonth(now).AddDate(-2, 0, 0)
	var monthlyReturns []float64

	for monthStart := twoYearsAgo; monthStart.Before(now); monthStart = monthStart.AddDate(0, 1, 0) {
		monthEnd := monthStart.AddDate(0, 1, 0)
		if monthEnd.After(now) {
			break
		}

		startPostings := query.Init(db).Like("Assets:%").
			NotAccountPrefix("Assets:Checking").
			UntilDate(monthStart).All()
		startPostings = service.PopulateMarketPriceAt(db, startPostings, monthStart)

		endPostings := query.Init(db).Like("Assets:%").
			NotAccountPrefix("Assets:Checking").
			UntilDate(monthEnd).All()
		endPostings = service.PopulateMarketPriceAt(db, endPostings, monthEnd)

		newContribs := query.Init(db).Like("Assets:%").
			NotAccountPrefix("Assets:Checking").
			Where("date between ? AND ?", monthStart, monthEnd).All()
		contribAmount := accounting.CostSum(newContribs)

		startValue := accounting.CurrentBalance(startPostings)
		endValue := accounting.CurrentBalance(endPostings)

		if startValue.GreaterThan(decimal.Zero) {
			returnPct := endValue.Sub(startValue).Sub(contribAmount).Div(startValue)
			r, _ := returnPct.Float64()
			monthlyReturns = append(monthlyReturns, r)
		}
	}

	monthsCovered = len(monthlyReturns)
	if monthsCovered < 3 {
		return decimal.Zero, decimal.Zero, monthsCovered
	}

	sum := 0.0
	for _, r := range monthlyReturns {
		sum += r
	}
	meanMonthly := sum / float64(monthsCovered)
	annualReturnFloat := math.Pow(1+meanMonthly, 12) - 1
	annualReturn = decimal.NewFromFloat(annualReturnFloat * 100).Round(2)

	sumSqDiff := 0.0
	for _, r := range monthlyReturns {
		diff := r - meanMonthly
		sumSqDiff += diff * diff
	}
	stdMonthly := math.Sqrt(sumSqDiff / float64(monthsCovered-1))
	volatilityFloat := stdMonthly * math.Sqrt(12)
	volatility = decimal.NewFromFloat(volatilityFloat * 100).Round(2)

	return annualReturn, volatility, monthsCovered
}

// seedPortfolio builds a ~3 year history with a priced mutual fund, a cash
// asset, excluded checking postings, purchases on month boundaries (which the
// BETWEEN semantics count in two adjacent months), mid-month purchases and a
// partial redemption.
func seedPortfolio(t *testing.T, db *gorm.DB) {
	t.Helper()
	start := date(2023, time.September, 1)
	nav := 100.0
	for i := 0; i < 38; i++ {
		monthStart := start.AddDate(0, i, 0)
		// Oscillating NAV so returns have non-trivial volatility.
		nav = nav * (1 + 0.01*float64((i%5)-1))
		seedFundPrice(t, db, "FUND", monthStart, nav)
		seedFundPrice(t, db, "FUND", monthStart.AddDate(0, 0, 14), nav*1.005)

		// Monthly SIP on the 1st (boundary date) bought at a slightly
		// different cost than the market price.
		units := 10.0
		seedPosting(t, db, monthStart, "Assets:Equity:FUND", "FUND", units, units*nav*0.998)
		seedPosting(t, db, monthStart, "Assets:Checking", config.DefaultCurrency(), -units*nav*0.998, -units*nav*0.998)

		// Mid-month cash deposit.
		seedPosting(t, db, monthStart.AddDate(0, 0, 10), "Assets:Cash", config.DefaultCurrency(), 500, 500)
		seedPosting(t, db, monthStart.AddDate(0, 0, 10), "Assets:Checking", config.DefaultCurrency(), -500, -500)

		if i%7 == 3 {
			// Partial redemption.
			seedPosting(t, db, monthStart.AddDate(0, 0, 20), "Assets:Equity:FUND", "FUND", -4, -4*nav)
		}
	}
}

func TestComputeReturnMetricsMatchesLegacy(t *testing.T) {
	loadTestConfig(t)
	db := openTestDB(t)
	seedPortfolio(t, db)

	nows := []time.Time{
		date(2026, time.November, 1),
		date(2026, time.October, 17),
		date(2025, time.March, 31),
		date(2024, time.January, 15), // fewer than 24 months of history
		date(2023, time.October, 20), // fewer than 3 months covered
	}
	for _, now := range nows {
		t.Run(now.Format("2006-01-02"), func(t *testing.T) {
			wantReturn, wantVol, wantMonths := legacyComputeReturnMetrics(db, now)
			gotReturn, gotVol, gotMonths := computeReturnMetrics(db, now)

			assert.Equal(t, wantMonths, gotMonths, "months covered")
			assert.Truef(t, wantReturn.Equal(gotReturn), "annual return: want %s got %s", wantReturn, gotReturn)
			assert.Truef(t, wantVol.Equal(gotVol), "volatility: want %s got %s", wantVol, gotVol)
		})
	}

	// Sanity check that the fixture exercises the full math path.
	_, vol, months := computeReturnMetrics(db, date(2026, time.November, 1))
	assert.Equal(t, 24, months)
	assert.True(t, vol.GreaterThan(decimal.Zero))
}

func TestExtractProfileReturnMetricsMatchLegacy(t *testing.T) {
	loadTestConfig(t)
	db := openTestDB(t)
	seedPortfolio(t, db)

	utils.SetNow("2026-11-01")
	t.Cleanup(utils.UnsetNow)

	profile := ExtractProfile(db)
	wantReturn, wantVol, wantMonths := legacyComputeReturnMetrics(db, utils.Now())

	assert.Equal(t, wantMonths, profile.PriceMonthsCovered)
	assert.Truef(t, wantReturn.Equal(profile.HistoricalReturn), "want %s got %s", wantReturn, profile.HistoricalReturn)
	assert.Truef(t, wantVol.Equal(profile.ReturnVolatility), "want %s got %s", wantVol, profile.ReturnVolatility)
}

func TestComputeReturnMetricsNoPrices(t *testing.T) {
	loadTestConfig(t)
	db := openTestDB(t)
	seedPosting(t, db, date(2025, time.January, 1), "Assets:Cash", config.DefaultCurrency(), 1000, 1000)

	r, v, m := computeReturnMetrics(db, date(2026, time.November, 1))
	assert.True(t, r.IsZero())
	assert.True(t, v.IsZero())
	assert.Equal(t, 0, m)
}

func TestMonthlyReturnsFromPostingsBoundarySemantics(t *testing.T) {
	loadTestConfig(t)
	b0 := date(2025, time.January, 1)
	b1 := date(2025, time.February, 1)
	b2 := date(2025, time.March, 1)

	mk := func(d time.Time, amt float64) posting.Posting {
		return posting.Posting{Date: d, Amount: decimal.NewFromFloat(amt), MarketAmount: decimal.NewFromFloat(amt)}
	}
	// Deliberately unsorted input.
	postings := []posting.Posting{
		mk(b1, 50), // on boundary: in month 0 and month 1 contributions
		mk(date(2024, time.June, 1), 1000),
		mk(b0, 100),                        // on b0: part of start value and month 0 contribution
		mk(date(2025, time.March, 2), 999), // after window: ignored
	}

	var seen []int
	valueAt := func(ps []posting.Posting, _ time.Time) decimal.Decimal {
		seen = append(seen, len(ps))
		return accounting.CurrentBalance(ps)
	}

	returns := monthlyReturnsFromPostings(postings, []time.Time{b0, b1, b2}, valueAt)
	assert.Equal(t, []int{2, 3, 3}, seen, "valueAt called once per boundary with date<=boundary prefix")
	require.Len(t, returns, 2)
	// Month 0: start=1100, end=1150, contrib=150 => (1150-1100-150)/1100
	assert.InDelta(t, -100.0/1100.0, returns[0], 1e-12)
	// Month 1: start=1150, end=1150, contrib=50 => -50/1150
	assert.InDelta(t, -50.0/1150.0, returns[1], 1e-12)
}
