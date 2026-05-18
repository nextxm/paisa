package server

import (
	"encoding/json"
	"errors"

	"github.com/ananthakumaran/paisa/internal/model/dashboard_snapshot"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

func buildDashboardSnapshotPayload(db *gorm.DB) ([]byte, error) {
	return json.Marshal(GetDashboard(db))
}

func RefreshDashboardSnapshot(db *gorm.DB) error {
	payload, err := buildDashboardSnapshotPayload(db)
	if err != nil {
		return err
	}

	return db.Transaction(func(tx *gorm.DB) error {
		return dashboard_snapshot.Replace(tx, payload)
	})
}

func getDashboardSnapshotPayload(db *gorm.DB) ([]byte, bool) {
	snapshot, err := dashboard_snapshot.Get(db)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Warn("Failed to load dashboard snapshot; falling back to live query path")
		}
		return nil, false
	}
	if snapshot.SchemaVersion != dashboard_snapshot.SchemaVersion {
		log.WithField("schema_version", snapshot.SchemaVersion).
			Warn("Dashboard snapshot schema version mismatch; falling back to live query path")
		return nil, false
	}
	if len(snapshot.Payload) == 0 || !json.Valid(snapshot.Payload) {
		log.Warn("Dashboard snapshot payload is invalid; falling back to live query path")
		return nil, false
	}
	return snapshot.Payload, true
}
