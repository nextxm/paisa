package drawdown

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/taxation"
	"github.com/shopspring/decimal"
)

func TestEffectiveTaxRateUsesTaxBurdenOverAmount(t *testing.T) {
	tax := taxation.Tax{ShortTerm: decimal.NewFromInt(15), LongTerm: decimal.NewFromInt(5)}
	rate := effectiveTaxRate(tax, decimal.NewFromInt(200))
	if !rate.Equal(decimal.NewFromFloat(0.1)) {
		t.Fatalf("rate = %s, want 0.1", rate)
	}
}

func TestGlobMatchSupportsPrefixGlob(t *testing.T) {
	if !globMatch("Assets:Equity:Index", "Assets:Equity:*") {
		t.Fatal("expected prefix glob match")
	}
	if globMatch("Assets:Debt:Fund", "Assets:Equity:*") {
		t.Fatal("unexpected match for non-prefix account")
	}
}

func TestSortPrefersLowerTaxRateThenLongerHolding(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	lots := []availableLot{
		{Posting: posting.Posting{Date: now.AddDate(-1, 0, 0)}, SortTaxRate: decimal.NewFromFloat(0.10), SortTaxAmount: decimal.NewFromInt(10), HoldingDays: 365, Commodity: config.Commodity{TaxCategory: config.Equity}},
		{Posting: posting.Posting{Date: now.AddDate(-2, 0, 0)}, SortTaxRate: decimal.NewFromFloat(0.05), SortTaxAmount: decimal.NewFromInt(10), HoldingDays: 730, Commodity: config.Commodity{TaxCategory: config.Debt}},
		{Posting: posting.Posting{Date: now.AddDate(-3, 0, 0)}, SortTaxRate: decimal.NewFromFloat(0.05), SortTaxAmount: decimal.NewFromInt(10), HoldingDays: 1095, Commodity: config.Commodity{TaxCategory: config.Equity}},
	}

	sortLotsByTaxCost(lots)

	if lots[0].HoldingDays != 1095 {
		t.Fatalf("first lot holding days = %d, want 1095", lots[0].HoldingDays)
	}
	if !lots[2].SortTaxRate.Equal(decimal.NewFromFloat(0.10)) {
		t.Fatalf("last lot tax rate = %s, want 0.10", lots[2].SortTaxRate)
	}
}

func TestBucketMatchesLotSupportsTaxCategoryAndHoldingPeriod(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	lot := availableLot{
		Posting:          posting.Posting{Account: "Assets:Equity:Index", Date: now.AddDate(-2, 0, 0)},
		Commodity:        config.Commodity{TaxCategory: config.Equity},
		PriceDate:        now,
		CurrentUnitPrice: decimal.NewFromInt(100),
	}

	if !bucketMatchesLot(lot, DrawdownBucket{TaxCategory: config.Equity}) {
		t.Fatal("expected equity category to match")
	}
	if bucketMatchesLot(lot, DrawdownBucket{TaxCategory: config.Debt}) {
		t.Fatal("unexpected category match for debt bucket")
	}
	if !bucketMatchesLot(lot, DrawdownBucket{HoldingPeriodMonths: 12}) {
		t.Fatal("expected holding period >= 12 months to match")
	}
	if bucketMatchesLot(lot, DrawdownBucket{HoldingPeriodMonths: 36}) {
		t.Fatal("unexpected match when holding period filter is not satisfied")
	}
	if bucketMatchesLot(lot, DrawdownBucket{AccountGlob: "Assets:Debt:*"}) {
		t.Fatal("unexpected match for non-matching account glob")
	}
}

func TestBucketMatchesLotSupportsExplicitAssignedAccounts(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	lot := availableLot{
		Posting:   posting.Posting{Account: "Assets:Equity:Index", Date: now.AddDate(-2, 0, 0)},
		Commodity: config.Commodity{TaxCategory: config.Equity},
		PriceDate: now,
	}

	bucket := DrawdownBucket{
		Accounts:    []string{"Assets:Equity:Index"},
		AccountGlob: "Assets:Debt:*",
	}
	if !bucketMatchesLot(lot, bucket) {
		t.Fatal("expected explicit account assignment to match even when glob does not")
	}

	bucket.Accounts = []string{"Assets:Debt:Fund"}
	if bucketMatchesLot(lot, bucket) {
		t.Fatal("unexpected match for bucket without assigned account")
	}
}

func TestOrderLotsByBucketsUsesBucketPriority(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	lots := []availableLot{
		{
			Posting:          posting.Posting{Account: "Assets:Equity:One", Date: now.AddDate(-3, 0, 0)},
			Commodity:        config.Commodity{TaxCategory: config.Equity},
			SortTaxRate:      decimal.NewFromFloat(0.01),
			SortTaxAmount:    decimal.NewFromInt(1),
			HoldingDays:      1095,
			CurrentUnitPrice: decimal.NewFromInt(100),
			PriceDate:        now,
			BucketIndex:      1,
		},
		{
			Posting:          posting.Posting{Account: "Assets:Debt:One", Date: now.AddDate(-1, 0, 0)},
			Commodity:        config.Commodity{TaxCategory: config.Debt},
			SortTaxRate:      decimal.NewFromFloat(0.50),
			SortTaxAmount:    decimal.NewFromInt(50),
			HoldingDays:      365,
			CurrentUnitPrice: decimal.NewFromInt(100),
			PriceDate:        now,
			BucketIndex:      0,
		},
	}

	buckets := []DrawdownBucket{
		{TaxCategory: config.Debt},
		{TaxCategory: config.Equity},
	}

	ordered := orderLotsByBuckets(lots, buckets)
	if len(ordered) != 2 {
		t.Fatalf("ordered lots length = %d, want 2", len(ordered))
	}
	if ordered[0].Commodity.TaxCategory != config.Debt {
		t.Fatalf("first lot category = %s, want debt", ordered[0].Commodity.TaxCategory)
	}
	if ordered[1].Commodity.TaxCategory != config.Equity {
		t.Fatalf("second lot category = %s, want equity", ordered[1].Commodity.TaxCategory)
	}
}

func TestOrderLotsByBucketsDropsUnmatchedLotsWhenBucketsProvided(t *testing.T) {
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	lots := []availableLot{
		{
			Posting:          posting.Posting{Account: "Assets:Equity:One", Date: now.AddDate(-2, 0, 0)},
			Commodity:        config.Commodity{TaxCategory: config.Equity},
			SortTaxRate:      decimal.NewFromFloat(0.10),
			SortTaxAmount:    decimal.NewFromInt(10),
			HoldingDays:      730,
			CurrentUnitPrice: decimal.NewFromInt(100),
			PriceDate:        now,
			BucketIndex:      0,
		},
		{
			Posting:          posting.Posting{Account: "Assets:Debt:One", Date: now.AddDate(-2, 0, 0)},
			Commodity:        config.Commodity{TaxCategory: config.Debt},
			SortTaxRate:      decimal.NewFromFloat(0.10),
			SortTaxAmount:    decimal.NewFromInt(10),
			HoldingDays:      730,
			CurrentUnitPrice: decimal.NewFromInt(100),
			PriceDate:        now,
			BucketIndex:      -1,
		},
	}

	ordered := orderLotsByBuckets(lots, []DrawdownBucket{{AccountGlob: "Assets:Equity:*"}})
	if len(ordered) != 1 {
		t.Fatalf("ordered lots length = %d, want 1", len(ordered))
	}
	if ordered[0].Posting.Account != "Assets:Equity:One" {
		t.Fatalf("matched account = %s, want Assets:Equity:One", ordered[0].Posting.Account)
	}
}

func TestApplyBucketTaxCategoryOverrideUsesOverrideWhenPresent(t *testing.T) {
	original := config.Equity
	bucket := DrawdownBucket{OverrideTaxCategory: config.Debt}
	if got := applyBucketTaxCategoryOverride(original, bucket); got != config.Debt {
		t.Fatalf("override category = %s, want debt", got)
	}

	if got := applyBucketTaxCategoryOverride(original, DrawdownBucket{}); got != original {
		t.Fatalf("category without override = %s, want %s", got, original)
	}
}
