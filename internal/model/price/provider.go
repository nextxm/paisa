package price

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// ProviderRateLimit describes a provider's preferred request concurrency and
// pacing so sync orchestration can avoid overloading strict APIs.
type ProviderRateLimit struct {
	// MaxConcurrentRequests is the maximum number of in-flight requests that
	// should be made to this provider. Values <= 0 are treated as 1.
	MaxConcurrentRequests int
	// MinIntervalBetweenRequests is the minimum gap between request starts for a
	// single provider. A zero value means no enforced delay.
	MinIntervalBetweenRequests time.Duration
}

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

	// RateLimit describes how requests to this provider should be paced.
	RateLimit() ProviderRateLimit

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

	// GetPricesBatch fetches full price histories for multiple commodities from
	// the same provider in a single call when the provider supports batching.
	// The returned map is keyed by the commodity code supplied in codes.
	// Callers may apply additional filtering (for example incremental since
	// filtering) after the batch fetch completes.
	GetPricesBatch(codes []string, commodityNames []string) (map[string][]*Price, error)
}

// GetPricesBatchSequentially adapts single-code GetPrices implementations to
// the batched PriceProvider contract. Providers that do not support a true
// batch API can use this helper to preserve behaviour while sync orchestration
// groups commodities by provider.
func GetPricesBatchSequentially(provider interface {
	GetPrices(code string, commodityName string, since time.Time) ([]*Price, error)
}, codes []string, commodityNames []string) (map[string][]*Price, error) {
	if len(codes) != len(commodityNames) {
		return nil, ErrMismatchedBatchInputs
	}

	pricesByCode := make(map[string][]*Price, len(codes))
	for i, code := range codes {
		prices, err := provider.GetPrices(code, commodityNames[i], time.Time{})
		if err != nil {
			return nil, err
		}
		pricesByCode[code] = prices
	}

	return pricesByCode, nil
}

var ErrMismatchedBatchInputs = errors.New("price provider batch inputs must have matching lengths")
