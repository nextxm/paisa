---
description: "Design decisions, trade-offs, and extension guidelines for Paisa"
---

# Design

## Design Goals

Paisa favors practical self-hosted usability over distributed-system complexity.

Primary goals:

1. Keep setup simple for personal finance users.
2. Preserve transparency by building on plain-text ledger journals.
3. Provide rich analytics and editing workflows in a browser/desktop UI.
4. Keep operational footprint small (single process, SQLite, embedded assets).

## Current Design Choices

### Backend design

1. Language/runtime: Go for backend services and CLI.
2. Web stack: Gin router + JSON APIs.
3. Persistence: SQLite via GORM.
4. Domain organization: package-per-concern under `internal/`.

Trade-off:

- Fast development and portability, but some business logic is tightly coupled to handler flows.

### Frontend design

1. SvelteKit SPA (`ssr = false`) with static build output.
2. Local state via Svelte stores in `src/store.ts`.
3. REST API access through a shared `ajax` helper in `src/lib/utils.ts`.

Trade-off:

- Simple deployment and fast client UX, but auth token management currently relies on browser storage.

### Authentication design

1. Optional user accounts in config (`internal/config/config.go`).
2. API auth middleware validates `X-Auth` header for API routes.
3. Rate limiter slows repeated invalid credentials.

Trade-off:

- Works for personal/self-hosted use, but lacks hardened session semantics expected in hostile networks.

### Sync design

1. Journal sync validates and parses journal files.
2. Price/portfolio sync fetches external market data.
3. DB is refreshed in place by model upsert logic.

Trade-off:

- Functional and straightforward, but partial-failure handling and transactional consistency can be improved.

## Design Principles For Future Changes

1. Keep web and desktop behavior aligned by reusing backend handlers/services.
2. Shift domain orchestration out of route handlers into explicit service boundaries.
3. Prefer deterministic, testable pure functions for financial calculations.
4. Treat filesystem operations and external scrapers as side-effect boundaries.
5. Keep read paths fast and stable; isolate write/sync paths for observability.

## Extension Guidelines

### Adding a new analytics capability

1. Add domain model/service logic under `internal/` first.
2. Add API route in `internal/server/` with clear request/response shape.
3. Add typed frontend integration in `src/lib/`.
4. Add route/view and store wiring in `src/routes/` and `src/store.ts` as needed.
5. Add regression fixtures/tests when output shape is stable.

### Adding a write path

1. Enforce `readonly` behavior consistently.
2. Validate request payloads with strict binding.
3. Wrap multi-step persistence changes in DB transactions.
4. Emit structured logs for auditability.

### Adding external data providers

1. Implement provider behind scraper abstraction.
2. Add provider-specific tests with deterministic fixtures.
3. Add timeout/retry/backoff policy and failure classification.
4. Ensure cache invalidation strategy is explicit.

## Non-Goals (Current Scope)

1. Multi-tenant user isolation.
2. Distributed storage and horizontal write scaling.
3. Full enterprise IAM/OIDC integration.

## Design Debt To Track

1. Global config mutability and thread-safety concerns.
2. Inconsistent API error response contracts.
3. Handler-level policy checks repeated across endpoints.
4. Limited formal migration/version management for DB schema.
