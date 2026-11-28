package accounting

import (
	"sync"
	"testing"

	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestAccountCache_ConcurrentReadAndClear(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&posting.Posting{}))
	require.NoError(t, db.Create(&posting.Posting{Account: "Assets:Checking", Commodity: "INR"}).Error)
	require.NoError(t, db.Create(&posting.Posting{Account: "Expenses:Food", Commodity: "INR"}).Error)

	ClearCache()
	t.Cleanup(ClearCache)

	assert.True(t, IsLeafAccount(db, "Assets:Checking"))
	assert.True(t, IsLeafAccount(db, "Expenses:Food"))
	assert.False(t, IsLeafAccount(db, "Assets:Unknown"))

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = AllAccounts(db)
			_ = IsLeafAccount(db, "Assets:Checking")
		}()
		go func() {
			defer wg.Done()
			ClearCache()
		}()
	}
	wg.Wait()
}
