# Paisa – Copilot Instructions

## Project Overview
Paisa is a personal finance manager that wraps [ledger-cli](https://ledger-cli.org/), [hledger](https://hledger.org/), or [Beancount](https://beancount.github.io/) as its accounting engine. The Go backend parses plain-text journal files via one of those CLIs, stores results in SQLite, and serves a SvelteKit SPA via a REST API.

## Architecture

```
ledger file → ledger CLI (ledger/hledger/beancount) → SQLite (via GORM) → Gin REST API → SvelteKit SPA
```

- **`cmd/`** – Cobra CLI entry points (`serve`, `update`, `init`, `version`)
- **`internal/`** – All backend logic; never imported from outside the module
  - **`ledger/`** – `Ledger` interface with three implementations (`LedgerCLI`, `HLedgerCLI`, `Beancount`); `ledger.Cli()` selects based on `paisa.yaml`
  - **`model/`** – GORM models (`posting.Posting` is the central entity); `model.SyncJournal()` is the main ingestion pipeline
  - **`query/`** – Fluent builder for DB queries on postings (e.g., `query.Init(db).LastNMonths(3).All()`)
  - **`accounting/`** – Domain logic (balances, XIRR, grouping) that operates on `[]posting.Posting` slices in-memory
  - **`server/`** – Gin router, one file per route group; all mutating endpoints go through `ReadonlyMiddleware`
  - **`scraper/`** – Price provider plug-ins (Yahoo, MutualFund API, NPS, Metal, AlphaVantage)
  - **`cache/`** – Clears in-memory caches across packages before every sync
  - **`config/`** – Loads/validates `paisa.yaml`; use `config.GetConfig()` everywhere
- **`src/`** – SvelteKit frontend
  - **`src/lib/utils.ts`** – Mirrors Go structs as TypeScript interfaces (e.g., `Posting`); keep both in sync when changing fields
  - **`src/store.ts`** – Svelte stores for shared UI state (editor state, date range, month)
  - **`src/routes/(app)/`** – Protected SPA routes; `login/` is the only public route
- **`web/`** – Embeds compiled frontend assets into the Go binary via `//go:embed`

## Developer Workflows

| Task | Command |
|---|---|
| Full dev mode (Go + Vite with HMR) | `make develop` |
| Go only (hot-reload via nodemon) | `make serve` |
| Frontend build (watch) | `make watch` |
| Run all tests | `make test` |
| JS unit tests only | `bun test --preload ./src/happydom.ts src` |
| Go tests only | `go test ./...` |
| Regression tests (spawns real server) | `unset PAISA_CONFIG && TZ=UTC bun test tests` |
| Regenerate regression fixtures | `unset PAISA_CONFIG && REGENERATE=true TZ=UTC bun test tests` |
| Rebuild Lezer parsers | `make parser` |

- Default server port is **7500**; override with `--port` or `-p`.
- Set `PAISA_DEBUG=true` to enable GORM SQL logging.
- `--now <YYYY-MM-DD>` freezes `utils.Now()` for deterministic testing (used in regression tests and `make serve-now`).

## Key Patterns

### Sync Pipeline
`POST /api/sync` → `cache.Clear()` → `model.SyncJournal(db)` (runs ledger CLI, upserts postings/prices to SQLite) → handlers recompute from DB. Always invalidate the cache before reading fresh data.

### Security & Authentication
- All `/api/*` routes (except `/api/auth/login`) require an `X-Auth` header validated by `TokenAuthMiddleware`.
- **Session tokens** (UUID, no colon): validated via `session.FindByToken(db, token)` with a 24-hour TTL. Tokens are stored in the `sessions` SQLite table.
- **Legacy auth** (`username:password` with colon in `X-Auth`): only enabled when `config.AllowLegacyAuth = true`.
- Passwords are stored as `sha256:<hex>` in `paisa.yaml`; always use `crypto/subtle.ConstantTimeCompare()` for comparison.
- Write endpoints are additionally guarded by `ReadonlyMiddleware` (rate limit: 6 req/min per user, 3-request burst via `throttled`).
- Login: `POST /api/auth/login` → returns UUID session token; Logout: `POST /api/auth/logout` → deletes the session.

### Error Handling
All API errors must use the standardized envelope from `internal/server/apierror.go`:
```json
{ "error": { "code": "<ErrorCode>", "message": "<human-readable text>" } }
```
Use the provided helpers (never construct raw error JSON manually):
- `AbortWithError(c, status, code, message)` – writes error + aborts handler chain
- `RespondError(c, status, code, message)` – writes error without aborting
- `BindJSONOrError(c, dst)` – binds JSON body, returns `false` and sends 400 on failure

Defined error codes: `INVALID_REQUEST` (400), `INTERNAL_ERROR` (500), `UNAUTHORIZED` (401), `TOO_MANY_REQUESTS` (429), `READONLY` (write-blocked).

### Adding a New API Endpoint
1. Add handler file in `internal/server/` (follow existing files like `budget.go`).
2. Register the route in `internal/server/server.go` in `Build()`.
3. Use `writeGroup.POST(...)` for any mutating endpoint (enforces `ReadonlyMiddleware`).
4. Return errors via `RespondError(c, status, ErrCode..., message)`.

### Adding a Price Provider
Implement `price.PriceProvider` interface and register in `scraper/scraper.go`:`GetAllProviders()` and `GetProviderByCode()`.

### Frontend ↔ Backend Contract
- TypeScript interfaces in `src/lib/utils.ts` mirror Go structs — update both together.
- API calls use the `ajax` helper from `src/lib/utils.ts` which handles loading state and error toasts.
- Dates from the API are ISO strings; always parse with `dayjs(...)` on the frontend.

### Configuration
- User config lives in `paisa.yaml` (path resolved via XDG or `--config` flag).
- Schema is validated with JSON Schema; `config.GetSchema()` returns the embedded schema for the UI.
- Multi-currency and multi-ledger-dialect support is gated on `config.GetConfig().LedgerCli`.

### Desktop (Wails) Build
The `desktop/` directory wraps the same Go backend as a native app using [Wails v2](https://wails.io/).
- **Entry point**: `desktop/main.go` — passes `server.Build(&app.db, false).Handler()` directly as the Wails `AssetServer`, so the identical Gin router handles all API calls; no separate HTTP port is opened.
- **Startup**: `desktop/app.go` `App.startup()` calls `cmd.InitConfig()` and `utils.OpenDB()`, mirroring `cmd/serve.go` but without Cobra.
- **Dev mode**: run `wails dev -tags webkit2_40` from `desktop/`; for production use `wails build -tags webkit2_40`.
- **GPU policy** on Linux: controlled by `PAISA_GPU_POLICY` env var (`always` | `never` | `ondemand`); defaults to `never` due to a Wails/WebKit2 issue.
- The desktop build does **not** use `//go:embed` in `web/`; assets are served live by Wails's asset server during dev.

### Import Templates
Bank/broker statements (CSV, XLS, XLSX, PDF) are converted to ledger entries entirely in the browser using Handlebars templates.
- **Pipeline**: `spreadsheet.ts`:`parse()` → rows as `Record<A-Z, string>` (columns mapped to letters) → Handlebars template renders ledger text.
- **Templates**: stored in `internal/model/template/templates/*.handlebars` (also user-configurable via `paisa.yaml` `import_templates`); each row is exposed as `ROW` with column letter keys (`ROW.A`, `ROW.B`, …).
- **Helpers** (`src/lib/template_helpers.ts`): custom Handlebars helpers including `amount`, `predictAccount` (TF-IDF cosine similarity against existing accounts), `isDate`, `negate`, `round`, `eq`, `gt`, `lt`, etc.
- **Adding a helper**: export it from `template_helpers.ts`; it is automatically registered in both the editor (`handlebars_parser.ts` highlights it) and the Handlebars runtime.
- **Tests**: `src/lib/import.test.ts` reads fixtures from `fixture/import/<template-name>/` — one CSV/XLS input paired with an expected `.ledger` output.

## Code Style
- **Go**: formatted with `gofmt` (enforced by `make lint`). No custom linter config — keep code `gofmt`-clean.
- **TypeScript / Svelte**: Prettier with `printWidth: 100` and `trailingComma: "none"`. Run `./node_modules/.bin/prettier --write src` to auto-format.
- **ESLint**: `@typescript-eslint/recommended` + `plugin:svelte/recommended`; `@typescript-eslint/no-explicit-any` is disabled (use of `any` is allowed).
- Run `make lint` to validate all style rules before committing.

## Testing Conventions
- **Regression tests** (`tests/`) spawn a real `./paisa serve` binary against fixture directories in `tests/fixture/` (one per currency/dialect: `inr`, `eur`, `inr-hledger`, `eur-hledger`, `inr-beancount`). They compare API responses against stored JSON snapshots; run with `REGENERATE=true` to update snapshots.
- **Go unit tests** live alongside source files (`*_test.go`); run with `go test ./...`.
- **JS unit tests** use Bun's test runner; files end in `.test.ts` under `src/lib/`.
