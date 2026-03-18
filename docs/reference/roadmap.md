---
description: "Prioritized roadmap to improve security, reliability, and maintainability"
---

# Roadmap

Date: 2026-03-17

This roadmap is organized by impact and execution sequence.

## Guiding Outcomes

1. Harden authentication and deployment security.
2. Improve reliability and predictable upgrades.
3. Increase delivery velocity with stronger tests and clearer module boundaries.

## Phase 1: Stabilize Foundations (0-6 weeks)

### Milestones

1. Standardize API error envelope and remove non-essential `log.Fatal` calls from request paths.
2. Create explicit write-policy middleware for `readonly` enforcement.
3. Establish baseline engineering quality gate in CI:
   - Backend tests.
   - Frontend checks.
   - Regression snapshot run.

### Deliverables

1. Error-handling guidelines and helper package.
2. Shared middleware for write guards.
3. CI status checks required for merges.

### Success metrics

1. 0 known request-path hard crashes during normal API failures.
2. 100% write endpoints passing readonly-policy tests.
3. CI green on all protected branches.

## Phase 2: Security Hardening (6-12 weeks)

### Milestones

1. Replace persistent credential token pattern with session-based auth.
2. Introduce token/session expiry and logout invalidation.
3. Publish secure reverse-proxy deployment examples (TLS required).

### Deliverables

1. Session management module and migration path from current auth format.
2. Security-focused tests (auth replay, brute force, unauthorized access).
3. Updated user-authentication docs with threat-model guidance.

### Success metrics

1. No password-derived tokens persisted in browser storage.
2. 100% API auth tests passing for expired/invalid sessions.
3. Documented secure deployment path for internet-facing use.

## Phase 3: Data and Upgrade Reliability (12-20 weeks)

### Milestones

1. Introduce schema versioning and explicit migrations.
2. Make sync/update workflows transaction-safe where appropriate.
3. Improve sync observability and partial-failure reporting.

### Deliverables

1. Migration command/tooling and migration files.
2. Transactional guards for multi-step update flows.
3. Structured logs and sync result diagnostics.

### Success metrics

1. Reproducible schema upgrades and downgrade strategy.
2. Reduced sync-related support incidents.
3. Deterministic recovery behavior for partial external failures.

## Phase 4: Maintainability and Scalability (20+ weeks)

### Milestones

1. Extract service layer interfaces from handler-heavy modules.
2. Expand integration and domain test coverage.
3. Add lightweight observability (request IDs, core metrics, timings).

### Deliverables

1. Refactored handler/service boundaries for major domains.
2. Test coverage targets for critical packages.
3. Operational dashboard for sync/API health.

### Success metrics

1. Faster cycle time for adding new analytics endpoints.
2. Lower regression rate in release cycles.
3. Improved mean time to diagnose production issues.

## Commit-Sized Deliverables Plan

This plan converts the roadmap into small, reviewable commits that can be shipped independently.

### Wave 1: Stabilize Foundations

#### C1. Add standard API error envelope

Scope:

1. Introduce shared error response helper and error codes.
2. Apply helper to a first slice of endpoints in `internal/server/server.go`.

Definition of done:

1. Error payload shape is stable and documented.
2. Existing successful responses remain unchanged.

#### C2. Convert fatal request-path errors in editor endpoints

Scope:

1. Replace request-path `log.Fatal` calls in editor handlers with returned errors.
2. Preserve existing behavior for startup-only failures.

Definition of done:

1. Invalid editor operations return non-2xx with structured error envelope.
2. No process termination from editor request failures.

Depends on: C1

#### C3. Convert fatal request-path errors in sheet endpoints

Scope:

1. Replace request-path `log.Fatal` calls in sheet handlers with returned errors.

Definition of done:

1. Failed sheet operations return structured errors.
2. No process termination from sheet request failures.

Depends on: C1

#### C4. Introduce write-policy middleware for readonly mode

Scope:

1. Add middleware that blocks mutating routes when readonly is enabled.
2. Remove duplicated readonly checks from first batch of write endpoints.

Definition of done:

1. Write endpoints show consistent readonly failure response.
2. Duplicate policy checks are reduced in router code.

Depends on: C1

#### C5. Add API tests for readonly and error envelope

Scope:

1. Add integration tests covering readonly and standardized errors.

Definition of done:

1. CI test suite fails if contract regresses.

Depends on: C1, C4

### Wave 2: Security Hardening

#### C6. Add session model and server-side token issuer

Scope:

1. Add session data model with expiry fields.
2. Add login endpoint that issues server-generated session token.

Definition of done:

1. Login returns session artifact with expiry metadata.

Depends on: C1

#### C7. Add session validation middleware alongside legacy auth

Scope:

1. Extend middleware to accept new session token format.
2. Keep legacy token path temporarily for migration window.

Definition of done:

1. Both auth modes work behind a feature flag/config switch.

Depends on: C6

#### C8. Frontend migration to session auth

Scope:

1. Update login/logout and `ajax` auth behavior in `src/lib/utils.ts`.
2. Remove password-derived token writes from browser storage.

Definition of done:

1. Browser no longer stores `username:sha256(password)` token format.
2. Login, logout, and unauthorized redirect still work.

Depends on: C6, C7

#### C9. Enforce session expiry and revocation

Scope:

1. Add middleware checks for expired or revoked sessions.
2. Add logout invalidation endpoint and tests.

Definition of done:

1. Expired sessions are rejected reliably.
2. Logout invalidates active session immediately.

Depends on: C7

#### C10. Publish secure deployment docs

Scope:

1. Add reverse-proxy TLS examples and secure defaults.
2. Update user-auth docs with session model and threat guidance.

Definition of done:

1. Internet-facing deployment path includes mandatory HTTPS guidance.

Depends on: C8, C9

### Wave 3: Data and Upgrade Reliability

#### C11. Add migration framework skeleton

Scope:

1. Introduce migration table and migration runner command.
2. Add first baseline migration.

Definition of done:

1. Fresh and existing installs can report schema version.

#### C12. Move selected models from AutoMigrate to explicit migrations

Scope:

1. Migrate core tables from implicit AutoMigrate flow.
2. Remove duplicate model migration calls.

Definition of done:

1. Core tables are created via versioned migrations.
2. Startup does not duplicate schema operations.

Depends on: C11

#### C13. Add transactional sync boundary

Scope:

1. Wrap journal sync write stages in DB transaction.
2. Add rollback behavior for partial failures.

Definition of done:

1. Partial sync failure does not leave half-applied writes.

Depends on: C11

#### C14. Add sync diagnostics and structured logs

Scope:

1. Add sync result summary object and structured log fields.
2. Surface sync diagnostics through API response where appropriate.

Definition of done:

1. Operators can identify failure stage and counts from logs and API.

Depends on: C13

### Wave 4: Maintainability and Scale

#### C15. Extract service layer for first vertical slice

Scope:

1. Move one major endpoint family from handler orchestration into service boundary.
2. Keep API contract unchanged.

Definition of done:

1. Handler code shrinks and service unit tests cover core logic.

Depends on: C1, C5

#### C16. Expand integration and security test matrix

Scope:

1. Add regression scenarios for auth expiry, readonly writes, and sync rollback.
2. Add CI job matrix for Go plus frontend checks plus regression suite.

Definition of done:

1. Test matrix prevents regressions in the top roadmap risk areas.

Depends on: C9, C13, C15

## Immediate Backlog Candidates

1. Add a P0 security epic for session-based auth migration.
2. Add a P1 reliability epic for fatal-error elimination and API error contract.
3. Add a P1 data epic for migration/versioning strategy.
4. Add a P2 quality epic for integration/security test expansion.

## Issue Templates (C1-C16)

Copy each block into an issue tracker as-is.

### C1 - Standard API error envelope

Title: C1: Introduce standardized API error envelope

Labels: area/backend, type/refactor, priority/P1

Body:

1. Problem: API error payloads are inconsistent across handlers.
2. Scope:
   - Add shared error response helper and error code set.
   - Apply to first endpoint slice in router handlers.
3. Acceptance criteria:
   - Error payload shape is documented and stable.
   - Successful payloads are unchanged.
4. Out of scope:
   - Full handler migration in this issue.

### C2 - Editor fatal errors

Title: C2: Remove request-path process exits from editor endpoints

Labels: area/backend, type/bug, priority/P1

Body:

1. Problem: Editor request failures can terminate process.
2. Scope:
   - Replace request-path fatal exits in editor handlers with structured errors.
3. Acceptance criteria:
   - Invalid editor requests return non-2xx error envelope.
   - No request-path process termination in editor flow.
4. Depends on: C1

### C3 - Sheet fatal errors

Title: C3: Remove request-path process exits from sheet endpoints

Labels: area/backend, type/bug, priority/P1

Body:

1. Problem: Sheet request failures can terminate process.
2. Scope:
   - Replace request-path fatal exits in sheet handlers with structured errors.
3. Acceptance criteria:
   - Failed sheet operations return error envelope.
   - No request-path process termination in sheet flow.
4. Depends on: C1

### C4 - Readonly middleware

Title: C4: Add readonly write-policy middleware

Labels: area/backend, type/refactor, priority/P1

Body:

1. Problem: Readonly checks are duplicated and inconsistent.
2. Scope:
   - Add middleware to block mutating endpoints when readonly is enabled.
   - Remove duplicate inline checks from first endpoint batch.
3. Acceptance criteria:
   - Consistent readonly failure response for write endpoints.
   - Router code has reduced duplicated policy checks.
4. Depends on: C1

### C5 - Readonly and error contract tests

Title: C5: Add integration tests for readonly policy and error envelope

Labels: area/testing, type/test, priority/P1

Body:

1. Problem: No guardrails for readonly and error contract regressions.
2. Scope:
   - Add integration tests for readonly writes and standardized error shape.
3. Acceptance criteria:
   - CI fails on contract regressions.
4. Depends on: C1, C4

### C6 - Session model and login issuer

Title: C6: Add session model and server-side session token issuance

Labels: area/security, area/backend, type/feature, priority/P0

Body:

1. Problem: Current token model is password-derived.
2. Scope:
   - Add session model with expiry fields.
   - Add login endpoint that returns session token.
3. Acceptance criteria:
   - Login issues session token and expiry metadata.
4. Depends on: C1

### C7 - Dual-mode auth middleware

Title: C7: Support session validation in auth middleware with legacy fallback

Labels: area/security, area/backend, type/feature, priority/P0

Body:

1. Problem: Migration requires temporary compatibility window.
2. Scope:
   - Extend auth middleware for session tokens.
   - Keep legacy token path behind config flag.
3. Acceptance criteria:
   - Both auth modes operate correctly when enabled.
4. Depends on: C6

### C8 - Frontend auth migration

Title: C8: Migrate frontend auth flow to server-issued sessions

Labels: area/security, area/frontend, type/feature, priority/P0

Body:

1. Problem: Frontend stores password-derived token in browser storage.
2. Scope:
   - Update login/logout/ajax auth handling to session model.
   - Remove password-derived token write path.
3. Acceptance criteria:
   - No username:sha256(password) token persisted.
   - Login/logout/unauthorized redirect still function.
4. Depends on: C6, C7

### C9 - Session expiry and revocation

Title: C9: Enforce session expiry and logout revocation

Labels: area/security, area/backend, type/feature, priority/P0

Body:

1. Problem: Session lifecycle controls are incomplete.
2. Scope:
   - Enforce expiry in middleware.
   - Add logout invalidation endpoint and tests.
3. Acceptance criteria:
   - Expired sessions are rejected.
   - Logout invalidates session immediately.
4. Depends on: C7

### C10 - Secure deployment docs

Title: C10: Publish TLS-first deployment guidance and auth model updates

Labels: area/docs, area/security, type/docs, priority/P1

Body:

1. Problem: Secure deployment guidance is incomplete for internet exposure.
2. Scope:
   - Add reverse proxy TLS examples.
   - Update authentication docs for session model and threat guidance.
3. Acceptance criteria:
   - Internet-facing deployment instructions require HTTPS.
4. Depends on: C8, C9

### C11 - Migration framework skeleton

Title: C11: Introduce database migration framework and schema version table

Labels: area/data, area/backend, type/feature, priority/P1

Body:

1. Problem: Schema evolution is implicit.
2. Scope:
   - Add migration runner and schema version tracking.
   - Add baseline migration.
3. Acceptance criteria:
   - Fresh and existing installs report schema version.

### C12 - Shift core tables to explicit migrations

Title: C12: Move core model creation from AutoMigrate to explicit migrations

Labels: area/data, area/backend, type/refactor, priority/P1

Body:

1. Problem: AutoMigrate is duplicated and hard to reason about.
2. Scope:
   - Move selected core tables to versioned migrations.
   - Remove duplicate migration calls.
3. Acceptance criteria:
   - Core schema managed by migration files.
   - Startup schema operations are deterministic.
4. Depends on: C11

### C13 - Transactional sync writes

Title: C13: Add transactional boundary for sync write stages

Labels: area/data, area/backend, type/bug, priority/P1

Body:

1. Problem: Partial sync failures can leave half-applied writes.
2. Scope:
   - Wrap sync write stages in DB transaction.
   - Add rollback behavior for stage failure.
3. Acceptance criteria:
   - Partial failures do not persist partial writes.
4. Depends on: C11

### C14 - Sync diagnostics

Title: C14: Add structured sync diagnostics and operator-visible summaries

Labels: area/observability, area/backend, type/feature, priority/P2

Body:

1. Problem: Sync troubleshooting lacks stage-level diagnostics.
2. Scope:
   - Add sync result summary object and structured logging fields.
   - Expose diagnostics in API response where relevant.
3. Acceptance criteria:
   - Operators can identify failed stage and counts from logs/API.
4. Depends on: C13

### C15 - Service extraction first slice

Title: C15: Extract first handler family into service-layer boundary

Labels: area/architecture, area/backend, type/refactor, priority/P2

Body:

1. Problem: Handler layer owns too much orchestration logic.
2. Scope:
   - Move one major endpoint family into service boundary.
   - Preserve API contract.
3. Acceptance criteria:
   - Handler complexity reduced.
   - Service unit tests cover extracted logic.
4. Depends on: C1, C5

### C16 - Test matrix expansion

Title: C16: Expand integration and security test matrix

Labels: area/testing, area/security, type/test, priority/P2

Body:

1. Problem: Core risk areas are under-tested.
2. Scope:
   - Add regression scenarios for auth expiry, readonly writes, sync rollback.
   - Add CI matrix for backend tests, frontend checks, and regression suite.
3. Acceptance criteria:
   - CI blocks merges when top-risk scenarios regress.
4. Depends on: C9, C13, C15

## Recommended PR Batching

Use this merge order to minimize conflicts and rework.

### Batch A - Contracts and stability baseline

1. PR-A1: C1
2. PR-A2: C2 + C3
3. PR-A3: C4 + C5

Target outcome:

1. Stable API error contract and readonly policy baseline with tests.

### Batch B - Authentication migration

1. PR-B1: C6 + C7
2. PR-B2: C8
3. PR-B3: C9 + C10

Target outcome:

1. Session auth in production with docs and migration-safe compatibility path.

### Batch C - Data reliability

1. PR-C1: C11
2. PR-C2: C12
3. PR-C3: C13 + C14

Target outcome:

1. Versioned schema path and transactional sync with improved diagnostics.

### Batch D - Architecture and quality

1. PR-D1: C15
2. PR-D2: C16

Target outcome:

1. Clearer service boundaries and durable regression coverage.

## Batch Exit Criteria

### After Batch A

1. No request-path process exits in edited handlers.
2. Error contract tests green.

### After Batch B

1. Password-derived browser token path removed.
2. Session expiry and logout invalidation verified.

### After Batch C

1. Schema version command reports expected state.
2. Sync rollback behavior validated by tests.

### After Batch D

1. Selected endpoint family is service-oriented.
2. CI matrix covers core security and data integrity regressions.
