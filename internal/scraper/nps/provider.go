package nps

import (
	"time"

	"github.com/ananthakumaran/paisa/internal/model/nps/scheme"
	"github.com/ananthakumaran/paisa/internal/model/price"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Compile-time check: PriceProvider must satisfy price.PriceProvider.
var _ price.PriceProvider = (*PriceProvider)(nil)

type PriceProvider struct {
}

func (p *PriceProvider) Code() string {
	return "com-purifiedbytes-nps"
}

func (p *PriceProvider) Label() string {
	return "Purified Bytes NPS India"
}

func (p *PriceProvider) Description() string {
	return "Supports all national pension scheme (nps) funds in India."
}

func (p *PriceProvider) RateLimit() price.ProviderRateLimit {
	return price.ProviderRateLimit{MaxConcurrentRequests: 1}
}

func (p *PriceProvider) AutoCompleteFields() []price.AutoCompleteField {
	return []price.AutoCompleteField{
		{Label: "PFM", ID: "pfm", Help: "Pension Fund Manager"},
		{Label: "Scheme Name", ID: "scheme"},
	}
}

func (p *PriceProvider) AutoComplete(db *gorm.DB, field string, filter map[string]string) []price.AutoCompleteItem {
	count := scheme.Count(db)
	if count == 0 {
		schemes, err := GetSchemes()
		if err != nil {
			log.Error("Failed to fetch NPS schemes: ", err)
			return []price.AutoCompleteItem{}
		}
		scheme.UpsertAll(db, schemes)
	} else {
		log.Info("Using cached results")
	}

	switch field {
	case "pfm":
		return scheme.GetPFMCompletions(db)
	case "scheme":
		return scheme.GetSchemeNameCompletions(db, filter["pfm"])
	}
	return []price.AutoCompleteItem{}
}

func (p *PriceProvider) ClearCache(db *gorm.DB) {
	db.Exec("DELETE FROM nps_schemes")
}

func (p *PriceProvider) GetPrices(code string, commodityName string, since time.Time) ([]*price.Price, error) {
	prices, err := GetNav(code, commodityName)
	if err != nil {
		return nil, err
	}
	return price.FilterSince(prices, since), nil
}

func (p *PriceProvider) GetPricesBatch(codes []string, commodityNames []string) (map[string][]*price.Price, error) {
	return price.GetPricesBatchSequentially(p, codes, commodityNames)
}
