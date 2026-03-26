package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// buildSankeyRouter constructs a minimal Gin engine wired to GetSankeyHandler.
func buildSankeyRouter(t *testing.T, db *gorm.DB) *gin.Engine {
	t.Helper()
	r := gin.New()
	r.GET("/api/sankey", func(c *gin.Context) {
		GetSankeyHandler(db, c)
	})
	return r
}

// seedPostings inserts a slice of posting rows into the test DB.
func seedPostings(t *testing.T, db *gorm.DB, postings []posting.Posting) {
	t.Helper()
	for i := range postings {
		require.NoError(t, db.Create(&postings[i]).Error)
	}
}

// parseTestDay parses a YYYY-MM-DD date for test data seeding.
func parseTestDay(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(err)
	}
	return t
}

// ---------------------------------------------------------------------------
// classifySankeyAccount unit tests
// ---------------------------------------------------------------------------

func TestClassifySankeyAccount(t *testing.T) {
	cases := []struct {
		account string
		want    SankeyNodeKind
	}{
		{"Income:Salary", SankeyKindIncome},
		{"Income", SankeyKindIncome},
		{"Expenses:Food", SankeyKindExpense},
		{"Expenses:Tax", SankeyKindExpense},
		{"Expenses", SankeyKindExpense},
		{"Assets:Checking", SankeyKindAsset},
		{"Assets:Savings", SankeyKindAsset},
		{"Assets", SankeyKindAsset},
		{"Liabilities:CreditCard", SankeyKindLiability},
		{"Liabilities", SankeyKindLiability},
		{"Equity:OpeningBalance", SankeyKindEquity},
		{"Equity", SankeyKindEquity},
		{"Unknown", SankeyKindOther},
		{"", SankeyKindOther},
	}
	for _, tc := range cases {
		t.Run(tc.account, func(t *testing.T) {
			assert.Equal(t, tc.want, classifySankeyAccount(tc.account))
		})
	}
}

// ---------------------------------------------------------------------------
// sankeyPeriodBounds unit tests
// ---------------------------------------------------------------------------

func TestSankeyPeriodBounds_Month(t *testing.T) {
	utils.SetNow("2024-03-15")
	defer utils.UnsetNow()

	from, to := sankeyPeriodBounds("month")
	assert.Equal(t, "2024-03-01", from.Format("2006-01-02"))
	assert.Equal(t, "2024-03-31", to.Format("2006-01-02"))
}

func TestSankeyPeriodBounds_Quarter_Q1(t *testing.T) {
	utils.SetNow("2024-02-10")
	defer utils.UnsetNow()

	from, to := sankeyPeriodBounds("quarter")
	assert.Equal(t, "2024-01-01", from.Format("2006-01-02"))
	assert.Equal(t, "2024-03-31", to.Format("2006-01-02"))
}

func TestSankeyPeriodBounds_Quarter_Q3(t *testing.T) {
	utils.SetNow("2024-08-20")
	defer utils.UnsetNow()

	from, to := sankeyPeriodBounds("quarter")
	assert.Equal(t, "2024-07-01", from.Format("2006-01-02"))
	assert.Equal(t, "2024-09-30", to.Format("2006-01-02"))
}

func TestSankeyPeriodBounds_Year(t *testing.T) {
	utils.SetNow("2024-09-01")
	defer utils.UnsetNow()

	from, to := sankeyPeriodBounds("year")
	assert.Equal(t, "2024-01-01", from.Format("2006-01-02"))
	assert.Equal(t, "2024-12-31", to.Format("2006-01-02"))
}

// ---------------------------------------------------------------------------
// computeSankeyGraph unit tests
// ---------------------------------------------------------------------------

// TestComputeSankeyGraph_Empty verifies that no postings yields empty slices.
func TestComputeSankeyGraph_Empty(t *testing.T) {
	nodes, links := computeSankeyGraph(nil)
	assert.Empty(t, nodes)
	assert.Empty(t, links)

	nodes2, links2 := computeSankeyGraph([]posting.Posting{})
	assert.Empty(t, nodes2)
	assert.Empty(t, links2)
}

// TestComputeSankeyGraph_BasicFlow verifies a simple income→asset→expense flow.
//
// Transaction 1: Income:Salary (-1000) → Assets:Checking (+1000)
// Transaction 2: Assets:Checking (-200) → Expenses:Food (+200)
func TestComputeSankeyGraph_BasicFlow(t *testing.T) {
	txnID1 := "txn-001"
	txnID2 := "txn-002"
	postings := []posting.Posting{
		{TransactionID: txnID1, Date: parseTestDay("2024-03-01"), Account: "Income:Salary",
			Amount: decimal.NewFromFloat(-1000), Commodity: "INR"},
		{TransactionID: txnID1, Date: parseTestDay("2024-03-01"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(1000), Commodity: "INR"},
		{TransactionID: txnID2, Date: parseTestDay("2024-03-05"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-200), Commodity: "INR"},
		{TransactionID: txnID2, Date: parseTestDay("2024-03-05"), Account: "Expenses:Food",
			Amount: decimal.NewFromFloat(200), Commodity: "INR"},
	}

	nodes, links := computeSankeyGraph(postings)

	// Expect all 3 accounts as nodes.
	assert.Len(t, nodes, 3)
	nodeNames := make(map[string]SankeyNodeKind)
	for _, n := range nodes {
		nodeNames[n.Name] = n.Kind
	}
	assert.Equal(t, SankeyKindIncome, nodeNames["Income:Salary"])
	assert.Equal(t, SankeyKindAsset, nodeNames["Assets:Checking"])
	assert.Equal(t, SankeyKindExpense, nodeNames["Expenses:Food"])

	// Expect two directed links.
	assert.Len(t, links, 2)
	linkMap := make(map[string]SankeyLink)
	for _, l := range links {
		linkMap[l.Source+"->"+l.Target] = l
	}

	l1, ok := linkMap["Income:Salary->Assets:Checking"]
	require.True(t, ok, "expected Income:Salary→Assets:Checking link")
	assert.True(t, l1.Value.Equal(decimal.NewFromFloat(1000)))
	assert.Equal(t, 1, l1.TxnCount)

	l2, ok := linkMap["Assets:Checking->Expenses:Food"]
	require.True(t, ok, "expected Assets:Checking→Expenses:Food link")
	assert.True(t, l2.Value.Equal(decimal.NewFromFloat(200)))
	assert.Equal(t, 1, l2.TxnCount)
}

// TestComputeSankeyGraph_TxnCountAggregation verifies that two transactions
// through the same account pair increment TxnCount.
func TestComputeSankeyGraph_TxnCountAggregation(t *testing.T) {
	postings := []posting.Posting{
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-50), Commodity: "INR"},
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Expenses:Food",
			Amount: decimal.NewFromFloat(50), Commodity: "INR"},
		{TransactionID: "t2", Date: parseTestDay("2024-03-10"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-30), Commodity: "INR"},
		{TransactionID: "t2", Date: parseTestDay("2024-03-10"), Account: "Expenses:Food",
			Amount: decimal.NewFromFloat(30), Commodity: "INR"},
	}

	_, links := computeSankeyGraph(postings)
	require.Len(t, links, 1)
	assert.Equal(t, 2, links[0].TxnCount)
	assert.True(t, links[0].Value.Equal(decimal.NewFromFloat(80)))
}

// TestComputeSankeyGraph_SelfLinkDropped verifies that self-links are omitted.
func TestComputeSankeyGraph_SelfLinkDropped(t *testing.T) {
	postings := []posting.Posting{
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Assets:A",
			Amount: decimal.NewFromFloat(-100), Commodity: "INR"},
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Assets:A",
			Amount: decimal.NewFromFloat(100), Commodity: "INR"},
	}

	_, links := computeSankeyGraph(postings)
	assert.Empty(t, links, "self-links must be dropped")
}

// TestComputeSankeyGraph_DeterministicOrder verifies that nodes and links are
// returned in a stable, sorted order regardless of insertion sequence.
func TestComputeSankeyGraph_DeterministicOrder(t *testing.T) {
	postings := []posting.Posting{
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Income:Salary",
			Amount: decimal.NewFromFloat(-1000), Commodity: "INR"},
		{TransactionID: "t1", Date: parseTestDay("2024-03-01"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(1000), Commodity: "INR"},
		{TransactionID: "t2", Date: parseTestDay("2024-03-05"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-200), Commodity: "INR"},
		{TransactionID: "t2", Date: parseTestDay("2024-03-05"), Account: "Expenses:Food",
			Amount: decimal.NewFromFloat(200), Commodity: "INR"},
	}

	nodes1, links1 := computeSankeyGraph(postings)
	nodes2, links2 := computeSankeyGraph(postings)

	require.Equal(t, len(nodes1), len(nodes2))
	for i := range nodes1 {
		assert.Equal(t, nodes1[i].ID, nodes2[i].ID)
	}
	require.Equal(t, len(links1), len(links2))
	for i := range links1 {
		assert.Equal(t, links1[i].Source, links2[i].Source)
		assert.Equal(t, links1[i].Target, links2[i].Target)
	}
}

// ---------------------------------------------------------------------------
// HTTP handler integration tests
// ---------------------------------------------------------------------------

// TestGetSankeyHandler_EmptyRange verifies that an empty DB returns a valid
// payload with empty nodes/links but a populated meta.
func TestGetSankeyHandler_EmptyRange(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?from=2024-03-01&to=2024-03-31", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp SankeyResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Empty(t, resp.Nodes)
	assert.Empty(t, resp.Links)
	assert.Equal(t, "2024-03-01", resp.Meta.From)
	assert.Equal(t, "2024-03-31", resp.Meta.To)
	assert.Equal(t, "month", resp.Meta.Period)
}

// TestGetSankeyHandler_WithData verifies that seeded postings produce the
// expected nodes, links, and meta totals.
func TestGetSankeyHandler_WithData(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	seedPostings(t, db, []posting.Posting{
		{TransactionID: "tx1", Date: parseTestDay("2024-03-01"), Account: "Income:Salary",
			Amount: decimal.NewFromFloat(-5000), Commodity: "INR"},
		{TransactionID: "tx1", Date: parseTestDay("2024-03-01"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(5000), Commodity: "INR"},
		{TransactionID: "tx2", Date: parseTestDay("2024-03-10"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-500), Commodity: "INR"},
		{TransactionID: "tx2", Date: parseTestDay("2024-03-10"), Account: "Expenses:Groceries",
			Amount: decimal.NewFromFloat(500), Commodity: "INR"},
	})
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?from=2024-03-01&to=2024-03-31", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp SankeyResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	assert.Len(t, resp.Nodes, 3)
	assert.Len(t, resp.Links, 2)
	assert.True(t, resp.Meta.TotalInflow.Equal(decimal.NewFromFloat(5000)),
		"totalInflow must equal the income flow")
	assert.True(t, resp.Meta.TotalOutflow.Equal(decimal.NewFromFloat(500)),
		"totalOutflow must equal the expense flow")
}

// TestGetSankeyHandler_MultiCurrency verifies that the Sankey handler converts
// native commodities to the target currency when specified, and zeros-out
// unconvertible flows.
func TestGetSankeyHandler_MultiCurrency(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)

	// Seed exchange rate: 1 CAD = 60 INR (so 1 INR = 1/60 CAD)
	utils.SetNow("2024-03-15")
	defer utils.UnsetNow()

	rate := decimal.NewFromFloat(60)
	require.NoError(t, db.Exec("INSERT INTO prices (date, commodity_name, quote_commodity, value, commodity_type) VALUES (?, ?, ?, ?, ?)",
		parseTestDay("2024-03-01"), "CAD", "INR", rate, "manual").Error)

	seedPostings(t, db, []posting.Posting{
		// A transaction in INR: 60,000 INR
		{TransactionID: "tx1", Date: parseTestDay("2024-03-05"), Account: "Income:Salary",
			Amount: decimal.NewFromFloat(-60000), Commodity: "INR"},
		{TransactionID: "tx1", Date: parseTestDay("2024-03-05"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(60000), Commodity: "INR"},
		// A transaction in an unknown currency
		{TransactionID: "tx2", Date: parseTestDay("2024-03-10"), Account: "Assets:Checking",
			Amount: decimal.NewFromFloat(-500), Commodity: "UNKNOWN"},
		{TransactionID: "tx2", Date: parseTestDay("2024-03-10"), Account: "Expenses:Magic",
			Amount: decimal.NewFromFloat(500), Commodity: "UNKNOWN"},
	})
	r := buildSankeyRouter(t, db)

	// Fetch Sankey in CAD
	req := httptest.NewRequest(http.MethodGet, "/api/sankey?from=2024-03-01&to=2024-03-31&currency=CAD", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp SankeyResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))

	// The unknown currency is zeroed out and dropped.
	// 60,000 INR -> CAD = 60,000 / 60 = 1,000 CAD.
	assert.Len(t, resp.Links, 1, "Only the convertible flow should remain")
	assert.True(t, resp.Links[0].Value.Equal(decimal.NewFromFloat(1000)), "60k INR should become 1k CAD")
	assert.True(t, resp.Meta.TotalInflow.Equal(decimal.NewFromFloat(1000)))
}

// TestGetSankeyHandler_PeriodDefault verifies that omitting all params returns
// a response whose meta reflects the current month period.
func TestGetSankeyHandler_PeriodDefault(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-06-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp SankeyResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "month", resp.Meta.Period)
	assert.Equal(t, "2024-06-01", resp.Meta.From)
	assert.Equal(t, "2024-06-30", resp.Meta.To)
}

// TestGetSankeyHandler_PeriodQuarter verifies that period=quarter sets bounds
// to the current calendar quarter.
func TestGetSankeyHandler_PeriodQuarter(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-05-15")
	defer utils.UnsetNow()

	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?period=quarter", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp SankeyResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	assert.Equal(t, "quarter", resp.Meta.Period)
	assert.Equal(t, "2024-04-01", resp.Meta.From)
	assert.Equal(t, "2024-06-30", resp.Meta.To)
}

// TestGetSankeyHandler_InvalidPeriod returns 400 for an unknown period value.
func TestGetSankeyHandler_InvalidPeriod(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?period=week", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestGetSankeyHandler_InvalidFromDate returns 400 for a malformed 'from' date.
func TestGetSankeyHandler_InvalidFromDate(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?from=not-a-date", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestGetSankeyHandler_InvalidToDate returns 400 for a malformed 'to' date.
func TestGetSankeyHandler_InvalidToDate(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?to=31/12/2024", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
}

// TestGetSankeyHandler_FromAfterTo returns 400 when from is after to.
func TestGetSankeyHandler_FromAfterTo(t *testing.T) {
	loadTestConfig(t, false)
	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey?from=2024-12-31&to=2024-01-01", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
	detail := decodeErrorEnvelope(t, rec)
	assert.Equal(t, ErrCodeInvalidRequest, detail.Code)
	assert.Contains(t, detail.Message, "from")
}

// TestGetSankeyHandler_ResponseShape verifies the canonical JSON keys are present.
func TestGetSankeyHandler_ResponseShape(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2024-03-15")
	defer utils.UnsetNow()

	db := openTestDB(t)
	r := buildSankeyRouter(t, db)

	req := httptest.NewRequest(http.MethodGet, "/api/sankey", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]json.RawMessage
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&body))
	_, hasNodes := body["nodes"]
	_, hasLinks := body["links"]
	_, hasMeta := body["meta"]
	assert.True(t, hasNodes, "response must contain 'nodes'")
	assert.True(t, hasLinks, "response must contain 'links'")
	assert.True(t, hasMeta, "response must contain 'meta'")
}
