package investment_income_snapshot

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// SnapshotName is the singleton investment income snapshot row key.
	SnapshotName = "investment_income"
	// SchemaVersion tracks the snapshot payload contract persisted in SQLite.
	SchemaVersion = 1
)

// InvestmentIncomeSnapshot stores the materialized /api/income/investment response payload.
type InvestmentIncomeSnapshot struct {
	Name          string    `gorm:"primaryKey;not null"`
	SchemaVersion int       `gorm:"not null"`
	Payload       []byte    `gorm:"type:blob;not null"`
	UpdatedAt     time.Time `gorm:"not null"`
}

func (InvestmentIncomeSnapshot) TableName() string {
	return "investment_income_snapshots"
}

func Get(db *gorm.DB) (InvestmentIncomeSnapshot, error) {
	var snapshot InvestmentIncomeSnapshot
	if err := db.Where("name = ?", SnapshotName).First(&snapshot).Error; err != nil {
		return InvestmentIncomeSnapshot{}, err
	}
	return snapshot, nil
}

func Replace(tx *gorm.DB, payload []byte) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"schema_version", "payload", "updated_at"}),
	}).Create(&InvestmentIncomeSnapshot{
		Name:          SnapshotName,
		SchemaVersion: SchemaVersion,
		Payload:       append([]byte(nil), payload...),
		UpdatedAt:     time.Now(),
	}).Error
}
