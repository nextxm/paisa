package cache

import (
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupCacheTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)
	require.NoError(t, db.AutoMigrate(&Cache{}))
	return db
}

func TestLookup_UpsertsExpiredEntryWithoutDuplicates(t *testing.T) {
	db := setupCacheTestDB(t)

	calls := 0
	val := Lookup(db, "k1", func() string {
		calls++
		return "v1"
	})
	assert.Equal(t, "v1", val)
	assert.Equal(t, 1, calls)

	// Force-expire the entry.
	require.NoError(t, db.Model(&Cache{}).Where("1 = 1").Update("expires_at", time.Now().Add(-1*time.Hour)).Error)

	val = Lookup(db, "k1", func() string {
		calls++
		return "v2"
	})
	assert.Equal(t, "v2", val)
	assert.Equal(t, 2, calls)

	var count int64
	require.NoError(t, db.Model(&Cache{}).Count(&count).Error)
	assert.Equal(t, int64(1), count)
}

func TestLookup_ConcurrentAccess(t *testing.T) {
	db := setupCacheTestDB(t)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			v := Lookup(db, idx%3, func() int { return idx % 3 })
			assert.Equal(t, idx%3, v)
		}(i)
	}
	wg.Wait()
}
