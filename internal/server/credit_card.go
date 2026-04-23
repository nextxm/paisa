package server

import (
	"sort"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/server/liabilities"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type CreditCardSummary struct {
	Account          string                                `json:"account"`
	Network          string                                `json:"network"`
	Number           string                                `json:"number"`
	Currency         string                                `json:"currency"`
	Balance          decimal.Decimal                       `json:"balance"`
	Bills            []CreditCardBill                      `json:"bills"`
	CreditLimit      decimal.Decimal                       `json:"creditLimit"`
	YearlySpends     map[string]map[string]decimal.Decimal `json:"yearlySpends"`
	ExpirationDate   time.Time                             `json:"expirationDate"`
	OriginalBalances []liabilities.OriginalCurrencyBalance `json:"originalBalances"`
}

type CreditCardBill struct {
	StatementStartDate   time.Time       `json:"statementStartDate"`
	StatementEndDate     time.Time       `json:"statementEndDate"`
	DueDate              time.Time       `json:"dueDate"`
	PaidDate             *time.Time      `json:"paidDate"`
	Currency             string          `json:"currency"`
	Credits              decimal.Decimal `json:"credits"`
	Debits               decimal.Decimal `json:"debits"`
	DebitsRunningBalance decimal.Decimal
	OpeningBalance       decimal.Decimal           `json:"openingBalance"`
	ClosingBalance       decimal.Decimal           `json:"closingBalance"`
	Postings             []posting.Posting         `json:"postings"`
	Transactions         []transaction.Transaction `json:"transactions"`
}

func GetCreditCards(db *gorm.DB) gin.H {
	creditCards := []CreditCardSummary{}

	for _, creditCardConfig := range config.GetConfig().CreditCards {
		ps := query.Init(db).Where("account = ?", creditCardConfig.Account).All()
		creditCards = append(creditCards, buildCreditCard(db, creditCardConfig, ps, false))
	}

	return gin.H{"creditCards": creditCards}
}

func GetCreditCard(db *gorm.DB, account string) gin.H {
	for _, creditCardConfig := range config.GetConfig().CreditCards {
		if creditCardConfig.Account == account {
			ps := query.Init(db).Where("account = ?", creditCardConfig.Account).All()
			creditCard := buildCreditCard(db, creditCardConfig, ps, true)
			return gin.H{"creditCard": creditCard, "found": true}
		}
	}

	return gin.H{"found": false}
}

func yearlySpends(db *gorm.DB, date time.Time, postings []posting.Posting) map[string]map[string]decimal.Decimal {
	yearlySpends := make(map[string]map[string]decimal.Decimal)
	for year, ps := range utils.GroupByYearCutoffAt(postings, date) {
		spends := lo.Filter(ps, func(p posting.Posting, _ int) bool {
			return p.Amount.IsNegative() || service.IsContraPostingRefund(db, p)
		})

		yearlySpends[year] = make(map[string]decimal.Decimal)
		for month, ps := range utils.GroupByMonth(spends) {
			yearlySpends[year][month] = accounting.CostSum(ps).Neg()
		}
	}
	return yearlySpends
}

// getPrimaryCommodity returns the most common commodity in the postings.
// If no postings, returns the default currency.
func getPrimaryCommodity(ps []posting.Posting) string {
	if len(ps) == 0 {
		return config.DefaultCurrency()
	}

	commodityCount := make(map[string]int)
	for _, p := range ps {
		if utils.IsCurrency(p.Commodity) {
			commodityCount[config.DefaultCurrency()]++
		} else {
			commodityCount[p.Commodity]++
		}
	}

	if len(commodityCount) == 0 {
		return config.DefaultCurrency()
	}

	maxCommodity := config.DefaultCurrency()
	maxCount := 0
	for commodity, count := range commodityCount {
		if count > maxCount {
			maxCount = count
			maxCommodity = commodity
		}
	}

	return maxCommodity
}

func buildCreditCard(db *gorm.DB, creditCardConfig config.CreditCard, ps []posting.Posting, includePostings bool) CreditCardSummary {
	primaryCommodity := getPrimaryCommodity(ps)
	bills := computeBills(db, creditCardConfig, ps, includePostings, primaryCommodity)
	balance := decimal.Zero
	if len(bills) > 0 {
		balance = bills[len(bills)-1].ClosingBalance
	}

	expirationDate, err := time.ParseInLocation("2006-01-02", creditCardConfig.ExpirationDate, config.TimeZone())
	if err != nil {
		log.Fatal(err)
	}

	ys := make(map[string]map[string]decimal.Decimal)
	if includePostings {
		ys = yearlySpends(db, expirationDate, ps)
	}
	originalBalances := computeCreditCardOriginalBalances(ps)
	return CreditCardSummary{
		Account:          creditCardConfig.Account,
		Network:          creditCardConfig.Network,
		Number:           creditCardConfig.Number,
		Currency:         primaryCommodity,
		Balance:          balance,
		Bills:            bills,
		CreditLimit:      decimal.NewFromInt(int64(creditCardConfig.CreditLimit)),
		YearlySpends:     ys,
		ExpirationDate:   expirationDate,
		OriginalBalances: originalBalances,
	}
}

// computeCreditCardOriginalBalances aggregates the outstanding balance per
// native currency without any FX conversion to the default currency.
func computeCreditCardOriginalBalances(ps []posting.Posting) []liabilities.OriginalCurrencyBalance {
	dc := config.DefaultCurrency()
	currencyAmt := make(map[string]decimal.Decimal)

	for _, p := range ps {
		if utils.IsCurrency(p.Commodity) {
			currencyAmt[dc] = currencyAmt[dc].Add(p.Amount)
		} else {
			currencyAmt[p.Commodity] = currencyAmt[p.Commodity].Add(p.Quantity)
		}
	}

	var result []liabilities.OriginalCurrencyBalance
	for currency, amount := range currencyAmt {
		negated := amount.Neg()
		if !negated.IsZero() {
			result = append(result, liabilities.OriginalCurrencyBalance{Currency: currency, Amount: negated})
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Currency < result[j].Currency
	})
	return result
}

func computeBills(db *gorm.DB, creditCardConfig config.CreditCard, ps []posting.Posting, includePostings bool, primaryCommodity string) []CreditCardBill {
	// Filter postings to only include the primary commodity
	filteredPs := lo.Filter(ps, func(p posting.Posting, _ int) bool {
		return (utils.IsCurrency(p.Commodity) && primaryCommodity == config.DefaultCurrency()) ||
			(!utils.IsCurrency(p.Commodity) && p.Commodity == primaryCommodity)
	})

	bills := []CreditCardBill{}

	grouped := accounting.GroupByMonthlyBillingCycle(filteredPs, creditCardConfig.StatementEndDay)

	balance := decimal.Zero
	creditsRunningBalance := decimal.Zero
	debitsRunningBalance := decimal.Zero
	unpaidBill := 0

	for _, month := range utils.SortedKeys(grouped) {
		statementEndDate, err := time.ParseInLocation("2006-01", month, config.TimeZone())
		if err != nil {
			log.Fatal(err)
		}

		statementEndDate = statementEndDate.AddDate(0, 0, creditCardConfig.StatementEndDay-1)
		statementStartDate := statementEndDate.AddDate(0, -1, 1)

		var dueDate time.Time
		if creditCardConfig.StatementEndDay < creditCardConfig.DueDay {
			dueDate = utils.BeginningOfMonth(statementEndDate).AddDate(0, 0, creditCardConfig.DueDay-1)
		} else {
			dueDate = utils.BeginningOfMonth(statementEndDate).AddDate(0, 1, creditCardConfig.DueDay-1)
		}

		bill := CreditCardBill{
			StatementStartDate: statementStartDate,
			StatementEndDate:   statementEndDate,
			DueDate:            dueDate,
			Currency:           primaryCommodity,
			OpeningBalance:     balance,
			Postings:           []posting.Posting{},
			Transactions:       []transaction.Transaction{},
		}

		transactionIDs := map[string]bool{}

		for _, p := range grouped[month] {
			// Use Quantity (original amount) instead of Amount (converted amount)
			amount := p.Quantity
			if utils.IsCurrency(p.Commodity) {
				amount = p.Amount
			}

			balance = balance.Add(amount.Neg())

			if amount.IsPositive() {
				creditsRunningBalance = creditsRunningBalance.Add(amount)
				bill.Credits = bill.Credits.Add(amount)
				for unpaidBill < len(bills) {
					if bills[unpaidBill].DebitsRunningBalance.LessThanOrEqual(creditsRunningBalance) {
						paidDate := p.Date
						bills[unpaidBill].PaidDate = &paidDate
						unpaidBill++
					} else {
						break
					}
				}
			} else {
				bill.Debits = bill.Debits.Add(amount.Neg())
				debitsRunningBalance = debitsRunningBalance.Add(amount.Neg())
			}

			if includePostings {
				bill.Postings = append(bill.Postings, p)
				transactionIDs[p.TransactionID] = true
			}

		}

		bill.DebitsRunningBalance = debitsRunningBalance
		bill.ClosingBalance = balance
		bill.Transactions = lo.Map(lo.Keys(transactionIDs), func(id string, _ int) transaction.Transaction {
			t, _ := transaction.GetById(db, id)
			return t
		})
		accounting.SortTransactionAsc(bill.Transactions)
		bills = append(bills, bill)
	}

	return bills
}
