package server

import (
	"strings"
	"sync"

	"github.com/ananthakumaran/paisa/internal/model/metadata"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type snapshotKind string

var snapshotRefreshLocks = map[snapshotKind]*sync.Mutex{
	snapshotKindDashboard:        {},
	snapshotKindProjection:       {},
	snapshotKindInvestmentIncome: {},
	snapshotKindNetworth:         {},
	snapshotKindGain:             {},
}

func snapshotLock(kind snapshotKind) *sync.Mutex {
	if mu, ok := snapshotRefreshLocks[kind]; ok {
		return mu
	}
	return nil
}

const (
	snapshotKindDashboard        snapshotKind = "dashboard"
	snapshotKindProjection       snapshotKind = "projection"
	snapshotKindInvestmentIncome snapshotKind = "investment_income"
	snapshotKindNetworth         snapshotKind = "networth"
	snapshotKindGain             snapshotKind = "gain"
)

const (
	dashboardSnapshotDirtyKey        = "snapshot_dirty_dashboard"
	projectionSnapshotDirtyKey       = "snapshot_dirty_projection"
	investmentIncomeSnapshotDirtyKey = "snapshot_dirty_investment_income"
	networthSnapshotDirtyKey         = "snapshot_dirty_networth"
	gainSnapshotDirtyKey             = "snapshot_dirty_gain"
)

func parseSnapshotKind(raw string) snapshotKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(snapshotKindDashboard):
		return snapshotKindDashboard
	case string(snapshotKindProjection), "networth_projection":
		return snapshotKindProjection
	case string(snapshotKindInvestmentIncome), "income", "investment":
		return snapshotKindInvestmentIncome
	case string(snapshotKindNetworth):
		return snapshotKindNetworth
	case string(snapshotKindGain):
		return snapshotKindGain
	default:
		return ""
	}
}

func snapshotDirtyKey(kind snapshotKind) string {
	switch kind {
	case snapshotKindDashboard:
		return dashboardSnapshotDirtyKey
	case snapshotKindProjection:
		return projectionSnapshotDirtyKey
	case snapshotKindInvestmentIncome:
		return investmentIncomeSnapshotDirtyKey
	case snapshotKindNetworth:
		return networthSnapshotDirtyKey
	case snapshotKindGain:
		return gainSnapshotDirtyKey
	default:
		return ""
	}
}

func setSnapshotDirty(db *gorm.DB, kind snapshotKind, dirty bool) error {
	key := snapshotDirtyKey(kind)
	if key == "" {
		return nil
	}
	value := "false"
	if dirty {
		value = "true"
	}
	return metadata.Set(db, key, value)
}

func isSnapshotDirty(db *gorm.DB, kind snapshotKind) bool {
	key := snapshotDirtyKey(kind)
	if key == "" {
		return false
	}
	value, err := metadata.GetOrDefault(db, key, "false")
	if err != nil {
		log.WithError(err).WithField("snapshot", kind).Warn("snapshot: failed reading dirty marker")
		return false
	}
	return strings.EqualFold(value, "true")
}

func markAllSnapshotsDirty(db *gorm.DB) error {
	kinds := []snapshotKind{
		snapshotKindDashboard,
		snapshotKindProjection,
		snapshotKindInvestmentIncome,
		snapshotKindNetworth,
		snapshotKindGain,
	}
	return db.Transaction(func(tx *gorm.DB) error {
		for _, kind := range kinds {
			if err := setSnapshotDirty(tx, kind, true); err != nil {
				return err
			}
		}
		return nil
	})
}

func refreshSnapshotByKind(db *gorm.DB, kind snapshotKind) error {
	var err error
	switch kind {
	case snapshotKindDashboard:
		err = RefreshDashboardSnapshot(db)
	case snapshotKindProjection:
		err = RefreshNetworthProjectionSnapshot(db)
	case snapshotKindInvestmentIncome:
		err = RefreshInvestmentIncomeSnapshot(db)
	case snapshotKindNetworth:
		err = RefreshNetworthSnapshot(db)
	case snapshotKindGain:
		err = RefreshGainSnapshot(db)
	default:
		return nil
	}
	if err != nil {
		return err
	}
	return setSnapshotDirty(db, kind, false)
}

func ensureSnapshotFresh(db *gorm.DB, kind snapshotKind) error {
	if !isSnapshotDirty(db, kind) {
		return nil
	}
	if mu := snapshotLock(kind); mu != nil {
		mu.Lock()
		defer mu.Unlock()
	}
	if !isSnapshotDirty(db, kind) {
		return nil
	}
	return refreshSnapshotByKind(db, kind)
}
