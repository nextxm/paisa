# CHANGELOG

### Unreleased — Future changes

#### Features

- **FX impact decomposition and currency exposure surfaces** — Added multi-currency attribution across net worth and asset views.
  - Backend `/api/networth` timeline points now include `contribution`, `investment_return`, and `fx_impact`, with decomposition logic that separates cumulative FX movement from local investment return for non-default-currency holdings.
  - Added backend `GET /api/currency-exposure` to return denomination-level portfolio exposure (`currency`, `amount`, `percentage`).
  - Updated assets net worth UI with FX overlay toggle, added decomposition metrics cards, and surfaced FX/contribution details in timeline tooltips.
  - Added currency exposure donut widget on **Assets → Allocation** and **Ledger → FX Rates** pages.
  - Added backend-focused tests validating FX decomposition consistency and currency exposure grouping totals.

- **Fix More → Logs runtime errors** — Resolved a Logs page failure caused by Workbox navigation fallback and fragile client rendering.
  - Updated PWA `navigateFallback` to `/index.html` so Workbox always serves a precached navigation shell (avoids `non-precached-url` for `/`).
  - Hardened `/more/logs` rendering by removing direct `window` access in markup, guarding virtualized rows, and safely formatting log timestamps.
  - Wrapped logs fetch in error handling to avoid unhandled promise rejections when the API request fails.

- **Svelte 5 state management: complete class-based adapters and decouple UI/persisted state (P2.3)** — Completed the remaining Svelte 5 state modernisation tasks (#231 #232 #233):
  - Added `commandPaletteOpen`, `cashflowExpenseDepthAllowed`, and `cashflowIncomeDepthAllowed` to the `UIState` class in `src/lib/state/ui.svelte.ts` so all transient UI state is accessible via a single rune-compatible entry point (`uiState.<prop>.current`).
  - Added `editorLeftWidth`, `editorRightWidth`, `editorLeftCollapsed`, `editorRightCollapsed`, and `configSidebarCollapsed` to the `PersistedState` class in `src/lib/state/persisted.svelte.ts`, completing coverage of every persisted store.
  - Moved the non-persisted `cashflowExpenseDepthAllowed`, `cashflowIncomeDepthAllowed`, and `setCashflowDepthAllowed` from `persisted_store.ts` to `store.ts`, enforcing a clear boundary: `persisted_store.ts` contains only localStorage-backed stores while `store.ts` owns all transient runtime state.
  - Updated `Navbar.svelte` and the yearly cash-flow page to import the moved stores from `store.ts`.

- **YoY Analysis: aligned top controls and denser KPI cards** — Fixed the `/analysis/yoy` top-bar control alignment so both dropdowns and the CSV action align cleanly on the same baseline. Tightened KPI card spacing/padding to reduce empty whitespace, expanded the KPI strip with new standalone cards for biggest category mover and efficiency snapshot (expense load + best net month), and tuned card heights/type scale for a more balanced 6-card layout on desktop and tablet.

- **Cash Flow Monthly: aligned multi-currency summary card numbers** — Refined the Monthly Cash Flow summary cards (Income, Expenses, Taxes, Net Flow) so each value now uses a consistent amount-and-currency grid with tabular numerals. This improves readability for mixed-commodity months by lining up figures cleanly across rows and cards.

- **YoY Analysis: richer dashboard layout with deeper insights** — Revamped the `/analysis/yoy` page with a stronger visual hierarchy and additional analytical elements. Added headline KPI cards (spending, income, net savings, savings rate), a hero summary band for selected comparison range, a category movers table (YoY change + expense share), and a monthly net profile panel that highlights highest expense and best net months. Existing YoY line/bar charts and CSV export remain intact, now presented within a more polished, responsive layout.

- **Cash Flow Monthly: polished inflow/outflow breakdown layout** — Fixed the vertical divider spacing regression in the monthly cash-flow page so the outflow section no longer crowds the divider. Also refined inflow/outflow breakdown tables with improved row spacing, clearer amount alignment, and subtle hover treatment for better readability.

- **Dev server: fixed Svelte virtual CSS parsing regression** — Adjusted Vite polyfill configuration to avoid intercepting Svelte virtual style modules in development, preventing PostCSS errors like `Unknown word` on `.svelte?type=style` requests.

- **Navigation: breadcrumb alpha tag spacing fix** — Fixed breadcrumb label/tag overlap by aligning breadcrumb items with inline-flex layout and dedicated alpha tag spacing, preventing the `alpha` badge from colliding with menu text.

- **MoM Analysis: negative trajectory values bounded correctly** — Fixed the MoM expense trajectory chart so windows containing negative values no longer spill outside the chart area. The y-axis now uses real min/max series bounds (with padding), the filled area anchors to the zero baseline, and values are clamped to chart space.

- **MoM Analysis: actual-currency filtering dropdown** — In actual currency mode, a new currency dropdown now allows filtering all MoM visuals and tables to a single selected currency (for example, CAD). When a currency is selected, charts, summary cards, timeline, variance/composition, and breakdown rows all render only that currency's data. Leaving it as `All currencies` preserves the current multi-currency behavior.

- **MoM Analysis: toggle between default and actual currency view** — Modified the MoM analysis page to support viewing expense data in two modes:
  - **Default Currency**: All amounts converted to and displayed in the default currency (INR by default). Reflects how much was spent in default currency terms.
  - **Actual Currency**: Amounts displayed in their original transaction currencies (USD, EUR, etc.). When multiple currencies exist for the same dimension (e.g., "Groceries USD" and "Groceries EUR"), they appear as separate rows in the breakdown table so you can track spending by original commodity.
  - Backend now returns `original_amount` field on each posting, preserving the amount in the original commodity before any conversion. Frontend toggles between using `amount` (converted to default) and `original_amount` (native commodity) for all calculations and charts.
  - In actual currency view, dimensions are grouped by both category/payee/account AND commodity, so mixed-currency spending is shown separately — e.g., "Groceries" becomes "Groceries (USD)" and "Groceries (EUR)" if both exist.
  - No server roundtrip on currency toggle — all amounts are pre-calculated and sent with each posting.

- **MoM Analysis: client-side currency conversion** — Added a "Currency" selector to the MoM analysis page control panel. On load, the page fetches latest FX rates for all configured currency pairs from the new `GET /api/expense/latest-rates` endpoint and stores them locally. Switching currencies instantly re-derives all charts and tables via a `$derived` converted postings layer — no extra network request on each switch. The new backend endpoint (`GetLatestRates`) returns a `rates[base][quote]` map plus the default currency so the UI can build the selector and apply conversions without server-side re-processing.

- **Month-on-Month (MoM) Analysis – Phase 5 & Layout Polish** — Continued improvements to the MoM analysis page:
  - **MoM Delta Chart** — New D3 diverging bar chart (right of the Monthly Timeline) showing month-over-month change as positive (red) or negative (green) bars. Includes hover tooltips, percentage labels on large bars, and a zero baseline, making directional month changes immediately visible at a glance.
  - **Monthly Timeline compacted to left half** — The timeline table now occupies 5/12 columns with a `white-space: nowrap` style to prevent wrapping, while the MoM Delta chart fills the freed 7/12 columns to the right. The table now shows percentage-only MoM column (hover for absolute value) for better scannability.
  - **Signals sidebar removed** — Redundant "Largest Movers" and "30-Day Momentum" sidebar replaced by the richer Dimension Variance Chart above (added in Phase 3). Breakdown table promoted to full width.
  - **Breakdown table MoM column** — Now shows percentage change (e.g. `+12.3%`) as primary value with absolute amount on hover, matching the Timeline table style.
  - **Help tooltips on metrics** — Volatility card now explains Coefficient of Variation via title attribute with thresholds (Low <15%, Medium 15-30%, High >30%). 3M Avg and MoM column headers carry descriptive `title` attributes. Volatility value includes a ✓ or ⚠ indicator.
  - **Volatility card expanded** — Added trend direction ("Up/Down this month") and 3M Avg directly into the Volatility card, replacing the previous sparse layout.

- **Month-on-Month (MoM) Analysis Page – Phase 1-4 Complete** — Launched a sophisticated month-on-month analysis page with advanced visualizations and compact layout:
  - **Phase 1: Compact Layout** — Redesigned control panel (single-row flexbox), 3-column summary cards (Latest Month, Range Highlights, Volatility & Trend), reduced padding/fonts throughout for dense information display.
  - **Phase 2: Expense Trajectory Chart (Hero)** — Interactive D3 line chart showing actual expense trajectory + 3-month moving average overlay, with hover tooltips displaying month, expense, 3M avg, and MoM % change. Includes area fill for visual depth, grid lines, and responsive sizing.
  - **Phase 3: Dimension Variance Chart** — Grouped bar chart (Previous vs Current month) for top movers (categories/payees/accounts), sorted by absolute change. Color-coded by direction (red=increase, green=decrease), with value labels and interactive hover details. Helps identify what drove month-over-month changes.
  - **Phase 4: Dimension Composition Chart** — Stacked area chart showing how category/payee/account breakdown shifts over selected month window. Interactive legend for toggling visibility. Answers "How has spending distribution changed?"
  - **Multi-currency Support** — All charts and tables respect the `report_currency` dropdown; data is converted server-side via `/api/expense` with `report_currency` parameter using available FX rates.
  - **Supporting Tables** — Compact Monthly Timeline (month, total, MoM change, 3M avg) and Breakdown table (by selected dimension) with sparkline trends for each category.
  - **Design Philosophy** — Charts are primary; tables are supporting reference. Opposite of YoY's basic approach. Delivers rich trend analysis for detailed expense insights.

- **MoM analysis currency controls** — The MoM analysis page now supports dual display modes: original transaction currency and report currency. Users can pick currencies from configured settings (`currencies`) via selectors. In report-currency mode, `/api/expense` now accepts `report_currency` and returns expense amounts/trends converted from the default currency using available FX rates.

- **Avoid app refresh after Quick Add and Editor save** — Saving via Quick Add transaction creation or Ledger Editor no longer triggers a global app refresh/remount. This prevents the perceived page refresh right after writes while keeping the existing explicit sync workflow (`Please sync to see changes`).

- **Frontend typing fix for `/api/config` quick-add fetch** — Fixed `ajax("/api/config", { background: true })` typing so GET calls with options retain the config response shape (including `accounts`), resolving `svelte-check` failures in quick-add launch paths.

- **Global Command Palette (Ctrl+K)** — Added a global command palette accessible via `Ctrl+K` (or `Cmd+K` on Mac) that allows quick navigation between all pages, launching the Quick Add Transaction modal, and searching currency-related views. The palette features fuzzy search, keyboard navigation (arrow keys, Enter to select, Escape to close), and a search button in the navbar for mouse users.

- **Natural Language Transaction Parser (Phase 1)** — New `internal/parser` package enables parsing natural language text input into structured transactions with confidence scoring and interactive suggestions.
  - **Core Parser** — 8-step extraction pipeline: normalize → date → amount → payee → hints → account matching → direction → confidence. Handles 10+ transaction scenarios (expenses, income, transfers, refunds, etc.).
  - **Regex-based Extraction** — High-performance pattern matching for dates (ISO 8601, month names, relative), amounts (currency symbols, codes, word forms), account hints (from/to/via), and payment methods. <1ms per parse operation.
  - **TF-IDF Account Matching** — Cosine similarity-based account disambiguation against known accounts in ledger, providing top 3 suggestions for low-confidence matches.
  - **Confidence Scoring** — Per-field scoring (date, amount, payee, from/to accounts, direction) with weighted average (amount 30%, from/to 25% each, payee 15%, date 5%). Auto-create threshold: 0.85 (configurable).
  - **Configurable Keywords** — Expense/income/transfer markers and payment method hints loaded from `paisa.yaml` for user customization without code changes.
  - **Unit Tests** — 18+ test cases covering all 10 transaction scenarios plus edge cases (empty input, long input, missing fields, special characters). All tests passing.
  - **Phase 2 backend APIs** — Added `POST /api/parser/parse` for parser preview and `POST /api/parser/create-transaction` for parse + append + sync flow with optional user overrides.
  - **Training log persistence** — Added schema migration v8 to create `parser_training_log` and log parser predictions plus user-confirmed values for future model training.
  - **Phase 2 frontend quick-add polish** — Added explicit “Clear Parsed State” action in Quick Add modal and extracted parser submit/suggestion helpers for focused parser-assisted quick-add tests (parse mapping, suggestion selection, create payload path). - **Parser Enhancements for Compact Formats** — Improved parsing of compact transaction descriptions (e.g., "20 cad groceries bmo cc at no frills") with:
    - **Category hint extraction** — Detects expense/income markers in text and uses them as "to" account hints for better category matching (e.g., "groceries" → "Expenses:Groceries").
    - **Payment method expansion** — Enhances payment method hints by extracting bank/card names (e.g., "bmo cc" → "bmo credit card") for improved TF-IDF account matching (+0.5 similarity boost for matching bank names).
    - **Robust amount extraction** — Fixed regex patterns for both prefix ($15) and suffix (15$, 15 CAD) amount formats, with case-insensitive currency matching (cad, CAD, etc.).
    - **Improved account matching** — Enhanced similarity scoring with substring matching (+0.4 boost) and explicit bank/card keyword matching for better identification of financial accounts.
    - **Cleaner payee extraction** — Payee field now excludes currencies, categories, payment methods, bank names, dates, and prepositions, returning only the merchant name (e.g., "no frills" instead of "cad no frills, bmo credit card").
- **QuickAdd Modal Enhancements** — Improved the QuickAdd modal with the following updates:
  - Added a "Clear Parsed State" button to reset the parser-assisted flow.
  - Extracted parser submit and suggestion helpers for better modularity and testing.
  - Added focused UI tests for parser-assisted quick-add behavior, including parse mapping, suggestion selection, and create payload path.

- **Quick Add account selection and Neo card matching improvements** — Quick Add now uses explicit account dropdowns for From/To account selection, and parser account matching now retains card context in hints with token alias matching (`cc` ↔ `credit card`) so multi-token matches like "neo cc" rank above single-token matches (preventing cases like "neo cc" matching `Assets:Crypto:Neo` over Neo card liabilities).

- **Span-masking architecture for NLP parser** — Refactored the transaction parser to use non-destructive span tracking instead of text mutation, improving maintainability and auditability.
  - **Immutable source text** — Original input text is preserved throughout extraction; consumed regions are tracked via byte offset ranges instead of modifying the text.
  - **Ordered extraction pipeline** — Extractions happen in dependency order (date → amount → from account → to account → payee → narration), with each step recording which spans have been consumed.
  - **Unconsumed text fallback** — Each extraction step can access full source text if needed for context, but downstream steps skip previously consumed spans to prevent double-counting tokens.
  - **Span tracking data structures** — Added `Span` (start/end byte offsets) and `SpanMask` (source + consumed spans list) types with helper methods (`RecordSpan()`, `GetUnconsumedText()`).
  - **Span-aware extractors** — `extractDate()` and `extractAmount()` now use `FindStringSubmatchIndex()` to locate and record consumed byte ranges, preparing the pipeline for full span-masked extraction.
  - **5 new tests** — Comprehensive validation of span initialization, recording, unconsumed text extraction, and prevention of token double-counting across extraction steps. All 34 parser tests passing.

- **Compact bare-token matching for from/to accounts** — Improved hint fallback for keyword-less inputs (e.g., "15 inr icici hyd for shopping clothing") by extracting tokens directionally from account roots.
  - **From hint fallback** now considers only `Assets:` / `Liabilities:` token sets.
  - **To hint fallback** now considers only `Expenses:` / `Income:` token sets.
  - Prevents missing category matches such as `Expenses:Shopping` when explicit markers or "to"/"from" keywords are absent.

- **Payee extraction order fix for compact phrases** — Parser now prefers explicit merchant segments (`at <merchant>`) when initial payee extraction resolves to payment-method text.
  - Fixes cases like "20 cad from bmo cc for groceries at no frills" incorrectly returning "bmo credit card".
  - Both orderings now resolve payee consistently to "no frills":
    - "20 cad from bmo cc at no frills for groceries"
    - "20 cad from bmo cc for groceries at no frills"

- **Joint role-aware account matching** — Parser now selects account pairs using full remaining text plus role hints instead of relying only on explicit from/to phrase captures.
  - Scores from-account candidates from `Assets:`/`Liabilities:` (or `Income:` for income direction) and to-account candidates from role-appropriate account roots.
  - Blends role hint score with full-text score so compact phrases still map correctly when explicit markers are missing.
  - Prevents fragile behavior where only one side (from or to) is inferred from bare tokens.

- **Transfer phrase support (`transfer` and `xfer`)** — Parser now reliably handles compact transfer phrasing such as "transfer 20 cad from icici hyd to hdfc" and "xfer 20 cad from icici hyd to hdfc".
  - Added `xfer -> transfer` normalization.
  - Added explicit `from <account> to <account>` hint extraction.
  - Direction detection now considers full text context (not only extracted hints), improving transfer classification.

- **Quick Add parser create payload type fix** — Fixed parser-assisted Quick Add submission to coerce form values to strings before POSTing to `/api/parser/create-transaction`.
  - Prevents `400 Bad Request` errors like `cannot unmarshal number into go struct CreateParsedTransactionRequest` when parser-returned numeric amounts were sent back as JSON numbers.
  - Added frontend regression tests for numeric parser amount coercion in quick add parser utilities.

- **Epic 6: Account balance snapshots as-of date** — Assets balance and account detail flows now support historical as-of views for reconciliation.
  - **Subtask 6.1 (Backend – Date filter on balance endpoints)** — Added `as_of_date` (`YYYY-MM-DD`) support to `GET /api/assets/balance`, `GET /api/gain/:account`, and new `GET /api/account/:account/balance`. Date defaults to today, rejects invalid format/future dates with `400 INVALID_REQUEST`, and excludes postings after the selected date.
  - **Subtask 6.2 (Frontend – Date picker on balance pages)** — Added "View as of" date pickers on Assets → Balance and account detail pages; changing the date reloads balance data without page reload and displays the selected as-of date.
  - **Subtask 6.3 (Frontend – Historical balance trend view)** — Added account-level "View Trend" with 6M/12M presets, custom start/end date filters, responsive SVG trend line, and current-balance marker.

- **Granular progress reporting for sync jobs** — Users can now see per-commodity progress during long price sync operations instead of a static "Syncing…" spinner.
  - `Job` (worker package) gains `items_completed` and `total_items` integer fields, serialised as JSON and exposed via `GET /api/jobs/:id`.
  - `DetailedJobFn` now receives a thread-safe `progress func(completed, total int)` callback. `runDetailed` creates the callback and updates the job fields under the registry lock, eliminating any data race.
  - `syncCommodities` accepts a `progressFn func(completed, total int)` parameter and calls it after each commodity result is processed (the results loop is sequential, so the counter is exact).
  - The navbar `SyncingIndicator` switches from "Syncing…" to an "X / Y" label while the price-scraper stage is active.
  - The Sync History overlay shows a Bulma `<progress>` bar labelled "X of Y commodities" for any running job that has reported `total_items > 0`.
  - The TypeScript `Job` interface in `utils.ts` mirrors the new `items_completed` and `total_items` fields. The `runningJob` derived store (in `src/lib/stores/jobs.ts`) exposes the active non-terminal job for consumption by `SyncingIndicator`.

- **Year-over-Year "Until year" selector** — The Year-over-Year analysis page now includes an "Until year" dropdown alongside "Years to compare". Users can select an end year (e.g. 2025) so the comparison covers the N years up to and including that year (e.g. last 3 years until 2025 = 2023, 2024, 2025). Defaults to the current year, preserving existing behaviour. The `/api/expense` and `/api/income` endpoints now accept an optional `until_year` query parameter.

- **Month-over-Month analysis page (`/analysis/mom`)** — Added a dedicated Analysis view for MoM expense trends across multiple angles.
  - Selectable analysis window (6/12/24 months) and end month.
  - Breakdown switches for category, payee, and account views.
  - Monthly timeline table with MoM deltas and 3-month moving averages.
  - Top contributors with share-of-month and compact sparklines.
  - Additional insight cards for range highs/lows, biggest movers, and strongest 30-day momentum signals.

- **Incremental price sync (delta updates)** — `SyncCommodities` now performs incremental syncs instead of fetching and replacing the full price history on every run.
  - `PriceProvider.GetPrices` accepts a new `since time.Time` parameter. Providers use it to filter returned prices to those on or after the start-of-day of `since`; a zero value means fetch the full history (first run).
  - `syncCommodities` reads the `last_price_sync` metadata timestamp and forwards it to every provider as `since`, enabling incremental fetches after the first sync.
  - The sync API now accepts `force_prices: true` to bypass `last_price_sync` and fetch the full commodity price history on demand. The Prices page exposes this via a new **Force Refresh** action.
  - `UpsertAllByTypeNameAndID` now uses a pure UPSERT (INSERT … ON CONFLICT DO UPDATE) without first deleting existing rows. Historical prices are preserved across syncs; the same date's value is updated in place if the provider returns a corrected figure.
  - New `price.FilterSince(prices, since)` helper: filters a `[]*Price` slice to entries on or after the start-of-day of `since` (UTC). Zero `since` returns the slice unmodified.

- **Incremental journal sync with transaction-level change tracking** — `SyncJournal` now performs delta updates to the `postings` table instead of a full DELETE + INSERT on every sync run, significantly reducing I/O for frequent small journal edits.
  - **File-level hash skip** — before invoking the ledger CLI at all, `SyncJournal` computes a combined SHA-256 hash of all included journal files (`ledger.Cli().Files()`) and compares it against the value stored in `metadata` under `journal_hash`. If the hash matches the sync returns `SyncResult{Skipped: true}` immediately, eliminating all CLI and database work when nothing has changed.
  - **Transaction-level content hash** — each posting now carries a `transaction_hash` column (schema migration v9) containing a deterministic SHA-256 hash of the full set of postings belonging to the same `TransactionID`. Postings within a transaction are sorted by account name before hashing so that re-ordering within a transaction does not produce a spurious "changed" signal.
  - **`posting.DeltaUpsert`** — new function that replaces `posting.UpsertAll` in the sync path. It loads the existing `(transaction_id, transaction_hash)` pairs with a lightweight indexed query, classifies each incoming transaction as added / updated / removed / unchanged, and issues only the SQL writes that are necessary. Unchanged transactions — the common case when only one or two transactions are appended to a large journal — are skipped entirely.
  - **`force_journal: true` sync option** — the `POST /api/sync` request body now accepts `force_journal: true` to bypass both the file-level hash check and the transaction-level delta path and instead perform a full `DELETE all + INSERT all` replace. This mirrors the existing `force_prices` flag and is the recommended escape hatch when the `transaction_hash` index may be stale (e.g. after a manual DB edit, data import, or migration from an older build). `SyncJournal` also clears the cached journal hash before attempting the work so a subsequent ordinary sync will not silently skip.
  - **Post-sync actions are unaffected** — `cache.WarmCache`, `service.WarmXIRRCache`, and `account_balance.RefreshFromPostings` all operate on the full `postings` table after the delta write completes, so XIRR calculations, balance computations, and market-price cache warming always see a fully consistent picture regardless of whether an incremental or full-replace sync was used.
  - **`SyncResult` delta counters** — `SyncResult` now includes `PostingsAdded`, `PostingsUpdated`, `PostingsRemoved`, and `PostingsUnchanged` counts so operators can observe the incremental-sync efficiency at a glance.
  - **`posting.StampTransactionHash` / `posting.ComputeTransactionHash`** — exported helpers for stamping and computing per-transaction hashes, available for use in tests and future tooling.

#### Documentation

- **Reference documentation updated** — All new features from this release cycle are now documented in the reference section:
  - [Parser API](docs/reference/parser.md) — natural-language parse/create endpoints, override fields, confidence handling, readonly behavior, and error envelope.
  - [Journal](docs/reference/journal.md) — added Natural Language Quick Add section with parser workflow and link to parser API reference.
  - MkDocs navigation now includes parser docs under Reference.
  - [Dashboard](docs/reference/dashboard.md) — new page covering the recent-transactions widget, monthly cashflow widget, reconciliation widget, and account drill-down.
  - [Accounts](docs/reference/accounts.md) — account notes, account reconciliation (badges, dashboard widget, API), and account-level transaction history.
  - [Analysis](docs/reference/analysis.md) — Year-over-Year comparison charts (`/analysis/yoy`).
  - [Cash Flow](docs/reference/cash-flow.md) — 30-day spending trends with sparklines and the monthly cashflow widget.
  - [Commodities](docs/reference/commodities.md) — price export feature (Ledger / hLedger / Beancount formats, single-file and ZIP).
  - [Import](docs/reference/import.md) — import preview (dry-run), import presets (save / load / delete), and built-in presets.
  - [Sync](docs/reference/sync.md) — async sync with job polling, sync history overlay, price & journal freshness indicators, Firefly III webhook integration, and Firefly III reconciliation (Labs).
  - [Configuration](docs/reference/config.md) — `add_journal_path`, `enable_reconciliation`, `firefly` block, and `labs` block.

#### Performance

- **Parallel commodity price fetching in sync** — `SyncCommodities` now fetches price histories using a bounded worker pool (5 concurrent fetches) with goroutines and `sync.WaitGroup`, reducing end-to-end sync time when many commodities are configured while avoiding unbounded provider request fan-out.

- **Materialized account balance summary table** — Introduces `account_balances` table (migration v7) that stores pre-computed per-`(account, commodity)` balance totals updated atomically on every journal sync.
  - `account_balance.RefreshFromPostings(tx, postings)` — computes sums in-memory from the already-loaded postings slice and atomically replaces the entire `account_balances` table within the sync transaction. Forecast postings are excluded.
  - `account_balance.ByAccount(db, account)` — O(1) index lookup for a single account's balance rows, replacing the previous O(N) full `postings` table scan.
  - `account_balance.All(db)` — returns all materialized balance rows ordered by account and commodity.
  - `SyncJournal` now calls `RefreshFromPostings` inside the same DB transaction as `posting.UpsertAll`, guaranteeing the summary table is always consistent with the postings table.
  - Unit tests added for `RefreshFromPostings`, `All`, and `ByAccount` covering aggregation, forecast exclusion, idempotent replacement, and empty-slice clearing.

- **SQL-level aggregation for balance queries** — Introduces `query.GroupSum()` as a reusable SQL aggregation primitive and improves the `ComputeBreakdowns` algorithm.
  - `query.GroupSum()` — new method on `Query` that returns per `(account, commodity)` aggregated `SUM(amount)` and `SUM(quantity)` rows directly from the database, as a lightweight alternative to `All()` when only totals are needed.
  - `ComputeBreakdowns` — internal O(A × N) loop replaced with a two-phase O(N + A × C) approach: postings are first grouped by effective account in O(N), then each breakdown group collects from the pre-built index (O(A × C) where C is the number of distinct leaf accounts).
  - Unit tests added for `GroupSum` and `ComputeBreakdowns`.

#### New features

- **Epic 12: Account Reconciliation Status Tracker** — Added account reconciliation metadata storage, reconciliation status badges, and a dashboard reconciliation summary widget.
  - **Subtask 12.1 (Backend – Account Reconciliation Metadata)** — Added schema migration v6 with new `account_reconciliation` table (`account`, `last_reconciled_date`, `frequency_days`) and new APIs: `GET /api/accounts/reconciliation`, `GET /api/accounts/:account/reconciliation`, and `PATCH /api/accounts/:account/reconciliation`. Responses include computed `days_since` and `is_overdue`.
  - **Subtask 12.2 (Frontend – Reconciliation Status Badge)** — Assets balance rows and account detail pages now show color-coded reconciliation badges ("Last reconciled: ..."). Clicking a badge opens reconciliation actions on the account detail page.
  - **Subtask 12.3 (Frontend – Reconciliation Dashboard Widget)** — Dashboard now includes an "Account Reconciliation" widget with up-to-date/overdue counts and quick links to reconcile overdue accounts.
  - **Subtask 12.4 (Frontend & Backend – Optional Feature Toggle)** — Reconciliation is now disabled by default and can be enabled via `enable_reconciliation` in the configuration. All reconciliation-related UI components and API fetches are gated by this flag.
  - **Subtask 12.5 (Frontend – Navigation-free Reconciliation)** — Reconciliation badges now open a global modal in-place rather than navigating away, preserving page state across Dashboard, Assets, and Account detail pages.

- **Epic: Import feature improvements (Subtask 1)** — Added `POST /api/import/preview` to parse CSV content in dry-run mode and return row-by-row preview data with validation status/error messages before committing anything to journal files.
  - **Subtask 2 (Frontend preview table)** — Added reusable `ImportPreviewTable.svelte` with per-row valid/invalid badges, error messages, and include/exclude checkboxes (plus select-all) for import previews.
  - **Subtask 3 (Confirm-and-write flow)** — Import page now saves only selected preview rows via `POST /api/editor/save` and shows loading state while confirming/writing.
  - **Subtask 4 (Import preset CRUD API)** — Added `GET/POST/DELETE /api/import/presets` backed by a new SQLite model + migration for reusable import presets (name, column mappings, date format, default accounts, delimiter).
  - **Subtask 5 (Preset selector UI)** — Added `PresetSelector.svelte` on the import page with preset dropdown and “Save Current” action.
  - **Subtask 6 (Built-in presets)** — Added built-in import presets for common formats: Generic Bank CSV, Chase Credit Card CSV, SBI Account Statement CSV, and ICICI Credit Card CSV.

- **Epic 11: Year-over-Year Comparison Charts** — Added backend multi-year series support and a new YoY analysis experience for comparing spending and income trends across calendar years.
  - **Subtask 11.1 (Backend – Multi-Year Expense/Income Data)** — `GET /api/expense` and `GET /api/income` now accept an optional `years` query parameter (default `1`, max `10`) and return `multi_year` data shaped as `{ "<year>": { month: { "YYYY-MM": amount }, total } }`. Leap-day transactions are naturally included in February aggregates.
  - **Subtask 11.2 (Frontend – YoY Comparison Chart)** — Added reusable `YoYChart.svelte` component supporting line and grouped-bar modes with Jan–Dec alignment, year legends, and month-level hover tooltips across all selected years.
  - **Subtask 11.3 (Frontend – YoY Analysis Page)** — Added `/analysis/yoy` page with configurable year range (2–5 years), spending + income YoY charts, category YoY chart (line/bar toggle), computed YoY insights, and CSV export.

- **Epic 4: Month-over-Month Spending Trends** — Added rolling 30-day spending trend comparison to the Monthly Expenses page. Each expense category now shows its current 30-day total alongside the previous 30-day total, a variance amount, a colour-coded percentage change (↑ red / ↓ green), and a 6-month sparkline bar chart.
  - **Subtask 4.1 (Backend – Monthly Expense Trends)** — `GetExpense()` in `internal/server/expense.go` now computes and returns a `trends` array. Each entry contains `category`, `current_month`, `previous_month`, `variance`, and `variance_pct` (null when there are no previous-period expenses). Rolling windows: today−30 to today (current) and today−60 to today−30 (previous). `Expenses:Tax` postings are excluded. Six unit tests added in `internal/server/expense_test.go`.
  - **Subtask 4.2 (Frontend – Trend Indicators)** — New `ExpenseTrendCard.svelte` component renders the category name, current amount, previous amount, and a coloured arrow + percentage badge. The Monthly Expenses page now displays a "30-Day Spending Trends" grid below the calendar using this component.
  - **Subtask 4.3 (Frontend – Monthly Trend Sparkline)** — New `SparklineChart.svelte` component renders a compact SVG bar chart for up to 6 months of per-category spending history. The last bar (current month) is highlighted in the category colour; a dashed red average line is overlaid. Sparklines are embedded inside each `ExpenseTrendCard` when more than one month of history is available.

- **Epic 3: Simple Account Notes / Metadata** — Users can now attach free-text notes to any account (e.g. "Emergency fund", "Company 401k") without editing the ledger file.
  - **Subtask 3.1 (Backend – DB)** — New `account_notes` SQLite table with a unique index on `account` name. Added as schema migration v4 via `internal/model/account_note` package.
  - **Subtask 3.2 (Backend – API)** — Four new REST endpoints: `GET /api/account_notes` (list all notes), `GET /api/account_notes/:account` (fetch one note), `POST /api/account_notes/upsert` (create/update), `POST /api/account_notes/delete` (remove). All write endpoints require the `ReadonlyMiddleware` guard.
  - **Subtask 3.3 (Frontend – Detail Page)** — New `/accounts/[name]/` overview page with a textarea for composing and saving a note, plus a delete button. Notes are persisted immediately via the new API.
  - **Subtask 3.4 (Frontend – List Widget)** — The `/accounts/[name]/transactions` page now fetches and displays an existing note inline in the page header as a highlighted tag, and includes a "Notes" button to navigate to the notes editor.

- **Epic 2: Recent Transactions Widget on Dashboard** — Added a feed of the 15 most recent transactions to the main dashboard for at-a-glance activity overview.
  - **Subtask 2.1 (Backend)** — `GET /api/transaction` now accepts optional `limit` and `offset` query parameters for server-friendly pagination. Both parameters are applied at the transaction level (after grouping postings) so every returned transaction includes all of its postings.
  - **Subtask 2.2 (Frontend)** — New `RecentTransactionsWidget.svelte` component encapsulates the recent-transactions feed. It accepts a `transactions` prop and an optional `limit` prop (default 15) and renders each entry via `TransactionCard`.
  - **Subtask 2.3 (Frontend)** — Dashboard (`+page.svelte`) now uses `RecentTransactionsWidget` instead of inline transaction rendering, keeping the layout clean and the widget reusable.

- **Account-Level Transaction Drill-Down** — Users can now click on any account balance widget on the dashboard to view a filtered transaction history for that specific account. A new route `/accounts/[name]/transactions` displays all transactions touching the selected account. The `GET /api/transaction` endpoint now accepts an optional `account` query parameter to return only transactions for the given account prefix.

- **Price & Journal Freshness Tracking** — Added visual indicators to the navigation bar to track the freshness of your financial data. The "Update Prices" icon turns amber after 24 hours and red after 48 hours. The "Sync Journal" icon turns amber if any journal files have been modified since the last sync.

- **Firefly III Webhook Integration** — Added a dedicated `/api/webhooks/firefly` endpoint that automatically parses and imports transactions from Firefly III via webhooks. Supported transactions are appended to the configured `add_journal_path` journal file.

- **Firefly III Reconciliation (Labs)** — Introduced a new balance reconciliation tool to compare Paisa (Ledger) account balances with Firefly III data. The feature is hidden by default and can be enabled under the new "Labs" configuration section. Includes support for ignoring specific accounts and case-insensitive account matching.

- **Price Export Feature** — Users can now export their commodity price history in Ledger, hLedger, or Beancount formats. The export supports filtering by commodity and provides options for single-file or multi-file (ZIP) output.

- **Dynamic Multi-Currency Reporting** — Improved the Income Statement and Market services to use period-end or dynamic exchange rates for foreign currency transactions, ensuring more accurate marked-to-market financial reporting across different timeframes.

- **Monthly Cashflow Widget** — Added a new multi-currency monthly cashflow breakdown widget to the dashboard. The widget provides a high-level view of income vs. expenses using original ledger quantities.

- **Advanced Journal Configuration** — Added `add_journal_path` to the configuration, allowing users to specify a dedicated journal file for transactions added via the API or webhooks, keeping the main journal file clean.

- **Mobile & PWA Enhancements** — Implemented a responsive navigation layout with a mobile-optimized side menu. Improved PWA support with better icon sizing, manifest validation, and consistent theme switching across mobile devices.

- **Assets Balance flat view + exports** — Added a Flat Accounts toggle on the Assets → Balance page and support for exporting the current view to CSV or Excel. Exports now respect the selected display mode (hierarchy or flat), and `/api/assets/balance` accepts `flat=true` to return non-rollup account rows.

- **Epic: Refining Paisa Configuration Interface** — Redesigned the configuration screen to improve navigation and layout. Transitioned from a monolithic JSON form to a structured, sidebar-based interface with categorized sections for better discoverability and user experience.

- **Epic 2: Architectural Alignment & Cleanup** — Finalized structural patterns and removed legacy Svelte 4 compatibility:
  - **Snippet Transition (2.1)** — `Modal.svelte` migrated from named `<slot>` elements to typed snippet props (`head`, `body`, `foot`). All consumers updated to use `{#snippet head(close)}...{/snippet}` blocks: `FileModal`, `PriceCodeSearchModal`, `DiffViewModal`, `SyncHistoryOverlay`, and the import page inline modal.
  - **Callback Prop Migration (2.2)** — `createEventDispatcher` removed from `FileModal.svelte` (replaced with `onsave` callback) and `PriceCodeSearchModal.svelte` (replaced with `onselect` callback). All four `FileModal` call-sites and `JsonSchemaForm`'s `PriceCodeSearchModal` usage updated to pass callback props directly.
  - **Global Store Evolution (2.3)** — New `src/lib/state/ui.svelte.ts` (`UIState`) and `src/lib/state/persisted.svelte.ts` (`PersistedState`) class-based wrappers created using `fromStore`. Components can now access stores via `uiState.<prop>.current` or continue using the existing `$store` syntax.
  - **Lifecycle & Context Update (2.4)** — `Navbar.svelte` `onMount`/`onDestroy` lifecycle hooks replaced with `$effect` (cleanup via returned function). Root `+layout.svelte` and `(app)/+layout.svelte` updated from `<slot />` to `{@render children()}` with typed `Snippet` prop.
  - **Final Switch-Over (2.5)** — `componentApi: 4` compatibility shim removed from `svelte.config.js`. All remaining Svelte 4 deprecation warnings resolved: `Dropzone`, `Spinner`, `ZeroState`, `PostingGroup` slots converted to snippet props; `JsonSchemaForm` `<svelte:self>` replaced with self-import; `LastNMonths` `options` array made `$derived`; `MonthPicker` `selectedYear` initialised from raw prop value; `ThemeSwitcher` initial store call decoupled from reactive variable. `isBurger` in `(app)/+layout.svelte` declared with `$state()`.
  - **prettier-plugin-svelte** bumped to `^3.3` to support `{@render ...}` syntax in formatting.
  - `svelte-check` now reports **0 errors and 0 warnings** across the entire frontend.

- **Epic 1: Component Modernization (Runes & Event Syntax)** — Systematically migrated all Svelte components and route pages from Svelte 4 syntax to Svelte 5 runes:
  - All `export let` props converted to `$props()` / `$bindable()`
  - All `$:` reactive statements converted to `$derived()` or `$effect()`
  - All mutable local state annotated with `$state()`
  - All DOM event attributes (`on:click`, `on:change`, `on:keydown`, etc.) replaced with lowercase equivalents (`onclick`, `onchange`, `onkeydown`, etc.)
  - `on:*|preventDefault` modifiers inlined as `e.preventDefault()` calls
  - `createEventDispatcher` replaced with callback props (`ondrop`, `onselect`, `onpreview`, `onsave`)
  - All callers of modernized components updated to pass callback props instead of `on:event` listeners

- **Svelte 5 runes migration (P2.3)** — Converted 15 remaining `src/lib/components/` files and 5 route pages from Svelte 4 syntax to Svelte 5 runes: `export let` → `$props()` / `$bindable()`, `$:` derived expressions → `$derived()`, `$:` side-effect blocks → `$effect()`, and mutable local state → `$state()`. `createEventDispatcher` in `BulkEditForm`, `DiffViewModal`, and `FileTree` replaced with callback props (`onpreview`, `onsave`, `onselect`); all callers updated. `<svelte:self>` in `FileTree` replaced with an explicit self-import. Goals pages (`savings`, `retirement`) converted all `onMount`-assigned variables to `$state()` for correct reactivity.

- **Svelte 5 upgrade & UI modernization (P2.2)** — Upgrades the frontend framework to Svelte 5 and begins the incremental migration to rune-based reactivity:
  - **Svelte 5** (`^5.0.0`), **svelte-check** (`^4.0.0`), **@sveltejs/vite-plugin-svelte** (`^4.0.0`), and **eslint-plugin-svelte** (`^3.0.0`) bumped in `package.json` (#226).
  - **Svelte 4 compatibility layer** enabled in `svelte.config.js` via `compilerOptions.compatibility.componentApi: 4`, allowing all existing components to keep working while new ones adopt runes (#227).
  - **Modal.svelte migrated to Svelte 5 runes** — props use `$props()` / `$bindable()`; internal state uses `$state()`; `on:click` replaced with `onclick`. Bulma structural classes (`modal`, `modal-background`, `modal-card`, `modal-card-head/body/foot`, `is-active`) replaced with DaisyUI equivalents (`du-modal`, `du-modal-box`, `du-modal-backdrop`, `du-modal-open`) and Tailwind flex utilities (#228 #229).
  - **BoxedTabs.svelte migrated to Svelte 5 runes** — `export let` replaced with `$props()` / `$bindable()`; `$:` block replaced with `$effect()`; `on:click` replaced with `onclick` (#229).
  - **AccountTree.svelte** — new Svelte 5 rune-based component that renders a hierarchical `AccountNode[]` tree with keyboard-accessible expand/collapse and single-node selection via a `$bindable` `selected` prop. Depth-limited auto-expansion (first two levels open by default) and focus ring via DaisyUI/Tailwind utilities (#230).
  - **Modal callers updated** — `FileModal`, `DiffViewModal`, `PriceCodeSearchModal`, and `SyncHistoryOverlay` updated to use DaisyUI button classes (`du-btn`), Tailwind layout utilities, and the new `onclick` event syntax.

- **Local JSON price provider** — A new built-in price provider (`local-json`) allows users to maintain custom commodity prices in a plain JSON file on the local filesystem. No network access is required; the file is read on every price-update run. The code field in `paisa.yaml` is the path to the JSON file (absolute, or relative to the config directory). See [Commodities](docs/reference/commodities.md) for the full JSON format specification and examples.
- **Extensible PriceProvider interface** — `internal/model/price.PriceProvider` is now fully documented with explicit return-value semantics for every method, making it straightforward to implement a custom provider. Compile-time interface-satisfaction checks (`var _ price.PriceProvider = ...`) have been added to every built-in provider package to catch drift early.

#### Bug fixes

- **Journal sync posting writes optimized** — `posting.UpsertAll` now performs replacement
  inserts using GORM `CreateInBatches` instead of one-row-at-a-time inserts, reducing sync
  time on large journals while preserving atomic replace behavior.

- **Error toast close button fixed** — Removed an unbound inline `delete` button from the global runtime error toast markup in `src/hooks.client.ts` and now rely on Bulma Toast's built-in dismiss control (`dismissible: true`). The visible close X now dismisses error boxes correctly.

- **Config mobile sidebar auto-close on selection** — On mobile widths, choosing a section in More → Configuration now automatically collapses the sidebar overlay so the selected section content is immediately visible.

- **Regression: Configuration schema descriptions** — Updated all regression test fixtures to match the latest configuration schema metadata. This ensures that intentional improvements to configuration documentation (like detailed tooltips and hover text) don't break the regression suite.

- **Income Statement start/end balance fixed** — `startingBalance` and `endingBalance` are now computed directly from the actual balance sheet (assets at market value + liabilities at book value), mirroring how the Networth page works. Previously they were derived from an income-flow reconstruction formula that had two bugs: (1) liabilities were negated with the wrong sign, and (2) the current fiscal year's `endingBalance` was priced at the FY end date (a future date) rather than today. The new approach guarantees that the End Balance row and the Networth page always agree for the current year, and correctly handles liability-funded transactions.

- **Navbar effect recursion fixed** — Reworked breadcrumb/nav selection in `src/lib/components/Navbar.svelte` to resolve selection from the current path in a pure helper (`src/lib/navbar_selection.ts`) instead of reading and mutating the same reactive state inside one `$effect`. This prevents startup/runtime `effect_update_depth_exceeded` crashes on built apps.

- **Embedded static asset routing fixed** — Updated `internal/server/server.go` to serve root-level PWA assets (`/manifest.webmanifest`, `/sw.js`, Workbox files, and `/pwa-*.png`) from embedded `web/static` instead of falling through `NoRoute` to `index.html`. This resolves `Manifest: Line: 1, column: 1, Syntax error` when using the Go-served app.

- **Dashboard startup loop fixed** — Updated `src/routes/(app)/+page.svelte` to remove a read-after-write reactive pattern in the dashboard `$effect` (`selectedExpenses`), preventing a recursive update cycle that could raise `effect_update_depth_exceeded` on load when the current month has no expense bucket.

- **Manifest endpoint syntax error fixed** — Added `static/manifest.webmanifest` so `/manifest.webmanifest` serves valid JSON instead of fallback HTML, resolving browser console errors like `Manifest: Line: 1, column: 1, Syntax error`.

- **Startup effect loop hardening** — Updated `src/lib/components/Actions.svelte` to stabilize the `obscure` store subscription (track previous value correctly and unsubscribe on destroy), preventing repeated `refresh()` cascades that could trigger `effect_update_depth_exceeded` on app load.

- **Assets Analysis render crash fixed** — `src/routes/(app)/assets/analysis/+page.svelte` now initializes the commodity color mapper with a safe fallback function and explicit callable type so the page no longer throws `TypeError: ... is not a function` during first render while async data is still settling.

- **Negative SVG width warnings fixed** — Clamped stacked bar segment widths in `src/lib/gain.ts` and `src/lib/liabilities/interest.ts` to `Math.max(0, ...)` to avoid invalid `<rect width>` values caused by floating-point precision drift.

- **Interest chart null-safety hardening** — `src/lib/liabilities/interest.ts` now guards missing DOM containers and empty timelines, uses safe fallbacks for current overview values, and aligns D3 tick formatter typing with strict TypeScript checks without changing chart behavior.

- **Frontend check dependency resolution** — Installed missing Node packages from `package.json` so generated Connect/Protobuf client modules resolve correctly during `npm run check`.

- **AccountTree compatibility fix** — `src/lib/components/AccountTree.svelte` was migrated from rune-specific APIs to compatible `export let` props + reactive state/effects, restoring type-safe recursive bindings and removing compile errors.

- **Mobile navbar burger alignment fix** — Updated `src/lib/components/Navbar.svelte` to enforce left-aligned burger placement on mobile (`.navbar-brand .navbar-burger.mobile-drawer-toggle`) and replaced invalid self-closing non-void tags in the navbar markup to avoid ambiguous render behavior.

- **UI warning cleanup (actions/sync modal)** — Updated `src/lib/components/SyncHistoryOverlay.svelte` and `src/lib/components/Actions.svelte` to remove ambiguous self-closing non-void icon tags, and improved `src/lib/components/Modal.svelte` backdrop semantics by using a real button instead of a clickable label.

- **Global non-void self-closing cleanup** — Applied a repo-wide Svelte markup normalization pass replacing ambiguous self-closing non-void HTML tags (for example `<i />`, `<div />`, `<span />`) with explicit opening/closing tags across `src/**/*.svelte`.

- **Closeable runtime error toasts** — Updated global client error handling in `src/hooks.client.ts` to show dismissible bottom-right toasts with sanitized stack/message content instead of center-screen modal-style error blocks, so users can always close or ignore transient errors.

- **Remaining Svelte accessibility warnings cleared** — Replaced lingering click-only anchors with semantic buttons, added missing labels to icon-only controls, fixed a remaining self-closing non-void table body, and restored `src/lib/components/BoxedTabs.svelte` to compatible classic Svelte props/reactivity so `svelte-check` now reports 0 errors and 0 warnings.

- **Repo line endings normalized** — Added a repository-level `.gitattributes` policy to keep source files on LF across platforms (with Windows-native script exceptions), preventing recurring Windows/Linux newline churn that was causing `gofmt -l .` lint failures from formatting-only diffs.

- **Client error toast null-safety + mobile navbar action overflow fix** — Hardened `src/hooks.client.ts` to safely format `null`/non-Error runtime exceptions (preventing `Cannot read properties of null (reading 'stack')` while rendering error toasts), updated `src/store.ts` to allow nullable `accountTfIdf` initialization, and adjusted mobile navbar action placement so top-right controls no longer get truncated on narrow screens.

- **Responsive navbar action placement fix** — Updated `src/lib/components/Navbar.svelte` so hamburger-layout breakpoints remain consistent up to tablet width (preventing burger drift before desktop menu switch) and added top-right action icons (`SyncingIndicator`, theme toggle, and `Actions`) in hamburger mode while hiding duplicate drawer-end actions.

- **Docker build fix** — Replaced `svelte-file-dropzone` (incompatible with Svelte 5) with a local self-contained `Dropzone.svelte` component in `src/lib/components/`. The local component matches the same API (`multiple`, `accept`, `inputElement` props; dispatches `drop` event with `{ acceptedFiles, fileRejections }`).

- **Dashboard crash on Svelte 5 fixed** — Removed `@egjs/svelte-grid` usage from dashboard and credit-card detail routes and replaced it with native CSS grid wrappers. This avoids runtime failures like `TypeError: Class constructor ... cannot be invoked without 'new'` caused by legacy class-based Svelte components during route hydration.

- **Frontend build emits sourcemaps** — Enabled `build.sourcemap` in `vite.config.js` so minified hashed chunks can be mapped back to original source during debugging.

- **Mobile navbar accessibility fix** — Closing the burger menu now blurs any focused descendant before hiding the menu, and the closed menu is marked `inert` to prevent focus from remaining inside hidden navigation content.

- **PWA cache/update hardening** — Switched to `registerType: "autoUpdate"`, enabled `skipWaiting`, `clientsClaim`, and `cleanupOutdatedCaches`, and disabled dev-time service worker registration to reduce stale asset/manifest mismatches during local development.

- **Manifest and dashboard action robustness** — Updated the app manifest link to use `%sveltekit.assets%/manifest.webmanifest` and changed the dashboard "Setup Demo" trigger to a semantic `<button>`.

- **Theme toggle duplication on desktop fixed** — The navbar theme switcher was appearing twice on desktop (in both `mobile-top-actions` and `menu-actions-row`) due to missing CSS specificity. Added `display: none !important` to `mobile-top-actions` to ensure it only shows on mobile (≤1023px breakpoint), and explicitly set `display: flex` for `menu-actions-row` to show it on desktop.

- **MonthPicker non-selected month styling improved** — Non-selected month buttons in the month picker dropdown now have a subtle hover background and outline, making them appear clearly interactive and clickable instead of plain text. Selected months remain highlighted in blue with bold text.

- **Local stale-chunk recovery** — Added a client bootstrap safeguard that, on local/dev hosts, unregisters existing service workers, clears Cache Storage, and performs a one-time reload to avoid mixed old/new chunk runtime errors such as `TypeError: ... is not a function`.

- **AutoComplete no longer crashes the server** — The `in-mfapi` (MF API) and `com-purifiedbytes-nps` providers previously called `log.Fatal` when the autocomplete cache could not be populated, killing the server process. They now log the error at `Error` level and return an empty suggestion list instead.

- **Typed API via Protobuf/Connect (P3.1)** — Eliminates hand-maintained TypeScript interfaces and `ajax` fetch wrappers for selected endpoints by driving the API contract from a `.proto` schema, giving compile-time type safety on both sides.
  - `proto/api.proto` — defines `Transaction`, `Posting`, `AccountBalance`, `AccountNode`, and the `PaisaService` service with a `GetAccountTree` RPC. Package name `paisa.v1`; Go package `github.com/ananthakumaran/paisa/internal/gen/paisa/v1;paisav1` (#234).
  - `internal/gen/` — committed generated Go stubs (`api.pb.go`, `paisav1connect/api.connect.go`) produced by `protoc-gen-go` + `protoc-gen-connect-go` (#235).
  - **Connect-Go integrated into Gin** — `connectrpc.com/connect` is added as a dependency; `PaisaService` endpoints are mounted at `/connect/paisa.v1.PaisaService/…` alongside the existing REST routes. `TokenAuthMiddleware` now protects `/connect/` paths with the same session-token logic as `/api/` paths (#235).
  - **`GetAccountTree` endpoint** — `internal/server/connect_service.go` implements `PaisaServiceHandler.GetAccountTree`, which converts the flat `[]string` from `accounting.AllAccounts` into a hierarchical `AccountNode` tree sorted alphabetically at each level. Intermediate-only nodes (not direct leaf accounts) have `is_leaf: false` (#236).
  - `src/lib/gen/api_pb.ts` — committed generated TypeScript types (produced by `protoc-gen-es@2.x`). Regenerate with `npm run generate:proto` or `make proto` (#237).
  - **`src/lib/connect_client.ts`** — typed `PaisaService` Connect transport client. Uses `createConnectTransport` from `@connectrpc/connect-web` targeting `/connect`; injects the same `X-Auth` session token as the REST `ajax` helper (#237, #238).
  - **`src/lib/account_tree.ts`** — `fetchAccountTree()` calls `paisaClient.getAccountTree()` with the same `loading` store behaviour as `ajax`; `flattenAccountTree()` converts the tree back to a flat `string[]` for backward-compatible consumption (#238).
  - `make proto` / `npm run generate:proto` — regeneration targets for both backend and frontend stubs (#237).

- **Sync history/log overlay** — A "Sync History" button (clock-rotate-left icon) is added to the header action bar. Clicking it opens a modal overlay that lists all known background jobs in reverse-chronological order, showing status badge (colour-coded: success/danger/info/warning), truncated job ID, created-at timestamp, started/finished timestamps, wall-clock duration, error message snippet (for failed jobs), and an expandable list of per-step `details`. A "Clear history" button resets the in-memory jobs store. The overlay can be opened and closed at any time without interrupting an in-progress sync. A count badge on the history button shows the total number of tracked jobs (P1.3).
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
