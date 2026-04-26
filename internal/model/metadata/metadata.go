package metadata

import (
	"errors"

	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Metadata stores arbitrary key/value pairs for application state.
// Key uniqueness is enforced by the uniqueIndex tag and the v3 migration.
type Metadata struct {
	ID    uint   `gorm:"primaryKey" json:"id"`
	Key   string `gorm:"uniqueIndex;not null" json:"key"`
	Value string `gorm:"not null" json:"value"`
}

// Get returns the value for key, or ("", false) when the key is absent.
// Unexpected database errors are logged and treated as a missing key.
func Get(db *gorm.DB, key string) (string, bool) {
	var m Metadata
	err := db.Where("key = ?", key).First(&m).Error
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Errorf("metadata: Get(%q) failed", key)
		}
		return "", false
	}
	return m.Value, true
}

// Set inserts or updates the value for key using an atomic upsert so that
// concurrent callers cannot produce a uniqueness violation.
func Set(db *gorm.DB, key, value string) error {
	return db.Exec(
		"INSERT INTO metadata (key, value) VALUES (?, ?) "+
			"ON CONFLICT (key) DO UPDATE SET value = excluded.value",
		key, value,
	).Error
}

// Delete removes the entry for key. No error is returned if the key is absent.
func Delete(db *gorm.DB, key string) error {
	return db.Where("key = ?", key).Delete(&Metadata{}).Error
}
