package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/kelindar/binary"
	"github.com/mitchellh/hashstructure/v2"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Cache struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	ExpiresAt time.Time `json:"expires_at"`
	HashKey   string    `gorm:"uniqueIndex" json:"hash_key"`
	Value     []byte    `gorm:"type:BLOB" json:"item"`
}

func DeleteExpired(db *gorm.DB) error {
	err := db.Exec("DELETE FROM caches WHERE expires_at < ?", time.Now()).Error
	if err != nil {
		return err
	}
	return nil
}

var (
	gcMu                   sync.Mutex
	lastGarbageCollectedAt time.Time
)

func maybeCollectExpired(db *gorm.DB) {
	now := time.Now()
	gcMu.Lock()
	shouldRun := lastGarbageCollectedAt.IsZero() || now.Sub(lastGarbageCollectedAt) > 24*time.Hour
	if shouldRun {
		lastGarbageCollectedAt = now
	}
	gcMu.Unlock()

	if shouldRun {
		_ = DeleteExpired(db)
	}
}

func Lookup[I any, K any](db *gorm.DB, key K, fallback func() I) I {
	var item I
	var cache Cache

	maybeCollectExpired(db)

	hash, err := hashstructure.Hash(key, hashstructure.FormatV2, nil)
	hashKey := fmt.Sprintf("%d", hash)
	if err == nil {
		err := db.Where("hash_key = ?", hashKey).First(&cache).Error
		if err == nil && time.Now().Before(cache.ExpiresAt) {
			if err := binary.Unmarshal(cache.Value, &item); err == nil {
				return item
			}
		}
	}

	item = fallback()
	bytes, err := binary.Marshal(item)
	if err == nil {
		cache = Cache{
			ExpiresAt: time.Now().Add(24 * time.Hour),
			HashKey:   hashKey,
			Value:     bytes,
		}
		db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "hash_key"}},
			DoUpdates: clause.AssignmentColumns([]string{"expires_at", "value"}),
		}).Create(&cache)
	}
	return item
}
