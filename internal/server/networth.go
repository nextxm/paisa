package server

import (
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type Networth struct {
	Date                time.Time       `json:"date"`
	InvestmentAmount    decimal.Decimal `json:"investmentAmount"`
	WithdrawalAmount    decimal.Decimal `json:"withdrawalAmount"`
	GainAmount          decimal.Decimal `json:"gainAmount"`
	Contribution        decimal.Decimal `json:"contribution"`
	InvestmentReturn    decimal.Decimal `json:"investment_return"`
	FXImpact            decimal.Decimal `json:"fx_impact"`
	BalanceAmount       decimal.Decimal `json:"balanceAmount"`
	BalanceUnits        decimal.Decimal `json:"balanceUnits"`
	NetInvestmentAmount decimal.Decimal `json:"netInvestmentAmount"`
}

type networthRunningSum struct {
	investment   decimal.Decimal
	withdrawal   decimal.Decimal
	balance      decimal.Decimal
	balanceUnits decimal.Decimal
}

type networthFXState struct {
	localValue decimal.Decimal
	quote      string
	rate       decimal.Decimal
	hasRate    bool
}

func GetNetworth(db *gorm.DB, reportCurrency string) gin.H {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()

	postings = service.PopulateMarketPrice(db, postings)
	networthTimeline := computeNetworthTimeline(db, postings, false, utils.ToDate(utils.Now()))
	xirr := service.XIRR(db, postings)
	if reportCurrency != "" && reportCurrency != config.DefaultCurrency() {
		networthTimeline = convertNetworthTimelineToReportCurrency(db, networthTimeline, reportCurrency)
	}
	return gin.H{"networthTimeline": networthTimeline, "xirr": xirr}
}

// convertNetworthTimelineToReportCurrency converts each Networth point's amounts
// to reportCurrency.  Each point is converted at the exchange rate on that date.
// When no rate is available for a given date, the point is kept unchanged.
func convertNetworthTimelineToReportCurrency(db *gorm.DB, timeline []Networth, reportCurrency string) []Networth {
	result := make([]Networth, 0, len(timeline))
	defaultCurrency := config.DefaultCurrency()
	for _, n := range timeline {
		rate, ok := service.GetRate(db, defaultCurrency, reportCurrency, n.Date)
		if !ok {
			result = append(result, n)
			continue
		}
		result = append(result, Networth{
			Date:                n.Date,
			InvestmentAmount:    n.InvestmentAmount.Mul(rate),
			WithdrawalAmount:    n.WithdrawalAmount.Mul(rate),
			GainAmount:          n.GainAmount.Mul(rate),
			Contribution:        n.Contribution.Mul(rate),
			InvestmentReturn:    n.InvestmentReturn.Mul(rate),
			FXImpact:            n.FXImpact.Mul(rate),
			BalanceAmount:       n.BalanceAmount.Mul(rate),
			BalanceUnits:        n.BalanceUnits,
			NetInvestmentAmount: n.NetInvestmentAmount.Mul(rate),
		})
	}
	return result
}

func GetCurrentNetworth(db *gorm.DB) gin.H {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()
	postings = service.PopulateMarketPrice(db, postings)
	networthTimeline := computeNetworthTimeline(db, postings, false, utils.ToDate(utils.Now()))
	networth := Networth{}
	if len(networthTimeline) > 0 {
		networth = networthTimeline[len(networthTimeline)-1]
	}
	xirr := service.XIRR(db, postings)
	return gin.H{"networth": networth, "xirr": xirr}
}

func computeNetworth(db *gorm.DB, postings []posting.Posting) Networth {
	if len(postings) == 0 {
		return Networth{}
	}

	networthTimeline := computeNetworthTimeline(db, postings, false, utils.ToDate(utils.Now()))
	if len(networthTimeline) == 0 {
		return Networth{}
	}
	return networthTimeline[len(networthTimeline)-1]
}

func computeNetworthTimeline(db *gorm.DB, postings []posting.Posting, computeBalanceUnits bool, asOfDate time.Time) []Networth {
	var networths []Networth

	var p posting.Posting

	if len(postings) == 0 {
		return []Networth{}
	}

	accumulator := make(map[string]networthRunningSum)
	fxStateByCommodity := make(map[string]networthFXState)
	cumulativeFXImpact := decimal.Zero

	end := utils.EndOfDay(asOfDate)
	for start := postings[0].Date; start.Before(end); start = start.AddDate(0, 0, 1) {
		dayEnd := utils.EndOfDay(start)
		for len(postings) > 0 && !postings[0].Date.After(dayEnd) {
			p, postings = postings[0], postings[1:]
			rs := accumulator[p.Commodity]
			isInterest := service.IsInterest(db, p)
			isInterestRepayment := service.IsInterestRepayment(db, p)
			isStockSplit := service.IsStockSplit(db, p)
			isCapitalGains := service.IsCapitalGains(p)
			isInvestmentIncome := isInvestmentIncomePosting(p)

			if isInterest || isInterestRepayment {
				rs.balance = rs.balance.Add(p.Amount)
			} else if isCapitalGains || isInvestmentIncome {
				rs.withdrawal = rs.withdrawal.Add(p.Amount.Neg())
			} else {
				if p.Amount.GreaterThan(decimal.Zero) && !isStockSplit {
					rs.investment = rs.investment.Add(service.GetMarketPrice(db, p, p.Date))
				}

				if p.Amount.LessThan(decimal.Zero) && !isStockSplit {
					rs.withdrawal = rs.withdrawal.Add(service.GetMarketPrice(db, p, p.Date).Neg())
				}

				rs.balance = rs.balance.Add(service.GetMarketPrice(db, p, dayEnd))
				rs.balanceUnits = rs.balanceUnits.Add(p.Quantity)
			}

			accumulator[p.Commodity] = rs

		}

		var investment decimal.Decimal = decimal.Zero
		var withdrawal decimal.Decimal = decimal.Zero
		var balance decimal.Decimal = decimal.Zero
		var balanceUnits decimal.Decimal = decimal.Zero
		dayFXImpact := decimal.Zero

		for commodity, rs := range accumulator {
			investment = investment.Add(rs.investment)
			withdrawal = withdrawal.Add(rs.withdrawal)

			if utils.IsCurrency(commodity) {
				balance = balance.Add(rs.balance)
			} else {
				if computeBalanceUnits {
					balanceUnits = balanceUnits.Add(rs.balanceUnits)
				}
				price := service.GetUnitPrice(db, commodity, dayEnd)
				if !price.Value.Equal(decimal.Zero) {
					balance = balance.Add(rs.balanceUnits.Mul(price.Value))
				} else {
					balance = balance.Add(rs.balance)
				}
			}

			dayFXImpact = dayFXImpact.Add(computeFXImpactForCommodity(db, commodity, rs, dayEnd, fxStateByCommodity))
		}

		gain := balance.Add(withdrawal).Sub(investment)
		netInvestment := investment.Sub(withdrawal)
		cumulativeFXImpact = cumulativeFXImpact.Add(dayFXImpact)
		investmentReturn := gain.Sub(cumulativeFXImpact)
		networths = append(networths, Networth{
			Date:                start,
			InvestmentAmount:    investment,
			WithdrawalAmount:    withdrawal,
			GainAmount:          gain,
			Contribution:        netInvestment,
			InvestmentReturn:    investmentReturn,
			FXImpact:            cumulativeFXImpact,
			BalanceAmount:       balance,
			BalanceUnits:        balanceUnits,
			NetInvestmentAmount: netInvestment,
		})

		if len(postings) == 0 && balance.Abs().LessThan(decimal.NewFromFloat(0.01)) {
			break
		}
	}
	return networths
}

func computeFXImpactForCommodity(
	db *gorm.DB,
	commodity string,
	rs networthRunningSum,
	dayEnd time.Time,
	fxStateByCommodity map[string]networthFXState,
) decimal.Decimal {
	localValue, quote, rate, hasRate, trackFX := commodityFXState(db, commodity, rs.balanceUnits, dayEnd)
	if !trackFX || localValue.IsZero() {
		delete(fxStateByCommodity, commodity)
		return decimal.Zero
	}

	prev, hasPrev := fxStateByCommodity[commodity]
	fxStateByCommodity[commodity] = networthFXState{
		localValue: localValue,
		quote:      quote,
		rate:       rate,
		hasRate:    hasRate,
	}

	if !hasPrev || !prev.hasRate || !hasRate || prev.quote != quote {
		return decimal.Zero
	}

	return prev.localValue.Mul(rate.Sub(prev.rate))
}

func commodityFXState(
	db *gorm.DB,
	commodity string,
	quantity decimal.Decimal,
	date time.Time,
) (localValue decimal.Decimal, quote string, rate decimal.Decimal, hasRate bool, trackFX bool) {
	defaultCurrency := config.DefaultCurrency()
	if commodity == defaultCurrency {
		return decimal.Zero, "", decimal.Zero, false, false
	}

	if service.IsForeignCurrency(commodity) {
		r, ok := service.GetRate(db, commodity, defaultCurrency, date)
		return quantity, commodity, r, ok, true
	}

	nativePrice, nativeQuote, ok := service.GetNativeUnitPrice(db, commodity, date)
	if !ok || nativeQuote == "" || nativeQuote == defaultCurrency {
		return decimal.Zero, "", decimal.Zero, false, false
	}

	r, found := service.GetRate(db, nativeQuote, defaultCurrency, date)
	return quantity.Mul(nativePrice), nativeQuote, r, found, true
}

func commodityDenominationCurrency(db *gorm.DB, commodity string, date time.Time) string {
	defaultCurrency := config.DefaultCurrency()
	if commodity == defaultCurrency {
		return defaultCurrency
	}

	if service.IsForeignCurrency(commodity) {
		return commodity
	}

	_, nativeQuote, ok := service.GetNativeUnitPrice(db, commodity, date)
	if ok && nativeQuote != "" {
		return nativeQuote
	}

	return defaultCurrency
}
