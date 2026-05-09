package model

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openSyncCommoditiesTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, migration.RunMigrations(db))
	return db
}

type trackingPriceProvider struct {
	current      atomic.Int64
	max          atomic.Int64
	singleCalls  atomic.Int64
	batchCalls   atomic.Int64
	batchSizesMu sync.Mutex
	batchSizes   []int
}

func (p *trackingPriceProvider) Name() string {
	return "tracking-provider"
}

func (p *trackingPriceProvider) Code() string {
	return "tracking-provider"
}

func (p *trackingPriceProvider) Label() string {
	return "Tracking Provider"
}

func (p *trackingPriceProvider) Description() string {
	return "tracking provider for sync concurrency tests"
}

func (p *trackingPriceProvider) AutoCompleteFields() []price.AutoCompleteField {
	return nil
}

func (p *trackingPriceProvider) AutoComplete(*gorm.DB, string, map[string]string) []price.AutoCompleteItem {
	return []price.AutoCompleteItem{}
}

func (p *trackingPriceProvider) ClearCache(*gorm.DB) {
}

func (p *trackingPriceProvider) GetPrices(code string, commodityName string, since time.Time) ([]*price.Price, error) {
	p.singleCalls.Add(1)
	current := p.current.Add(1)
	for {
		maxSeen := p.max.Load()
		if current <= maxSeen || p.max.CompareAndSwap(maxSeen, current) {
			break
		}
	}

	time.Sleep(100 * time.Millisecond)
	p.current.Add(-1)

	return []*price.Price{
		{
			CommodityType:  config.Stock,
			CommodityID:    code,
			CommodityName:  commodityName,
			Date:           time.Now().UTC(),
			QuoteCommodity: "INR",
			Value:          decimal.NewFromInt(1),
		},
	}, nil
}

func (p *trackingPriceProvider) GetPricesBatch(codes []string, commodityNames []string) (map[string][]*price.Price, error) {
	p.batchCalls.Add(1)
	p.batchSizesMu.Lock()
	p.batchSizes = append(p.batchSizes, len(codes))
	p.batchSizesMu.Unlock()

	current := p.current.Add(1)
	for {
		maxSeen := p.max.Load()
		if current <= maxSeen || p.max.CompareAndSwap(maxSeen, current) {
			break
		}
	}

	time.Sleep(100 * time.Millisecond)
	p.current.Add(-1)

	pricesByCode := make(map[string][]*price.Price, len(codes))
	for i, code := range codes {
		pricesByCode[code] = []*price.Price{
			{
				CommodityType:  config.Stock,
				CommodityID:    code,
				CommodityName:  commodityNames[i],
				Date:           time.Now().UTC(),
				QuoteCommodity: "INR",
				Value:          decimal.NewFromInt(1),
			},
		}
	}

	return pricesByCode, nil
}

func TestSyncCommodities_UsesBoundedConcurrentFetching(t *testing.T) {
	db := openSyncCommoditiesTestDB(t)
	provider := &trackingPriceProvider{}

	commodities := make([]config.Commodity, 0, 12)
	for i := 0; i < 12; i++ {
		// Pair commodities by provider so syncCommodities has 6 provider-level
		// batch jobs to schedule across the 5-worker pool.
		commodities = append(commodities, config.Commodity{
			Name: fmt.Sprintf("Commodity-%d", i),
			Type: config.Stock,
			Price: config.Price{
				Provider: fmt.Sprintf("test-provider-%d", i/2),
				Code:     fmt.Sprintf("C%d", i),
			},
		})
	}

	result, err := syncCommodities(db, commodities, func(string) price.PriceProvider {
		return provider
	}, 5, false, nil)
	require.NoError(t, err)
	assert.Empty(t, result.Failures)

	assert.Greater(t, provider.max.Load(), int64(1), "expected concurrent fetches")
	assert.LessOrEqual(t, provider.max.Load(), int64(5), "must not exceed worker limit")
	assert.Zero(t, provider.singleCalls.Load(), "batched provider groups should not fall back to single fetches")
	assert.Equal(t, int64(6), provider.batchCalls.Load(), "expected one batch fetch per provider group")
	assert.ElementsMatch(t, []int{2, 2, 2, 2, 2, 2}, provider.batchSizes)

	var count int64
	require.NoError(t, db.Model(&price.Price{}).Count(&count).Error)
	assert.Equal(t, int64(len(commodities)), count)
}

// sincePriceProvider is a stub PriceProvider that records the since argument
// received during GetPrices so tests can assert its value.
type sincePriceProvider struct {
	receivedSince time.Time
	prices        []*price.Price
}

func (p *sincePriceProvider) Code() string        { return "since-stub" }
func (p *sincePriceProvider) Label() string       { return "Since Stub" }
func (p *sincePriceProvider) Description() string { return "" }
func (p *sincePriceProvider) AutoCompleteFields() []price.AutoCompleteField {
	return []price.AutoCompleteField{}
}
func (p *sincePriceProvider) AutoComplete(_ *gorm.DB, _ string, _ map[string]string) []price.AutoCompleteItem {
	return []price.AutoCompleteItem{}
}
func (p *sincePriceProvider) ClearCache(_ *gorm.DB) {}
func (p *sincePriceProvider) GetPrices(_ string, _ string, since time.Time) ([]*price.Price, error) {
	p.receivedSince = since
	return p.prices, nil
}
func (p *sincePriceProvider) GetPricesBatch(codes []string, commodityNames []string) (map[string][]*price.Price, error) {
	return price.GetPricesBatchSequentially(p, codes, commodityNames)
}

// TestSyncCommodities_PassesSinceToProvider verifies that syncCommodities reads
// the last_price_sync metadata value and forwards it as the since argument to
// GetPrices.  When no metadata exists the zero time is passed (full sync).
func TestSyncCommodities_PassesSinceToProvider(t *testing.T) {
	db := openSyncCommoditiesTestDB(t)

	stub := &sincePriceProvider{
		prices: []*price.Price{
			{
				Date:           time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				CommodityType:  config.MutualFund,
				CommodityID:    "FUND1",
				CommodityName:  "TestFund",
				Value:          decimal.NewFromFloat(100.0),
				QuoteCommodity: "INR",
			},
		},
	}

	commodities := []config.Commodity{
		{
			Name: "TestFund",
			Type: config.MutualFund,
			Price: config.Price{
				Provider: "since-stub",
				Code:     "FUND1",
			},
		},
	}
	getProvider := func(_ string) price.PriceProvider { return stub }

	// First call: no metadata → since must be zero (full history fetch).
	_, err := syncCommodities(db, commodities, getProvider, 1, false, nil)
	require.NoError(t, err)
	assert.True(t, stub.receivedSince.IsZero(), "since must be zero when no last_price_sync metadata exists")

	// Store a last_price_sync timestamp.
	lastSync := time.Date(2024, 5, 15, 10, 0, 0, 0, time.UTC)
	require.NoError(t, metadata.Set(db, LastPriceSyncKey, lastSync.Format(time.RFC3339)))

	// Second call: since must match the stored timestamp.
	stub.receivedSince = time.Time{} // reset
	_, err = syncCommodities(db, commodities, getProvider, 1, false, nil)
	require.NoError(t, err)
	assert.Equal(t, lastSync.UTC().Truncate(time.Second), stub.receivedSince.UTC().Truncate(time.Second),
		"since must match the last_price_sync metadata value")

	// Force refresh: ignore metadata and fetch full history again.
	stub.receivedSince = lastSync
	_, err = syncCommodities(db, commodities, getProvider, 1, true, nil)
	require.NoError(t, err)
	assert.True(t, stub.receivedSince.IsZero(), "force refresh must ignore last_price_sync metadata")
}

// TestSyncCommodities_IncrementalUpsertPreservesHistory verifies that on an
// incremental sync (where the provider returns only new prices), previously
// stored prices are not deleted – they remain in the DB alongside the new rows.
func TestSyncCommodities_IncrementalUpsertPreservesHistory(t *testing.T) {
	db := openSyncCommoditiesTestDB(t)

	// First full sync: provider returns two historical prices.
	firstSyncPrices := []*price.Price{
		{
			Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(180.0),
			QuoteCommodity: "USD",
		},
		{
			Date:           time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(181.0),
			QuoteCommodity: "USD",
		},
	}
	stub := &sincePriceProvider{prices: firstSyncPrices}
	commodities := []config.Commodity{
		{
			Name:  "Apple",
			Type:  config.Stock,
			Price: config.Price{Provider: "since-stub", Code: "AAPL"},
		},
	}
	getProvider := func(_ string) price.PriceProvider { return stub }

	_, err := syncCommodities(db, commodities, getProvider, 1, false, nil)
	require.NoError(t, err)

	var count int64
	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(2), count, "first full sync should persist 2 rows")

	// Second incremental sync: provider returns only the newest price.
	// The existing 2 rows must not be deleted.
	stub.prices = []*price.Price{
		{
			Date:           time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			CommodityType:  config.Stock,
			CommodityID:    "AAPL",
			CommodityName:  "Apple",
			Value:          decimal.NewFromFloat(182.0),
			QuoteCommodity: "USD",
		},
	}
	_, err = syncCommodities(db, commodities, getProvider, 1, false, nil)
	require.NoError(t, err)

	db.Model(&price.Price{}).Count(&count)
	assert.Equal(t, int64(3), count, "incremental sync must add new prices without removing historical rows")
}

// TestSyncCommodities_ReportsProgress verifies that syncCommodities invokes
// the progressFn callback once per commodity and that the final call reports
// completed == total.
func TestSyncCommodities_ReportsProgress(t *testing.T) {
	db := openSyncCommoditiesTestDB(t)

	stub := &sincePriceProvider{
		prices: []*price.Price{
			{
				Date:           time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC),
				CommodityType:  config.Stock,
				CommodityID:    "X",
				CommodityName:  "CommodityX",
				Value:          decimal.NewFromInt(1),
				QuoteCommodity: "USD",
			},
		},
	}

	commodities := []config.Commodity{
		{Name: "CommodityX", Type: config.Stock, Price: config.Price{Provider: "since-stub", Code: "X"}},
		{Name: "CommodityY", Type: config.Stock, Price: config.Price{Provider: "since-stub", Code: "Y"}},
		{Name: "CommodityZ", Type: config.Stock, Price: config.Price{Provider: "since-stub", Code: "Z"}},
	}
	getProvider := func(_ string) price.PriceProvider { return stub }

	var calls []struct{ completed, total int }
	progressFn := func(completed, total int) {
		calls = append(calls, struct{ completed, total int }{completed, total})
	}

	_, err := syncCommodities(db, commodities, getProvider, 1, false, progressFn)
	require.NoError(t, err)

	assert.Len(t, calls, len(commodities), "progressFn must be called once per commodity")
	last := calls[len(calls)-1]
	assert.Equal(t, len(commodities), last.completed, "final call must have completed == len(commodities)")
	assert.Equal(t, len(commodities), last.total, "total must equal len(commodities) on every call")
	for _, c := range calls {
		assert.Equal(t, len(commodities), c.total, "total must be constant across all progress calls")
		assert.Greater(t, c.completed, 0, "completed must always be positive")
	}
}
