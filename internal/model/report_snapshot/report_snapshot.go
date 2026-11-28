package report_snapshot

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ReportSnapshot stores a materialized JSON response payload keyed by report name.
type ReportSnapshot struct {
	Name          string    `gorm:"primaryKey;not null"`
	SchemaVersion int       `gorm:"not null"`
	Payload       []byte    `gorm:"type:blob;not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (ReportSnapshot) TableName() string {
	return "report_snapshots"
}

func Get(db *gorm.DB, name string) (ReportSnapshot, error) {
	var snapshot ReportSnapshot
	if err := db.Where("name = ?", name).First(&snapshot).Error; err != nil {
		return ReportSnapshot{}, err
	}
	return snapshot, nil
}

func Replace(tx *gorm.DB, name string, schemaVersion int, payload []byte) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"schema_version", "payload", "updated_at"}),
	}).Create(&ReportSnapshot{
		Name:          name,
		SchemaVersion: schemaVersion,
		Payload:       append([]byte(nil), payload...),
		UpdatedAt:     time.Now(),
	}).Error
}
