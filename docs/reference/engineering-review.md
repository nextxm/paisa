---
description: "Engineering review of major risks and remediation priorities"
---

# Engineering Review

Date: 2026-03-17

This review focuses on major architecture, reliability, security, and maintainability issues identified from repository inspection.

## Critical Issues

### 1. Credential token model is weak for hostile networks

Evidence:

1. Frontend stores auth token in local storage as `username:sha256(password)` in `src/lib/utils.ts`.
2. Middleware compares against config-stored hash in `internal/server/server.go`.
3. Existing documentation already warns HTTPS is required in `docs/reference/user-authentication.md`.

Impact:

1. Token replay is possible if intercepted over HTTP.
2. Token lifetime is effectively unbounded.
3. Browser storage exposure (for example through XSS) gives durable API access.

Remediation priority: P0

Recommended actions:

1. Move to server-issued session tokens with expiry.
2. Prefer secure, HTTP-only, SameSite cookies for browser auth.
3. Add explicit network hardening guidance and reverse-proxy examples for TLS termination.

### 2. Database schema evolution relies on implicit `AutoMigrate`

Evidence:

1. Auto migration invoked from serve/startup/sync paths (`cmd/serve.go`, `desktop/app.go`, `internal/model/model.go`).
2. No explicit migration version files or rollback strategy.

Impact:

1. Hard to reason about deterministic upgrades/downgrades.
2. Risk during major schema changes.

Remediation priority: P1

Recommended actions:

1. Introduce explicit migration tooling and schema version table.
2. Keep `AutoMigrate` only for development mode if desired.

## High Priority Issues

### 3. Error handling often fails closed via process termination

Evidence:

1. `log.Fatal` usage in core startup/config/server flows (`cmd/root.go`, `internal/config/config.go`, `internal/server/server.go`).

Impact:

1. Single operation failures can terminate the service.
2. Reduced resilience for long-running deployments.

Remediation priority: P1

Recommended actions:

1. Return typed errors to callers and API clients.
2. Reserve process exit for unrecoverable startup failures.
3. Standardize API error envelopes.

### 4. Handler layer owns both policy and orchestration logic

Evidence:

1. `internal/server/server.go` includes route registration, readonly checks, payload binding, and direct domain invocation.

Impact:

1. Repetition across many endpoints.
2. Harder testing and policy consistency.

Remediation priority: P1

Recommended actions:

1. Introduce dedicated middleware/policy gates for write operations.
2. Move endpoint orchestration into service methods.

## Medium Priority Issues

### 5. Test coverage is narrow for critical paths

Evidence:

1. Backend tests are limited; only a small number of package tests exist.
2. Snapshot regression test exists in `tests/regression.test.ts` but requires local toolchain not available in this environment.

Impact:

1. Refactors risk behavioral regressions.
2. Security and concurrency regressions are likely to slip.

Remediation priority: P2

Recommended actions:

1. Add service-level unit tests for sync/auth/error paths.
2. Add API integration tests for auth, readonly mode, and editor write paths.
3. Add security-focused tests for token handling and brute-force controls.

### 6. Config state is global and mutable

Evidence:

1. Config singleton is process-global in `internal/config/config.go`.

Impact:

1. Limits test isolation.
2. Potential race and coupling risks as concurrency increases.

Remediation priority: P2

Recommended actions:

1. Introduce immutable config snapshots and dependency injection.
2. Avoid hidden global reads in deep domain layers.

## Observed Strengths

1. Clear domain-oriented package structure in `internal/`.
2. Shared backend across web and desktop reduces feature divergence.
3. Path traversal mitigation helper exists (`BuildSubPath` in `internal/utils/utils.go`).
4. Config schema validation and docs are already mature for user-facing configuration.

## Validation Limits

1. Could not execute `go test ./...` because `go` is unavailable in current terminal environment.
2. Could not execute frontend checks because `svelte-kit` dependencies are unavailable in current terminal environment.

These runtime/toolchain checks should be re-run in CI or a fully provisioned dev machine as part of remediation.
