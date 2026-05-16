package server

import (
	"errors"

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

	return db.Transaction(func(tx *gorm.DB) error {
		return projection_snapshot.Replace(
			tx,
			inputs.CurrentNetworth,
			inputs.MonthlyContribution,
			inputs.SavingsRate,
			inputs.AnnualExpenses,
		)
	})
}

func getProjectionBaseInputs(db *gorm.DB) projectionBaseInputs {
	snapshot, err := projection_snapshot.Get(db)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			log.WithError(err).Warn("Failed to load projection snapshot; falling back to live query path")
		}
		return computeProjectionBaseInputs(db)
	}
	if snapshot.SchemaVersion != projection_snapshot.SchemaVersion {
		log.WithField("schema_version", snapshot.SchemaVersion).
			Warn("Projection snapshot schema version mismatch; falling back to live query path")
		return computeProjectionBaseInputs(db)
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
