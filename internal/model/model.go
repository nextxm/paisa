package model

import (
	"fmt"
	"strings"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	"github.com/ananthakumaran/paisa/internal/model/commodity"
	"github.com/ananthakumaran/paisa/internal/model/portfolio"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper"
	"github.com/ananthakumaran/paisa/internal/scraper/india"
	"github.com/ananthakumaran/paisa/internal/scraper/mutualfund"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// SyncResult holds per-stage outcomes and aggregate counts for a sync run.
// It is returned by SyncJournal so that callers can surface stage-level
// diagnostics to operators via logs or API responses.
type SyncResult struct {
	FailedStage  string `json:"failed_stage,omitempty"`
	Message      string `json:"message,omitempty"`
	PostingCount int    `json:"posting_count"`
	PriceCount   int    `json:"price_count"`
}

func SyncJournal(db *gorm.DB) (SyncResult, error) {
	log.WithFields(log.Fields{"stage": "journal.validate"}).Info("Syncing transactions from journal")

	errors, _, err := ledger.Cli().ValidateFile(config.GetJournalPath())
	if err != nil {
		var message string
		if len(errors) == 0 {
			message = err.Error()
		} else {
			for _, e := range errors {
				message += e.Message + "\n\n"
			}
			message = strings.TrimRight(message, "\n")
		}
		log.WithFields(log.Fields{"stage": "journal.validate", "error": message}).Error("Journal validation failed")
		return SyncResult{FailedStage: "journal.validate", Message: message}, err
	}

	prices, err := ledger.Cli().Prices(config.GetJournalPath())
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.prices", "error": err}).Error("Journal price extraction failed")
		return SyncResult{FailedStage: "journal.prices", Message: err.Error()}, err
	}

	postings, err := ledger.Cli().Parse(config.GetJournalPath(), prices)
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.parse", "error": err}).Error("Journal parsing failed")
		return SyncResult{FailedStage: "journal.parse", Message: err.Error()}, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := price.UpsertAllByType(tx, config.Unknown, prices); err != nil {
			return err
		}
		return posting.UpsertAll(tx, postings)
	})
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.db_write", "error": err}).Error("Journal database write failed")
		return SyncResult{FailedStage: "journal.db_write", Message: err.Error()}, err
	}

	result := SyncResult{
		PostingCount: len(postings),
		PriceCount:   len(prices),
	}
	log.WithFields(log.Fields{
		"stage":         "journal",
		"posting_count": result.PostingCount,
		"price_count":   result.PriceCount,
	}).Info("Journal sync completed")
	return result, nil
}

func SyncCommodities(db *gorm.DB) error {
	log.WithFields(log.Fields{"stage": "commodities"}).Info("Fetching commodities price history")
	commodities := lo.Shuffle(commodity.All())

	var errors []error
	for _, commodity := range commodities {
		name := commodity.Name
		log.WithFields(log.Fields{"stage": "commodities", "commodity": name}).Info("Fetching commodity")
		code := commodity.Price.Code
		var prices []*price.Price
		var err error

		provider := scraper.GetProviderByCode(commodity.Price.Provider)
		prices, err = provider.GetPrices(code, name)

		if err != nil {
			log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "error": err}).Error("Failed to fetch commodity prices")
			errors = append(errors, fmt.Errorf("Failed to fetch price for %s: %w", name, err))
			continue
		}

		if err := price.UpsertAllByTypeNameAndID(db, commodity.Type, name, code, prices); err != nil {
			log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "error": err}).Error("Failed to save commodity prices")
			errors = append(errors, fmt.Errorf("Failed to save price for %s: %w", name, err))
		}
	}

	if len(errors) > 0 {
		var message string
		for _, error := range errors {
			message += error.Error() + "\n"
		}
		return fmt.Errorf("%s", strings.Trim(message, "\n"))
	}
	return nil
}

func SyncCII(db *gorm.DB) error {
	log.WithFields(log.Fields{"stage": "cii"}).Info("Fetching taxation related info")
	ciis, err := india.GetCostInflationIndex()
	if err != nil {
		log.WithFields(log.Fields{"stage": "cii", "error": err}).Error("Failed to fetch CII")
		return fmt.Errorf("Failed to fetch CII: %w", err)
	}
	if err := cii.UpsertAll(db, ciis); err != nil {
		return fmt.Errorf("Failed to save CII: %w", err)
	}
	return nil
}

func SyncPortfolios(db *gorm.DB) error {
	log.WithFields(log.Fields{"stage": "portfolios"}).Info("Fetching commodities portfolio")
	commodities := commodity.FindByType(config.MutualFund)

	return db.Transaction(func(tx *gorm.DB) error {
		for _, commodity := range commodities {
			if commodity.Price.Provider != "in-mfapi" {
				continue
			}

			name := commodity.Name
			log.WithFields(log.Fields{"stage": "portfolios", "commodity": name}).Info("Fetching portfolio")
			portfolios, err := mutualfund.GetPortfolio(commodity.Price.Code, commodity.Name)

			if err != nil {
				log.WithFields(log.Fields{"stage": "portfolios", "commodity": name, "error": err}).Error("Failed to fetch portfolio")
				return fmt.Errorf("Failed to fetch portfolio for %s: %w", name, err)
			}

			if err := portfolio.UpsertAll(tx, commodity.Type, commodity.Price.Code, portfolios); err != nil {
				return fmt.Errorf("Failed to save portfolio for %s: %w", name, err)
			}
		}
		return nil
	})
}
