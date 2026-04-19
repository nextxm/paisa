package xirr

import (
	"math"
	"sort"
	"time"

	"github.com/samber/lo"
	"github.com/shopspring/decimal"
)

func daysBetween(start, end time.Time) float64 {
	const millisecondsPerDay = 1000 * 60 * 60 * 24
	millisBetween := end.Sub(start).Milliseconds()
	return float64(millisBetween) / millisecondsPerDay
}

type Transaction struct {
	Years  float64
	Amount float64
}

type Cashflow struct {
	Date   time.Time
	Amount float64
}

func npv(transactions []Transaction, r float64) float64 {
	v := 0.0
	for _, tx := range transactions {
		base := 1.0 + r
		if base <= 1e-15 {
			// Avoid division by zero or extremely small numbers that overflow
			if tx.Amount > 0 {
				v += 1e100 // Large positive
			} else {
				v -= 1e100 // Large negative
			}
			continue
		}
		v += tx.Amount / math.Pow(base, tx.Years)
	}
	return v
}

func newtonXIRR(transactions []Transaction, initialGuess float64) (float64, bool) {
	x := initialGuess
	const MAX_TRIES = 100
	const EPSILON = 1.0e-9

	for tries := 0; tries < MAX_TRIES; tries++ {
		fxs := 0.0
		dfxs := 0.0
		for _, tx := range transactions {
			base := 1.0 + x
			if base <= 1e-12 {
				return 0, false
			}

			powBaseYears := math.Pow(base, tx.Years)
			fx := tx.Amount / powBaseYears
			dfx := (-tx.Years * tx.Amount) / (powBaseYears * base)
			fxs += fx
			dfxs += dfx
		}

		if dfxs == 0 || math.IsNaN(dfxs) {
			return 0, false
		}

		xNew := x - fxs/dfxs
		if math.IsNaN(xNew) || math.IsInf(xNew, 0) {
			return 0, false
		}

		if math.Abs(xNew-x) <= EPSILON {
			return xNew, true
		}
		x = xNew
	}
	return 0, false
}

func calculateXIRR(transactions []Transaction, initialGuess float64) float64 {
	// Try the guess first.
	if x, ok := newtonXIRR(transactions, initialGuess); ok {
		return x
	}

	// 1. Fine-grained search from the negative end to maintain compatibility with
	// existing tests (favors the lower root when multiple exist).
	// Range: -99.9% to 1,000%.
	for g := -0.999; g <= 10.0; g += 0.01 {
		if x, ok := newtonXIRR(transactions, g); ok {
			return x
		}
	}

	// 2. Extra guesses for extremely high returns (up to 1,000,000%).
	for _, g := range []float64{20.0, 50.0, 100.0, 1000.0, 10000.0} {
		if x, ok := newtonXIRR(transactions, g); ok {
			return x
		}
	}

	// 3. Robust bisection fallback.
	low, high := -0.9999999999, 100000.0
	fLow, fHigh := npv(transactions, low), npv(transactions, high)
	if fLow*fHigh < 0 {
		for i := 0; i < 100; i++ {
			mid := (low + high) / 2
			fMid := npv(transactions, mid)
			if math.Abs(fMid) < 1e-7 || (high-low)/2 < 1e-9 {
				return mid
			}
			if fMid*fLow < 0 {
				high = mid
				fHigh = fMid
			} else {
				low = mid
				fLow = fMid
			}
		}
	} else if math.Abs(fLow) < 1e-7 {
		return low
	} else if math.Abs(fHigh) < 1e-7 {
		return high
	}

	return 0
}

func XIRR(cashflows []Cashflow) decimal.Decimal {
	if len(cashflows) < 2 {
		return decimal.Zero
	}

	// XIRR only exists if there is at least one positive and one negative cashflow.
	hasPos := false
	hasNeg := false
	for _, cf := range cashflows {
		if cf.Amount > 0 {
			hasPos = true
		} else if cf.Amount < 0 {
			hasNeg = true
		}
	}
	if !hasPos || !hasNeg {
		return decimal.Zero
	}

	sort.Slice(cashflows, func(i, j int) bool { return cashflows[i].Date.Before(cashflows[j].Date) })
	transactions := lo.Map(cashflows, func(cf Cashflow, _ int) Transaction {
		return Transaction{
			Years:  daysBetween(cashflows[0].Date, cf.Date) / 365,
			Amount: cf.Amount,
		}
	})
	return decimal.NewFromFloat(calculateXIRR(transactions, 0.1) * 100).Round(2)
}
