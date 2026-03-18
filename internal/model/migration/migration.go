package migration

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/model/cache"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	mutualfundModel "github.com/ananthakumaran/paisa/internal/model/mutualfund/scheme"
	npsModel "github.com/ananthakumaran/paisa/internal/model/nps/scheme"
	"github.com/ananthakumaran/paisa/internal/model/portfolio"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/model/session"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SchemaVersion records an applied migration version in the database.
type SchemaVersion struct {
	Version   int       `gorm:"primaryKey"`
	AppliedAt time.Time `gorm:"not null"`
}

type migrationFn func(db *gorm.DB) error

type step struct {
	Version int
	Apply   migrationFn
}

// steps is the ordered list of all database migrations.
var steps = []step{
	{Version: 1, Apply: v1Baseline},
}

// v1Baseline is the initial migration that creates all tables for existing models.
func v1Baseline(db *gorm.DB) error {
	return db.AutoMigrate(
		&npsModel.Scheme{},
		&mutualfundModel.Scheme{},
		&posting.Posting{},
		&price.Price{},
		&portfolio.Portfolio{},
		&cii.CII{},
		&cache.Cache{},
		&session.Session{},
	)
}

// RunMigrations initializes the schema_versions table, applies any unapplied
// migrations in version order, and logs the current schema version.
// It is safe to call multiple times; already-applied migrations are skipped.
func RunMigrations(db *gorm.DB) error {
	if err := db.AutoMigrate(&SchemaVersion{}); err != nil {
		return fmt.Errorf("failed to initialize schema_versions table: %w", err)
	}

	for _, m := range steps {
		var count int64
		if err := db.Model(&SchemaVersion{}).Where("version = ?", m.Version).Count(&count).Error; err != nil {
			return fmt.Errorf("failed to query schema_versions: %w", err)
		}
		if count > 0 {
			continue
		}

		log.Infof("Applying database migration v%d", m.Version)
		if err := m.Apply(db); err != nil {
			return fmt.Errorf("migration v%d failed: %w", m.Version, err)
		}

		if err := db.Create(&SchemaVersion{Version: m.Version, AppliedAt: time.Now()}).Error; err != nil {
			return fmt.Errorf("failed to record migration v%d: %w", m.Version, err)
		}
	}

	var current SchemaVersion
	if err := db.Order("version desc").First(&current).Error; err == nil {
		log.Infof("Database schema version: %d", current.Version)
	}

	return nil
}

// CurrentVersion returns the highest applied schema version, or 0 if none.
func CurrentVersion(db *gorm.DB) int {
	var current SchemaVersion
	if err := db.Order("version desc").First(&current).Error; err != nil {
		return 0
	}
	return current.Version
}
