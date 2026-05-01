# CHANGELOG

### Unreleased — Future changes

#### New features

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
