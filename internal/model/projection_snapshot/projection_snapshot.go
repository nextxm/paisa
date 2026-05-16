package projection_snapshot

import (
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	// SnapshotName is the singleton projection snapshot row key.
	SnapshotName = "networth_projection"
	// SchemaVersion tracks the persisted projection-input contract.
	SchemaVersion = 1
)

// ProjectionSnapshot stores the precomputed base inputs for
// GET /api/networth/projection.
type ProjectionSnapshot struct {
	Name                string          `gorm:"primaryKey;not null"`
	SchemaVersion       int             `gorm:"not null"`
	CurrentNetworth     decimal.Decimal `gorm:"not null"`
	MonthlyContribution decimal.Decimal `gorm:"not null"`
	SavingsRate         decimal.Decimal `gorm:"not null"`
	AnnualExpenses      decimal.Decimal `gorm:"not null"`
	UpdatedAt           time.Time       `gorm:"not null"`
}

func (ProjectionSnapshot) TableName() string {
	return "projection_snapshots"
}

func Get(db *gorm.DB) (ProjectionSnapshot, error) {
	var snapshot ProjectionSnapshot
	if err := db.Where("name = ?", SnapshotName).First(&snapshot).Error; err != nil {
		return ProjectionSnapshot{}, err
	}
	return snapshot, nil
}

func Replace(
	tx *gorm.DB,
	currentNetworth decimal.Decimal,
	monthlyContribution decimal.Decimal,
	savingsRate decimal.Decimal,
	annualExpenses decimal.Decimal,
) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"schema_version", "current_networth", "monthly_contribution", "savings_rate", "annual_expenses", "updated_at"}),
	}).Create(&ProjectionSnapshot{
		Name:                SnapshotName,
		SchemaVersion:       SchemaVersion,
		CurrentNetworth:     currentNetworth,
		MonthlyContribution: monthlyContribution,
		SavingsRate:         savingsRate,
		AnnualExpenses:      annualExpenses,
		UpdatedAt:           time.Now(),
	}).Error
}
