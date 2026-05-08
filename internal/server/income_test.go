package server

import (
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetIncome_MultiYearSeriesUsesPositiveAmounts(t *testing.T) {
	loadTestConfig(t, false)
	utils.SetNow("2025-03-20")
	defer utils.UnsetNow()

	db := openTestDB(t)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "income-1",
		Date:          parseDay("2025-01-10"),
		Account:       "Income:Salary",
		Amount:        decimal.NewFromFloat(-2500),
		Commodity:     "INR",
	}).Error)
	require.NoError(t, db.Create(&posting.Posting{
		TransactionID: "income-2",
		Date:          parseDay("2024-01-10"),
		Account:       "Income:Salary",
		Amount:        decimal.NewFromFloat(-2000),
		Commodity:     "INR",
	}).Error)

	response := GetIncome(db, 2, 0)
	series := response["multi_year"].(map[string]YoYMonthlySeries)

	assert.True(t, series["2025"].Total.Equal(decimal.NewFromFloat(2500)))
	assert.True(t, series["2024"].Total.Equal(decimal.NewFromFloat(2000)))
	assert.True(t, series["2025"].Month["2025-01"].Equal(decimal.NewFromFloat(2500)))
	assert.True(t, series["2024"].Month["2024-01"].Equal(decimal.NewFromFloat(2000)))
}
