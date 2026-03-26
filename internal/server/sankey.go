package server

import (
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const sankeyDateLayout = "2006-01-02"

// sankeyEpsilon is the minimum absolute flow value included in the response.
// Links with |value| < sankeyEpsilon are dropped to avoid rendering clutter.
var sankeyEpsilon = decimal.NewFromFloat(0.001)

// SankeyNodeKind classifies a ledger account for Sankey visualization.
type SankeyNodeKind string

const (
	SankeyKindIncome    SankeyNodeKind = "income"
	SankeyKindAsset     SankeyNodeKind = "asset"
	SankeyKindLiability SankeyNodeKind = "liability"
	SankeyKindExpense   SankeyNodeKind = "expense"
	SankeyKindEquity    SankeyNodeKind = "equity"
	SankeyKindOther     SankeyNodeKind = "other"
)

// SankeyNode represents a unique account in the Sankey flow graph.
type SankeyNode struct {
	ID   string         `json:"id"`
	Name string         `json:"name"`
	Kind SankeyNodeKind `json:"kind"`
}

// sankeyLinkKey identifies a directed flow between two node IDs.
type sankeyLinkKey struct {
	Source string
	Target string
}

// SankeyLink represents an aggregated directed flow between two nodes.
type SankeyLink struct {
	Source   string          `json:"source"`
	Target   string          `json:"target"`
	Value    decimal.Decimal `json:"value"`
	TxnCount int             `json:"txnCount"`
}

// SankeyMeta holds request metadata echoed back in the response.
type SankeyMeta struct {
	From         string          `json:"from"`
	To           string          `json:"to"`
	Period       string          `json:"period"`
	Currency     string          `json:"currency"`
	TotalInflow  decimal.Decimal `json:"totalInflow"`
	TotalOutflow decimal.Decimal `json:"totalOutflow"`
}

// SankeyResponse is the full payload returned by GET /api/sankey.
type SankeyResponse struct {
	Nodes []SankeyNode `json:"nodes"`
	Links []SankeyLink `json:"links"`
	Meta  SankeyMeta   `json:"meta"`
}

// sankeyQueryParams holds the validated query parameters for GET /api/sankey.
type sankeyQueryParams struct {
	From     string `form:"from"`
	To       string `form:"to"`
	Period   string `form:"period"`
	Currency string `form:"currency"`
}

// classifySankeyAccount returns the SankeyNodeKind for the given account name
// based on its top-level prefix.
func classifySankeyAccount(account string) SankeyNodeKind {
	top := strings.SplitN(account, ":", 2)[0]
	switch top {
	case "Income":
		return SankeyKindIncome
	case "Expenses":
		return SankeyKindExpense
	case "Assets":
		return SankeyKindAsset
	case "Liabilities":
		return SankeyKindLiability
	case "Equity":
		return SankeyKindEquity
	default:
		return SankeyKindOther
	}
}

// sankeyPeriodBounds returns the [from, to] inclusive date bounds for the
// given period string relative to utils.Now().
func sankeyPeriodBounds(period string) (time.Time, time.Time) {
	now := utils.Now()
	switch period {
	case "quarter":
		// Align to the start of the current calendar quarter.
		month := int(now.Month())
		quarterStartMonth := time.Month(((month-1)/3)*3 + 1)
		start := time.Date(now.Year(), quarterStartMonth, 1, 0, 0, 0, 0, now.Location())
		end := utils.EndOfMonth(start.AddDate(0, 2, 0))
		return start, end
	case "year":
		start := time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, now.Location())
		end := utils.EndOfMonth(time.Date(now.Year(), time.December, 1, 0, 0, 0, 0, now.Location()))
		return start, end
	default: // "month"
		start := utils.BeginningOfMonth(now)
		end := utils.EndOfMonth(now)
		return start, end
	}
}

// computeSankeyGraph derives Sankey nodes and links from a slice of postings.
//
// For each transaction the function allocates flows from "source" postings
// (amount < 0, i.e. the account that money leaves) to "destination" postings
// (amount > 0, i.e. the account that money enters).  Self-links and links
// below sankeyEpsilon are dropped.
func computeSankeyGraph(postings []posting.Posting) ([]SankeyNode, []SankeyLink) {
	if len(postings) == 0 {
		return []SankeyNode{}, []SankeyLink{}
	}

	nodeSet := make(map[string]SankeyNodeKind)
	linkValues := make(map[sankeyLinkKey]decimal.Decimal)
	linkTxnIDs := make(map[sankeyLinkKey]map[string]struct{})

	transactions := transaction.Build(postings)

	for _, txn := range transactions {
		from := lo.Filter(txn.Postings, func(p posting.Posting, _ int) bool {
			return p.Amount.IsNegative()
		})
		to := lo.Filter(txn.Postings, func(p posting.Posting, _ int) bool {
			return p.Amount.IsPositive()
		})

		for _, f := range from {
			nodeSet[f.Account] = classifySankeyAccount(f.Account)
			// remainingToAllocate tracks the unallocated portion of this source
			// posting; we greedily assign it to destination postings in order
			// until the source amount is fully consumed.
			remainingToAllocate := f.Amount.Neg() // convert negative amount to positive budget
			// toCopy is a local copy so we can safely mutate Amount inline to
			// track how much of each destination posting is still unallocated.
			toCopy := make([]posting.Posting, len(to))
			copy(toCopy, to)
			for remainingToAllocate.GreaterThan(sankeyEpsilon) && len(toCopy) > 0 {
				dest := toCopy[0]
				nodeSet[dest.Account] = classifySankeyAccount(dest.Account)

				if dest.Account == f.Account {
					// skip self-links
					toCopy = toCopy[1:]
					continue
				}

				// Allocate as much of the destination as the remaining source allows.
				var flow decimal.Decimal
				if dest.Amount.GreaterThan(remainingToAllocate) {
					// Destination is larger than remaining source: consume source fully.
					flow = remainingToAllocate
					toCopy[0].Amount = dest.Amount.Sub(remainingToAllocate)
					remainingToAllocate = decimal.Zero
				} else {
					// Destination is smaller or equal: consume destination fully, move on.
					flow = dest.Amount
					remainingToAllocate = remainingToAllocate.Sub(dest.Amount)
					toCopy = toCopy[1:]
				}

				if flow.GreaterThan(sankeyEpsilon) {
					key := sankeyLinkKey{Source: f.Account, Target: dest.Account}
					linkValues[key] = linkValues[key].Add(flow)
					if linkTxnIDs[key] == nil {
						linkTxnIDs[key] = make(map[string]struct{})
					}
					linkTxnIDs[key][txn.ID] = struct{}{}
				}
			}
		}
	}

	// Build deterministically ordered node slice.
	nodeNames := lo.Keys(nodeSet)
	sort.Strings(nodeNames)
	nodes := lo.Map(nodeNames, func(name string, _ int) SankeyNode {
		return SankeyNode{ID: name, Name: name, Kind: nodeSet[name]}
	})

	// Build deterministically ordered link slice, dropping zero/tiny values.
	type rawLink struct {
		key   sankeyLinkKey
		value decimal.Decimal
		count int
	}
	var rawLinks []rawLink
	for key, val := range linkValues {
		if val.GreaterThan(sankeyEpsilon) {
			rawLinks = append(rawLinks, rawLink{
				key:   key,
				value: val,
				count: len(linkTxnIDs[key]),
			})
		}
	}
	sort.Slice(rawLinks, func(i, j int) bool {
		if rawLinks[i].key.Source != rawLinks[j].key.Source {
			return rawLinks[i].key.Source < rawLinks[j].key.Source
		}
		return rawLinks[i].key.Target < rawLinks[j].key.Target
	})

	links := lo.Map(rawLinks, func(r rawLink, _ int) SankeyLink {
		return SankeyLink{
			Source:   r.key.Source,
			Target:   r.key.Target,
			Value:    r.value,
			TxnCount: r.count,
		}
	})

	return nodes, links
}

// normalizeSankeyCurrency converts each posting's Amount from its native Commodity
// to the target currency using historical FX rates.
func normalizeSankeyCurrency(db *gorm.DB, postings []posting.Posting, targetCurrency string) []posting.Posting {
	for i, p := range postings {
		if p.Commodity == "" || p.Commodity == targetCurrency {
			continue
		}

		rate, ok := service.GetRate(db, p.Commodity, targetCurrency, p.Date)
		if !ok {
			// Fallback: If no FX rate exists on or before the transaction date,
			// try to fetch the most recent (latest) known rate instead of skipping.
			rate, ok = service.GetRate(db, p.Commodity, targetCurrency, utils.EndOfToday())
		}

		if ok {
			postings[i].Amount = p.Amount.Mul(rate)
		}
	}
	return postings
}

// GetSankeyHandler handles GET /api/sankey.
//
// Query parameters (all optional):
//
//	from     – start date in YYYY-MM-DD format
//	to       – end date in YYYY-MM-DD format
//	period   – "month" | "quarter" | "year" (default: "month"); used to derive
//	           from/to when they are omitted
//	currency – reporting currency label echoed in the meta (default: config default)
func GetSankeyHandler(db *gorm.DB, c *gin.Context) {
	var q sankeyQueryParams
	if err := c.ShouldBindQuery(&q); err != nil {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest, err.Error())
		return
	}

	// Validate period.
	period := q.Period
	if period == "" {
		period = "month"
	}
	if period != "month" && period != "quarter" && period != "year" {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			"invalid 'period': expected month, quarter, or year")
		return
	}

	// Parse explicit date bounds.
	var from, to time.Time
	if q.From != "" {
		t, err := time.Parse(sankeyDateLayout, q.From)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'from' date: expected YYYY-MM-DD format")
			return
		}
		from = t
	}
	if q.To != "" {
		t, err := time.Parse(sankeyDateLayout, q.To)
		if err != nil {
			RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
				"invalid 'to' date: expected YYYY-MM-DD format")
			return
		}
		to = t
	}

	// Fall back to period-derived bounds for any unset endpoint.
	if from.IsZero() || to.IsZero() {
		pFrom, pTo := sankeyPeriodBounds(period)
		if from.IsZero() {
			from = pFrom
		}
		if to.IsZero() {
			to = pTo
		}
	}

	// Validate ordering.
	if from.After(to) {
		RespondError(c, http.StatusBadRequest, ErrCodeInvalidRequest,
			"'from' must not be after 'to'")
		return
	}

	// Fetch postings within the date range.
	postings := query.Init(db).
		Where("date >= ?", from).
		Where("date <= ?", to).
		All()

	currency := q.Currency
	if currency == "" {
		currency = config.DefaultCurrency()
	}

	postings = normalizeSankeyCurrency(db, postings, currency)

	nodes, links := computeSankeyGraph(postings)

	// Derive totals from links for the meta block.
	totalInflow := decimal.Zero
	totalOutflow := decimal.Zero
	for _, l := range links {
		srcKind := classifySankeyAccount(l.Source)
		dstKind := classifySankeyAccount(l.Target)
		if srcKind == SankeyKindIncome {
			totalInflow = totalInflow.Add(l.Value)
		}
		if dstKind == SankeyKindExpense {
			totalOutflow = totalOutflow.Add(l.Value)
		}
	}

	c.JSON(http.StatusOK, SankeyResponse{
		Nodes: nodes,
		Links: links,
		Meta: SankeyMeta{
			From:         from.Format(sankeyDateLayout),
			To:           to.Format(sankeyDateLayout),
			Period:       period,
			Currency:     currency,
			TotalInflow:  totalInflow,
			TotalOutflow: totalOutflow,
		},
	})
}
