package model

import (
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
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
	current atomic.Int64
	max     atomic.Int64
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

func (p *trackingPriceProvider) GetPrices(code string, commodityName string) ([]*price.Price, error) {
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

func TestSyncCommodities_UsesBoundedConcurrentFetching(t *testing.T) {
	db := openSyncCommoditiesTestDB(t)
	provider := &trackingPriceProvider{}

	commodities := make([]config.Commodity, 0, 12)
	for i := 0; i < 12; i++ {
		commodities = append(commodities, config.Commodity{
			Name: fmt.Sprintf("Commodity-%d", i),
			Type: config.Stock,
			Price: config.Price{
				Provider: "test-provider",
				Code:     fmt.Sprintf("C%d", i),
			},
		})
	}

	result, err := syncCommodities(db, commodities, func(string) price.PriceProvider {
		return provider
	}, 5)
	require.NoError(t, err)
	assert.Empty(t, result.Failures)

	assert.Greater(t, provider.max.Load(), int64(1), "expected concurrent fetches")
	assert.LessOrEqual(t, provider.max.Load(), int64(5), "must not exceed worker limit")

	var count int64
	require.NoError(t, db.Model(&price.Price{}).Count(&count).Error)
	assert.Equal(t, int64(len(commodities)), count)
}
