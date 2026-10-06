package query

import (
	"context"
	"errors"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	dbutil "github.com/ananthakumaran/paisa/internal/db"
	sqlcdb "github.com/ananthakumaran/paisa/internal/db/sqlc"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// AccountCommoditySum holds aggregated totals for a single (account, commodity)
// pair.  It is returned by Query.GroupSum and used as a lightweight alternative
// to loading every individual posting row when only totals are required.
type AccountCommoditySum struct {
	Account   string
	Commodity string
	Amount    decimal.Decimal
	Quantity  decimal.Decimal
}

type Query struct {
	context                 *gorm.DB
	order                   string
	includeForecast         bool
	limit                   int
	offset                  int
	hasFromDate             bool
	fromDate                time.Time
	hasToDate               bool
	toDate                  time.Time
	statusFilter            string
	creditOnly              bool
	accountPrefixes         []string
	excludedAccountPrefixes []string
	likePatterns            []string
	notLikePatterns         []string
	commodities             []string
	excludedAccounts        []string
	accountFilter           string
	useLegacy               bool
}

func Init(db *gorm.DB) *Query {
	return &Query{context: db, order: "ASC", includeForecast: false}
}

func (q *Query) Desc() *Query {
	q.order = "DESC"
	return q
}

func (q *Query) Forecast() *Query {
	q.includeForecast = true
	return q
}

func (q *Query) Limit(n int) *Query {
	q.context = q.context.Limit(n)
	q.limit = n
	return q
}

func (q *Query) Offset(n int) *Query {
	q.context = q.context.Offset(n)
	q.offset = n
	return q
}

func (q *Query) Clone() *Query {
	return &Query{
		context:                 q.context.Session(&gorm.Session{}),
		order:                   q.order,
		includeForecast:         q.includeForecast,
		limit:                   q.limit,
		offset:                  q.offset,
		hasFromDate:             q.hasFromDate,
		fromDate:                q.fromDate,
		hasToDate:               q.hasToDate,
		toDate:                  q.toDate,
		statusFilter:            q.statusFilter,
		creditOnly:              q.creditOnly,
		accountPrefixes:         append([]string(nil), q.accountPrefixes...),
		excludedAccountPrefixes: append([]string(nil), q.excludedAccountPrefixes...),
		likePatterns:            append([]string(nil), q.likePatterns...),
		notLikePatterns:         append([]string(nil), q.notLikePatterns...),
		commodities:             append([]string(nil), q.commodities...),
		excludedAccounts:        append([]string(nil), q.excludedAccounts...),
		accountFilter:           q.accountFilter,
		useLegacy:               q.useLegacy,
	}
}

func (q *Query) setFromDate(date time.Time) {
	if !q.hasFromDate || date.After(q.fromDate) {
		q.fromDate = date
		q.hasFromDate = true
	}
}

func (q *Query) setToDate(date time.Time) {
	if !q.hasToDate || date.Before(q.toDate) {
		q.toDate = date
		q.hasToDate = true
	}
}

func (q *Query) BeforeNMonths(n int) *Query {
	monthStart := utils.BeginningOfMonth(utils.Now())
	start := monthStart.AddDate(0, -(n - 1), 0)
	q.context = q.context.Where("date < ?", start)
	q.setToDate(start.Add(-time.Nanosecond))
	return q
}

func (q *Query) UntilToday() *Query {
	end := utils.EndOfToday()
	q.context = q.context.Where("date < ?", end)
	q.setToDate(end.Add(-time.Nanosecond))
	return q
}

func (q *Query) UntilDate(date time.Time) *Query {
	end := utils.EndOfDay(date)
	q.context = q.context.Where("date <= ?", end)
	q.setToDate(end)
	return q
}

func (q *Query) UntilThisMonthEnd() *Query {
	end := utils.EndOfMonth(utils.Now())
	q.context = q.context.Where("date <= ?", end)
	q.setToDate(end)
	return q
}

func (q *Query) LastNMonths(n int) *Query {
	monthStart := utils.BeginningOfMonth(utils.Now())
	start := monthStart.AddDate(0, -(n - 1), 0)
	end := monthStart.AddDate(0, 1, 0)
	q.context = q.context.Where("date >= ? and date < ?", start, end)
	q.setFromDate(start)
	q.setToDate(end.Add(-time.Nanosecond))
	return q
}

func (q *Query) Commodities(commodities []config.Commodity) *Query {
	q.context = q.context.Where("commodity in ?", lo.Map(commodities, func(c config.Commodity, _ int) string { return c.Name }))
	q.commodities = lo.Map(commodities, func(c config.Commodity, _ int) string { return c.Name })
	return q
}

func (q *Query) Status(status string) *Query {
	if status == "cleared" || status == "pending" {
		q.context = q.context.Where("status = ?", status)
		q.statusFilter = status
	}
	return q
}

func (q *Query) Credit() *Query {
	q.context = q.context.Where("amount > 0")
	q.creditOnly = true
	return q
}

func (q *Query) AccountPrefix(account ...string) *Query {
	query := "account like ? or account = ?"
	for range account[1:] {
		query += " or account like ? or account = ?"
	}

	args := make([]interface{}, len(account)*2)
	for i, a := range account {
		args[i*2] = a + ":%"
		args[i*2+1] = a
	}
	q.context = q.context.Where(query, args...)
	q.accountPrefixes = append(q.accountPrefixes, account...)
	return q
}

func (q *Query) NotAccountPrefix(accounts ...string) *Query {
	for _, account := range accounts {
		q.context = q.context.Where("account not like ? and account != ?", account+":%", account)
	}
	q.excludedAccountPrefixes = append(q.excludedAccountPrefixes, accounts...)
	return q
}

func (q *Query) NotInactive() *Query {
	conf := config.GetConfig()
	for _, account := range conf.InactiveAccounts {
		q.NotAccountPrefix(account)
	}

	var inactiveAccounts []string
	for _, a := range conf.Accounts {
		if a.Inactive {
			inactiveAccounts = append(inactiveAccounts, a.Name)
		}
	}
	if len(inactiveAccounts) > 0 {
		q.context = q.context.Where("account not in ?", inactiveAccounts)
		q.excludedAccounts = append(q.excludedAccounts, inactiveAccounts...)
	}

	return q
}

func (q *Query) Like(accounts ...string) *Query {
	query := "account like ?"
	for range accounts[1:] {
		query += " or account like ?"
	}

	args := make([]interface{}, len(accounts))
	for i, a := range accounts {
		args[i] = a
	}
	q.context = q.context.Where(query, args...)
	q.likePatterns = append(q.likePatterns, accounts...)
	return q
}

func (q *Query) NotLike(account string) *Query {
	q.context = q.context.Where("account not like ?", account)
	q.notLikePatterns = append(q.notLikePatterns, account)
	return q
}

func (q *Query) Where(query interface{}, args ...interface{}) *Query {
	q.context = q.context.Where(query, args...)

	queryString, ok := query.(string)
	if !ok {
		q.useLegacy = true
		return q
	}

	switch {
	case queryString == "account = ?" && len(args) == 1:
		if account, ok := args[0].(string); ok {
			q.accountFilter = account
			return q
		}
	case queryString == "date between ? AND ?" && len(args) == 2:
		start, startOK := args[0].(time.Time)
		end, endOK := args[1].(time.Time)
		if startOK && endOK {
			q.setFromDate(start)
			q.setToDate(end)
			return q
		}
	}

	q.useLegacy = true
	return q
}

func (q *Query) postingParams() (sqlcdb.ListPostingsAscParams, sqlcdb.ListPostingsDescParams) {
	baseAsc := sqlcdb.ListPostingsAscParams{
		Forecast:          dbutil.NullBool(q.includeForecast),
		Column2:           dbutil.BoolFlag(q.hasFromDate),
		Date:              dbutil.NullTime(q.fromDate),
		Column4:           dbutil.BoolFlag(q.hasToDate),
		Date_2:            dbutil.NullTime(q.toDate),
		Column6:           q.statusFilter,
		Column7:           dbutil.BoolFlag(q.creditOnly),
		Column8:           q.accountFilter,
		JsonArrayLength:   dbutil.JSONStringArray(q.commodities),
		JsonArrayLength_2: dbutil.JSONStringArray(q.accountPrefixes),
		JsonArrayLength_3: dbutil.JSONStringArray(q.excludedAccountPrefixes),
		JsonArrayLength_4: dbutil.JSONStringArray(q.likePatterns),
		JsonArrayLength_5: dbutil.JSONStringArray(q.notLikePatterns),
		JsonArrayLength_6: dbutil.JSONStringArray(q.excludedAccounts),
		Offset:            int64(q.offset),
		Column16:          int64(q.limit),
	}
	baseDesc := sqlcdb.ListPostingsDescParams{
		Forecast:          baseAsc.Forecast,
		Column2:           baseAsc.Column2,
		Date:              baseAsc.Date,
		Column4:           baseAsc.Column4,
		Date_2:            baseAsc.Date_2,
		Column6:           baseAsc.Column6,
		Column7:           baseAsc.Column7,
		Column8:           baseAsc.Column8,
		JsonArrayLength:   baseAsc.JsonArrayLength,
		JsonArrayLength_2: baseAsc.JsonArrayLength_2,
		JsonArrayLength_3: baseAsc.JsonArrayLength_3,
		JsonArrayLength_4: baseAsc.JsonArrayLength_4,
		JsonArrayLength_5: baseAsc.JsonArrayLength_5,
		JsonArrayLength_6: baseAsc.JsonArrayLength_6,
		Offset:            baseAsc.Offset,
		Column16:          baseAsc.Column16,
	}
	return baseAsc, baseDesc
}

func (q *Query) groupParams() sqlcdb.GroupPostingSumsParams {
	return sqlcdb.GroupPostingSumsParams{
		Forecast:          dbutil.NullBool(q.includeForecast),
		Column2:           dbutil.BoolFlag(q.hasFromDate),
		Date:              dbutil.NullTime(q.fromDate),
		Column4:           dbutil.BoolFlag(q.hasToDate),
		Date_2:            dbutil.NullTime(q.toDate),
		Column6:           q.statusFilter,
		Column7:           dbutil.BoolFlag(q.creditOnly),
		Column8:           q.accountFilter,
		JsonArrayLength:   dbutil.JSONStringArray(q.commodities),
		JsonArrayLength_2: dbutil.JSONStringArray(q.accountPrefixes),
		JsonArrayLength_3: dbutil.JSONStringArray(q.excludedAccountPrefixes),
		JsonArrayLength_4: dbutil.JSONStringArray(q.likePatterns),
		JsonArrayLength_5: dbutil.JSONStringArray(q.notLikePatterns),
		JsonArrayLength_6: dbutil.JSONStringArray(q.excludedAccounts),
	}
}

func mapPosting(row sqlcdb.Posting) posting.Posting {
	mapped := posting.Posting{
		ID:                   uint(row.ID),
		TransactionID:        row.TransactionID.String,
		Payee:                row.Payee.String,
		Account:              row.Account.String,
		Commodity:            row.Commodity.String,
		Quantity:             row.Quantity,
		Amount:               row.Amount,
		OriginalAmount:       row.OriginalAmount,
		Status:               row.Status.String,
		TagRecurring:         row.TagRecurring.String,
		TagPeriod:            row.TagPeriod.String,
		TransactionBeginLine: uint64(row.TransactionBeginLine.Int64),
		TransactionEndLine:   uint64(row.TransactionEndLine.Int64),
		FileName:             row.FileName.String,
		Forecast:             row.Forecast.Bool,
		Note:                 row.Note.String,
		TransactionNote:      row.TransactionNote.String,
		TransactionHash:      row.TransactionHash.String,
	}
	if row.Date.Valid {
		mapped.Date = utils.ToDate(row.Date.Time)
	}
	mapped.PreserveOriginalAmount()
	return mapped
}

func parseDecimal(value string) decimal.Decimal {
	if value == "" {
		return decimal.Zero
	}
	d, err := decimal.NewFromString(value)
	if err != nil {
		log.Fatal(err)
	}
	return d
}

func (q *Query) GroupSum() []AccountCommoditySum {
	if q.useLegacy {
		return q.legacyGroupSum()
	}

	rows, err := dbutil.Queries(q.context).GroupPostingSums(context.Background(), q.groupParams())
	if err != nil {
		log.Fatal(err)
	}
	return lo.Map(rows, func(row sqlcdb.GroupPostingSumsRow, _ int) AccountCommoditySum {
		return AccountCommoditySum{
			Account:   row.Account.String,
			Commodity: row.Commodity.String,
			Amount:    parseDecimal(row.Amount),
			Quantity:  parseDecimal(row.Quantity),
		}
	})
}

func (q *Query) All() []posting.Posting {
	if q.useLegacy {
		return q.legacyAll()
	}

	queries := dbutil.Queries(q.context)
	ascParams, descParams := q.postingParams()
	var (
		rows []sqlcdb.Posting
		err  error
	)
	if q.order == "DESC" {
		rows, err = queries.ListPostingsDesc(context.Background(), descParams)
	} else {
		rows, err = queries.ListPostingsAsc(context.Background(), ascParams)
	}
	if err != nil {
		log.Fatal(err)
	}
	return lo.Map(rows, func(row sqlcdb.Posting, _ int) posting.Posting { return mapPosting(row) })
}

func (q *Query) First() *posting.Posting {
	if q.useLegacy {
		return q.legacyFirst()
	}

	clone := q.Clone()
	clone.limit = 1
	clone.offset = 0
	postings := clone.All()
	if len(postings) == 0 {
		return nil
	}
	return &postings[0]
}

func (q *Query) legacyGroupSum() []AccountCommoditySum {
	var sums []AccountCommoditySum
	q.context = q.context.Where("forecast = ?", q.includeForecast)
	result := q.context.Model(&posting.Posting{}).
		Select("account, commodity, SUM(amount) as amount, SUM(quantity) as quantity").
		Group("account, commodity").
		Scan(&sums)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return sums
}

func (q *Query) legacyAll() []posting.Posting {
	var postings []posting.Posting

	q.context = q.context.Where("forecast = ?", q.includeForecast)
	result := q.context.Order("date " + q.order + ", amount desc, account asc").Find(&postings)
	if result.Error != nil {
		log.Fatal(result.Error)
	}
	return lo.Map(postings, func(p posting.Posting, _ int) posting.Posting {
		p.Date = utils.ToDate(p.Date)
		p.PreserveOriginalAmount()
		return p
	})
}

func (q *Query) legacyFirst() *posting.Posting {
	var posting posting.Posting
	q.context = q.context.Where("forecast = ?", q.includeForecast)
	result := q.context.Order("date " + q.order + ", amount desc, account asc").First(&posting)

	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil
		}
		log.Fatal(result.Error)
	}
	posting.Date = utils.ToDate(posting.Date)
	posting.PreserveOriginalAmount()
	return &posting
}
