package migration

import (
	"fmt"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/account_balance"
	"github.com/ananthakumaran/paisa/internal/model/account_note"
	"github.com/ananthakumaran/paisa/internal/model/account_reconciliation"
	"github.com/ananthakumaran/paisa/internal/model/cache"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	"github.com/ananthakumaran/paisa/internal/model/import_preset"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
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
	{Version: 3, Apply: v3AddMetadata},
	{Version: 4, Apply: v4AddAccountNotes},
	{Version: 5, Apply: v5AddImportPresets},
	{Version: 6, Apply: v6AddAccountReconciliation},
	{Version: 7, Apply: v7AddAccountBalances},
	{Version: 8, Apply: v8AddParserTrainingLog},
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

// v3AddMetadata creates the metadata key/value table with a unique index on key.
func v3AddMetadata(db *gorm.DB) error {
	if err := db.AutoMigrate(&metadata.Metadata{}); err != nil {
		return fmt.Errorf("v3: AutoMigrate metadata failed: %w", err)
	}
	return nil
}

// v4AddAccountNotes creates the account_notes table for per-account user notes.
func v4AddAccountNotes(db *gorm.DB) error {
	if err := db.AutoMigrate(&account_note.AccountNote{}); err != nil {
		return fmt.Errorf("v4: AutoMigrate account_notes failed: %w", err)
	}
	return nil
}

// v5AddImportPresets creates the import_presets table for reusable import mappings.
func v5AddImportPresets(db *gorm.DB) error {
	if err := db.AutoMigrate(&import_preset.Preset{}); err != nil {
		return fmt.Errorf("v5: AutoMigrate import_presets failed: %w", err)
	}
	return nil
}

// v6AddAccountReconciliation creates the account_reconciliation table for per-account reconciliation metadata.
func v6AddAccountReconciliation(db *gorm.DB) error {
	if err := db.AutoMigrate(&account_reconciliation.AccountReconciliation{}); err != nil {
		return fmt.Errorf("v6: AutoMigrate account_reconciliation failed: %w", err)
	}
	return nil
}

// v7AddAccountBalances creates the account_balances materialized-summary table that
// stores pre-computed per-(account, commodity) balance totals refreshed on every sync.
func v7AddAccountBalances(db *gorm.DB) error {
	if err := db.AutoMigrate(&account_balance.AccountBalance{}); err != nil {
		return fmt.Errorf("v7: AutoMigrate account_balances failed: %w", err)
	}
	return nil
}

// v8AddParserTrainingLog creates parser_training_log table for NL parser learning data.
func v8AddParserTrainingLog(db *gorm.DB) error {
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS parser_training_log (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		created_at DATETIME,
		input_text TEXT,
		predicted_date DATETIME,
		predicted_amount TEXT,
		predicted_currency TEXT,
		predicted_payee TEXT,
		predicted_from_account TEXT,
		predicted_to_account TEXT,
		predicted_direction TEXT,
		confidence_date REAL,
		confidence_amount REAL,
		confidence_currency REAL,
		confidence_payee REAL,
		confidence_from_account REAL,
		confidence_to_account REAL,
		confidence_direction REAL,
		confidence_overall REAL,
		actual_date DATETIME,
		actual_amount TEXT,
		actual_currency TEXT,
		actual_payee TEXT,
		actual_from_account TEXT,
		actual_to_account TEXT,
		actual_direction TEXT,
		user_corrected NUMERIC,
		correction_notes TEXT,
		suggestions_shown INTEGER,
		suggestion_used INTEGER,
		time_to_confirm INTEGER
	)`).Error; err != nil {
		return fmt.Errorf("v8: create parser_training_log table failed: %w", err)
	}

	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_parser_training_log_created_at ON parser_training_log(created_at)`).Error; err != nil {
		return fmt.Errorf("v8: create idx_parser_training_log_created_at failed: %w", err)
	}
	if err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_parser_training_log_user_corrected ON parser_training_log(user_corrected)`).Error; err != nil {
		return fmt.Errorf("v8: create idx_parser_training_log_user_corrected failed: %w", err)
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
