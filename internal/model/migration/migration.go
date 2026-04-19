package migration

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
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
	{Version: 2, Apply: v2AddQuoteCommodity},
	{Version: 3, Apply: v3PostingsIndexes},
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

// v2AddQuoteCommodity adds the quote_commodity and source columns to the prices
// table, backfills existing rows with the configured default currency, and
// creates indexes for efficient pair-aware lookups.
func v2AddQuoteCommodity(db *gorm.DB) error {
	// AutoMigrate adds the new columns to any existing prices table; for a
	// fresh install the table was already created with all columns by v1.
	if err := db.AutoMigrate(&price.Price{}); err != nil {
		return fmt.Errorf("v2: AutoMigrate prices failed: %w", err)
	}

	// Determine the default currency for the backfill.
	dc := config.DefaultCurrency()
	if dc == "" {
		dc = "INR"
	}

	// Backfill existing rows that pre-date this migration (quote_commodity = '' or NULL).
	if err := db.Exec(
		"UPDATE prices SET quote_commodity = ? WHERE quote_commodity IS NULL OR quote_commodity = ''", dc,
	).Error; err != nil {
		return fmt.Errorf("v2: backfill quote_commodity failed: %w", err)
	}

	// Index on commodity_name for fast per-commodity history queries.
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_prices_commodity_name ON prices(commodity_name)",
	).Error; err != nil {
		return fmt.Errorf("v2: create idx_prices_commodity_name failed: %w", err)
	}

	// Index on quote_commodity for pair-aware queries.
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_prices_quote_commodity ON prices(quote_commodity)",
	).Error; err != nil {
		return fmt.Errorf("v2: create idx_prices_quote_commodity failed: %w", err)
	}

	// Unique index enforces no duplicate prices per (commodity_type, date, base, quote) tuple.
	if err := db.Exec(
		"CREATE UNIQUE INDEX IF NOT EXISTS idx_prices_type_date_base_quote " +
			"ON prices(commodity_type, date, commodity_name, quote_commodity)",
	).Error; err != nil {
		return fmt.Errorf("v2: create idx_prices_type_date_base_quote failed: %w", err)
	}

	return nil
}

// v3PostingsIndexes adds indexes on the postings table for the most common
// query patterns. Almost every API endpoint filters postings by:
//   - account prefix  (account LIKE 'Assets:%')
//   - date range      (date < ?)
//   - forecast flag   (forecast = false / true)
//
// Without indexes each of these clauses triggers a full table scan.
func v3PostingsIndexes(db *gorm.DB) error {
	// Ensure the postings table exists before attempting to index it.
	// In production this is guaranteed by v1Baseline; the AutoMigrate call is
	// a no-op when the table already exists, and protects against partial
	// install scenarios (e.g., isolated migration tests).
	if err := db.AutoMigrate(&posting.Posting{}); err != nil {
		return fmt.Errorf("v3: AutoMigrate postings failed: %w", err)
	}

	// Single-column index on account for prefix / equality lookups.
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_postings_account ON postings(account)",
	).Error; err != nil {
		return fmt.Errorf("v3: create idx_postings_account failed: %w", err)
	}

	// Single-column index on date for time-range filtering.
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_postings_date ON postings(date)",
	).Error; err != nil {
		return fmt.Errorf("v3: create idx_postings_date failed: %w", err)
	}

	// Composite index covers the combined (forecast=false, date<X) filter
	// present in every query.Init(db).UntilToday() call.
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_postings_forecast_date ON postings(forecast, date)",
	).Error; err != nil {
		return fmt.Errorf("v3: create idx_postings_forecast_date failed: %w", err)
	}

	// Composite covers account-prefix + date-range queries (most common:
	// account LIKE 'Assets:%' AND forecast = false AND date < Y).
	if err := db.Exec(
		"CREATE INDEX IF NOT EXISTS idx_postings_account_date ON postings(account, date)",
	).Error; err != nil {
		return fmt.Errorf("v3: create idx_postings_account_date failed: %w", err)
	}

	return nil
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
