package metadata

import (
	"errors"

	"gorm.io/gorm"
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

// Set inserts or updates the value for the given key.
func Set(db *gorm.DB, key, value string) error {
	return db.Where(Metadata{Key: key}).
		Assign(Metadata{Value: value}).
		FirstOrCreate(&Metadata{Key: key}).Error
}
