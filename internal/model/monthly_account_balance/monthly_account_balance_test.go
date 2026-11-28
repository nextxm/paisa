package monthly_account_balance

import (
	"testing"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRefreshFromPostings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&MonthlyAccountBalance{}))

	jan := time.Date(2024, 1, 10, 0, 0, 0, 0, time.UTC)
	feb := time.Date(2024, 2, 5, 0, 0, 0, 0, time.UTC)

	postings := []*posting.Posting{
		{Date: jan, Account: "Expenses:Food", Commodity: "INR", Quantity: decimal.NewFromInt(100), Amount: decimal.NewFromInt(100)},
		{Date: jan, Account: "Expenses:Food", Commodity: "INR", Quantity: decimal.NewFromInt(250), Amount: decimal.NewFromInt(250)},
		{Date: feb, Account: "Expenses:Food", Commodity: "INR", Quantity: decimal.NewFromInt(50), Amount: decimal.NewFromInt(50)},
		{Date: feb, Account: "Expenses:Food", Commodity: "INR", Quantity: decimal.NewFromInt(999), Amount: decimal.NewFromInt(999), Forecast: true},
	}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		return RefreshFromPostings(tx, postings)
	}))

	rows, err := All(db)
	require.NoError(t, err)
	require.Len(t, rows, 2)

	assert.Equal(t, "2024-01", rows[0].Month)
	assert.True(t, rows[0].Amount.Equal(decimal.NewFromInt(350)))

	assert.Equal(t, "2024-02", rows[1].Month)
	assert.True(t, rows[1].Amount.Equal(decimal.NewFromInt(50)))
}
