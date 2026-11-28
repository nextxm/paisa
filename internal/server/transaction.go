package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"

	"gorm.io/gorm"
)

func sortTransactionsDesc(transactions []transaction.Transaction) {
	sort.Slice(transactions, func(i, j int) bool {
		if transactions[i].Date.Equal(transactions[j].Date) {
			return transactions[i].ID > transactions[j].ID
		}
		return transactions[i].Date.After(transactions[j].Date)
	})
}

func GetTransactions(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	postings = accounting.PopulateBalance(postings)
	transactions := transaction.Build(postings)
	sortTransactionsDesc(transactions)

	return gin.H{"transactions": transactions}
}

// GetTransactionsHandler handles GET /api/transaction with optional query parameters:
//   - ?q=<text>          – full-text filter on payee and transaction narration
//   - ?amount_min=<n>    – minimum absolute posting amount inside a transaction
//   - ?amount_max=<n>    – maximum absolute posting amount inside a transaction
//   - ?account=<prefix>  – filter to transactions touching the given account prefix
//   - ?commodity=<code>  – filter to transactions containing the given commodity
//   - ?date_from=<date>  – inclusive lower date bound (YYYY-MM-DD)
//   - ?date_to=<date>    – inclusive upper date bound (YYYY-MM-DD)
//   - ?limit=<n>         – return at most n transactions (applied after building transactions)
//   - ?offset=<n>        – skip the first n transactions (applied after building transactions)
func GetTransactionsHandler(db *gorm.DB, c *gin.Context) {
	filters := parseTransactionFilters(c)

	q := query.Init(db).Desc()
	if filters.Account != "" {
		q = q.AccountPrefix(filters.Account)
	}
	if filters.DateFrom != nil {
		q = q.Where("date >= ?", *filters.DateFrom)
	}
	if filters.DateTo != nil {
		q = q.Where("date <= ?", *filters.DateTo)
	}
	if filters.Query != "" {
		like := "%" + strings.ToLower(filters.Query) + "%"
		q = q.Where("LOWER(payee) LIKE ? OR LOWER(transaction_note) LIKE ?", like, like)
	}
	postings := q.All()
	postings = accounting.PopulateBalance(postings)
	transactions := transaction.Build(postings)

	sortTransactionsDesc(transactions)
	if filters.Commodity != "" || filters.AmountMin != nil || filters.AmountMax != nil {
		matcher := newTransactionFilterMatcher(filters)
		transactions = lo.Filter(transactions, func(t transaction.Transaction, _ int) bool {
			return matcher(t)
		})
	}

	// Apply offset and limit at the transaction level to preserve transaction integrity.
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if n, err := strconv.Atoi(offsetStr); err == nil && n > 0 {
			if n >= len(transactions) {
				transactions = nil
			} else {
				transactions = transactions[n:]
			}
		}
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if n, err := strconv.Atoi(limitStr); err == nil && n > 0 && n < len(transactions) {
			transactions = transactions[:n]
		}
	}

	c.JSON(http.StatusOK, gin.H{"transactions": transactions})
}

type transactionFilters struct {
	Query     string
	AmountMin *float64
	AmountMax *float64
	Account   string
	Commodity string
	DateFrom  *time.Time
	DateTo    *time.Time
}

func parseTransactionFilters(c *gin.Context) transactionFilters {
	filters := transactionFilters{
		Query:     strings.TrimSpace(c.Query("q")),
		Account:   strings.TrimSpace(c.Query("account")),
		Commodity: strings.TrimSpace(c.Query("commodity")),
	}

	if v, err := strconv.ParseFloat(c.Query("amount_min"), 64); err == nil {
		filters.AmountMin = lo.ToPtr(v)
	}
	if v, err := strconv.ParseFloat(c.Query("amount_max"), 64); err == nil {
		filters.AmountMax = lo.ToPtr(v)
	}
	if v, err := time.Parse("2006-01-02", c.Query("date_from")); err == nil {
		date := v
		filters.DateFrom = &date
	}
	if v, err := time.Parse("2006-01-02", c.Query("date_to")); err == nil {
		date := utils.EndOfDay(v)
		filters.DateTo = &date
	}

	return filters
}

func newTransactionFilterMatcher(filters transactionFilters) func(transaction.Transaction) bool {
	var minDec, maxDec decimal.Decimal
	hasMin := filters.AmountMin != nil
	if hasMin {
		minDec = decimal.NewFromFloat(*filters.AmountMin)
	}
	hasMax := filters.AmountMax != nil
	if hasMax {
		maxDec = decimal.NewFromFloat(*filters.AmountMax)
	}

	return func(t transaction.Transaction) bool {
		if filters.Commodity != "" {
			hasCommodity := lo.SomeBy(t.Postings, func(p posting.Posting) bool {
				return strings.EqualFold(p.Commodity, filters.Commodity)
			})
			if !hasCommodity {
				return false
			}
		}

		if hasMin || hasMax {
			matched := lo.SomeBy(t.Postings, func(p posting.Posting) bool {
				amount := p.Amount.Abs()
				if hasMin && amount.LessThan(minDec) {
					return false
				}
				if hasMax && amount.GreaterThan(maxDec) {
					return false
				}
				return true
			})
			if !matched {
				return false
			}
		}

		return true
	}
}

func GetBalancedPostings(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	transactions := transaction.Build(postings)
	balancePostings := accounting.BuildBalancedPostings(transactions)

	return gin.H{"balancedPostings": balancePostings}
}

func GetLatestTransactions(db *gorm.DB) []transaction.Transaction {
	postings := query.Init(db).Desc().Limit(200).All()
	postings = accounting.PopulateBalance(postings)
	transactions := transaction.Build(postings)
	sortTransactionsDesc(transactions)

	return transactions
}
