package dashboard_snapshot

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// SnapshotName is the singleton dashboard snapshot row key.
	SnapshotName = "dashboard"
	// SchemaVersion tracks the snapshot payload contract persisted in SQLite.
	SchemaVersion = 1
)

// DashboardSnapshot stores the materialized /api/dashboard response payload.
type DashboardSnapshot struct {
	Name          string    `gorm:"primaryKey;not null"`
	SchemaVersion int       `gorm:"not null"`
	Payload       []byte    `gorm:"type:blob;not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (DashboardSnapshot) TableName() string {
	return "dashboard_snapshots"
}

func Get(db *gorm.DB) (DashboardSnapshot, error) {
	var snapshot DashboardSnapshot
	if err := db.Where("name = ?", SnapshotName).First(&snapshot).Error; err != nil {
		return DashboardSnapshot{}, err
	}
	return snapshot, nil
}

func Replace(tx *gorm.DB, payload []byte) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"schema_version", "payload", "updated_at"}),
	}).Create(&DashboardSnapshot{
		Name:          SnapshotName,
		SchemaVersion: SchemaVersion,
		Payload:       append([]byte(nil), payload...),
		UpdatedAt:     time.Now(),
	}).Error
}
