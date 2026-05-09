package accounting

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/migration"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/ananthakumaran/paisa/internal/utils"
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
	require.NoError(t, migration.RunMigrations(db))
	return db
}

func loadAccountingTestConfig(t *testing.T) {
	t.Helper()
	orig := config.GetConfig()
	t.Cleanup(func() {
		_ = config.LoadConfig([]byte("journal_path: "+orig.JournalPath+"\ndb_path: "+orig.DBPath), "")
	})
	require.NoError(t, config.LoadConfig([]byte("journal_path: main.ledger\ndb_path: paisa.db\n"), ""))
}

func TestRunningBalance_WindowFunctionKeepsDailyCarryForward(t *testing.T) {
	loadAccountingTestConfig(t)
	utils.SetNow("2024-01-04")
	t.Cleanup(utils.UnsetNow)
	service.ClearPriceCache()
	t.Cleanup(service.ClearPriceCache)

	db := openTestDB(t)
	require.NoError(t, db.Create([]posting.Posting{
		{
			TransactionID: "t1",
			Date:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Equity:Gold",
			Commodity:     "GOLD",
			Quantity:      decimal.NewFromInt(10),
			Amount:        decimal.NewFromInt(100),
		},
		{
			TransactionID: "t2",
			Date:          time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			Account:       "Assets:Equity:Gold",
			Commodity:     "GOLD",
			Quantity:      decimal.NewFromInt(5),
			Amount:        decimal.NewFromInt(75),
		},
	}).Error)

	require.NoError(t, db.Create([]price.Price{
		{
			Date:           time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
			CommodityName:  "GOLD",
			CommodityType:  config.Unknown,
			QuoteCommodity: "INR",
			Value:          decimal.NewFromInt(10),
		},
		{
			Date:           time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC),
			CommodityName:  "GOLD",
			CommodityType:  config.Unknown,
			QuoteCommodity: "INR",
			Value:          decimal.NewFromInt(15),
		},
		{
			Date:           time.Date(2024, 1, 4, 0, 0, 0, 0, time.UTC),
			CommodityName:  "GOLD",
			CommodityType:  config.Unknown,
			QuoteCommodity: "INR",
			Value:          decimal.NewFromInt(20),
		},
	}).Error)

	posts := query.Init(db).Like("Assets:%").UntilToday().All()
	series := RunningBalance(db, posts)
	require.Len(t, series, 4)

	assert.True(t, decimal.NewFromInt(100).Equal(series[0].Value))
	assert.True(t, decimal.NewFromInt(100).Equal(series[1].Value))
	assert.True(t, decimal.NewFromInt(225).Equal(series[2].Value))
	assert.True(t, decimal.NewFromInt(300).Equal(series[3].Value))
}
