---
description: "RFC for multi-currency commodity price support, including journal ingestion and export"
---

# RFC: Multi-Currency Commodity Prices

Status: Draft  
Date: 2026-03-18  
Owner: Paisa Core Team

## Summary

Add first-class multi-currency commodity pricing across:

1. Journal ingestion (ledger, hledger, beancount).
2. External provider sync.
3. Valuation and analytics.
4. Price export back to ledger-family formats.

The current system stores one value per commodity and effectively treats it as quoted in default currency. This RFC introduces explicit price pairs and quote currency support while preserving backward compatibility.

## Goals

1. Ingest and persist commodity prices as base and quote currency pairs.
2. Support valuation in default currency and arbitrary report currency.
3. Preserve round-trip fidelity: DB -> export file -> sync.
4. Keep existing users functional without immediate config changes.

## Non-Goals

1. Real-time streaming FX pricing.
2. Intraday tick-level pricing.
3. Automated provider arbitration beyond deterministic precedence rules.

## Current Behavior and Gaps

1. Journal price parsing filters to default-currency targets for most directives.
2. Provider prices are stored without explicit quote currency.
3. Price lookup is keyed by commodity only, not pair.
4. Currency helper behavior assumes a single default currency.

This leads to loss of non-default quote information and prevents native multi-currency valuation.

## Proposed Data Model

Add quote currency and source metadata to the price model.

Proposed table shape:

1. id
2. date
3. base_commodity
4. quote_commodity
5. rate
6. source_type (journal | provider)
7. source_provider (nullable)
8. source_ref (nullable, provider code/ticker/scheme)
9. created_at
10. updated_at

Proposed uniqueness:

1. unique(date, base_commodity, quote_commodity, source_type, source_provider, source_ref)

Recommended indexes:

1. (base_commodity, quote_commodity, date)
2. (quote_commodity, base_commodity, date)
3. (source_type, source_provider)

Backward compatibility strategy:

1. Existing rows without quote info are migrated with quote_commodity = default currency at migration time.

## Ingestion Design

### Journal

1. Parse and keep all price directives (not only default-currency targets).
2. Store directed pair base -> quote with rate.
3. If parser receives the inverse pattern (for example default currency as commodity with foreign target), normalize into canonical base -> quote direction when deterministic.
4. Preserve file date semantics and timezone behavior as today.

### Providers

1. Extend provider contract to return explicit quote_commodity.
2. For providers already converting to default currency (for example some stock providers), either:
   - keep conversion behavior but set quote_commodity explicitly, or
   - store native quote + FX chain as separate pairs if available.
3. For providers that return NAV in known local currency, set quote_commodity explicitly.

## Rate Resolution and Valuation

Introduce a pair-aware resolver:

1. GetRate(base, quote, date) -> rate record.
2. Resolution order:
   - direct pair on or before date,
   - inverse pair on or before date,
   - optional one-hop cross via configured anchor currencies.
3. Deterministic precedence when multiple sources exist on same date:
   - journal overrides provider by default,
   - tie-break by latest updated_at.

Valuation API behavior:

1. Existing valuation endpoints continue to use default currency unless explicitly requested.
2. New optional query parameter report_currency enables alternate report currency.

## Export Design

Add export that writes DB prices to selected dialect format.

### Inputs

1. dialect: ledger | hledger | beancount
2. from_date (optional)
3. to_date (optional)
4. base_commodities (optional list)
5. quote_commodities (optional list)
6. source filter (optional)

### Output format

1. ledger: P YYYY/MM/DD HH:MM:SS BASE RATE QUOTE
2. hledger: P YYYY-MM-DD BASE RATE QUOTE
3. beancount: YYYY-MM-DD price BASE RATE QUOTE

Notes:

1. Keep deterministic sort order by date, base, quote, source.
2. Optionally emit source comments where dialect supports comments.

## API and CLI Proposal

### API

1. GET /api/price
   - add optional filters: base, quote, from, to, report_currency, source
2. GET /api/price/export
   - returns text output in requested dialect
3. POST /api/price/export
   - supports larger filter payloads and future options

### CLI

1. paisa prices export --dialect ledger --from 2020-01-01 --to 2026-12-31 --base NIFTY --quote INR
2. paisa prices export --dialect beancount --source journal > prices.beancount

## Migration Plan

### Phase 1: Schema and dual-write

1. Add new columns/table and migration.
2. Read old model, write both old and new on sync.
3. Add validation logs for pair completeness.

### Phase 2: Pair-aware reads

1. Switch service price resolution to pair-aware engine.
2. Keep old read path behind fallback flag.

### Phase 3: API and UI

1. Add pair filters and report currency selector.
2. Update price pages/charts for base/quote awareness.

### Phase 4: Export and cleanup

1. Ship dialect export endpoint and CLI command.
2. Remove old single-commodity assumptions after compatibility window.

## Config Additions (Optional)

1. price_anchor_currencies: [USD, EUR]
2. price_source_precedence:
   - journal
   - provider
3. allow_cross_rate: true

Defaults should preserve current behavior for users who do not opt in.

## Testing Plan

1. Parser tests:
   - ledger/hledger/beancount directives with mixed quote currencies.
2. Migration tests:
   - old DB upgrades and valuation parity in default currency mode.
3. Resolver tests:
   - direct, inverse, and cross-rate path resolution.
   - precedence tie-breaking across journal and provider rows.
4. Export tests:
   - deterministic output snapshots for all dialects.
5. Regression tests:
   - existing snapshots remain valid in default settings.

## Rollout and Risk Management

1. Feature flag: enable_multi_currency_prices.
2. Compatibility mode defaulted on for one release cycle.
3. Telemetry/log counters for missing pairs and failed conversions.
4. Clear fallback path to old resolver if severe issues are detected.

Primary risks:

1. Ambiguous provider quote currency for some feeds.
2. Performance regression from pair-aware lookups.
3. Behavioral changes in historical valuation due to richer data.

Mitigations:

1. Require quote currency from providers at ingestion boundary.
2. Add indexed query plan checks and cache pair trees.
3. Keep source precedence explicit and documented.

## Definition of Done

1. Multi-currency journal prices are persisted without dropping non-default quotes.
2. Provider prices include explicit quote currency.
3. Valuation works for default currency and requested report currency.
4. Export works for ledger, hledger, and beancount formats.
5. Existing users can upgrade without manual data edits.
6. Regression suite passes with compatibility mode enabled and disabled.

## Decisions Locked

1. Source precedence: journal overrides provider.
2. Cross-rate resolution: one hop only.
3. Export default: full history.

## GitHub Issue Action Plan

The items below are ready to be created as GitHub issues labeled `enhancement`.

### EPIC 1: Pair-Aware Price Schema and Migration

Suggested title: `enhancement: add base/quote price schema with backward-compatible migration`

Scope:

1. Add pair-aware schema for prices with base commodity, quote commodity, rate, and source metadata.
2. Add indexes and uniqueness constraints for deterministic lookups.
3. Add migration from existing rows to `quote_commodity = default_currency`.
4. Keep compatibility mode for existing single-currency behavior.

Acceptance criteria:

1. Existing databases migrate successfully without data loss.
2. Existing valuations in default currency remain unchanged in compatibility mode.
3. New pair-aware rows can be written and queried by date/base/quote.

Dependencies:

1. None.

Estimate:

1. 3-5 engineering days.

### EPIC 2: Journal Ingestion for Full Price Pairs

Suggested title: `enhancement: ingest full multi-currency journal price directives`

Scope:

1. Update ledger/hledger/beancount price parsers to retain non-default quote pairs.
2. Normalize parsed prices into canonical base -> quote direction.
3. Persist journal prices with source metadata (`source_type = journal`).
4. Add parser fixtures for mixed quote currencies.

Acceptance criteria:

1. Non-default quote prices are no longer dropped during sync.
2. Same input journal produces deterministic base/quote output rows.
3. Existing parser behavior for default-currency-only journals remains stable.

Dependencies:

1. EPIC 1.

Estimate:

1. 3-4 engineering days.

### EPIC 3: Provider Contract Upgrade with Explicit Quote Currency

Suggested title: `enhancement: require explicit quote currency in price providers`

Scope:

1. Extend provider contract to return quote commodity explicitly.
2. Update all built-in providers to populate quote commodity.
3. Persist provider prices with source metadata (`source_type = provider`).
4. Preserve current conversion behavior where already implemented.

Acceptance criteria:

1. Every provider row includes explicit quote commodity.
2. Provider sync succeeds across all current provider implementations.
3. Missing quote metadata fails fast with actionable errors.

Dependencies:

1. EPIC 1.

Estimate:

1. 3-5 engineering days.

### EPIC 4: Pair-Aware Resolver with One-Hop Cross Rate

Suggested title: `enhancement: implement pair-aware rate resolver with one-hop cross rates`

Scope:

1. Add `GetRate(base, quote, date)` style resolver.
2. Resolve in this order: direct pair, inverse pair, one-hop cross via anchors.
3. Enforce source precedence rule: journal overrides provider.
4. Add caching/index-aware lookup path for performance.

Acceptance criteria:

1. Resolver returns deterministic rates for direct, inverse, and one-hop cases.
2. When journal and provider both exist, journal value is selected.
3. Performance is within acceptable bounds for current dataset sizes.

Dependencies:

1. EPIC 1.
2. EPIC 2.
3. EPIC 3.

Estimate:

1. 4-6 engineering days.

### EPIC 5: API Enhancements for Pair Filtering and Report Currency

Suggested title: `enhancement: extend price APIs for base/quote filters and report currency`

Scope:

1. Extend `/api/price` with base/quote/date/source/report-currency filters.
2. Keep old API usage backward compatible where possible.
3. Add consistent error envelopes for invalid combinations.
4. Update API docs with request/response examples.

Acceptance criteria:

1. API supports pair-aware queries and returns deterministic ordering.
2. Existing clients continue to work in compatibility mode.
3. Invalid requests return standardized API errors.

Dependencies:

1. EPIC 4.

Estimate:

1. 2-3 engineering days.

### EPIC 6: Full-History Export for Ledger, Hledger, and Beancount

Suggested title: `enhancement: export full price history from DB to ledger-family formats`

Scope:

1. Add export endpoint and CLI command for `ledger`, `hledger`, `beancount`.
2. Export full history by default.
3. Support optional filters (date range, base, quote, source).
4. Ensure deterministic sort order for reproducible output.

Acceptance criteria:

1. Export output validates in selected target dialect.
2. Default export includes full available history.
3. Repeated export on unchanged data yields byte-for-byte stable output.

Dependencies:

1. EPIC 2.
2. EPIC 3.
3. EPIC 4.

Estimate:

1. 3-4 engineering days.

### EPIC 7: UI and UX for Multi-Currency Prices

Suggested title: `enhancement: add UI support for base/quote prices and report currency`

Scope:

1. Add base/quote selectors in relevant price screens.
2. Add report-currency selection for valuation views.
3. Show source metadata (journal/provider) where relevant.
4. Preserve existing UX defaults for current users.

Acceptance criteria:

1. Users can view and filter by base/quote pairs.
2. Users can switch report currency without breaking existing dashboards.
3. Default views remain compatible with old workflows.

Dependencies:

1. EPIC 5.

Estimate:

1. 4-6 engineering days.

### EPIC 8: Testing, Rollout, and Compatibility Hardening

Suggested title: `enhancement: complete test coverage and rollout plan for multi-currency pricing`

Scope:

1. Add parser, migration, resolver, API, and export regression tests.
2. Add feature flag rollout controls and fallback path.
3. Add release notes and migration guidance.
4. Add observability counters for missing pairs and conversion failures.

Acceptance criteria:

1. Regression suite passes with compatibility mode on and off.
2. Upgrade path is documented and validated on fixture datasets.
3. Rollback and fallback procedures are documented and tested.

Dependencies:

1. EPIC 1-7.

Estimate:

1. 3-5 engineering days.

## Recommended Issue Labels and Milestone

Labels:

1. `enhancement`
2. `backend`
3. `api`
4. `frontend`
5. `migration`
6. `testing`

Milestone suggestion:

1. `Multi-Currency Prices v1`
