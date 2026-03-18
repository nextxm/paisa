---
description: "Feature roadmap for Paisa"
---

# Feature Roadmap

Date: 2026-03-18

This roadmap focuses on user-facing product capabilities while staying aligned with Paisa's plain-text ledger model and local-first architecture.

## Product Themes

1. Faster onboarding to first useful insight.
2. Better day-to-day financial planning and decision support.
3. Deeper investment and tax workflows for advanced users.
4. Safe automation with clear user control and auditability.

## Q2 2026: Onboarding and Data Flow

### Planned features

1. Guided first-run setup flow:
   - Detect and validate ledger dialect early (ledger, hledger, beancount).
   - Check required config fields with actionable fixes.
2. Import workflow upgrades:
   - Import preview with row-level validation errors.
   - Reusable import profile presets for banks and brokers.
3. Sync transparency:
   - Last sync summary in UI with counts, warnings, and failures.
   - Suggested next steps when sync partially fails.
4. Budget starter pack:
   - One-click templates for common budgeting styles.
   - Auto-categorization hints using existing account history.

### Success criteria

1. Time-to-first-dashboard below 10 minutes for new users.
2. At least 50% fewer import-related support issues.
3. More successful first sync attempts on clean installs.

## Q3 2026: Planning and Insights

### Planned features

1. Forecasting workspace:
   - Monthly cashflow projection based on recurring entries and trends.
   - Best-case and conservative forecast views.
2. Goals improvements:
   - Goal progress projection and expected completion date.
   - Goal dependency support for milestone-based planning.
3. Budget intelligence:
   - Overrun risk flags before month-end.
   - Suggested reallocations between envelope categories.
4. Explainable analytics:
   - Drill-down cards that show how key metrics were computed.
   - Quick links from chart points to underlying journal postings.

### Success criteria

1. Increased use of budgets and goals pages week over week.
2. Reduced month-end budget surprises for active users.
3. Faster diagnosis of anomalies from analytics pages.

## Q4 2026: Investments and Tax Depth

### Planned features

1. Portfolio diagnostics:
   - Attribution views by asset class, account, and instrument.
   - Exposure heatmap for concentration risk.
2. Capital gains workflow:
   - Tax lot explorer with realized and unrealized breakouts.
   - Filters for financial-year and jurisdiction-specific reporting windows.
3. Corporate actions support:
   - Split, bonus, and merger event helpers.
   - Reconciliation checks to prevent position drift.
4. Multi-currency reporting:
   - Performance in base currency and source currency.
   - FX impact decomposition in returns views.

### Success criteria

1. More complete tax reporting coverage without manual spreadsheets.
2. Fewer portfolio reconciliation mismatches after imports.
3. Higher retention among investment-focused users.

## Q1 2027: Automation and Ecosystem

### Planned features

1. Scheduled operations:
   - Configurable background sync schedules.
   - Optional end-of-day recomputation job.
2. Integrations foundation:
   - Personal API keys and scoped token controls.
   - Webhook events for sync completion and failure.
3. Provider ecosystem:
   - Standardized provider contract for new price/import plugins.
   - Community provider registry documentation.
4. Mobile-friendly workflow:
   - PWA enhancements for quick capture and review.
   - Improved responsive UX for dashboard and transactions.

### Success criteria

1. More users running unattended sync successfully.
2. Growing contributor adoption of extension points.
3. Better mobile usage completion rates for key tasks.

## Guardrails

1. Keep ledger files as source of truth; generated state remains reproducible.
2. Preserve backward compatibility in public API contracts where possible.
3. Ship each quarter in small, testable increments with regression coverage.
4. Treat security and readonly guarantees as release blockers for write features.

## Not Planned in This Window

1. Cloud-hosted multi-tenant Paisa service.
2. Real-time market data streaming infrastructure.
3. Proprietary closed-source plugin marketplace.
