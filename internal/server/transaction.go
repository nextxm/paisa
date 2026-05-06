package server

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/gin-gonic/gin"

	"gorm.io/gorm"
)

func GetTransactions(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	transactions := transaction.Build(postings)

	sort.Slice(transactions, func(i, j int) bool { return transactions[i].ID > transactions[j].ID })
	sort.SliceStable(transactions, func(i, j int) bool { return transactions[i].Date.After(transactions[j].Date) })

	return gin.H{"transactions": transactions}
}

// GetTransactionsHandler handles GET /api/transaction with optional query parameters:
//   - ?account=<prefix>  – filter to transactions touching the given account prefix
//   - ?limit=<n>         – return at most n transactions (applied after building transactions)
//   - ?offset=<n>        – skip the first n transactions (applied after building transactions)
func GetTransactionsHandler(db *gorm.DB, c *gin.Context) {
	account := c.Query("account")

	q := query.Init(db).Desc()
	if account != "" {
		q = q.AccountPrefix(account)
	}
	postings := q.All()
	postings = accounting.PopulateBalance(postings)
	transactions := transaction.Build(postings)

	sort.Slice(transactions, func(i, j int) bool { return transactions[i].ID > transactions[j].ID })
	sort.SliceStable(transactions, func(i, j int) bool { return transactions[i].Date.After(transactions[j].Date) })

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

func GetBalancedPostings(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	transactions := transaction.Build(postings)
	balancePostings := accounting.BuildBalancedPostings(transactions)

	return gin.H{"balancedPostings": balancePostings}
}

func GetLatestTransactions(db *gorm.DB) []transaction.Transaction {
	postings := query.Init(db).Desc().Limit(200).All()
	transactions := transaction.Build(postings)

	sort.Slice(transactions, func(i, j int) bool { return transactions[i].ID > transactions[j].ID })
	sort.SliceStable(transactions, func(i, j int) bool { return transactions[i].Date.After(transactions[j].Date) })

	return transactions
}
