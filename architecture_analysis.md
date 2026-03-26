# Paisa Architectural Review & Strategy

This document provides a comprehensive analysis of the Paisa application architecture, highlighting its core strengths and detailing seven key areas for long-term improvement.

## 🏗️ Architectural Overview

Paisa is built as a **Modular Monolith** designed to bridge the gap between **Plain Text Accounting (PTA)** and modern, data-driven financial tools.

### 1. Hybrid Data Strategy

Paisa employs a unique hybrid storage model:

- **Primary Source of Truth**: User-managed PTA files (Ledger/hledger format).
- **Secondary Cache/Store**: A SQLite database (`paisa.db`) managed via GORM for high-performance querying and external market data storage.
- **Sync Mechanism**: A validation and parsing layer that translates text-based double-entry records into a relational schema.

### 2. Backend Architecture (Go)

- **API Engine**: Built on **Gin**, providing a robust REST interface.
- **Service Layer**: Logic is partitioned into specialized domain packages (`accounting`, `budget`, `taxation`, `xirr`), promoting decoupling.
- **Multi-Platform Support**: Uses **Wails** to share backend Go logic between web and desktop interfaces.

### 3. Frontend Architecture (SvelteKit)

- **Reactive UI**: Built with **SvelteKit 2**, ensuring a snappy user experience.
- **Visualization Stack**: Extensive use of **D3.js** for analytical charts and **CodeMirror 6** for an integrated, syntax-highlighted ledger editor.

---

## 🚀 Expanded Architectural Improvements

Below are deep-dives into the four most critical improvements for the application's maturity.

### 1️⃣ Incremental Synchronization & Differential Parsing

Currently, Paisa performs a full re-sync and cache-clear whenever the ledger file changes. This creates a linear performance degradation as the user's financial history grows.

- **Implementation Strategy**:
  - **Content Hashing**: Store a cryptographic hash (SHA-256) of the ledger file. If the hash remains unchanged, skip the sync process entirely.
  - **Transaction-Level Tracking**: Generate stable hashes for individual transaction blocks. Compare these hashes during the parse phase to identify only new or modified entries.
  - **Selective DB Update**: Use a "Upsert" strategy based on transaction hashes rather than a full "Delete-and-Recreate" approach.
  - **Performance Win**: Reduces sync time from seconds to milliseconds for multi-year ledger files.

### 2️⃣ Asynchronous Background Processing

Lengthy operations—such as scraping mutual fund NAVs or performing intensive XIRR calculations—currently block the main HTTP thread.

- **Implementation Strategy**:
  - **Task Workers**: Implement an internal Go worker queue to process syncs and scrapers in the background.
  - **Real-Time Status**: Introduce **Server-Sent Events (SSE)** or **WebSockets** so the UI can provide live progress updates (e.g., "Retrieved 12/15 prices...").
  - **Non-Blocking UI**: Allow users to continue navigating their cached data while a background sync is in progress.
  - **Reliability**: Implement automatic retries with exponential backoff for external API providers.

### 4️⃣ Plugin-Based Scraper & Engine Architecture

Regional finance logic (e.g., Indian Mutual Funds) and market scrapers are currently hardcoded into the core `internal/` packages.

- **Implementation Strategy**:
  - **RPC-Based Architecture**: Use a system like **Hashicorp go-plugin** to allow standalone binaries (written in Python, Node, or Go) to act as financial adapters.
  - **Embedded Scripting**: Support small **Lua** or **JavaScript** (via Goja) scripts for custom bank CSV importers or local tax rules.
  - **Provider Interface**: Standardize the data exchange format, ensuring the core remains agnostic to whether it is fetching stock prices from NASDAQ or gold rates from a regional API.

### 7️⃣ Structured GraphQL or Typed API Layer

The REST API's current growth path leads to over-fetching and a lack of strict typing between the Go models and Svelte stores.

- **Implementation Strategy**:
  - **GraphQL Schema**: Define a unified financial schema. This allows the frontend to request exactly what is needed for a widget (e.g., just the balances and the XIRR for a specific portfolio) in a single round-trip.
  - **Type Generation**: Use tools like `graphql-codegen` or `Connect-Go` to generate TypeScript interfaces directly from the backend models.
  - **Analytical Power**: GraphQL's nested structure is naturally suited for the hierarchical nature of financial accounts (e.g., `Assets:Investments:MutualFunds`).

---

## 🛠️ Other Recommended Enhancements

- **Consolidated Design System**: Standardizing on **TailwindCSS + DaisyUI** to reduce CSS bloat and styling conflicts.
- **Formalized Hexagonal Architecture**: Enforcing stricter "Ports and Adapters" to allow easier swapping of the database (e.g., SQLite to Postgres) or the parsing engine.
- **Multi-Tenancy Support**: Adding session-based data isolation for family or multi-workspace hosting.
