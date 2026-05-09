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
	BalanceAmount       decimal.Decimal `json:"balanceAmount"`
	BalanceUnits        decimal.Decimal `json:"balanceUnits"`
	NetInvestmentAmount decimal.Decimal `json:"netInvestmentAmount"`
}

func normalizeWindowDecimal(value decimal.Decimal) decimal.Decimal {
	rounded := value.Round(10)
	if value.Sub(rounded).Abs().LessThan(decimal.NewFromFloat(1e-12)) {
		return rounded
	}
	return value
}

func GetNetworth(db *gorm.DB, reportCurrency string) gin.H {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()

	postings = service.PopulateMarketPrice(db, postings)
	networthTimeline := computeNetworthTimeline(db, postings, false)
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
	networthTimeline := computeNetworthTimeline(db, postings, false)
	networth := Networth{}
	if len(networthTimeline) > 0 {
		networth = networthTimeline[len(networthTimeline)-1]
	}
	xirr := service.XIRR(db, postings)
	return gin.H{"networth": networth, "xirr": xirr}
}

func computeNetworth(db *gorm.DB, postings []posting.Posting) Networth {
	var networth Networth

	if len(postings) == 0 {
		return networth
	}

	var investment decimal.Decimal = decimal.Zero
	var withdrawal decimal.Decimal = decimal.Zero
	var balance decimal.Decimal = decimal.Zero

	now := utils.EndOfToday()
	for _, p := range postings {
		isInterest := service.IsInterest(db, p)
		isInterestRepayment := service.IsInterestRepayment(db, p)
		isStockSplit := service.IsStockSplit(db, p)
		isCapitalGains := service.IsCapitalGains(p)

		if isInterest || isInterestRepayment {
			balance = balance.Add(p.Amount)
		} else if isCapitalGains {
			withdrawal = withdrawal.Add(p.Amount.Neg())
		} else {
			if p.Amount.GreaterThan(decimal.Zero) && !isStockSplit {
				investment = investment.Add(service.GetMarketPrice(db, p, p.Date))
			}

			if p.Amount.LessThan(decimal.Zero) && !isStockSplit {
				withdrawal = withdrawal.Add(service.GetMarketPrice(db, p, p.Date).Neg())
			}

			balance = balance.Add(service.GetMarketPrice(db, p, now))
		}
	}

	gain := balance.Add(withdrawal).Sub(investment)
	netInvestment := investment.Sub(withdrawal)
	networth = Networth{
		Date:                now,
		InvestmentAmount:    investment,
		WithdrawalAmount:    withdrawal,
		GainAmount:          gain,
		BalanceAmount:       balance,
		NetInvestmentAmount: netInvestment,
	}

	return networth
}

func computeNetworthTimeline(db *gorm.DB, postings []posting.Posting, computeBalanceUnits bool) []Networth {
	var networths []Networth

	if len(postings) == 0 {
		return []Networth{}
	}

	type RunningSum struct {
		investment   decimal.Decimal
		withdrawal   decimal.Decimal
		balance      decimal.Decimal
		balanceUnits decimal.Decimal
	}

	type runningNetworthRow struct {
		Day          string
		Commodity    string
		Investment   decimal.Decimal
		Withdrawal   decimal.Decimal
		Balance      decimal.Decimal
		BalanceUnits decimal.Decimal
	}

	ids := make([]uint, 0, len(postings))
	for _, p := range postings {
		ids = append(ids, p.ID)
	}

	defaultCurrency := config.DefaultCurrency()
	var rows []runningNetworthRow
	result := db.Raw(`
WITH filtered AS (
	SELECT
		DATE(date) AS day,
		commodity,
		amount,
		quantity,
		account,
		transaction_id,
		payee
	FROM postings
	WHERE id IN ?
),
classified AS (
	SELECT
		f.day,
		f.commodity,
		f.amount,
		f.quantity,
		CASE WHEN f.commodity = ? AND EXISTS (
			SELECT 1
			FROM postings ip
			WHERE DATE(ip.date) = f.day
				AND ip.account LIKE 'Income:Interest:%'
				AND ip.amount = -f.amount
				AND ip.payee = f.payee
			LIMIT 1
		) THEN 1 ELSE 0 END AS is_interest,
		CASE WHEN f.commodity = ? AND (
			f.account LIKE 'Expenses:Interest:%'
			OR EXISTS (
				SELECT 1
				FROM postings ip
				WHERE DATE(ip.date) = f.day
					AND ip.account LIKE 'Expenses:Interest:%'
					AND ip.amount = -f.amount
					AND ip.payee = f.payee
				LIMIT 1
			)
		) THEN 1 ELSE 0 END AS is_interest_repayment,
		CASE WHEN f.account LIKE 'Income:CapitalGains:%' THEN 1 ELSE 0 END AS is_capital_gains,
		CASE WHEN f.commodity <> ? AND NOT EXISTS (
			SELECT 1
			FROM postings tp
			WHERE tp.transaction_id = f.transaction_id
				AND (tp.commodity = ? OR tp.account != f.account)
			LIMIT 1
		) THEN 1 ELSE 0 END AS is_stock_split
	FROM filtered f
),
deltas AS (
	SELECT
		day,
		commodity,
		CASE WHEN is_interest = 1 OR is_interest_repayment = 1 THEN amount ELSE 0 END AS balance_interest_delta,
		CASE WHEN is_capital_gains = 1 THEN -amount ELSE 0 END AS capital_withdrawal_delta,
		CASE
			WHEN is_interest = 0 AND is_interest_repayment = 0 AND is_capital_gains = 0 AND amount > 0 AND is_stock_split = 0
			THEN amount ELSE 0
		END AS investment_delta,
		CASE
			WHEN is_interest = 0 AND is_interest_repayment = 0 AND is_capital_gains = 0 AND amount < 0 AND is_stock_split = 0
			THEN -amount ELSE 0
		END AS withdrawal_delta,
		CASE
			WHEN is_interest = 0 AND is_interest_repayment = 0 AND is_capital_gains = 0
			THEN amount ELSE 0
		END AS balance_delta,
		CASE
			WHEN is_interest = 0 AND is_interest_repayment = 0 AND is_capital_gains = 0
			THEN quantity ELSE 0
		END AS balance_units_delta
	FROM classified
),
daily AS (
	SELECT
		day,
		commodity,
		SUM(investment_delta) AS investment_delta,
		SUM(capital_withdrawal_delta + withdrawal_delta) AS withdrawal_delta,
		SUM(balance_interest_delta + balance_delta) AS balance_delta,
		SUM(balance_units_delta) AS balance_units_delta
	FROM deltas
	GROUP BY day, commodity
),
running AS (
	SELECT
		day,
		commodity,
		SUM(investment_delta) OVER (
			PARTITION BY commodity
			ORDER BY day
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		) AS investment,
		SUM(withdrawal_delta) OVER (
			PARTITION BY commodity
			ORDER BY day
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		) AS withdrawal,
		SUM(balance_delta) OVER (
			PARTITION BY commodity
			ORDER BY day
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		) AS balance,
		SUM(balance_units_delta) OVER (
			PARTITION BY commodity
			ORDER BY day
			ROWS BETWEEN UNBOUNDED PRECEDING AND CURRENT ROW
		) AS balance_units
	FROM daily
)
SELECT day, commodity, investment, withdrawal, balance, balance_units
FROM running
ORDER BY day, commodity
`, ids, defaultCurrency, defaultCurrency, defaultCurrency, defaultCurrency).Scan(&rows)
	if result.Error != nil {
		return []Networth{}
	}
	for i := range rows {
		rows[i].Investment = normalizeWindowDecimal(rows[i].Investment)
		rows[i].Withdrawal = normalizeWindowDecimal(rows[i].Withdrawal)
		rows[i].Balance = normalizeWindowDecimal(rows[i].Balance)
		rows[i].BalanceUnits = normalizeWindowDecimal(rows[i].BalanceUnits)
	}

	accumulator := make(map[string]RunningSum)
	rowIndex := 0

	end := utils.EndOfToday()
	for start := postings[0].Date; start.Before(end); start = start.AddDate(0, 0, 1) {
		dayEnd := utils.EndOfDay(start)
		day := start.Format("2006-01-02")
		for rowIndex < len(rows) && rows[rowIndex].Day == day {
			accumulator[rows[rowIndex].Commodity] = RunningSum{
				investment:   rows[rowIndex].Investment,
				withdrawal:   rows[rowIndex].Withdrawal,
				balance:      rows[rowIndex].Balance,
				balanceUnits: rows[rowIndex].BalanceUnits,
			}
			rowIndex++
		}

		var investment decimal.Decimal = decimal.Zero
		var withdrawal decimal.Decimal = decimal.Zero
		var balance decimal.Decimal = decimal.Zero
		var balanceUnits decimal.Decimal = decimal.Zero

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

		}

		gain := balance.Add(withdrawal).Sub(investment)
		netInvestment := investment.Sub(withdrawal)
		networths = append(networths, Networth{
			Date:                start,
			InvestmentAmount:    investment,
			WithdrawalAmount:    withdrawal,
			GainAmount:          gain,
			BalanceAmount:       balance,
			BalanceUnits:        balanceUnits,
			NetInvestmentAmount: netInvestment,
		})

		if rowIndex == len(rows) && balance.Abs().LessThan(decimal.NewFromFloat(0.01)) {
			break
		}
	}
	return networths
}
