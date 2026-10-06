# Life Projection Engine — Architecture & Implementation Plan

## 1. Vision

Transform Paisa from a **backward-looking ledger** into a **forward-looking financial planning tool** by building a Life Projection Engine that answers: *"Given how I actually live and invest, what does my financial future look like — and what levers can I pull to change it?"*

The engine uses the rich historical data already in Paisa's ledger (income growth, spending patterns, savings rates, investment returns) to project probabilistic financial outcomes against user-defined life goals.

---

## 2. Architecture Overview

```mermaid
graph TB
  subgraph "Data Layer (existing)"
    L[Ledger Files] --> P[Postings/SQLite]
    S[Price Scraper] --> PR[Prices/SQLite]
    P --> Q[Query Builder]
    PR --> MKT[Market Service]
  end

  subgraph "Inference Engine (new)"
    Q --> FD[Financial DNA Extractor]
    MKT --> FD
    FD --> |income_growth, expense_trend<br/>savings_rate, asset_returns| DNA[FinancialProfile]
  end

  subgraph "Goal System (new, extends existing)"
    CFG[paisa.yaml goals] --> GM[Goal Manager]
    GM --> |GoalDefinition[]| SIM
  end

  subgraph "Simulation Engine (new)"
    DNA --> SIM[Monte Carlo Simulator]
    GM --> SIM
    SIM --> |N scenarios × T months| RES[SimulationResult]
    RES --> AGG[Aggregator / Percentile Calculator]
  end

  subgraph "What-If Layer (new)"
    WI[What-If Parameters] --> SIM
  end

  subgraph "Tax Strategy (new, extends existing)"
    TAX[Taxation Module] --> DD[Drawdown Optimizer]
    DD --> SIM
  end

  subgraph "API Layer"
    AGG --> API["/api/projection/*"]
    API --> FE[SvelteKit Frontend]
  end
```

---

## 3. Key Design Decisions

### 3.1 Where does simulation run?

**Go backend.** The simulation is CPU-bound (1000+ Monte Carlo paths × 360+ months). Go's performance characteristics make this natural. The frontend receives pre-aggregated percentile bands and renders them.

> [!IMPORTANT]
> The existing projection page duplicates the Go projection logic in TypeScript (see [+page.svelte](file:///d:/Git/paisa/paisa/src/routes/(app)/planning/projection/+page.svelte#L112-L132)). The new design should consolidate projection logic server-side to avoid drift, while keeping lightweight client-side recalculation for slider interactions via a debounced API call pattern.

### 3.2 Deterministic vs Stochastic

The existing projection uses **deterministic** 3-scenario CAGR (conservative/expected/optimistic). The new engine adds **Monte Carlo** as an opt-in layer while preserving the deterministic mode for backward compatibility.

### 3.3 Goal Storage

Goals already live in `paisa.yaml` (see [config.go](file:///d:/Git/paisa/paisa/internal/config/config.go#L77-L101)). We extend the existing `Goals` struct with a new `LifeGoal` type rather than creating a separate storage mechanism. This keeps the single-source-of-truth principle.

### 3.4 Snapshot Caching

Following the existing [projection_snapshot](file:///d:/Git/paisa/paisa/internal/model/projection_snapshot/projection_snapshot.go) pattern, expensive simulation results are cached in SQLite and invalidated on journal/price sync.

---

## 4. Component Deep-Dive

### 4.1 Financial DNA Extractor

Analyzes the ledger to infer the user's financial "genome" — the parameters that drive projections.

| Parameter | Source | Method |
|---|---|---|
| **Income Growth Rate** | `Income:*` postings, last 3 years | Year-over-year CAGR of annual income |
| **Expense Growth Rate** | `Expenses:*` postings (ex-Tax), last 3 years | Year-over-year CAGR of annual expenses |
| **Savings Rate** | Derived from income & investment postings | Existing logic in [networth_projection.go](file:///d:/Git/paisa/paisa/internal/server/networth_projection.go#L170-L214) |
| **Monthly Contribution** | Asset postings (ex-Checking), last 12 months | Existing logic, extended with trend |
| **Historical Returns** | Per-asset XIRR via [xirr.go](file:///d:/Git/paisa/paisa/internal/service/xirr.go) | Weighted by current allocation |
| **Return Volatility** | Price history std deviation | Annualized from monthly returns |
| **Asset Allocation** | Current portfolio breakdown | From existing allocation endpoint |
| **Current Net Worth** | Existing computation | From [networth_projection_snapshot.go](file:///d:/Git/paisa/paisa/internal/server/networth_projection_snapshot.go#L21-L29) |
| **Annual Expenses** | `Expenses:*` postings, last 24 months | Existing logic, inflation-adjusted |

**New Go package:** `internal/projection/dna/`

### 4.2 Goal System (Extended)

New config struct extending the existing goals:

```go
// New goal type in config.go
type LifeGoal struct {
    Name             string   `json:"name" yaml:"name"`
    Icon             string   `json:"icon" yaml:"icon"`
    Type             string   `json:"type" yaml:"type"`             // "milestone" | "recurring"
    TargetAmount     float64  `json:"target_amount" yaml:"target_amount"`
    TargetDate       string   `json:"target_date" yaml:"target_date"`       // "2030-06" (month-year)
    StartDate        string   `json:"start_date" yaml:"start_date"`         // recurring: "2026-01"
    EndDate          string   `json:"end_date" yaml:"end_date"`             // recurring: "2035-12"
    Frequency        string   `json:"frequency" yaml:"frequency"`           // "monthly" | "quarterly" | "yearly"
    InflationRate    *float64 `json:"inflation_rate" yaml:"inflation_rate"` // per-goal override (nil → use global)
    Priority         int      `json:"priority" yaml:"priority"`
    FundedBy         []string `json:"funded_by" yaml:"funded_by"`           // account globs
    MonthlyAlloc     float64  `json:"monthly_allocation" yaml:"monthly_allocation"`
}

// Extended Goals struct
type Goals struct {
    Retirement []RetirementGoal `json:"retirement" yaml:"retirement"`
    Savings    []SavingsGoal    `json:"savings" yaml:"savings"`
    Life       []LifeGoal       `json:"life" yaml:"life"`              // NEW
}
```

**Goal types:**
- **Milestone:** One-time outflow at `target_date` (e.g., house down payment ₹50L in 2030-06)
- **Recurring:** Repeated outflow from `start_date` to `end_date` at `frequency` (e.g., ₹2L/year vacation from 2026-01 to 2045-12, yearly)

**Inflation:** A global `inflation_rate` in the simulation config applies to all goals by default. Each goal can override it with its own `inflation_rate` (e.g., education inflation at 10% while global is 6%). A `nil` value means "use global."

### 4.3 Monte Carlo Simulator

```go
// internal/projection/simulator/simulator.go

type SimulationConfig struct {
    Iterations       int             // default 1000
    MonthsToProject  int             // default 360 (30 years)
    StartDate        time.Time
    CurrentNetworth  decimal.Decimal
    MonthlyContrib   decimal.Decimal
    ContribGrowthPct decimal.Decimal  // annual contribution growth rate
    ExpectedReturn   decimal.Decimal  // annualized mean return
    ReturnVolatility decimal.Decimal  // annualized std deviation
    InflationRate    decimal.Decimal  // global inflation rate
    Goals            []GoalCashflow   // scheduled outflows (milestone + recurring expanded)
}

type GoalCashflow struct {
    Name          string
    Month         int              // which month offset this outflow occurs (0-indexed from start)
    Amount        decimal.Decimal  // base amount (inflation applied during simulation)
    InflationRate decimal.Decimal  // per-goal inflation override; zero = use global
}

// ExpandGoalCashflows converts LifeGoal config entries into a flat list of
// GoalCashflow events. Milestone goals produce a single cashflow. Recurring
// goals are expanded into one cashflow per frequency period between start and end.

type SimulationResult struct {
    // Monthly percentile bands: P10, P25, P50, P75, P90
    Bands     map[string][]MonthlyPoint  // "p10", "p25", "p50", "p75", "p90"
    GoalProbs map[string]float64         // probability of meeting each goal
    FIREProb  float64                    // probability of FIRE
    FIREYear  map[string]int             // year of FIRE at each percentile
}

type MonthlyPoint struct {
    Date          time.Time       `json:"date"`
    BalanceAmount decimal.Decimal `json:"balance_amount"`
}
```

**Algorithm:** For each iteration:
1. Sample monthly returns from a log-normal distribution: `r ~ N(μ_monthly, σ_monthly)`
2. Apply: `balance[t] = balance[t-1] × (1 + r) + contribution[t] - expenses[t] - goals[t]`
3. Grow contributions by `contrib_growth_pct / 12` monthly
4. Inflate goal amounts by `goal.inflation_rate / 12` monthly
5. Track whether each goal is fully funded
6. Track FIRE crossing (balance > annual_expenses / SWR)

After all iterations, compute percentile bands and goal probabilities.

### 4.4 What-If Engine

Thin wrapper that clones a `SimulationConfig`, applies parameter overrides, and re-runs the simulation:

```go
// internal/projection/whatif/whatif.go

type Scenario struct {
    Name       string                     `json:"name"`
    Overrides  map[string]decimal.Decimal  `json:"overrides"`
    // Keys: "monthly_contribution", "expected_return", "inflation_rate",
    //        "income_growth", "expense_growth", etc.
}

type ComparisonResult struct {
    Baseline  SimulationResult            `json:"baseline"`
    Scenarios map[string]SimulationResult `json:"scenarios"`
}
```

### 4.5 Tax-Aware Drawdown (Phase 5)

Extends the existing [taxation](file:///d:/Git/paisa/paisa/internal/taxation/tax.go) module:

```go
// internal/projection/drawdown/drawdown.go

type DrawdownStrategy struct {
    Order []DrawdownBucket  // priority order for withdrawals
}

type DrawdownBucket struct {
    AccountGlob string
    TaxCategory config.TaxCategoryType
    HoldingPeriodMonths int
}

// OptimalDrawdown returns the tax-minimizing withdrawal order for a given amount
func OptimalDrawdown(db *gorm.DB, amount decimal.Decimal, buckets []DrawdownBucket) []Withdrawal
```

### 4.6 API Endpoints

Following existing patterns in [server.go](file:///d:/Git/paisa/paisa/internal/server/server.go):

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/projection/dna` | Financial DNA summary (inferred parameters) |
| `POST` | `/api/projection/simulate` | Run simulation with config overrides |
| `POST` | `/api/projection/whatif` | Compare baseline vs scenarios |
| `GET` | `/api/projection/goals` | Life goals with probability scores |
| `POST` | `/api/projection/drawdown` | Tax-optimal drawdown analysis |

### 4.7 Frontend

New route: `src/routes/(app)/planning/life/` with sub-routes:

- `/planning/life` — Dashboard: probability gauges, goal progress, net worth fan chart
- `/planning/life/goals` — Goal editor with inline simulation
- `/planning/life/whatif` — Interactive what-if playground
- `/planning/life/drawdown` — Tax-aware withdrawal strategy (Phase 5)

Key visualization: **Fan chart** showing percentile bands (P10–P90) with goal markers, FIRE line, and interactive overlays.

---

## 5. Phased Implementation

> [!IMPORTANT]  
> Each phase is a self-contained, deployable increment. Phases are ordered by value delivered and dependency chain. Later phases can be re-prioritized based on feedback.

---

### Phase 1: Financial DNA Extractor
**Goal:** Extract and expose the user's financial parameters from their ledger data.

**Scope:**
- New Go package `internal/projection/dna/` with `ExtractProfile(db) → FinancialProfile`
- Compute: income growth, expense growth, savings rate, monthly contribution, portfolio-weighted historical return, return volatility
- New API endpoint `GET /api/projection/dna`
- TypeScript interface `FinancialProfile` in `utils.ts`
- Simple frontend card on the existing projection page showing inferred parameters

**Files to create/modify:**

| Action | File |
|---|---|
| [NEW] | `internal/projection/dna/dna.go` |
| [NEW] | `internal/projection/dna/dna_test.go` |
| [MODIFY] | `internal/server/server.go` (register route) |
| [NEW] | `internal/server/projection_dna.go` (handler) |
| [MODIFY] | `src/lib/utils.ts` (add interface + ajax overload) |
| [MODIFY] | `src/routes/(app)/planning/projection/+page.svelte` (display DNA card) |

**Acceptance Criteria:**
- [ ] `GET /api/projection/dna` returns a JSON `FinancialProfile` with all parameters populated from ledger data
- [ ] Parameters are tested against fixture data (INR fixture) with expected ranges
- [ ] Projection page shows a "Your Financial DNA" card with inferred values
- [ ] Go unit tests cover each parameter derivation function
- [ ] `gofmt` and `prettier` pass on all touched files

---

### Phase 2: Monte Carlo Simulation Engine
**Goal:** Replace the deterministic 3-line projection with a probabilistic Monte Carlo engine.

**Scope:**
- New Go package `internal/projection/simulator/` with `Run(config) → SimulationResult`
- Percentile band calculation (P10, P25, P50, P75, P90)
- FIRE probability and crossing year
- New API endpoint `POST /api/projection/simulate`
- Frontend fan chart visualization replacing the current 3-line chart
- Preserve backward compatibility: existing `/api/networth/projection` remains unchanged

**Files to create/modify:**

| Action | File |
|---|---|
| [NEW] | `internal/projection/simulator/simulator.go` |
| [NEW] | `internal/projection/simulator/simulator_test.go` |
| [NEW] | `internal/server/projection_simulate.go` (handler) |
| [MODIFY] | `internal/server/server.go` (register route) |
| [MODIFY] | `src/lib/utils.ts` (add SimulationResult interface + ajax overload) |
| [NEW] | `src/lib/components/FanChart.svelte` (percentile band chart) |
| [NEW] | `src/routes/(app)/planning/life/+page.svelte` (new projection dashboard) |
| [NEW] | `src/routes/(app)/planning/life/+page.ts` (data loader) |

**Acceptance Criteria:**
- [ ] `POST /api/projection/simulate` runs 1000 iterations in < 500ms for a 30-year projection
- [ ] Result contains P10/P25/P50/P75/P90 bands with 360 monthly points each
- [ ] FIRE probability is calculated and returned (0–100%)
- [ ] Fan chart renders correctly with percentile bands and FIRE marker
- [ ] Existing `/api/networth/projection` endpoint continues working identically
- [ ] Go benchmark test confirms performance target
- [ ] Results are reproducible with a fixed random seed (test determinism)

---

### Phase 3: Life Goals System
**Goal:** Allow users to define life goals that integrate with the projection engine.

**Scope:**
- Extend `paisa.yaml` config with `goals.life[]` (LifeGoal struct)
- Update JSON schema for validation
- Goal manager that converts life goals into simulation cashflows
- Goal probability scores (% chance of meeting each goal)
- Frontend goal editor (CRUD via config API)
- Goal markers on the fan chart

**Files to create/modify:**

| Action | File |
|---|---|
| [MODIFY] | `internal/config/config.go` (add LifeGoal struct, extend Goals) |
| [MODIFY] | `internal/config/schema.json` (add life goal schema) |
| [NEW] | `internal/projection/goals/goals.go` (goal → cashflow conversion) |
| [NEW] | `internal/projection/goals/goals_test.go` |
| [MODIFY] | `internal/server/projection_simulate.go` (integrate goals) |
| [NEW] | `internal/server/projection_goals.go` (goal-specific endpoint) |
| [MODIFY] | `internal/server/server.go` (register routes) |
| [MODIFY] | `src/lib/utils.ts` (add LifeGoal interface) |
| [NEW] | `src/lib/components/GoalEditor.svelte` |
| [NEW] | `src/lib/components/GoalProbabilityGauge.svelte` |
| [MODIFY] | `src/routes/(app)/planning/life/+page.svelte` (integrate goals) |
| [NEW] | `src/routes/(app)/planning/life/goals/+page.svelte` |

**Acceptance Criteria:**
- [ ] `paisa.yaml` accepts `goals.life[]` with name, type, target_amount, target_date, inflation_rate, priority, funded_by
- [ ] Config schema validation rejects invalid goal definitions
- [ ] Each goal appears on the fan chart as a marker at its target date
- [ ] Probability score (0–100%) is computed for each goal based on Monte Carlo results
- [ ] Goals can be created, edited, and deleted via the UI (round-trips through config API)
- [ ] Goal priority ordering affects the simulation (higher priority goals get funded first)
- [ ] Regression fixture test validates goal probability for known scenarios

---

### Phase 4: What-If Scenario Engine
**Goal:** Let users compare their baseline trajectory against hypothetical scenarios.

**Scope:**
- What-if engine that runs multiple simulations with parameter overrides
- Side-by-side comparison visualization
- Pre-built scenario templates ("Job loss for 6 months", "Increase SIP by ₹10k", "Early retirement")
- Interactive slider-driven scenario builder
- New API endpoint `POST /api/projection/whatif`

**Files to create/modify:**

| Action | File |
|---|---|
| [NEW] | `internal/projection/whatif/whatif.go` |
| [NEW] | `internal/projection/whatif/whatif_test.go` |
| [NEW] | `internal/server/projection_whatif.go` (handler) |
| [MODIFY] | `internal/server/server.go` (register route) |
| [MODIFY] | `src/lib/utils.ts` (interfaces) |
| [NEW] | `src/routes/(app)/planning/life/whatif/+page.svelte` |
| [NEW] | `src/lib/components/ScenarioComparison.svelte` |

**Acceptance Criteria:**
- [ ] `POST /api/projection/whatif` accepts baseline config + up to 5 scenario overrides
- [ ] Each scenario runs a full Monte Carlo simulation independently
- [ ] Response includes baseline + all scenario results with goal probabilities
- [ ] Frontend shows overlay fan charts with toggle-able scenarios
- [ ] Pre-built scenario templates populate the form with realistic defaults
- [ ] Total response time for baseline + 3 scenarios < 2 seconds
- [ ] Go tests validate scenario override mechanics

---

### Phase 5: Tax-Aware Drawdown Strategy
**Goal:** Optimize withdrawal order across accounts to minimize tax liability.

**Scope:**
- Extend the existing [taxation](file:///d:/Git/paisa/paisa/internal/taxation/tax.go) module with drawdown analysis
- Account-level tax impact estimation for hypothetical withdrawals
- Optimal drawdown order recommendation
- Integration with the simulation engine (drawdown affects projected trajectory)
- New API endpoint `POST /api/projection/drawdown`
- Frontend drawdown strategy page with visual account ordering

**Files to create/modify:**

| Action | File |
|---|---|
| [NEW] | `internal/projection/drawdown/drawdown.go` |
| [NEW] | `internal/projection/drawdown/drawdown_test.go` |
| [MODIFY] | `internal/taxation/tax.go` (add hypothetical tax calc) |
| [NEW] | `internal/server/projection_drawdown.go` |
| [MODIFY] | `internal/server/server.go` (register route) |
| [MODIFY] | `src/lib/utils.ts` (interfaces) |
| [NEW] | `src/routes/(app)/planning/life/drawdown/+page.svelte` |
| [NEW] | `src/lib/components/DrawdownStrategy.svelte` |

**Acceptance Criteria:**
- [ ] `POST /api/projection/drawdown` returns an ordered list of withdrawal recommendations
- [ ] Each recommendation includes: account, amount, tax category, estimated tax, holding period
- [ ] Tax calculations are consistent with the existing capital gains module
- [ ] Drawdown order minimizes total tax for the requested withdrawal amount
- [ ] Frontend shows draggable account ordering with real-time tax impact
- [ ] Go tests validate tax optimization against known tax scenarios (equity LTCG, debt, etc.)

---

### Phase 6: Simulation Snapshot Caching & Polish
**Goal:** Optimize performance and UX for production use.

**Scope:**
- Snapshot caching for simulation results (following [projection_snapshot](file:///d:/Git/paisa/paisa/internal/model/projection_snapshot/projection_snapshot.go) pattern)
- Cache invalidation on journal/price sync
- Pre-computation during sync (like existing dashboard/projection snapshots)
- Navigation integration (add "Life Plan" to the Planning section)
- Responsive design and mobile optimization
- Loading states and error handling
- Documentation (docs page)

**Files to create/modify:**

| Action | File |
|---|---|
| [NEW] | `internal/model/simulation_snapshot/simulation_snapshot.go` |
| [MODIFY] | `internal/server/sync.go` (add simulation snapshot refresh) |
| [MODIFY] | `internal/server/snapshot_policy.go` (register snapshot kind) |
| [MODIFY] | `src/lib/components/Navbar.svelte` (add navigation) |
| [MODIFY] | `src/routes/(app)/planning/+page.ts` (update redirect) |
| [NEW] | `docs/reference/life-projection.md` |
| Polish | All `src/routes/(app)/planning/life/**` pages |

**Acceptance Criteria:**
- [ ] First load of the Life Plan page uses cached snapshot (< 200ms response)
- [ ] Simulation is automatically refreshed after journal sync
- [ ] Navigation bar includes "Life Plan" under Planning section
- [ ] All pages are responsive and work on mobile viewports
- [ ] Loading spinners show during simulation computation
- [ ] Error states are handled gracefully (no data, bad config, etc.)
- [ ] `make lint` passes on all touched files
- [ ] Documentation page explains the feature and configuration

---

## 6. Design Decisions (Resolved)

| Decision | Resolution |
|---|---|
| **Inflation modeling** | Global inflation rate with per-goal override. `nil` override = use global. |
| **Existing projection page** | Keep `/planning/projection` as-is. New engine lives at `/planning/life`. Both coexist. |
| **Recurring goals** | Required from Phase 3. Goals have `start_date`, `end_date` (month-year), and `frequency` (monthly/quarterly/yearly). |
| **Multi-currency** | Start with `default_currency` only. Multi-currency support deferred to a future enhancement. |
| **Monte Carlo iterations** | Default 1000. Expose as advanced setting (100–10000 range). |

---

## 7. Verification Plan

### Automated Tests
- **Go unit tests** for each new package (`dna`, `simulator`, `goals`, `whatif`, `drawdown`)
- **Go benchmark tests** for simulation performance (target: 1000 iterations × 360 months < 500ms)
- **Regression tests** against the INR fixture: known inputs should produce expected probability ranges
- **Frontend tests** (Bun) for new component logic (percentile calculation, chart data transformation)

### Manual Verification
- Run `make develop` and visually verify the Life Plan dashboard
- Test goal CRUD flow end-to-end (create goal → see probability → modify → see update)
- Test what-if scenarios with known outcomes (e.g., doubling contribution should halve FIRE timeline)
- Verify snapshot caching (first load fast, sync triggers refresh)
- Test on narrow viewport (mobile responsive)

### Scoped Validation Commands
```bash
# Go: touched packages
go test ./internal/projection/...
gofmt -l internal/projection/ internal/server/projection_*.go

# Frontend: touched files
npx prettier --write src/routes/\(app\)/planning/life/ src/lib/components/FanChart.svelte
npm run check
```
