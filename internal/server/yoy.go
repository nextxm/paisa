package server

import (
	"fmt"
	"strconv"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
)

type YoYMonthlySeries struct {
	Month map[string]decimal.Decimal `json:"month"`
	Total decimal.Decimal            `json:"total"`
}

func parseYearsParam(raw string) int {
	if raw == "" {
		return 1
	}

	years, err := strconv.Atoi(raw)
	if err != nil || years < 1 {
		return 1
	}

	if years > 10 {
		return 10
	}

	return years
}

func parseUntilYearParam(raw string) int {
	if raw == "" {
		return utils.Now().Year()
	}

	year, err := strconv.Atoi(raw)
	if err != nil || year < 1900 {
		return utils.Now().Year()
	}

	return year
}

func computeYoYMonthlySeries(postings []posting.Posting, years, untilYear int, amountFn func(decimal.Decimal) decimal.Decimal) map[string]YoYMonthlySeries {
	// Defensive normalization so direct callers (tests/helpers) are still safe
	// even when years is not parsed through parseYearsParam.
	if years < 1 {
		years = 1
	}

	if untilYear <= 0 {
		untilYear = utils.Now().Year()
	}

	if amountFn == nil {
		amountFn = func(amount decimal.Decimal) decimal.Decimal { return amount }
	}

	currentYear := untilYear
	series := map[string]YoYMonthlySeries{}
	for i := 0; i < years; i++ {
		year := currentYear - i
		months := map[string]decimal.Decimal{}
		for month := 1; month <= 12; month++ {
			months[fmt.Sprintf("%04d-%02d", year, month)] = decimal.Zero
		}

		series[strconv.Itoa(year)] = YoYMonthlySeries{
			Month: months,
			Total: decimal.Zero,
		}
	}

	for _, p := range postings {
		year := strconv.Itoa(p.Date.Year())
		current, ok := series[year]
		if !ok {
			continue
		}

		amount := amountFn(p.Amount)
		monthKey := p.Date.Format("2006-01")
		current.Month[monthKey] = current.Month[monthKey].Add(amount)
		current.Total = current.Total.Add(amount)
		series[year] = current
	}

	return series
}
