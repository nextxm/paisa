package server

import (
	"encoding/json"

	"github.com/ananthakumaran/paisa/internal/model/investment_income_snapshot"
	"github.com/ananthakumaran/paisa/internal/utils"
	"gorm.io/gorm"
)

func buildInvestmentIncomeSnapshotPayload(db *gorm.DB) ([]byte, error) {
	liveData := GetInvestmentIncome(db, utils.ToDate(utils.Now()))
	return json.Marshal(liveData)
}

// RefreshInvestmentIncomeSnapshot precomputes the entire investment income payload
// and saves it inside the SQLite snapshots table for instant O(1) reads.
func RefreshInvestmentIncomeSnapshot(db *gorm.DB) error {
	payload, err := buildInvestmentIncomeSnapshotPayload(db)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		return investment_income_snapshot.Replace(tx, payload)
	})
}

func getInvestmentIncomeSnapshotPayload(db *gorm.DB) ([]byte, bool) {
	snapshot, err := investment_income_snapshot.Get(db)
	if err != nil {
		return nil, false
	}
	if snapshot.SchemaVersion != investment_income_snapshot.SchemaVersion {
		return nil, false
	}
	return snapshot.Payload, true
}
