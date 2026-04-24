package stock

import (
	"testing"
	"time"

	"github.com/nextxm/paisa/internal/config"
	"github.com/nextxm/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAppendYahooPricesSkipsNilClose(t *testing.T) {
	close := 116.6
	result := Result{
		Timestamp: []int64{
			time.Date(2026, time.April, 20, 7, 0, 0, 0, time.UTC).Unix(),
			time.Date(2026, time.April, 21, 7, 0, 0, 0, time.UTC).Unix(),
		},
		Indicators: Indicators{
			Quote: []Quote{{Close: []*float64{&close, nil}}},
		},
	}

	prices := appendYahooPrices(nil, result, func(date time.Time, value float64) price.Price {
		return price.Price{
			Date:           date,
			CommodityType:  config.Stock,
			CommodityID:    "VOD.L",
			CommodityName:  "Vodafone",
			QuoteCommodity: "GBP",
			Value:          decimal.NewFromFloat(value),
		}
	})

	require.Len(t, prices, 1)
	assert.Equal(t, "2026-04-20", prices[0].Date.UTC().Format("2006-01-02"))
	assert.True(t, prices[0].Value.Equal(decimal.NewFromFloat(116.6)))
}
