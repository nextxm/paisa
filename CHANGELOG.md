# CHANGELOG

### Unreleased — Future changes

#### New features

- **Header syncing indicator** — A non-intrusive "Syncing…" badge with a spinning icon appears in the app header (navbar-end) while any background job is active (`isJobRunning` is true) and disappears with a smooth fade once the job completes. The indicator is hidden on mobile via `is-hidden-mobile` to keep the layout stable, and is declared with `aria-live="polite"` for accessibility. No layout shift occurs because the element is not present in the DOM when idle (P1.3).
- **Frontend jobs store** — New `src/lib/stores/jobs.ts` module provides a global Svelte store (`jobs`) that tracks background `Job` objects keyed by their ID. Exposes `upsert` (insert or replace), `updateById` (partial merge; returns `true` when the ID was found), `reset` (clear all), and `snapshot` (synchronous read). Derived stores `jobsList` (sorted by `created_at`) and `isJobRunning` (true while any job is pending or running) are exported for reactive UI consumption. The `jobs`, `jobsList`, and `isJobRunning` names are re-exported from `src/store.ts` for consistency with other app-wide stores (P1.3).
- **`Job` and `JobStatus` types in utils.ts** — Added `JobStatus` union type (`"pending" | "running" | "completed" | "failed"`) and `Job` interface mirroring the Go `worker.Job` struct. Date fields (`created_at`, `started_at`, `finished_at`) are typed as `string` because the ajax reviver only converts keys matching `/Date|date|time|now/`.
- **`POST /api/sync` ajax overload updated** — The TypeScript overload for `/api/sync` now declares the return type as `{ job_id: string }`, matching the `202 Accepted` response the backend already returns. A new overload for `GET /api/jobs/:id` returning `Job` is also added.
- **`src/lib/sync.ts` adapted for async API** — `sync()` now extracts `job_id` from the `202` response and immediately upserts a `pending` job into the jobs store. Network-level failures (non-2xx, connection errors) are surfaced as a Bulma toast; job-level failures will be surfaced via polling (P1.3-12).
- **`startPolling` — background job status polling** — `startPolling(jobId, onTerminal?, options?)` polls `GET /api/jobs/:id` in the background at 2-second intervals, updating the jobs store on each response. Polling stops automatically when the job reaches a terminal state (completed or failed) or after 150 attempts (~5 minutes). Up to 5 consecutive network errors are tolerated before polling aborts; the error counter resets on the next successful response. On failure, a Bulma toast is shown with the error message. The `onTerminal` callback is called with the final job, enabling callers (e.g. `Actions.svelte`) to trigger a data refresh. All timing and retry parameters are injectable for unit testing (P1.3).
- **`Actions.svelte` updated for async sync** — `syncWithLoader` now calls `startPolling` after `sync()` returns a `job_id`, deferring the data `refresh()` to the `onTerminal` callback rather than running it immediately. This ensures the UI reflects the completed sync result rather than stale data (P1.3).
- **`createJobsStore` exported** — The `createJobsStore` factory in `src/lib/stores/jobs.ts` is now exported so tests and tooling can create isolated store instances without sharing the module-level singleton.
- **Asynchronous POST /api/sync** — `POST /api/sync` now returns `202 Accepted` immediately with `{"job_id": "<uuid>"}` instead of blocking until the sync completes. The sync work is performed in the background via the `worker.Registry`; callers can poll the job status via `GET /api/jobs/:id` (P1.2). Readonly and authentication behaviour are preserved: the endpoint is still guarded by `ReadonlyMiddleware` and `TokenAuthMiddleware`.
- **GET /api/jobs/:id** — New read endpoint that returns the current state of a background job as a JSON object (fields: `id`, `status`, `created_at`, `started_at`, `finished_at`, `error`, `details`). Returns `404` with the standard error envelope when the job ID is unknown.
- **Background worker package** — New `internal/service/worker` package provides a thread-safe `Registry` for submitting and tracking background jobs. Each `Job` progresses through a defined state machine (`Pending → Running → Completed | Failed`) with creation, start, and finish timestamps. Callers submit a `func(context.Context) error` via `Registry.Submit` and receive a unique job ID; status can be polled with `Registry.Get` or enumerated with `Registry.List`. This lays the groundwork for the asynchronous sync API (P1.2).
- **Per-step job details** — `Job` now carries a `Details []string` field that holds per-step diagnostic messages accumulated during job execution. For example, individual commodity fetch failures from a price-scraper job are each recorded as a separate entry rather than collapsed into a single top-level error string. Present on both successful and failed jobs. `Registry.SubmitDetailed` accepts a `func(context.Context) ([]string, error)` callback and stores the returned details in the job.
- **Price scraper failures in job details** — Each commodity price fetch failure from `SyncCommodities` (e.g. network errors, missing quote currency) is now recorded as a separate entry in `Job.Details` so operators can identify the exact failing commodity without inspecting server logs.
- **XIRR pre-computation in sync job** — After a journal or price sync, `service.WarmXIRRCache` pre-computes XIRR for every investment account and stores the results in the SQLite computation cache. Any accounts whose Newton-Raphson XIRR solver does not converge are recorded in `Job.Details` as warnings so operators can identify data problems. Subsequent API calls to `/api/gain` and `/api/networth` are served from cache.
- **`xirr.XIRRWithConvergence`** — New exported function alongside the existing `XIRR` that returns `(decimal.Decimal, bool)`, where the bool reports whether the Newton-Raphson iteration converged. Callers that need to distinguish a genuine zero XIRR from a non-convergent solver can use this variant.
- **Explicit Job state-machine enforcement** — The `worker` package now defines a `validTransitions` map that enumerates every legal `JobStatus` transition. A `transition()` helper validates each state change at runtime and panics on any illegal move (e.g. Completed → Running), making invalid transitions immediately observable. `JobStatus.IsTerminal()` returns true for `Completed` and `Failed`, enabling callers to check whether a job has reached a terminal state without inspecting the status string directly.
- **Deterministic `Registry.List` ordering** — `Registry.List()` now returns jobs sorted by `CreatedAt` ascending (oldest job first), replacing the previous non-deterministic map-iteration order. All concurrent-access tests for `List`, `Get`, and `Submit` pass under `-race`.
- **SHA-256 file hash utility** — Added `SHA256File(path string) (string, error)` in `internal/utils/hash.go`. Streams file content in chunks for efficient large-file handling, and returns descriptive errors on open/read failures. This utility supports upcoming incremental sync checks (P1.1).
- **Metadata key/value table** — New `internal/model/metadata` package adds a `Metadata` model backed by a SQLite table (`metadata`) with a unique index on `key`. A new schema migration (v3) creates the table for both fresh installs and existing databases.
- **Metadata persistence API** — `metadata.Get(db, key)` returns the stored value or `gorm.ErrRecordNotFound` for missing keys; `metadata.GetOrDefault(db, key, default)` returns a caller-supplied fallback for missing keys; `metadata.Set(db, key, value)` performs an atomic `INSERT … ON CONFLICT DO UPDATE` upsert, covering both create and update paths consistently.
- **SyncJournal hash-skip optimisation** — `SyncJournal` now computes the SHA-256 hash of the journal file before invoking any ledger CLI commands. When the hash matches the value stored in the `journal_hash` metadata key from the last successful sync, all CLI validation and parsing work is skipped and the result carries `Skipped: true`. The hash is only persisted after a fully successful sync, so a partial failure never silently suppresses the next run.

### 0.8-beta (2026-04-26) — Multi-currency pricing rollout

#### New features

- **Multi-currency price schema** — `prices` table gains `quote_commodity` and
  `source` columns; a backward-compatible migration (v1 → v2) backfills
  `quote_commodity` from the ledger default currency for all existing rows.
- **Pair-aware rate resolver** — `service.GetRate` resolves exchange rates via
  direct pairs, inverse pairs, and one-hop cross rates through the configured
  default currency (e.g. INR).
- **Extended price API** — `GET /api/price` accepts optional `base`, `quote`,
  `from`, `to`, `source`, and `report_currency` query parameters; unfiltered
  calls continue to return the legacy map-keyed format for backward
  compatibility.
- **Price export endpoint** — `GET /api/price/export` exports the full price
  history as ledger, hledger, or beancount directives.
- **Rollback flag** — set `disable_multi_currency_prices: true` in `paisa.yaml`
  to disable cross-rate resolution and `report_currency` conversion and revert
  to pre-rollout behaviour without downgrading the binary.
- **UI Enhancements** — Improved Actions and Navbar components, added portfolio sync option to dropdown actions. Increased size of navbar action icons on mobile devices for better touchability.
- **Original Balances** — Added currency field and original balance display to Credit Card Summary, Liability breakdown, and asset breakdown for non-equity commodities.
- **New Configuration Fields** — Added `currencies` field to distinguish currencies from securities, and `provider_debug_http` to log provider HTTP requests.
- **FX Rates Page** — Added a new page for tracking exchange rates (FX Rates) with derived rate support.
- **Background Sync** — Implement periodic background sync for backend to improve data freshness, with schedule option for journal and price sync.

#### Bug fixes

- **Assets -> Gain page calculations** — Fixed units-vs-currency mismatch in XIRR, investment, and absolute return calculations by using historical market prices when ledger falls back to units.
- **Same-Day Price/Rate Race Conditions** — Fixed by using EndOfDay pivot.
- **Yahoo Provider Handling** — Fixed handling of nil close values and empty responses.
- **Currency Detection** — Fixed `IsForeignCurrency` detection by normalizing commodity input.
- **Net Worth Timeline** — Reconciled net worth discrepancies between dashboard and timeline by including today's transactions and latest prices.
- **Core Stability** — Fixed server crash in `GetUnitPrice` when prices are missing by replacing `log.Fatal` with a warning, and resolved USD->INR conversion for non-posting commodities and implicit journal quotes.
- **Missing Price & FX Reporting** — Enhanced missing price warnings with dates and added summary counts for missing FX rates during cache warming.
- **Mobile UI** — Re-enabled zoom functionality on mobile devices.
- **Dependencies** — Addressed multiple npm dependency vulnerabilities (kit, axios, handlebars, lodash, pdfjs-dist, vite).

#### Upgrade guide

1. Run `paisa update` (or restart the server) — the database migration runs
   automatically on startup; no manual steps are required.
2. Verify prices with `GET /api/price?base=<COMMODITY>` to confirm
   `quote_commodity` has been backfilled correctly.
3. If unexpected valuation changes are observed, set
   `disable_multi_currency_prices: true` in `paisa.yaml` and restart as a
   rollback measure.

#### Rollback procedure

1. Add `disable_multi_currency_prices: true` to `paisa.yaml`.
2. Restart the Paisa server — no database changes are required.
3. Report the regression so it can be investigated before re-enabling.

### 0.7.4 (2025-02-23)

- Update price data domain
- Fix NixOS build

### 0.7.3 (2025-02-23)

- Fix yahoo price fetcher
- Build fixes

### 0.7.1 (2024-10-20)

- Fix remote code execution [vulnerability](https://github.com/ananthakumaran/paisa/issues/294)

### 0.7.0 (2024-08-26)

- Add [docker image variant](https://github.com/ananthakumaran/paisa/pull/274) for hledger and beancount
- Bug fixes

### 0.6.6 (2024-02-10)

- Improve tables (make it sortable)
- Show tabulated value on allocation page
- Show invested value on goals page
- Bug fixes

### 0.6.5 (2024-02-02)

- Add Liabilities > [Credit Card](https://nextxm.github.io/paisa/reference/credit-cards) page
- Support password protected XLSX file
- Allow user to configure timezone
- Bug fixes

### 0.6.4 (2024-01-22)

- Add checking accounts balance to dashboard
- Improve template management UI
- Improve spinner and page transition
- Bug fixes

### 0.6.3 (2024-01-13)

- Introduce [Sheets](https://nextxm.github.io/paisa/reference/sheets/): A notepad calculator with access to your ledger
- Remove flat option from cashflow > yearly page
- Dockerimage now installs paisa to /usr/bin
- Improve legends rendering on all pages
- Allow user to cancel pdf password prompt
- Add new warning for missing assets accounts from allocation target
- Support hledger's balance assertion
- Bug fixes

### 0.6.2 (2023-12-23)

- New logo
- Allow goals to be reordered
- Show goals on the dashboard page
- Bug fixes

### 0.6.1 (2023-12-16)

- Add new price provider: [Alpha Vantage](https://nextxm.github.io/paisa/reference/commodities/#alpha-vantage)
- Make first day of the week configurable
- Support ledger strict mode
- Add user login support, go to `User Accounts` section in configuration page to enable it
- Show notes associated with a transaction/posting
- Bug fixes

### 0.6.0 (2023-12-09)

- Add individual account balance on goals page
- Add [keyboard shortcuts](https://nextxm.github.io/paisa/reference/editor/) to format/save file on editor page
- Add ability to search posting/transaction by note
- Add option to reverse the order of generated transactions on import page
- Add option to clear price cache
- Bug fixes

### 0.5.9 (2023-11-26)

- Improve postings page
- Add income statement page (Cash Flow > Income Statement)
- Bug fixes

### 0.5.8 (2023-11-18)

- Add ability to specify rate, target date or monthly contribution to
  [savings goal](https://nextxm.github.io/paisa/reference/goals/savings/)
- Improve price page
- Bug fixes

### 0.5.7 (2023-11-11)

- Add [goals](https://nextxm.github.io/paisa/reference/goals)
- Remove retirement page (available under goals)
- Bug fixes

#### Breaking Changes :rotating_light:

Retirement page has been moved under goals. If you have used
retirement, you need to setup a new [retirement goal](https://nextxm.github.io/paisa/reference/goals)

### 0.5.6 (2023-11-04)

- Add support for Income:CapitalGains
- Add option to control display precision
- Add new price provider for gold and silver (IBJA India)
- Add option to disable budget rollover
- Bug fixes

### 0.5.5 (2023-10-07)

- Support account icon customization
- Add beancount ledger client support

### 0.5.4 (2023-10-07)

- Add calendar view to recurring page
- Support [recurring period](https://nextxm.github.io/paisa/reference/recurring/#period) configuration
- Support European number format
- Bug fixes

### 0.5.3 (2023-09-30)

- Add Docker Image
- Add Linux Application (deb package)
- Move import templates to configuration file
- Bug fixes

#### Breaking Changes :rotating_light:

User's custom import templates used to be stored in Database, which is
a bad idea in hindsight. It's being moved to the configuration
file. With this change, all the data in paisa.db would be transient
and can be deleted and re created from the journal and configuration
files without any data loss.

If you have custom template, take a backup before you upgrade and add
it again via new version. If you have already upgraded, you can still
get the data directly from the db file using the following query
`sqlite3 paisa.db "select * from templates";`

### 0.5.2 (2023-09-22)

- Add Desktop app
- Support password protected PDF on import page
- Bug fixes

#### Breaking Changes :rotating_light:

- The structure of price code configuration has been updated to make
  it easier to add more price provider in the future. In addition to
  the code, the provider name also has to be added. Refer the
  [config](https://nextxm.github.io/paisa/reference/config/) documentation for more details

```diff
     type: mutualfund
-    code: 122639
+    price:
+      provider: in-mfapi
+      code: 122639
     harvest: 365
```

### 0.5.0 (2023-09-16)

- Add config page
- Embed ledger binary inside paisa
- Bug fixes

### 0.4.9 (2023-09-09)

- Add [search query](https://nextxm.github.io/paisa/reference/bulk-edit/#search) support in transaction page
- Spends at child accounts level would be included in the budget of
  parent account.
- Fix the windows build, which was broken by the recent changes to
  ledger import
- Bug fixes

### 0.4.8 (2023-09-01)

- Add budget
- Add hierarchial cash flow
- Switch from float64 to decimal
- Bug fixes

### 0.4.7 (2023-08-19)

- Add dark mode
- Add bulk transaction editor
