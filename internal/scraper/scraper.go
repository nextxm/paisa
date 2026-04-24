package scraper

import (
	"github.com/nextxm/paisa/internal/model/price"
	"github.com/nextxm/paisa/internal/scraper/metal"
	"github.com/nextxm/paisa/internal/scraper/mutualfund"
	"github.com/nextxm/paisa/internal/scraper/nps"
	"github.com/nextxm/paisa/internal/scraper/stock"
	log "github.com/sirupsen/logrus"
)

func GetAllProviders() []price.PriceProvider {
	return []price.PriceProvider{
		&stock.YahooPriceProvider{},
		&mutualfund.PriceProvider{},
		&stock.AlphaVantagePriceProvider{},
		&nps.PriceProvider{},
		&metal.PriceProvider{},
	}

}

func GetProviderByCode(code string) price.PriceProvider {
	switch code {
	case "in-mfapi":
		return &mutualfund.PriceProvider{}
	case "com-purifiedbytes-nps":
		return &nps.PriceProvider{}
	case "com-purifiedbytes-metal":
		return &metal.PriceProvider{}
	case "com-yahoo":
		return &stock.YahooPriceProvider{}
	case "co-alphavantage":
		return &stock.AlphaVantagePriceProvider{}
	}
	log.Fatal("Unknown price provider: ", code)
	return nil
}
