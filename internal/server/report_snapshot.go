package server

import (
	"encoding/json"
	"errors"

	"github.com/ananthakumaran/paisa/internal/model/report_snapshot"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const (
	networthSnapshotName          = "networth"
	networthSnapshotSchemaVersion = 1

	gainSnapshotName          = "gain"
	gainSnapshotSchemaVersion = 1
)

func RefreshNetworthSnapshot(db *gorm.DB) error {
	payload, err := json.Marshal(GetNetworth(db, ""))
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return report_snapshot.Replace(tx, networthSnapshotName, networthSnapshotSchemaVersion, payload)
	})
}

func RefreshGainSnapshot(db *gorm.DB) error {
	payload, err := json.Marshal(GetGain(db))
	if err != nil {
		return err
	}
	return db.Transaction(func(tx *gorm.DB) error {
		return report_snapshot.Replace(tx, gainSnapshotName, gainSnapshotSchemaVersion, payload)
	})
}

func getNetworthSnapshotPayload(db *gorm.DB) ([]byte, bool) {
	return getReportSnapshotPayload(db, snapshotKindNetworth, networthSnapshotName, networthSnapshotSchemaVersion)
}

func getGainSnapshotPayload(db *gorm.DB) ([]byte, bool) {
	return getReportSnapshotPayload(db, snapshotKindGain, gainSnapshotName, gainSnapshotSchemaVersion)
}

func getReportSnapshotPayload(db *gorm.DB, kind snapshotKind, name string, schemaVersion int) ([]byte, bool) {
	if err := ensureSnapshotFresh(db, kind); err != nil {
		log.WithError(err).WithField("snapshot", name).Warn("Failed to refresh report snapshot; falling back to live query path")
		return nil, false
	}

	snapshot, err := report_snapshot.Get(db, name)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).WithField("snapshot", name).Warn("Failed to load report snapshot; falling back to live query path")
		}
		return nil, false
	}
	if snapshot.SchemaVersion != schemaVersion || len(snapshot.Payload) == 0 || !json.Valid(snapshot.Payload) {
		return nil, false
	}
	return snapshot.Payload, true
}
