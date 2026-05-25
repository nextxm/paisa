package server

import (
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/ananthakumaran/paisa/internal/accounting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/model/transaction_tag"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"

	"gorm.io/gorm"
)

type transactionWithTags struct {
	transaction.Transaction
	Tags []string `json:"tags"`
}

func attachTags(db *gorm.DB, transactions []transaction.Transaction) ([]transactionWithTags, error) {
	ids := lo.Map(transactions, func(t transaction.Transaction, _ int) string { return t.ID })
	byTransaction, err := transaction_tag.ListByTransactionIDs(db, ids)
	if err != nil {
		return nil, err
	}
	return lo.Map(transactions, func(t transaction.Transaction, _ int) transactionWithTags {
		return transactionWithTags{Transaction: t, Tags: byTransaction[t.ID]}
	}), nil
}

func GetTransactions(db *gorm.DB) gin.H {
	postings := query.Init(db).Desc().All()
	postings = accounting.PopulateBalance(postings)
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
	tags := lo.FilterMap(strings.Split(c.Query("tags"), ","), func(tag string, _ int) (string, bool) {
		tag = strings.TrimSpace(tag)
		return tag, tag != ""
	})

	q := query.Init(db).Desc()
	if account != "" {
		q = q.AccountPrefix(account)
	}
	postings := q.All()
	postings = accounting.PopulateBalance(postings)
	transactions := transaction.Build(postings)
	if len(tags) > 0 {
		transactionIDs, err := transaction_tag.FindTransactionIDsByTags(db, tags)
		if err != nil {
			RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
			return
		}
		allowed := lo.SliceToMap(transactionIDs, func(id string) (string, struct{}) { return id, struct{}{} })
		transactions = lo.Filter(transactions, func(t transaction.Transaction, _ int) bool {
			_, ok := allowed[t.ID]
			return ok
		})
	}

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

	withTags, err := attachTags(db, transactions)
	if err != nil {
		RespondError(c, http.StatusInternalServerError, ErrCodeInternalError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"transactions": withTags})
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

	sort.Slice(transactions, func(i, j int) bool { return transactions[i].ID > transactions[j].ID })
	sort.SliceStable(transactions, func(i, j int) bool { return transactions[i].Date.After(transactions[j].Date) })

	return transactions
}
