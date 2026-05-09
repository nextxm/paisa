package model

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ananthakumaran/paisa/internal/config"
	"github.com/ananthakumaran/paisa/internal/ledger"
	"github.com/ananthakumaran/paisa/internal/model/account_balance"
	"github.com/ananthakumaran/paisa/internal/model/cii"
	"github.com/ananthakumaran/paisa/internal/model/commodity"
	"github.com/ananthakumaran/paisa/internal/model/metadata"
	"github.com/ananthakumaran/paisa/internal/model/portfolio"
	"github.com/ananthakumaran/paisa/internal/model/posting"
	"github.com/ananthakumaran/paisa/internal/model/price"
	"github.com/ananthakumaran/paisa/internal/scraper"
	"github.com/ananthakumaran/paisa/internal/scraper/india"
	"github.com/ananthakumaran/paisa/internal/scraper/mutualfund"
	"github.com/ananthakumaran/paisa/internal/utils"
	"github.com/samber/lo"
	log "github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// JournalHashKey is the metadata key used to persist the last-synced journal hash.
const JournalHashKey = "journal_hash"

// lastPriceSyncKey is the metadata key used to persist the last-synced price fetch time.
const LastPriceSyncKey = "last_price_sync"

// SyncResult holds per-stage outcomes and aggregate counts for a sync run.
// It is returned by SyncJournal so that callers can surface stage-level
// diagnostics to operators via logs or API responses.
type SyncResult struct {
	FailedStage  string `json:"failed_stage,omitempty"`
	Message      string `json:"message,omitempty"`
	PostingCount int    `json:"posting_count"`
	PriceCount   int    `json:"price_count"`
	// Skipped is true when the journal file hash matches the last-synced hash,
	// meaning no CLI parse or validation work was performed.
	Skipped bool `json:"skipped,omitempty"`
}

func SyncJournal(db *gorm.DB) (SyncResult, error) {
	journalPath := config.GetJournalPath()

	files, err := ledger.Cli().Files(journalPath)
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.files", "error": err}).
			Warn("Failed to list journal files; proceeding with root file only")
		files = []string{journalPath}
	}

	// Compute the SHA-256 hash of all journal files.  If hashing fails (e.g. a
	// file is missing) we log a warning and fall through to a full sync rather
	// than returning an error, so that the validate stage can surface the real
	// problem with a clearer message.
	currentHash, hashErr := utils.SHA256Files(files)
	if hashErr != nil {
		log.WithFields(log.Fields{"stage": "journal.hash", "error": hashErr}).
			Warn("Failed to compute journal hash; proceeding with full sync")
	}

	// When we have a valid current hash, compare it against the cached value.
	// A match means the file has not changed since the last successful sync, so
	// we can skip all expensive CLI work.
	if currentHash != "" {
		cachedHash, err := metadata.GetOrDefault(db, JournalHashKey, "")
		if err != nil {
			log.WithFields(log.Fields{"stage": "journal.hash", "error": err}).
				Warn("Failed to read cached journal hash; proceeding with full sync")
		} else if currentHash == cachedHash {
			log.WithFields(log.Fields{"stage": "journal.hash"}).
				Info("Journal unchanged (hash matches), skipping sync")
			return SyncResult{Skipped: true}, nil
		}
	}

	log.WithFields(log.Fields{"stage": "journal.validate"}).Info("Syncing transactions from journal")

	errors, _, err := ledger.Cli().ValidateFile(journalPath)
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

	prices, err := ledger.Cli().Prices(journalPath)
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.prices", "error": err}).Error("Journal price extraction failed")
		return SyncResult{FailedStage: "journal.prices", Message: err.Error()}, err
	}

	postings, err := ledger.Cli().Parse(journalPath, prices)
	if err != nil {
		log.WithFields(log.Fields{"stage": "journal.parse", "error": err}).Error("Journal parsing failed")
		return SyncResult{FailedStage: "journal.parse", Message: err.Error()}, err
	}

	err = db.Transaction(func(tx *gorm.DB) error {
		if err := price.UpsertAllByType(tx, config.Unknown, prices); err != nil {
			return err
		}
		if err := posting.UpsertAll(tx, postings); err != nil {
			return err
		}
		return account_balance.RefreshFromPostings(tx, postings)
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

	// Persist the journal hash only after a fully successful sync so that a
	// partial failure never causes a subsequent run to be silently skipped.
	if currentHash != "" {
		if err := metadata.Set(db, JournalHashKey, currentHash); err != nil {
			log.WithFields(log.Fields{"stage": "journal.hash", "error": err}).
				Warn("Failed to store journal hash; next sync will not be skipped")
		}
	}

	return result, nil
}

// SyncCommoditiesResult holds per-commodity outcomes for a price-scraper sync run.
type SyncCommoditiesResult struct {
	// Failures contains one human-readable message for each commodity whose
	// price could not be fetched or persisted.  An empty slice means all
	// commodities were synced successfully.
	Failures []string
}

const commodityFetchWorkers = 5

type commodityPriceFetchResult struct {
	commodity config.Commodity
	prices    []*price.Price
	err       error
}

type commodityBatchFetchJob struct {
	providerCode string
	commodities  []config.Commodity
}

type providerRateLimiter struct {
	mu          sync.Mutex
	nextAllowed time.Time
	minInterval time.Duration
}

func newProviderRateLimiter(minInterval time.Duration) *providerRateLimiter {
	return &providerRateLimiter{minInterval: minInterval}
}

func (l *providerRateLimiter) WaitTurn() {
	if l == nil || l.minInterval <= 0 {
		return
	}

	l.mu.Lock()
	wait := time.Until(l.nextAllowed)
	if wait < 0 {
		wait = 0
	}
	startAt := time.Now().Add(wait)
	l.nextAllowed = startAt.Add(l.minInterval)
	l.mu.Unlock()

	if wait > 0 {
		time.Sleep(wait)
	}
}

func normalizeProviderRateLimit(limit price.ProviderRateLimit, workers int) price.ProviderRateLimit {
	maxConcurrency := limit.MaxConcurrentRequests
	if maxConcurrency <= 0 {
		maxConcurrency = 1
	}
	if workers > 0 && maxConcurrency > workers {
		maxConcurrency = workers
	}
	limit.MaxConcurrentRequests = maxConcurrency
	return limit
}

func SyncCommodities(db *gorm.DB, forcePrices bool, progressFn func(completed, total int)) (SyncCommoditiesResult, error) {
	log.WithFields(log.Fields{"stage": "commodities"}).Info("Fetching commodities price history")
	return syncCommodities(db, lo.Shuffle(commodity.All()), scraper.GetProviderByCode, commodityFetchWorkers, forcePrices, progressFn)
}

func syncCommodities(db *gorm.DB, commodities []config.Commodity, getProviderByCode func(string) price.PriceProvider, workers int, forcePrices bool, progressFn func(completed, total int)) (SyncCommoditiesResult, error) {
	if workers <= 0 {
		workers = 1
	}
	// Determine the since timestamp from the last successful price sync.
	// A zero value means fetch the full history (first run or metadata missing).
	var since time.Time
	if forcePrices {
		log.WithFields(log.Fields{"stage": "commodities"}).Info("Force refresh requested; fetching full price history")
	} else if lastSyncStr, err := metadata.GetOrDefault(db, LastPriceSyncKey, ""); err == nil && lastSyncStr != "" {
		if t, parseErr := time.Parse(time.RFC3339, lastSyncStr); parseErr == nil {
			since = t
			log.WithFields(log.Fields{"stage": "commodities", "since": since.Format(time.RFC3339)}).Info("Performing incremental price sync")
		} else {
			log.WithFields(log.Fields{"stage": "commodities", "value": lastSyncStr, "error": parseErr}).
				Warn("Failed to parse last_price_sync metadata; falling back to full price history fetch")
			log.WithFields(log.Fields{"stage": "commodities"}).Info("No previous price sync found; fetching full price history")
		}
	} else {
		log.WithFields(log.Fields{"stage": "commodities"}).Info("No previous price sync found; fetching full price history")
	}

	var result SyncCommoditiesResult
	var errs []error
	batchJobs := groupCommodityBatchJobs(commodities)
	results := make(chan commodityPriceFetchResult, len(commodities))
	var wg sync.WaitGroup
	jobsByProvider := make(map[string][]commodityBatchFetchJob, len(batchJobs))
	for _, job := range batchJobs {
		jobsByProvider[job.providerCode] = append(jobsByProvider[job.providerCode], job)
	}

	for providerCode, providerJobs := range jobsByProvider {
		provider := getProviderByCode(providerCode)
		limit := normalizeProviderRateLimit(provider.RateLimit(), workers)
		limiter := newProviderRateLimiter(limit.MinIntervalBetweenRequests)

		jobs := make(chan commodityBatchFetchJob)
		providerWorkers := limit.MaxConcurrentRequests
		if providerWorkers > len(providerJobs) {
			providerWorkers = len(providerJobs)
		}

		for i := 0; i < providerWorkers; i++ {
			wg.Add(1)
			go func(providerCode string, provider price.PriceProvider, limiter *providerRateLimiter, limit price.ProviderRateLimit) {
				defer wg.Done()
				for job := range jobs {
					// For providers with strict pacing, fetch per commodity so the
					// throttle applies to every outbound request instead of one large
					// provider-local loop inside GetPricesBatchSequentially.
					if limit.MinIntervalBetweenRequests > 0 && len(job.commodities) > 1 {
						for _, commodity := range job.commodities {
							limiter.WaitTurn()
							name := commodity.Name
							log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "provider": providerCode}).
								Info("Fetching commodity (throttled)")
							prices, err := provider.GetPrices(commodity.Price.Code, name, since)
							results <- commodityPriceFetchResult{
								commodity: commodity,
								prices:    prices,
								err:       err,
							}
						}
						continue
					}

					if len(job.commodities) == 1 {
						commodity := job.commodities[0]
						name := commodity.Name
						limiter.WaitTurn()
						log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "provider": providerCode}).
							Info("Fetching commodity")
						prices, err := provider.GetPrices(commodity.Price.Code, name, since)
						results <- commodityPriceFetchResult{
							commodity: commodity,
							prices:    prices,
							err:       err,
						}
						continue
					}

					codes := make([]string, len(job.commodities))
					names := make([]string, len(job.commodities))
					for i, commodity := range job.commodities {
						codes[i] = commodity.Price.Code
						names[i] = commodity.Name
					}

					limiter.WaitTurn()
					log.WithFields(log.Fields{"stage": "commodities", "provider": providerCode, "count": len(job.commodities)}).
						Info("Fetching commodity batch")
					pricesByCode, err := provider.GetPricesBatch(codes, names)
					if err != nil {
						for _, commodity := range job.commodities {
							results <- commodityPriceFetchResult{
								commodity: commodity,
								err:       err,
							}
						}
						continue
					}
					for _, commodity := range job.commodities {
						results <- commodityPriceFetchResult{
							commodity: commodity,
							prices:    price.FilterSince(pricesByCode[commodity.Price.Code], since),
						}
					}
				}
			}(providerCode, provider, limiter, limit)
		}

		for _, providerJob := range providerJobs {
			jobs <- providerJob
		}
		close(jobs)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	total := len(commodities)
	itemsCompleted := 0
	for fetched := range results {
		itemsCompleted++
		if progressFn != nil {
			progressFn(itemsCompleted, total)
		}
		commodity := fetched.commodity
		name := commodity.Name
		prices := fetched.prices
		err := fetched.err

		if err != nil {
			log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "error": err}).Error("Failed to fetch commodity prices")
			msg := fmt.Sprintf("Failed to fetch price for %s: %s", name, err.Error())
			errs = append(errs, fmt.Errorf("%s", msg))
			result.Failures = append(result.Failures, msg)
			continue
		}

		// Validate that every returned price carries an explicit quote commodity.
		// A missing quote_commodity means the provider contract was not fulfilled,
		// so we fail fast with an actionable error rather than silently backfilling.
		missingQuote := false
		for _, p := range prices {
			if p.QuoteCommodity == "" {
				log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "provider": commodity.Price.Provider, "date": p.Date.Format("2006-01-02")}).
					Error("Provider returned price without quote_commodity")
				msg := fmt.Sprintf("provider %s returned price for %s on %s without quote_commodity: update the provider or set the quote currency explicitly", commodity.Price.Provider, name, p.Date.Format("2006-01-02"))
				errs = append(errs, fmt.Errorf("%s", msg))
				result.Failures = append(result.Failures, msg)
				missingQuote = true
				break
			}
		}
		if missingQuote {
			continue
		}

		// Stamp source metadata before persisting so every provider row is
		// identifiable as originating from a price provider (not the journal).
		for i := range prices {
			if prices[i].Source == "" {
				prices[i].Source = "provider"
			}
		}

		if err := price.UpsertAllByTypeNameAndID(db, commodity.Type, name, commodity.Price.Code, prices); err != nil {
			log.WithFields(log.Fields{"stage": "commodities", "commodity": name, "error": err}).Error("Failed to save commodity prices")
			msg := fmt.Sprintf("Failed to save price for %s: %s", name, err.Error())
			errs = append(errs, fmt.Errorf("%s", msg))
			result.Failures = append(result.Failures, msg)
		}
	}

	if len(errs) > 0 {
		var message string
		for _, e := range errs {
			message += e.Error() + "\n"
		}
		// Even with errors, we might have successfully updated some commodities,
		// so we record the sync attempt time.
		_ = metadata.Set(db, LastPriceSyncKey, time.Now().Format(time.RFC3339))
		return result, fmt.Errorf("%s", strings.Trim(message, "\n"))
	}
	_ = metadata.Set(db, LastPriceSyncKey, time.Now().Format(time.RFC3339))
	return result, nil
}

func groupCommodityBatchJobs(commodities []config.Commodity) []commodityBatchFetchJob {
	batches := make([]commodityBatchFetchJob, 0)
	indexByProvider := make(map[string]int, len(commodities))

	for _, commodity := range commodities {
		providerCode := commodity.Price.Provider
		if index, ok := indexByProvider[providerCode]; ok {
			batches[index].commodities = append(batches[index].commodities, commodity)
			continue
		}

		indexByProvider[providerCode] = len(batches)
		batches = append(batches, commodityBatchFetchJob{
			providerCode: providerCode,
			commodities:  []config.Commodity{commodity},
		})
	}

	return batches
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
