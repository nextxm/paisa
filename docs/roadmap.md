# Paisa Product Roadmap (2026-2028)

Date: 2026-03-18

This is the single source of truth for Paisa roadmap planning. It combines long-term product direction with practical quarter-by-quarter delivery goals, while preserving Paisa's local-first and plain-text ledger model.

## Vision

Paisa evolves from a robust local-first desktop finance manager into an intelligent financial partner that remains auditable, privacy-first, and user-controlled.

## Strategic Pillars

1. Zero-friction automation: reduce manual bookkeeping through better imports, OCR, and optional integrations.
2. Financial intelligence: move from historical reporting to explainable forecasting and decision support.
3. Privacy-first sync and mobility: enable cross-device workflows without compromising local ownership and encryption.
4. Extensible ecosystem: open provider and plugin extension points with clear contracts.

## Product Themes (2026-2027)

1. Faster onboarding to first useful insight.
2. Better day-to-day planning and decision support.
3. Deeper investment and tax workflows for advanced users.
4. Safe automation with explicit user control and auditability.

## Delivery Plan

### Q2 2026: Onboarding and Data Flow

Goal: improve first-run success and reduce import friction.

Planned capabilities:

1. Guided first-run setup flow:
    - Detect and validate ledger dialect early (ledger, hledger, beancount).
    - Validate required config fields with actionable fixes.
2. Import workflow upgrades:
    - Import preview with row-level validation errors.
    - Reusable import profile presets for banks and brokers.
3. Sync transparency:
    - Last sync summary in UI with counts, warnings, and failures.
    - Suggested next steps when sync partially fails.
4. Budget starter pack:
    - One-click templates for common budgeting styles.
    - Auto-categorization hints based on account history.

Success criteria:

1. Time-to-first-dashboard below 10 minutes for new users.
2. At least 50% fewer import-related support issues.
3. Higher first-sync success on clean installs.

### Q3 2026: Planning, Insights, and UX Refresh

Goal: make planning actionable and insights explainable.

Planned capabilities:

1. Forecasting workspace:
    - Monthly cashflow projections based on recurring entries and trends.
    - Best-case and conservative forecast modes.
2. Goals improvements:
    - Goal progress projection with expected completion date.
    - Goal dependencies for milestone-based plans.
3. Budget intelligence:
    - Overrun risk flags before month-end.
    - Suggested reallocations between categories.
4. Explainable analytics:
    - Drill-down cards showing how metrics are computed.
    - Links from charts to underlying journal postings.
5. UI and accessibility improvements:
    - Dashboard customization with drag-and-drop widgets.
    - Accessibility pass and improved responsive behavior.
6. Advanced charting:
    - Interactive flow views (for example, Sankey) for income/expense analysis.

Success criteria:

1. Week-over-week growth in budgets and goals usage.
2. Reduced month-end budget surprises for active users.
3. Faster anomaly diagnosis on analytics pages.

### Q4 2026: Investment and Tax Depth + Assisted Automation

Goal: improve investment workflows and begin safe AI assistance.

Planned capabilities:

1. Portfolio diagnostics:
    - Attribution by asset class, account, and instrument.
    - Concentration exposure heatmap.
2. Capital gains workflow:
    - Tax-lot explorer with realized and unrealized breakouts.
    - Financial-year and jurisdiction-aware filters.
3. Corporate actions support:
    - Helpers for split, bonus, and merger events.
    - Reconciliation checks to prevent position drift.
4. Multi-currency reporting:
    - Performance in base and source currencies.
    - FX impact decomposition in returns views.
5. AI transaction categorization (opt-in):
    - Local-first categorization suggestions that learn from user corrections.
    - Confidence and explainability signals before applying tags.

Success criteria:

1. More complete tax reporting without external spreadsheets.
2. Fewer portfolio reconciliation mismatches after imports.
3. Measurable adoption of suggestion-based categorization.

### Q1 2027: Operations, Integrations, and Mobile-First Workflow

Goal: support reliable unattended operation and broader ecosystem usage.

Planned capabilities:

1. Scheduled operations:
    - Configurable background sync schedules.
    - Optional end-of-day recomputation jobs.
2. Integrations foundation:
    - Personal API keys with scoped token controls.
    - Webhook events for sync completion and failure.
    - Price history export from database to ledger-family formats (ledger, hledger, beancount) with date-range and commodity filters.
    - Design and implementation details tracked in [Multi-Currency Commodity Prices RFC](./reference/multi-currency-prices-rfc.md).
3. Provider ecosystem:
    - Standardized contract for new price/import providers.
    - Community provider registry documentation.
4. Mobile-friendly workflow:
    - PWA enhancements for quick capture and review.
    - Improved responsive UX for dashboard and transactions.
5. Receipt capture groundwork:
    - Capture flow and OCR pipeline validation for statement/receipt input.

Success criteria:

1. More users running unattended sync successfully.
2. Increasing contributor adoption of extension points.
3. Better completion rates for key workflows on mobile.
4. Users can round-trip externally fetched prices by exporting DB history into a valid price file for their selected dialect.

### Q2-Q4 2027: Sync and Companion Experience

Goal: provide secure multi-device continuity.

Planned capabilities:

1. Encrypted remote sync (optional):
    - End-to-end encrypted sync via self-hosted and peer-to-peer options.
2. Mobile companion expansion:
    - Progressively expand from PWA-first flows to richer companion capabilities.
3. Automation maturity:
    - Intelligent recurring bill/subscription detection.
    - Optional trigger-based automations through local webhooks.
4. Retirement and long-horizon planning:
    - FIRE and retirement scenario modeling with Monte Carlo simulations.

Success criteria:

1. Reliable cross-device usage without privacy regressions.
2. Lower manual effort in recurring transaction management.
3. Strong usage retention for long-term planning features.

### 2028+: Platform and Ecosystem Scale

Goal: establish Paisa as a durable, extensible personal finance platform.

Planned capabilities:

1. Plugin framework and marketplace foundations:
    - Sandboxed runtime for scripts, importers, reports, and themes.
2. Dedicated multi-currency and digital asset engine:
    - Deeper handling of multi-currency and crypto assets.
3. Institutional connectors where privacy can be preserved:
    - Open banking connectors in supported regions.
4. Advanced wealth workflows:
    - Estate/trust-oriented planning and reporting workflows.

## Technical Enablers

### Backend

1. Continue evolving `internal/` domain boundaries for maintainability.
2. Use WebAssembly or Rust selectively for compute-heavy paths with measured gains.
3. Evaluate encrypted SQLite (for example SQLCipher) as a default option.
4. Expand prediction pipeline with explainable statistical and ML models.

### Frontend

1. Stay aligned with Svelte evolution, including Svelte 5 migration planning.
2. Build and adopt a reusable Paisa component and design system.
3. Improve analytics interaction patterns and accessibility consistency.

### Infrastructure and Desktop

1. Keep Docker-first deployment paths for self-hosters.
2. Strengthen Wails bridge integrations (for example system tray and global shortcuts).
3. Harden desktop authentication options, including biometric and hardware key support where feasible.

## Guardrails

1. Ledger files remain source of truth; generated state must stay reproducible.
2. Preserve backward compatibility in public APIs where practical.
3. Ship in small, testable increments with regression coverage.
4. Treat security and readonly guarantees as release blockers for write paths.
5. Keep advanced automation opt-in, explainable, and reversible.

## Not Planned in This Window

1. Cloud-hosted multi-tenant Paisa service.
2. Real-time market data streaming infrastructure.
3. Closed-source proprietary plugin marketplace.

> [!NOTE]
> This roadmap is a living document and will evolve with community feedback, delivery learnings, and technology changes.
