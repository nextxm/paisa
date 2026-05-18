package portfolio

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Portfolio{}))
	return db
}

func TestUpsertAllAndGetPortfolios(t *testing.T) {
	db := openTestDB(t)
	rows := []*Portfolio{
		{SecurityID: "INF001", SecurityName: "Fund A", Percentage: decimal.NewFromFloat(60)},
		{SecurityID: "INF002", SecurityName: "Fund B", Percentage: decimal.NewFromFloat(40)},
	}

	require.NoError(t, UpsertAll(db, config.MutualFund, "MF123", rows))

	portfolios := GetPortfolios(db, "MF123")
	require.Len(t, portfolios, 2)
	assert.Equal(t, "Fund A", portfolios[0].SecurityName)
	assert.True(t, portfolios[0].Percentage.Equal(decimal.NewFromFloat(60)))
	assert.Equal(t, "Fund B", portfolios[1].SecurityName)
}

func TestGetAllParentCommodityIDs(t *testing.T) {
	db := openTestDB(t)
	require.NoError(t, UpsertAll(db, config.MutualFund, "MF123", []*Portfolio{{SecurityID: "INF001", SecurityName: "Fund A", Percentage: decimal.NewFromFloat(100)}}))
	require.NoError(t, UpsertAll(db, config.MutualFund, "MF456", []*Portfolio{{SecurityID: "INF002", SecurityName: "Fund B", Percentage: decimal.NewFromFloat(100)}}))

	ids := GetAllParentCommodityIDs(db)
	assert.Equal(t, []string{"MF123", "MF456"}, ids)
}
