package accounting

import (
"sort"
"testing"

"github.com/ananthakumaran/paisa/internal/model/posting"
"github.com/glebarez/sqlite"
"github.com/stretchr/testify/require"
"gorm.io/gorm"
)

func openTestDB(t *testing.T) *gorm.DB {
t.Helper()
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
require.NoError(t, err)
require.NoError(t, db.AutoMigrate(&posting.Posting{}))
return db
}

func TestAllAccountsSorted(t *testing.T) {
// Reset global cache before test
ClearCache()

db := openTestDB(t)

// Insert postings in non-alphabetical order (insertion order)
accounts := []string{
"Income:Salary:Acme",
"Assets:Checking",
"Expenses:Rent",
"Assets:Equity:NIFTY",
"Assets:Equity:ABNB",
"Assets:Equity:AAPL",
"Income:CapitalGains:Equity:AAPL",
"Expenses:Charges",
"Assets:Dollar",
"Income:Interest:Checking",
}
for i, acc := range accounts {
p := posting.Posting{Account: acc, TransactionID: "t" + string(rune('0'+i)), Payee: "test"}
require.NoError(t, db.Create(&p).Error)
}

result := AllAccounts(db)

sorted := make([]string, len(result))
copy(sorted, result)
sort.Strings(sorted)

require.Equal(t, sorted, result, "AllAccounts should return accounts in alphabetical order")
}

func TestAllAccountsCacheInvalidation(t *testing.T) {
// Reset global cache before test
ClearCache()

db := openTestDB(t)

// Insert some accounts
p1 := posting.Posting{Account: "Zzz:Last", TransactionID: "t1", Payee: "test"}
require.NoError(t, db.Create(&p1).Error)

result1 := AllAccounts(db)
require.Equal(t, []string{"Zzz:Last"}, result1)

// Insert more accounts
p2 := posting.Posting{Account: "Aaa:First", TransactionID: "t2", Payee: "test"}
require.NoError(t, db.Create(&p2).Error)

// Without cache clear, still returns old data
result2 := AllAccounts(db)
require.Equal(t, []string{"Zzz:Last"}, result2, "cache should still return old data")

// After clearing cache, returns new sorted data
ClearCache()
result3 := AllAccounts(db)
require.Equal(t, []string{"Aaa:First", "Zzz:Last"}, result3, "after clear, should return sorted data")
}
