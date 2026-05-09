package price

import (
	"time"

	"gorm.io/gorm"
)

// AutoCompleteItem is a single suggestion returned by a provider's AutoComplete method.
type AutoCompleteItem struct {
	Label string `json:"label"`
	ID    string `json:"id"`
}

// AutoCompleteField describes one interactive field shown in the commodity
// price-provider configuration form.  InputType mirrors the HTML input type;
// an empty value defaults to a searchable dropdown populated via AutoComplete.
type AutoCompleteField struct {
	Label     string `json:"label"`
	ID        string `json:"id"`
	Help      string `json:"help"`
	InputType string `json:"inputType"`
}

// PriceProvider is the stable contract that every price-data source must
// satisfy.  Implementations are registered in internal/scraper/scraper.go and
// selected at commodity-sync time via the provider code stored in paisa.yaml.
//
// Return-value semantics that all implementations must honour:
//
//   - GetPrices: return (nil, non-nil error) on any failure that should be
//     reported to the caller.  Return (empty-or-nil slice, nil) when the
//     provider is reachable but has no data for the requested code – this is
//     not an error.  Implementations must never call log.Fatal.
//
//   - AutoComplete: return an empty (non-nil) slice on any error or when no
//     matching suggestions exist; errors should be logged at the Error level
//     but not propagated (the method has no error return).
//
//   - ClearCache: clear any in-memory or database-level cache accumulated by
//     this provider.  It is called before every full sync.  It must be safe to
//     call when no cache exists.
type PriceProvider interface {
	// Code returns the stable, unique identifier for this provider (e.g.
	// "com-yahoo").  The value must never change once published; it is stored
	// in user configuration files.
	Code() string

	// Label returns a short human-readable name shown in the UI provider list.
	Label() string

	// Description returns a longer human-readable description of what this
	// provider supports, shown as help text in the UI.
	Description() string

	// AutoCompleteFields describes the configuration fields that the UI should
	// render when the user selects this provider.  The returned slice must not
	// be nil; return an empty slice when no fields are needed.
	AutoCompleteFields() []AutoCompleteField

	// AutoComplete returns autocomplete suggestions for field identified by
	// field using the current partial values in filter.  On error, log at
	// Error level and return an empty (non-nil) slice.
	AutoComplete(db *gorm.DB, field string, filter map[string]string) []AutoCompleteItem

	// ClearCache clears any cached data held by this provider.  Called before
	// every full sync so stale data is not reused across runs.
	ClearCache(db *gorm.DB)

	// GetPrices fetches price history for the commodity identified by code
	// (provider-specific format) with the display name commodityName.
	// When since is non-zero, implementations should return only prices on or
	// after since's start-of-day (UTC); a zero since means fetch the full
	// history.  Return (nil, err) on failure; return (empty-or-nil slice, nil)
	// when the provider is reachable but has no data for the requested code –
	// this is not an error.  Implementations must never call log.Fatal.
	GetPrices(code string, commodityName string, since time.Time) ([]*Price, error)
}
