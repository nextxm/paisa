package server

import (
	"strings"

	"github.com/ananthakumaran/paisa/internal/model/metadata"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type snapshotKind string

const (
	snapshotKindDashboard        snapshotKind = "dashboard"
	snapshotKindProjection       snapshotKind = "projection"
	snapshotKindInvestmentIncome snapshotKind = "investment_income"
)

const (
	dashboardSnapshotDirtyKey        = "snapshot_dirty_dashboard"
	projectionSnapshotDirtyKey       = "snapshot_dirty_projection"
	investmentIncomeSnapshotDirtyKey = "snapshot_dirty_investment_income"
)

func parseSnapshotKind(raw string) snapshotKind {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case string(snapshotKindDashboard):
		return snapshotKindDashboard
	case string(snapshotKindProjection), "networth", "networth_projection":
		return snapshotKindProjection
	case string(snapshotKindInvestmentIncome), "income", "investment":
		return snapshotKindInvestmentIncome
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
	}
	for _, kind := range kinds {
		if err := setSnapshotDirty(db, kind, true); err != nil {
			return err
		}
	}
	return nil
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
	return refreshSnapshotByKind(db, kind)
}
