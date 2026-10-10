package import_rule

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Rule{}))
	return db
}

func TestRuleEvaluate(t *testing.T) {
	minAmt := 500.0
	rule := Rule{
		Name:           "Uber Rides",
		Enabled:        true,
		PayeePattern:   "Uber",
		PayeeMatchType: MatchContains,
		TxType:         TxTypeDebit,
		TargetAccount:  "Expenses:Transport:Taxi",
		Tags:           []string{"#commute"},
	}

	tx1 := CandidateTransaction{
		Payee:   "UBER *TRIP 1234",
		Amount:  250.0,
		IsDebit: true,
	}
	assert.True(t, rule.Evaluate(tx1))

	// Credit transaction shouldn't match
	txCredit := CandidateTransaction{
		Payee:   "UBER REFUND",
		Amount:  250.0,
		IsDebit: false,
	}
	assert.False(t, rule.Evaluate(txCredit))

	// Add MinAmount requirement
	rule.MinAmount = &minAmt
	assert.False(t, rule.Evaluate(tx1))

	txLarge := CandidateTransaction{
		Payee:   "UBER PREMIUM",
		Amount:  650.0,
		IsDebit: true,
	}
	assert.True(t, rule.Evaluate(txLarge))
}

func TestRuleRegexMatch(t *testing.T) {
	rule := Rule{
		Name:           "Amazon Purchases",
		Enabled:        true,
		PayeePattern:   `AMZN|AMAZON.*PAY`,
		PayeeMatchType: MatchRegex,
		TargetAccount:  "Expenses:Shopping",
	}

	assert.True(t, rule.Evaluate(CandidateTransaction{Payee: "AMZN Mktp US", IsDebit: true}))
	assert.True(t, rule.Evaluate(CandidateTransaction{Payee: "AMAZON RETAIL PAY", IsDebit: true}))
	assert.False(t, rule.Evaluate(CandidateTransaction{Payee: "WALMART", IsDebit: true}))
}

func TestRuleCRUD(t *testing.T) {
	db := openTestDB(t)

	rule := &Rule{
		Name:          "Grocery Rule",
		PayeePattern:  "Whole Foods",
		TargetAccount: "Expenses:Groceries",
		Priority:      10,
	}

	err := Save(db, rule)
	require.NoError(t, err)
	assert.NotZero(t, rule.ID)

	all, err := All(db)
	require.NoError(t, err)
	assert.Len(t, all, 1)
	assert.Equal(t, "Grocery Rule", all[0].Name)

	err = Delete(db, rule.ID)
	require.NoError(t, err)

	all, err = All(db)
	require.NoError(t, err)
	assert.Empty(t, all)
}
