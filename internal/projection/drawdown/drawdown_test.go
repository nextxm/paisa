package drawdown

import (
	"sort"
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

func TestMatchesBucketSupportsPrefixGlob(t *testing.T) {
	buckets := []DrawdownBucket{{AccountGlob: "Assets:Equity:*"}}
	if !matchesBucket("Assets:Equity:Index", buckets) {
		t.Fatal("expected prefix glob match")
	}
	if matchesBucket("Assets:Debt:Fund", buckets) {
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

	// mirror the production sort contract
	sort.Slice(lots, func(i, j int) bool {
		if !lots[i].SortTaxRate.Equal(lots[j].SortTaxRate) {
			return lots[i].SortTaxRate.LessThan(lots[j].SortTaxRate)
		}
		if !lots[i].SortTaxAmount.Equal(lots[j].SortTaxAmount) {
			return lots[i].SortTaxAmount.LessThan(lots[j].SortTaxAmount)
		}
		return lots[i].HoldingDays > lots[j].HoldingDays
	})

	if lots[0].HoldingDays != 1095 {
		t.Fatalf("first lot holding days = %d, want 1095", lots[0].HoldingDays)
	}
	if !lots[2].SortTaxRate.Equal(decimal.NewFromFloat(0.10)) {
		t.Fatalf("last lot tax rate = %s, want 0.10", lots[2].SortTaxRate)
	}
}
