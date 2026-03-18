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

func SyncJournal(db *gorm.DB) (string, error) {
	log.Info("Syncing transactions from journal")

	errors, _, err := ledger.Cli().ValidateFile(config.GetJournalPath())
	if err != nil {

		if len(errors) == 0 {
			return err.Error(), err
		}

		var message string
		for _, error := range errors {
			message += error.Message + "\n\n"
		}
		return strings.TrimRight(message, "\n"), err
	}

	prices, err := ledger.Cli().Prices(config.GetJournalPath())
	if err != nil {
		return err.Error(), err
	}

	postings, err := ledger.Cli().Parse(config.GetJournalPath(), prices)
	if err != nil {
		return err.Error(), err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := price.UpsertAllByType(tx, config.Unknown, prices); err != nil {
			return err
		}
		return posting.UpsertAll(tx, postings)
	})
	if err != nil {
		return err.Error(), err
	}

	return "", nil
}

func SyncCommodities(db *gorm.DB) error {
	log.Info("Fetching commodities price history")
	commodities := lo.Shuffle(commodity.All())

	var errors []error
	for _, commodity := range commodities {
		name := commodity.Name
		log.Info("Fetching commodity ", name)
		code := commodity.Price.Code
		var prices []*price.Price
		var err error

		provider := scraper.GetProviderByCode(commodity.Price.Provider)
		prices, err = provider.GetPrices(code, name)

		if err != nil {
			log.Error(err)
			errors = append(errors, fmt.Errorf("Failed to fetch price for %s: %w", name, err))
			continue
		}

		if err := price.UpsertAllByTypeNameAndID(db, commodity.Type, name, code, prices); err != nil {
			log.Error(err)
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
	log.Info("Fetching taxation related info")
	ciis, err := india.GetCostInflationIndex()
	if err != nil {
		log.Error(err)
		return fmt.Errorf("Failed to fetch CII: %w", err)
	}
	if err := cii.UpsertAll(db, ciis); err != nil {
		return fmt.Errorf("Failed to save CII: %w", err)
	}
	return nil
}

func SyncPortfolios(db *gorm.DB) error {
	log.Info("Fetching commodities portfolio")
	commodities := commodity.FindByType(config.MutualFund)

	return db.Transaction(func(tx *gorm.DB) error {
		for _, commodity := range commodities {
			if commodity.Price.Provider != "in-mfapi" {
				continue
			}

			name := commodity.Name
			log.Info("Fetching portfolio for ", name)
			portfolios, err := mutualfund.GetPortfolio(commodity.Price.Code, commodity.Name)

			if err != nil {
				log.Error(err)
				return fmt.Errorf("Failed to fetch portfolio for %s: %w", name, err)
			}

			if err := portfolio.UpsertAll(tx, commodity.Type, commodity.Price.Code, portfolios); err != nil {
				return fmt.Errorf("Failed to save portfolio for %s: %w", name, err)
			}
		}
		return nil
	})
}
