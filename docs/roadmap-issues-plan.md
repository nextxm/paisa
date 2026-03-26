# Implementation Roadmap -> Milestones and Issues

Repository: nextxm/paisa

## Milestones

1. P1: High-Performance Sync & Responsiveness
2. P2: Refactoring & Svelte 5 Modernization
3. P3: Scalability & Modernization

## Issue Breakdown

### P1: High-Performance Sync & Responsiveness

1. P1.1 Core: add SHA256 file hash utility
2. P1.1 Data model: add metadata table
3. P1.1 Persistence: metadata getters/setters
4. P1.1 SyncJournal: skip parse on unchanged hash
5. P1.2 Worker: create internal/service/worker package
6. P1.2 Worker: define Job state machine
7. P1.2 Worker: implement thread-safe JobRegistry
8. P1.2 API: make POST /api/sync asynchronous
9. P1.2 API: add GET /api/jobs/:id
10. P1.2 Integrate long-running tasks with worker
11. P1.3 Frontend: add jobs store
12. P1.3 Frontend: implement polling or SSE for job updates
13. P1.3 UX: header syncing indicator
14. P1.3 UX: sync history/log overlay
15. P1.3 UX: toast errors for critical failures

### P2: Refactoring & Svelte 5 Modernization

1. P2.1 Scraper: formalize PriceProvider interface
2. P2.1 Scraper: align existing providers to interface
3. P2.1 Prototype adapter-based importer plugin approach
4. P2.1 Docs: external provider JSON format
5. P2.2 Frontend: bump Svelte and SvelteKit
6. P2.2 Frontend: enable Svelte 4 compatibility layer
7. P2.2 CSS: replace Bulma classes with Tailwind/DaisyUI
8. P2.2 Components: port Modal and Tab logic to Svelte runes
9. P2.2 Components: rewrite AccountTree using Svelte 5 runes
10. P2.3 State: migrate src/store.ts to class-based manager
11. P2.3 State: replace writable/derived with runes
12. P2.3 State: decouple UI state from persisted config state

### P3: Scalability & Modernization

1. P3.1 API: create proto/api.proto
2. P3.1 Backend: integrate Connect-Go in Gin
3. P3.1 Backend: implement sample typed Account Tree endpoint
4. P3.1 Frontend: generate connect-web typed client
5. P3.1 Frontend: replace manual api.ts fetch calls with typed client
6. P3.2 Parser: prototype native Go ledger parser subset
7. P3.2 Parser: benchmark for 10x target vs external CLI
8. P3.2 Sync: implement trailing-lines incremental parse
9. P3.3 Config: add multi-workspace array to paisa.yaml
10. P3.3 UI: add workspace switcher
11. P3.3 Data: scope SQLite per workspace

## Script to Create On GitHub

Use scripts/create-roadmap-issues.ps1.

Example:

PowerShell:

$env:GITHUB_TOKEN = "<token-with-repo-scope>"
./scripts/create-roadmap-issues.ps1 -Owner nextxm -Repo paisa
