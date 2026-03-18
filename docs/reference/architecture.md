---
description: "System architecture of Paisa across CLI, web, and desktop modes"
---

# Architecture

## Overview

Paisa is a personal finance application built around double-entry ledger files, with three execution modes:

1. CLI mode for operations like `serve`, `update`, and `init`.
2. Web mode that serves a browser SPA and JSON API.
3. Desktop mode (Wails) that embeds the same backend API in a native shell.

The core architecture is a modular monolith:

1. A Go backend in `internal/` for domain logic, data access, and API handlers.
2. A SvelteKit frontend in `src/` that consumes backend APIs.
3. SQLite persistence managed through GORM.
4. Ledger CLI integration for parsing and validating source journal files.

## Runtime Entry Points

### CLI entry

- Binary entrypoint: `paisa.go`
- Command composition: `cmd/root.go`
- Operational commands:
  - `cmd/serve.go` starts HTTP server.
  - `cmd/update.go` syncs journal, prices, and portfolios.
  - `cmd/init.go` creates starter/demo data.

### Web server entry

- HTTP router build and API registration: `internal/server/server.go`
- Listener startup: `internal/server/server.go` (`Listen`)
- Static asset embedding: `web/web.go`

### Desktop entry

- Wails bootstrap: `desktop/main.go`
- Desktop app lifecycle and DB open/migrate: `desktop/app.go`
- The desktop asset server uses the same backend router from `internal/server/server.go`.

## Layered Structure

### Presentation Layer

- Frontend routes and UI: `src/routes/`, `src/lib/`
- API calls and auth token handling: `src/lib/utils.ts`
- App-level state stores: `src/store.ts`

### API Layer

- Gin router and middleware: `internal/server/server.go`
- Domain handlers grouped by concern:
  - Dashboard, cash flow, budget, net worth, income/expense.
  - Ledger editor and sheet editor.
  - Portfolio/prices/goals/credit cards.

### Domain and Service Layer

- Accounting primitives and account views: `internal/accounting/`
- Capital gains, market price and other business workflows: `internal/service/`
- Taxation and forecasting logic: `internal/taxation/`, `internal/prediction/`, `internal/xirr/`

### Data Access Layer

- DB open and helper utilities: `internal/utils/utils.go`
- Models and sync orchestration: `internal/model/model.go` and related `internal/model/*`
- Query helpers: `internal/query/`

### Integration Layer

- Ledger engine wrapper: `internal/ledger/`
- External price/portfolio sources: `internal/scraper/`

## High-Level Data Flow

1. User action in frontend route triggers API call via `ajax` in `src/lib/utils.ts`.
2. Request enters Gin router in `internal/server/server.go`.
3. Auth middleware validates optional `X-Auth` token for `/api/*` when users are configured.
4. Handler delegates to accounting/service/model/query modules.
5. Data is loaded from SQLite and/or refreshed via ledger/scraper flows.
6. JSON response is sent to frontend and rendered by Svelte components.

## Key Architectural Characteristics

### Strengths

1. Shared backend between web and desktop keeps behavior consistent.
2. Clear package boundaries in `internal/` by functional domain.
3. Config schema validation in `internal/config/` reduces malformed config risk.
4. Embed-based static serving simplifies deployment to a single binary.

### Current Constraints

1. Global mutable config singleton (`internal/config/config.go`) creates tight coupling.
2. HTTP handlers contain orchestration logic and domain calls in a single layer.
3. GORM `AutoMigrate`-driven schema evolution limits explicit migration control.
4. Auth is designed for single-user/self-hosted deployments, not multi-tenant SaaS.

## Primary Deployment Surfaces

1. Native binary (`paisa serve`), typically reverse-proxied in production.
2. Docker images (`Dockerfile`, `Dockerfile.hledger`, `Dockerfile.beancount`, `Dockerfile.all`).
3. Desktop artifacts from Wails in `desktop/build/`.
4. Static/documentation site via MkDocs (`mkdocs.yml`, `docs/`).

## Suggested Next Architectural Evolutions

1. Introduce explicit service interfaces for core domains (sync, reporting, editor).
2. Add DB migration versioning rather than relying only on implicit auto-migrations.
3. Move mutable write-protection checks to middleware/policy layer.
4. Add structured observability primitives (request IDs, metrics, trace correlation).
