package server

import (
	"sort"
	"strings"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/transaction"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ExpenseTrend holds month-over-month spending data for a single expense category.
type ExpenseTrend struct {
	Category      string           `json:"category"`
	CurrentMonth  decimal.Decimal  `json:"current_month"`
	PreviousMonth decimal.Decimal  `json:"previous_month"`
	Variance      decimal.Decimal  `json:"variance"`
	VariancePct   *decimal.Decimal `json:"variance_pct"`
}

type Node struct {
	ID   uint   `json:"id"`
	Name string `json:"name"`
}

type Link struct {
	Source uint            `json:"source"`
	Target uint            `json:"target"`
	Value  decimal.Decimal `json:"value"`
}

type Pair struct {
	Source uint `json:"source"`
	Target uint `json:"target"`
}

type Graph struct {
	Nodes []Node `json:"nodes"`
	Links []Link `json:"links"`
}

// expenseCategory returns the second segment of an account name
// (e.g. "Expenses:Groceries:Supermarket" → "Groceries").
func expenseCategory(account string) string {
	parts := strings.SplitN(account, ":", 3)
	if len(parts) >= 2 {
		return parts[1]
	}
	return account
}

// ComputeExpenseTrends calculates month-over-month spending trends for each
// expense category.  "Current month" is the rolling 30-day window ending
// today; "previous month" is the 30-day window immediately before that.
func ComputeExpenseTrends(db *gorm.DB) []ExpenseTrend {
	now := utils.Now()
	currentEnd := utils.EndOfDay(now)
	currentStart := utils.ToDate(now.AddDate(0, 0, -30))
	previousStart := utils.ToDate(now.AddDate(0, 0, -60))

	currentPostings := query.Init(db).
		Where("date >= ? and date <= ?", currentStart, currentEnd).
		Like("Expenses:%").
		NotAccountPrefix("Expenses:Tax").
		All()

	previousPostings := query.Init(db).
		Where("date >= ? and date < ?", previousStart, currentStart).
		Like("Expenses:%").
		NotAccountPrefix("Expenses:Tax").
		All()

	// Aggregate amounts per category.
	currentByCategory := make(map[string]decimal.Decimal)
	for _, p := range currentPostings {
		cat := expenseCategory(p.Account)
		currentByCategory[cat] = currentByCategory[cat].Add(p.Amount)
	}

	previousByCategory := make(map[string]decimal.Decimal)
	for _, p := range previousPostings {
		cat := expenseCategory(p.Account)
		previousByCategory[cat] = previousByCategory[cat].Add(p.Amount)
	}

	// Union of categories from both windows.
	seen := make(map[string]bool)
	for cat := range currentByCategory {
		seen[cat] = true
	}
	for cat := range previousByCategory {
		seen[cat] = true
	}

	categories := lo.Keys(seen)
	sort.Strings(categories)

	trends := make([]ExpenseTrend, 0, len(categories))
	for _, cat := range categories {
		current := currentByCategory[cat]
		previous := previousByCategory[cat]
		variance := current.Sub(previous)

		var variancePct *decimal.Decimal
		if !previous.IsZero() {
			pct := variance.Div(previous).Mul(decimal.NewFromInt(100)).Round(2)
			variancePct = &pct
		}

		trends = append(trends, ExpenseTrend{
			Category:      cat,
			CurrentMonth:  current,
			PreviousMonth: previous,
			Variance:      variance,
			VariancePct:   variancePct,
		})
	}

	return trends
}

func GetCurrentExpense(db *gorm.DB) map[string][]posting.Posting {
	expenses := query.Init(db).LastNMonths(3).Like("Expenses:%").NotAccountPrefix("Expenses:Tax").All()
	return utils.GroupByMonth(expenses)
}

func GetExpense(db *gorm.DB) gin.H {
	postings := query.Init(db).All()

	expenses := []posting.Posting{}
	incomes := []posting.Posting{}
	investments := []posting.Posting{}
	taxes := []posting.Posting{}
	liabilities := []posting.Posting{}

	for _, p := range postings {
		if utils.IsSameOrParent(p.Account, "Expenses:Tax") {
			taxes = append(taxes, p)
		} else if utils.IsSameOrParent(p.Account, "Expenses") {
			expenses = append(expenses, p)
		} else if utils.IsSameOrParent(p.Account, "Income") {
			incomes = append(incomes, p)
		} else if utils.IsSameOrParent(p.Account, "Assets") && !utils.IsSameOrParent(p.Account, "Assets:Checking") {
			investments = append(investments, p)
		} else if utils.IsSameOrParent(p.Account, "Liabilities") {
			liabilities = append(liabilities, p)
		}
	}

	graph := make(map[string]Graph)
	for fy, ps := range utils.GroupByFY(postings) {
		graph[fy] = sortGraph(computeHierarchyGraph(ps))
	}

	return gin.H{
		"expenses": expenses,
		"trends":   ComputeExpenseTrends(db),
		"month_wise": gin.H{
			"expenses":    utils.GroupByMonth(expenses),
			"incomes":     utils.GroupByMonth(incomes),
			"investments": utils.GroupByMonth(investments),
			"taxes":       utils.GroupByMonth(taxes),
			"liabilities": utils.GroupByMonth(liabilities)},
		"year_wise": gin.H{
			"expenses":    utils.GroupByFY(expenses),
			"incomes":     utils.GroupByFY(incomes),
			"investments": utils.GroupByFY(investments),
			"taxes":       utils.GroupByFY(taxes),
			"liabilities": utils.GroupByFY(liabilities)},
		"graph": graph}
}

func sortGraph(graph Graph) Graph {
	nodes := graph.Nodes
	sort.Slice(nodes, func(i, j int) bool {
		return graph.Nodes[i].Name < graph.Nodes[j].Name
	})

	links := graph.Links
	sort.Slice(links, func(i, j int) bool {
		return graph.Links[i].Source < graph.Links[j].Source || (graph.Links[i].Source == graph.Links[j].Source && graph.Links[i].Target < graph.Links[j].Target)
	})
	return Graph{
		Nodes: nodes,
		Links: links,
	}

}

func computeHierarchyGraph(postings []posting.Posting) Graph {
	nodes := make(map[string]Node)
	links := make(map[Pair]decimal.Decimal)

	var nodeID uint = 0

	transactions := transaction.Build(postings)

	for _, p := range postings {
		addNode(&nodeID, &nodes, p.Account)
	}

	for _, t := range transactions {
		from := lo.Filter(t.Postings, func(p posting.Posting, _ int) bool { return p.Amount.LessThan(decimal.Zero) })
		to := lo.Filter(t.Postings, func(p posting.Posting, _ int) bool { return p.Amount.GreaterThan(decimal.Zero) })

		for _, f := range from {
			for f.Amount.Abs().GreaterThan(decimal.NewFromFloat(0.1)) && len(to) > 0 {
				top := to[0]
				if top.Amount.GreaterThan(f.Amount.Neg()) {
					addLink(f.Account, top.Account, f.Amount.Neg(), &nodes, &links)
					top.Amount = top.Amount.Sub(f.Amount)
					f.Amount = decimal.Zero
				} else {
					addLink(f.Account, top.Account, top.Amount, &nodes, &links)
					f.Amount = f.Amount.Add(top.Amount)
					to = to[1:]
				}
			}
		}
	}

	return Graph{Nodes: lo.Values(nodes), Links: lo.Map(lo.Keys(links), func(k Pair, _ int) Link {
		return Link{Source: k.Source, Target: k.Target, Value: links[k]}
	})}

}

func addNode(nodeID *uint, nodes *map[string]Node, account string) {
	if account == "" {
		return
	}

	_, ok := (*nodes)[account]
	if !ok {
		if strings.HasPrefix(account, "Income:") || strings.HasPrefix(account, "Expenses:") {
			parts := strings.Split(account, ":")
			addNode(nodeID, nodes, strings.Join(parts[:len(parts)-1], ":"))

		}

		(*nodeID)++
		(*nodes)[account] = Node{ID: *nodeID, Name: account}
	}
}

func addLink(source string, target string, amount decimal.Decimal, nodes *map[string]Node, links *map[Pair]decimal.Decimal) {

	sparts := strings.Split(source, ":")
	if sparts[0] == "Income" {
		for len(sparts) > 1 {
			s := strings.Join(sparts, ":")
			t := strings.Join(sparts[:len(sparts)-1], ":")
			(*links)[Pair{Source: (*nodes)[s].ID, Target: (*nodes)[t].ID}] = (*links)[Pair{Source: (*nodes)[s].ID, Target: (*nodes)[t].ID}].Add(amount)
			sparts = sparts[:len(sparts)-1]
		}
		source = strings.Join(sparts, ":")

	}

	tparts := strings.Split(target, ":")
	if tparts[0] == "Expenses" {
		for len(tparts) > 1 {
			t := strings.Join(tparts, ":")
			s := strings.Join(tparts[:len(tparts)-1], ":")
			(*links)[Pair{Source: (*nodes)[s].ID, Target: (*nodes)[t].ID}] = (*links)[Pair{Source: (*nodes)[s].ID, Target: (*nodes)[t].ID}].Add(amount)
			tparts = tparts[:len(tparts)-1]
		}
		target = strings.Join(tparts, ":")
	}

	(*links)[Pair{Source: (*nodes)[source].ID, Target: (*nodes)[target].ID}] = (*links)[Pair{Source: (*nodes)[source].ID, Target: (*nodes)[target].ID}].Add(amount)
}
