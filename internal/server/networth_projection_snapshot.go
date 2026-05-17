package server

import (
	"errors"

	"github.com/ananthakumaran/paisa/internal/model"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/projection_snapshot"
	"github.com/ananthakumaran/paisa/internal/query"
	"github.com/ananthakumaran/paisa/internal/service"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type projectionBaseInputs struct {
	CurrentNetworth decimal.Decimal
	historicalProjectionInputs
}

func computeProjectionBaseInputs(db *gorm.DB) projectionBaseInputs {
	postings := query.Init(db).Like("Assets:%", "Income:CapitalGains:%", "Liabilities:%").UntilToday().All()
	postings = service.PopulateMarketPrice(db, postings)

	return projectionBaseInputs{
		CurrentNetworth:            computeNetworth(db, postings).BalanceAmount,
		historicalProjectionInputs: deriveProjectionInputs(db),
	}
}

func RefreshNetworthProjectionSnapshot(db *gorm.DB) error {
	inputs := computeProjectionBaseInputs(db)
	journalHash, _ := metadata.GetOrDefault(db, model.JournalHashKey, "")
	lastPriceSync, _ := metadata.GetOrDefault(db, model.LastPriceSyncKey, "")

	return db.Transaction(func(tx *gorm.DB) error {
		return projection_snapshot.Replace(
			tx,
			inputs.CurrentNetworth,
			inputs.MonthlyContribution,
			inputs.SavingsRate,
			inputs.AnnualExpenses,
			journalHash,
			lastPriceSync,
		)
	})
}

func getProjectionBaseInputs(db *gorm.DB) projectionBaseInputs {
	snapshot, err := projection_snapshot.Get(db)

	// Fetch latest sync metadata to compare
	currentJournalHash, _ := metadata.GetOrDefault(db, model.JournalHashKey, "")
	currentLastPriceSync, _ := metadata.GetOrDefault(db, model.LastPriceSyncKey, "")

	// Helper to recalculate, persist and return
	recalculateAndSave := func() projectionBaseInputs {
		inputs := computeProjectionBaseInputs(db)
		err := db.Transaction(func(tx *gorm.DB) error {
			return projection_snapshot.Replace(
				tx,
				inputs.CurrentNetworth,
				inputs.MonthlyContribution,
				inputs.SavingsRate,
				inputs.AnnualExpenses,
				currentJournalHash,
				currentLastPriceSync,
			)
		})
		if err != nil {
			log.WithError(err).Warn("Failed to persist updated projection snapshot")
		}
		return inputs
	}

	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Warn("Failed to load projection snapshot; recalculating")
		}
		return recalculateAndSave()
	}

	if snapshot.SchemaVersion != projection_snapshot.SchemaVersion {
		log.WithField("schema_version", snapshot.SchemaVersion).
			Warn("Projection snapshot schema version mismatch; recalculating")
		return recalculateAndSave()
	}

	// Check if either of the sync timestamps / hashes have changed
	if snapshot.JournalHash != currentJournalHash || snapshot.LastPriceSync != currentLastPriceSync {
		log.Info("Sync state changed (journal or prices updated); recalculating projection snapshot")
		return recalculateAndSave()
	}

	return projectionBaseInputs{
		CurrentNetworth: snapshot.CurrentNetworth,
		historicalProjectionInputs: historicalProjectionInputs{
			MonthlyContribution: snapshot.MonthlyContribution,
			SavingsRate:         snapshot.SavingsRate,
			AnnualExpenses:      snapshot.AnnualExpenses,
		},
	}
}
