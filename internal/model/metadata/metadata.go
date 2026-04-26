package metadata

import (
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Metadata persists arbitrary key/value pairs in the database.
// Keys are unique; calling Set on an existing key upserts the value.
type Metadata struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"uniqueIndex;not null" json:"key"`
	Value string `gorm:"not null" json:"value"`
}

// Get returns the value for the given key, or gorm.ErrRecordNotFound if absent.
func Get(db *gorm.DB, key string) (string, error) {
	var m Metadata
	if err := db.Where("key = ?", key).First(&m).Error; err != nil {
		return "", err
	}
	return m.Value, nil
}

// GetOrDefault returns the stored value for key, or defaultValue when the key
// does not exist.  Any other database error is returned as-is.
func GetOrDefault(db *gorm.DB, key, defaultValue string) (string, error) {
	val, err := Get(db, key)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return defaultValue, nil
	}
	return val, err
}

// Set inserts or updates the value for the given key.  The operation is
// executed as a single atomic statement using INSERT … ON CONFLICT DO UPDATE,
// so concurrent callers are safe and the value is always written regardless of
// whether it is the zero value for its type.
func Set(db *gorm.DB, key, value string) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&Metadata{Key: key, Value: value}).Error
}
