package account_reconciliation

import (
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const DefaultFrequencyDays = 30

type AccountReconciliation struct {
	ID                 uint       `gorm:"primaryKey" json:"id"`
	Account            string     `gorm:"uniqueIndex;not null" json:"account"`
	LastReconciledDate *time.Time `json:"last_reconciled_date"`
	FrequencyDays      int        `gorm:"not null;default:30" json:"frequency_days"`
}

func (AccountReconciliation) TableName() string {
	return "account_reconciliation"
}

func Get(db *gorm.DB, account string) (*AccountReconciliation, error) {
	var reconciliation AccountReconciliation
	if err := db.Where("account = ?", account).First(&reconciliation).Error; err != nil {
		return nil, err
	}
	return &reconciliation, nil
}

func GetAll(db *gorm.DB) ([]AccountReconciliation, error) {
	var reconciliations []AccountReconciliation
	if err := db.Order("account asc").Find(&reconciliations).Error; err != nil {
		return nil, err
	}
	return reconciliations, nil
}

func Upsert(db *gorm.DB, account string, lastReconciledDate *time.Time, frequencyDays int) (*AccountReconciliation, error) {
	if frequencyDays <= 0 {
		frequencyDays = DefaultFrequencyDays
	}

	reconciliation := &AccountReconciliation{
		Account:            account,
		LastReconciledDate: lastReconciledDate,
		FrequencyDays:      frequencyDays,
	}

	if err := db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "account"}},
		DoUpdates: clause.AssignmentColumns([]string{"last_reconciled_date", "frequency_days"}),
	}).Create(reconciliation).Error; err != nil {
		return nil, err
	}

	return Get(db, account)
}
