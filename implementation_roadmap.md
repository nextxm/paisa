# Detailed Architectural Implementation Plan: Paisa v2.0

This document is the granular execution roadmap for the Paisa architecture evolution. It breaks down the high-level goals into atomic developer tasks.

---

## ⚡ Phase 1: High-Performance Sync & Responsiveness

**Focus**: Performance optimization and non-blocking user experience.

### Step 1.1: Incremental File Hashing (Sync Optimization)

- [ ] **Core Utility**: Create `internal/utils/hash.go` with a `SHA256File(path string) (string, error)` helper.
- [ ] **Data Model**:
  - [ ] Define a `Metadata` struct in `internal/model/model.go` (Key: string, Value: string).
  - [ ] Run migration for the `metadata` table.
- [ ] **Persistence Layer**:
  - [ ] Implement `GetMetadata(db, key)` and `SetMetadata(db, key, value)`.
- [ ] **Sync Logics**:
  - [ ] Update `SyncJournal` to fetch the cached hash from the database.
  - [ ] Skip the Ledger CLI validation and parsing if the file hash matches.
  - [ ] Only update the stored hash in SQLite _after_ a successful full sync.

### Step 1.2: Asynchronous Task Runner (Go Backend)

- [ ] **Job Manager**:
  - [ ] Create `internal/service/worker/` package.
  - [ ] Define a `Job` state machine (Pending, Running, Completed, Failed).
  - [ ] Implement a thread-safe `JobRegistry` to store current and historical job states.
- [ ] **API Refactor**:
  - [ ] Update `POST /api/sync` in `internal/server/sync.go` to return a `202 Accepted` with a `job_id`.
  - [ ] Add `GET /api/jobs/:id` for the frontend to poll status.
- [ ] **Long-Running Process Handling**:
  - [ ] Wrap XIRR calculations and price scrapers into the new Job system.

### Step 1.3: Svelte Interactive Progress UI

- [ ] **Reactivity**:
  - [ ] Create `src/lib/stores/jobs.ts` to manage global task state in the frontend.
  - [ ] Implement a polling mechanism (or SSE client) that starts when a `job_id` is received.
- [ ] **Components**:
  - [ ] Design a subtle "Syncing Records..." loader for the main header.
  - [ ] Implement a "Sync History" or "Log Viewer" overlay to see background job outcomes.
  - [ ] Add toast notifications for critical failures (e.g., "Ledger Parsing Error at line 142").

---

## 🔌 Phase 2: Refactoring & Svelte 5 Modernization

**Focus**: Maintenance, extensibility, and modern frontend paradigms.

### Step 2.1: Scraper Interface & Decoupling

- [ ] **Core Abstraction**:
  - [ ] Define a `PriceProvider` interface in `internal/scraper/`.
  - [ ] Refactor existing providers (Google Finance, Yahoo, MFAPI) to strictly implement this interface.
- [ ] **Plugin System Prototype**:
  - [ ] Explore a directory-based "adapter" approach for custom CSV/JSON importers.
  - [ ] Document the "External Provider" JSON format for local custom rates.

### Step 2.2: Design System & Svelte 5 Migration

- [ ] **Svelte 5 Bridge**:
  - [ ] Bump `svelte` and `@sveltejs/kit` versions in `package.json`.
  - [ ] Enable the Svelte 4 compatibility layer for existing components.
- [ ] **CSS Modernization**:
  - [ ] Identify and replace Bulma specific classes (e.g., `is-primary`, `level`, `columns`) with **Tailwind/DaisyUI** equivalents.
  - [ ] Port the `Modal` and `Tab` logic from Bulma JavaScript to pure Svelte 5 Runes.
- [ ] **Component Refactoring**:
  - [ ] Rewrite `AccountTree.svelte` to use **Svelte 5 Runes** (`$state`, `$derived`) for hierarchical rendering.

### Step 2.3: Global State (Stores) Modernization

- [ ] **Store Refactor**:
  - [ ] Migrate `src/store.ts` to a Svelte 5 class-based state manager.
  - [ ] Replace `writable` and `derived` with native `$state` and `$derived` runes.
  - [ ] Decouple UI state from persistent configuration state.

---

## 🔗 Phase 3: Scalability & Modernization

**Focus**: Network efficiency and native Go capabilities.

### Step 3.1: Typed API Prototype (Connect/Protobuf)

- [ ] **Schema Definition**: Create `proto/api.proto` with messages for `Transaction`, `Posting`, and `AccountBalance`.
- [ ] **Backend Implementation**:
  - [ ] Integrate **Connect-Go** into the Gin engine.
  - [ ] Implement a sample gRPC-compatible endpoint for fetching the `Account Tree`.
- [ ] **Frontend Client**:
  - [ ] Use `@bufbuild/connect-web` to generate a typed TypeScript client.
  - [ ] Replace the manual `fetch()` calls in `src/lib/api.ts` with the typed client.

### Step 3.2: Native Parsing & Incremental Logic

- [ ] **Go Parser Evaluation**:
  - [ ] Write a prototype native Go parser targeting the subset of Ledger format used in Paisa.
  - [ ] Target a 10x speed improvement over external CLI calls.
- [ ] **Incremental Logic**:
  - [ ] Read only the "trailing" lines of the ledger file (newest transactions) if the header hash matches.

### Step 3.3: Multi-Workspace Isolation

- [ ] **Configuration Expansion**: Update `paisa.yaml` to allow defining an array of workspace directories.
- [ ] **Session & State**:
  - [ ] Implement a workspace switcher in the UI.
  - [ ] Ensure the SQLite database remains scoped or namespaced per workspace.
