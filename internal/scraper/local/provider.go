// Package local implements a PriceProvider that reads price data from a
// user-supplied JSON file on the local filesystem.  This lets users maintain
// custom commodity prices without writing a full remote provider.
//
// The provider code is "local-json".  The commodity code field in paisa.yaml
// is the path to the JSON file (absolute or relative to the config directory).
//
// JSON format:
//
//	{
//	  "version": 1,
//	  "commodity": "MYFUND",
//	  "currency": "INR",
//	  "entries": [
//	    { "date": "2024-01-01", "value": "123.45" },
//	    { "date": "2024-02-01", "value": "124.00" }
//	  ]
//	}
//
// The top-level "commodity" and "currency" fields are optional defaults; they
// are overridden per-entry when an entry specifies its own "commodity" or
// "currency" field.  The commodity name passed by the caller at GetPrices time
// is used when neither the entry nor the top-level object supplies a commodity.
package local

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/shopspring/decimal"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Compile-time check: PriceProvider must satisfy price.PriceProvider.
var _ price.PriceProvider = (*PriceProvider)(nil)

// PriceProvider reads prices from a local JSON file.
type PriceProvider struct{}

func (p *PriceProvider) Code() string {
	return "local-json"
}

func (p *PriceProvider) Label() string {
	return "Local JSON File"
}

func (p *PriceProvider) Description() string {
	return "Reads price history from a JSON file on the local filesystem. " +
		"Useful for commodities whose prices are maintained manually or exported " +
		"from a custom source. The code field must be the path to the JSON file " +
		"(absolute, or relative to the config directory)."
}

func (p *PriceProvider) RateLimit() price.ProviderRateLimit {
	return price.ProviderRateLimit{MaxConcurrentRequests: 4}
}

func (p *PriceProvider) AutoCompleteFields() []price.AutoCompleteField {
	return []price.AutoCompleteField{
		{
			Label:     "File Path",
			ID:        "path",
			Help:      "Path to the JSON file (absolute or relative to config directory).",
			InputType: "text",
		},
	}
}

func (p *PriceProvider) AutoComplete(_ *gorm.DB, _ string, _ map[string]string) []price.AutoCompleteItem {
	return []price.AutoCompleteItem{}
}

func (p *PriceProvider) ClearCache(_ *gorm.DB) {}

// GetPrices reads the JSON file identified by code (a file path) and returns
// price entries found in it, filtered to those on or after since (start-of-day
// UTC).  When since is zero all entries are returned.  commodityName is used
// as a fallback when the file itself does not specify a commodity name.
func (p *PriceProvider) GetPrices(code string, commodityName string, since time.Time) ([]*price.Price, error) {
	path := resolveFilePath(code)
	log.Infof("Loading local JSON prices from %s", path)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("local-json: cannot read file %q: %w", path, err)
	}

	prices, err := parseLocalPrices(data, commodityName)
	if err != nil {
		return nil, err
	}
	return price.FilterSince(prices, since), nil
}

func (p *PriceProvider) GetPricesBatch(codes []string, commodityNames []string) (map[string][]*price.Price, error) {
	return price.GetPricesBatchSequentially(p, codes, commodityNames)
}

// resolveFilePath makes path absolute relative to the config directory when it
// is not already absolute.
func resolveFilePath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	configDir := config.GetConfigDir()
	if configDir == "" {
		return path
	}
	return filepath.Join(configDir, path)
}

// localPriceFile is the top-level structure of the JSON file.
type localPriceFile struct {
	// Version is reserved for future format changes; currently must be 1.
	Version int `json:"version"`
	// Commodity is the default commodity name for entries that do not specify one.
	Commodity string `json:"commodity"`
	// Currency is the default quote currency for entries that do not specify one.
	Currency string `json:"currency"`
	// Entries holds the individual price data points.
	Entries []localPriceEntry `json:"entries"`
}

// localPriceEntry is a single price data point inside the file.
type localPriceEntry struct {
	// Date in YYYY-MM-DD format.
	Date string `json:"date"`
	// Value is the price as a decimal string (e.g. "123.45").
	Value string `json:"value"`
	// Commodity overrides the file-level commodity for this entry.
	Commodity string `json:"commodity"`
	// Currency overrides the file-level currency for this entry.
	Currency string `json:"currency"`
}

// parseLocalPrices is the pure parsing function, exported for testing.
func parseLocalPrices(data []byte, fallbackCommodity string) ([]*price.Price, error) {
	var f localPriceFile
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("local-json: invalid JSON: %w", err)
	}

	if f.Version != 0 && f.Version != 1 {
		return nil, fmt.Errorf("local-json: unsupported format version %d (only version 1 is supported)", f.Version)
	}

	if len(f.Entries) == 0 {
		return []*price.Price{}, nil
	}

	defaultCurrency := f.Currency
	if defaultCurrency == "" {
		defaultCurrency = config.DefaultCurrency()
		if defaultCurrency == "" {
			defaultCurrency = "INR"
		}
	}

	defaultCommodity := f.Commodity
	if defaultCommodity == "" {
		defaultCommodity = fallbackCommodity
	}

	var prices []*price.Price
	for i, entry := range f.Entries {
		date, err := time.ParseInLocation("2006-01-02", entry.Date, config.TimeZone())
		if err != nil {
			return nil, fmt.Errorf("local-json: entry %d: invalid date %q: %w", i, entry.Date, err)
		}

		val, err := decimal.NewFromString(entry.Value)
		if err != nil {
			return nil, fmt.Errorf("local-json: entry %d: invalid value %q: %w", i, entry.Value, err)
		}

		commodity := entry.Commodity
		if commodity == "" {
			commodity = defaultCommodity
		}

		currency := entry.Currency
		if currency == "" {
			currency = defaultCurrency
		}

		prices = append(prices, &price.Price{
			Date:           date,
			CommodityType:  config.Unknown,
			CommodityID:    commodity,
			CommodityName:  commodity,
			Value:          val,
			QuoteCommodity: currency,
		})
	}

	return prices, nil
}
