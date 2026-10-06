# Life Projection Engine — Task Tracker

## Phase 1: Financial DNA Extractor

- [x] Create `internal/projection/dna/dna.go` — FinancialProfile struct + ExtractProfile function
- [x] Implement income growth rate calculation (3-year YoY CAGR)
- [x] Implement expense growth rate calculation (3-year YoY CAGR)
- [x] Implement return volatility calculation (annualized from monthly price returns)
- [x] Implement portfolio-weighted historical return
- [x] Leverage existing savings rate + monthly contribution logic
- [ ] Create `internal/projection/dna/dna_test.go` — unit tests
- [x] Create `internal/server/projection_dna.go` — API handler
- [x] Register `GET /api/projection/dna` route in `server.go`
- [x] Add `FinancialProfile` interface to `src/lib/utils.ts`
- [x] Add DNA summary card to projection page
- [x] Run `gofmt` and `go test` on touched files
- [x] Run `prettier` on touched frontend files

## Phase 2: Monte Carlo Simulation Engine

- [x] Create `internal/projection/simulator/simulator.go`
- [x] Implement Monte Carlo loop with log-normal returns
- [x] Percentile band aggregation (P10/P25/P50/P75/P90)
- [x] FIRE probability calculation
- [ ] Create `internal/projection/simulator/simulator_test.go` with benchmark
- [x] Create `internal/server/projection_simulate.go` — API handler
- [x] Register `POST /api/projection/simulate` route
- [x] Add simulation result interfaces to `src/lib/utils.ts`
- [x] Create fan chart visualization (D3-based in life page)
- [x] Create `src/routes/(app)/planning/life/+page.svelte` — dashboard
- [x] Add "Life Plan" navigation entry to Navbar
- [ ] Verify performance: 1000 iterations × 360 months < 500ms

## Phase 3: Life Goals System

- [x] Add `LifeGoal` struct to `internal/config/config.go`
- [x] Update `internal/config/schema.json`
- [x] Create `internal/projection/goals/goals.go` — goal→cashflow expansion
- [x] Support milestone goals (one-time at target_date)
- [x] Support recurring goals (start/end month-year, frequency)
- [x] Global inflation with per-goal override
- [x] Create `internal/projection/goals/goals_test.go`
- [x] Goal probability scoring from Monte Carlo
- [x] Create goal editor UI
- [x] Goal markers on fan chart

## Phase 4: What-If Scenario Engine

- [x] Create `internal/projection/whatif/whatif.go`
- [x] Scenario comparison logic (baseline + overrides)
- [x] Pre-built scenario templates
- [x] API endpoint + handler
- [x] Scenario comparison UI with overlay charts

## Phase 5: Tax-Aware Drawdown

- [x] Drawdown optimizer
- [x] Tax impact estimation
- [x] Drawdown strategy UI

Current slice delivered:
- Added `internal/projection/drawdown/drawdown.go` with ordered bucket strategy support (`account_glob`, `tax_category`, `holding_period_months`) and tax-aware FIFO lot ranking.
- Added `POST /api/projection/drawdown` with backend tests validating sort behavior, bucket filtering, and priority ordering.
- Added `/planning/life/drawdown` strategy editor with reorderable buckets and `DrawdownStrategy.svelte` output cards for withdrawal order + estimated tax impact.
- Added deeper coupling: drawdown analysis now optionally runs baseline vs post-drawdown Monte Carlo simulation and returns impact deltas (FIRE probability and timeline impact).

## Phase 6: Caching & Polish

- [x] Simulation snapshot caching
- [x] Cache invalidation on sync
- [x] Navigation integration
- [x] Documentation
- [x] Mobile responsive polish

Phase 6 delivered:
- Added Monte Carlo simulation result caching in `internal/projection/simulator` keyed by normalized simulation config.
- Wired cache invalidation into global `cache.Clear()` so sync/price refresh clears simulation cache alongside other runtime caches.
- Added cross-navigation buttons between `/planning/life`, `/planning/life/whatif`, `/planning/life/goals`, and `/planning/life/drawdown`.
- Updated drawdown metric cards for improved mobile layout density and readability.
