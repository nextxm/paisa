package main

import (
	"flag"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/server"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type endpointResult struct {
	Endpoint         string
	Samples          int
	P50MS            float64
	P95MS            float64
	TotalSQLCount    int64
	TotalSQLTimeMS   float64
	AvgSQLCount      float64
	AvgSQLTimeMS     float64
	AvgLatencyHeader float64
}

func main() {
	var (
		iterations = flag.Int("iterations", 40, "number of measured samples per endpoint")
		warmup     = flag.Int("warmup", 5, "warmup calls per endpoint")
		years      = flag.Int("years", 20, "years of synthetic monthly data to seed")
	)
	flag.Parse()

	if *iterations <= 0 {
		fatalf("iterations must be > 0")
	}
	if *warmup < 0 {
		fatalf("warmup must be >= 0")
	}
	if *years <= 0 {
		fatalf("years must be > 0")
	}

	db, cleanup, err := setupBenchmarkDB(*years)
	if err != nil {
		fatalf("setup failed: %v", err)
	}
	defer cleanup()

	router := server.Build(db, false)
	endpoints := []string{
		"/api/config",
		"/api/dashboard",
		"/api/networth/projection",
	}

	results := make([]endpointResult, 0, len(endpoints))
	for _, endpoint := range endpoints {
		result, err := runEndpoint(router, endpoint, *warmup, *iterations)
		if err != nil {
			fatalf("benchmark failed for %s: %v", endpoint, err)
		}
		results = append(results, result)
	}

	printReport(results, *iterations, *warmup, *years)
}

func setupBenchmarkDB(years int) (*gorm.DB, func(), error) {
	tempDir, err := os.MkdirTemp("", "paisa-perfbaseline-*")
	if err != nil {
		return nil, nil, err
	}

	dbPath := filepath.Join(tempDir, "paisa-perfbaseline.db")
	journalPath := filepath.Join(tempDir, "main.ledger")
	if err := os.WriteFile(journalPath, []byte("; synthetic benchmark dataset\n"), 0600); err != nil {
		return nil, nil, err
	}

	cfg := fmt.Sprintf("journal_path: %s\ndb_path: %s\ntime_zone: UTC\n", journalPath, dbPath)
	if err := config.LoadConfig([]byte(cfg), ""); err != nil {
		return nil, nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	if err := migration.RunMigrations(db); err != nil {
		return nil, nil, err
	}

	postings := makeSyntheticPostings(years)
	if err := db.CreateInBatches(postings, 500).Error; err != nil {
		return nil, nil, err
	}

	cleanup := func() {
		_ = os.RemoveAll(tempDir)
	}
	return db, cleanup, nil
}

func makeSyntheticPostings(years int) []posting.Posting {
	postings := make([]posting.Posting, 0, years*12*6)
	now := time.Now().UTC()
	start := time.Date(now.Year()-years, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := 0; i < years*12; i++ {
		monthDate := start.AddDate(0, i, 0)
		idPrefix := fmt.Sprintf("%d-%02d", monthDate.Year(), monthDate.Month())

		salary := decimal.NewFromInt(8000)
		rent := decimal.NewFromInt(2200)
		invest := decimal.NewFromInt(2500)

		postings = append(postings,
			buildPosting(idPrefix+"-salary", monthDate, "Salary", "Assets:Checking", salary),
			buildPosting(idPrefix+"-salary", monthDate, "Salary", "Income:Salary", salary.Neg()),
			buildPosting(idPrefix+"-rent", monthDate.AddDate(0, 0, 3), "Rent", "Expenses:Rent", rent),
			buildPosting(idPrefix+"-rent", monthDate.AddDate(0, 0, 3), "Rent", "Assets:Checking", rent.Neg()),
			buildPosting(idPrefix+"-invest", monthDate.AddDate(0, 0, 10), "Invest", "Assets:Investments:Index", invest),
			buildPosting(idPrefix+"-invest", monthDate.AddDate(0, 0, 10), "Invest", "Assets:Checking", invest.Neg()),
		)
	}

	return postings
}

func buildPosting(txID string, date time.Time, payee string, account string, amount decimal.Decimal) posting.Posting {
	return posting.Posting{
		TransactionID:        txID,
		Date:                 date,
		Payee:                payee,
		Account:              account,
		Commodity:            "INR",
		Quantity:             amount,
		Amount:               amount,
		OriginalAmount:       amount,
		TransactionBeginLine: 1,
		TransactionEndLine:   2,
		FileName:             "synthetic.ledger",
	}
}

func runEndpoint(router http.Handler, endpoint string, warmup int, iterations int) (endpointResult, error) {
	latencies := make([]float64, 0, iterations)
	var totalSQLCount int64
	var totalSQLTime float64
	var totalLatencyHeader float64

	for i := 0; i < warmup+iterations; i++ {
		req := httptest.NewRequest(http.MethodGet, endpoint, nil)
		rec := httptest.NewRecorder()

		start := time.Now()
		router.ServeHTTP(rec, req)
		elapsedMS := float64(time.Since(start).Nanoseconds()) / float64(time.Millisecond)

		if rec.Code != http.StatusOK {
			return endpointResult{}, fmt.Errorf("unexpected status %d", rec.Code)
		}

		sqlCount, err := strconv.ParseInt(rec.Header().Get("X-Paisa-Perf-SQL-Count"), 10, 64)
		if err != nil {
			return endpointResult{}, fmt.Errorf("invalid X-Paisa-Perf-SQL-Count: %w", err)
		}
		sqlTimeMS, err := strconv.ParseFloat(rec.Header().Get("X-Paisa-Perf-SQL-Time-Ms"), 64)
		if err != nil {
			return endpointResult{}, fmt.Errorf("invalid X-Paisa-Perf-SQL-Time-Ms: %w", err)
		}
		latencyHeaderMS, err := strconv.ParseFloat(rec.Header().Get("X-Paisa-Perf-Latency-Ms"), 64)
		if err != nil {
			return endpointResult{}, fmt.Errorf("invalid X-Paisa-Perf-Latency-Ms: %w", err)
		}

		if i >= warmup {
			latencies = append(latencies, elapsedMS)
			totalSQLCount += sqlCount
			totalSQLTime += sqlTimeMS
			totalLatencyHeader += latencyHeaderMS
		}
	}

	return endpointResult{
		Endpoint:         endpoint,
		Samples:          iterations,
		P50MS:            percentile(latencies, 50),
		P95MS:            percentile(latencies, 95),
		TotalSQLCount:    totalSQLCount,
		TotalSQLTimeMS:   totalSQLTime,
		AvgSQLCount:      float64(totalSQLCount) / float64(iterations),
		AvgSQLTimeMS:     totalSQLTime / float64(iterations),
		AvgLatencyHeader: totalLatencyHeader / float64(iterations),
	}, nil
}

func percentile(values []float64, p float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 1 {
		return sorted[0]
	}
	position := (p / 100) * float64(len(sorted)-1)
	lower := int(math.Floor(position))
	upper := int(math.Ceil(position))
	if lower == upper {
		return sorted[lower]
	}
	fraction := position - float64(lower)
	return sorted[lower] + (sorted[upper]-sorted[lower])*fraction
}

func printReport(results []endpointResult, iterations int, warmup int, years int) {
	fmt.Printf("paisa perf baseline\n")
	fmt.Printf("timestamp: %s\n", time.Now().UTC().Format(time.RFC3339))
	fmt.Printf("go: %s\n", runtime.Version())
	fmt.Printf("os/arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("cpus: %d\n", runtime.NumCPU())
	fmt.Printf("dataset: synthetic %d years (%d warmup + %d measured samples)\n\n", years, warmup, iterations)

	fmt.Println("| endpoint | p50 latency (ms) | p95 latency (ms) | total sql queries | total sql time (ms) | avg sql queries/request | avg sql time/request (ms) |")
	fmt.Println("|---|---:|---:|---:|---:|---:|---:|")
	for _, r := range results {
		fmt.Printf("| %s | %.2f | %.2f | %d | %.2f | %.2f | %.2f |\n",
			r.Endpoint,
			r.P50MS,
			r.P95MS,
			r.TotalSQLCount,
			r.TotalSQLTimeMS,
			r.AvgSQLCount,
			r.AvgSQLTimeMS,
		)
	}
}

func fatalf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
