# Simple Features Roadmap 2026

High-value, low-complexity feature epics with detailed subtasks for scoping and implementation.

---

## Epic 1: Account-Level Transaction Drill-Down

**Goal:** Allow users to click on any account balance or widget to view filtered transaction history for that account.

**Core Value Prop:** Quick access from dashboard to detailed transaction view; reduces clicks to answer "what happened in this account?"

**Technical Considerations:**
- Reuse existing `/api/transaction` endpoint with account filtering
- Create new route: `(app)/accounts/[name]/transactions`
- Leverage existing transaction UI components

### Subtask 1.1: Backend – Add Account Filter to Transaction Endpoint
**Description:** Extend `/api/transaction` handler to accept optional `account` query parameter.

**Acceptance Criteria:**
- `GET /api/transaction?account=Assets:Checking` returns only postings from that account
- Filter is case-insensitive and handles account hierarchy (e.g., `Assets:Checking` matches postings from `Assets:Checking` and children if configured)
- Returns empty array if account doesn't exist (graceful)
- Response time < 200ms for accounts with 1000+ transactions

**Implementation Notes:**
- Modify `GetTransactions()` in `internal/server/transaction.go`
- Use existing `query.Init(db).Like(account)` pattern from capital gains handler
- Ensure account name is validated/escaped
- Add test case with nested accounts

**Testing Strategy:**
- Unit test: verify filtering logic with various account formats
- Integration: call endpoint with real DB, verify only correct postings returned
- Edge case: empty account, nested account, special characters in account name

---

### Subtask 1.2: Frontend – Create Account Transactions Page
**Description:** Add new SvelteKit route `(app)/accounts/[name]/transactions/+page.svelte` to display filtered transactions.

**Acceptance Criteria:**
- Page loads transaction list for the account name in URL
- Breadcrumb shows: Dashboard > Assets > [Account Name] > Transactions
- Account name is URL-decoded and displayed prominently at top
- Uses existing Transaction table component
- Back button returns to appropriate parent page
- Mobile responsive

**Implementation Notes:**
- Create `src/routes/(app)/accounts/[name]/transactions/+page.svelte`
- Create `+page.ts` server load function that fetches account name and calls `/api/transaction?account=[name]`
- Reuse `TransactionTable` or `Posting` display component
- Add breadcrumb navigation via store or prop
- Handle 404 if account doesn't exist (check against `config.accounts` list)

**Testing Strategy:**
- Unit: test URL decoding, account existence check
- Integration: load page for valid/invalid accounts, verify data loads
- Visual: check responsive layout on mobile/tablet
- Cross-browser: ensure history back works correctly

---

### Subtask 1.3: Frontend – Add Drill-Down Clickable Areas
**Description:** Make account balances on dashboard and asset pages clickable to navigate to account transactions.

**Acceptance Criteria:**
- Dashboard: clicking "Assets: $X" navigates to account transactions page
- Asset page: account name/balance links to transactions for that account
- Cursor changes to pointer on hover (visual feedback)
- Link uses account hierarchy path (e.g., `Assets/Checking`)
- Works for all account types (Assets, Liabilities, Income, Expenses)

**Implementation Notes:**
- Modify dashboard widget display (wherever account balances are shown)
- Modify asset balance page table row rendering
- Use `<a href="/accounts/{encodeURIComponent(account)}/transactions">`
- Ensure account names are properly URL-encoded (spaces, special chars)

**Testing Strategy:**
- Click each account balance type on dashboard, verify navigation
- Verify URL encoding for accounts with spaces/special characters
- Test back/forward browser buttons

---

## Epic 2: Recent Transactions Widget on Dashboard

**Goal:** Display a feed of the 10–15 most recent transactions on the main dashboard.

**Core Value Prop:** At-a-glance activity overview; helps users catch data entry errors and unexpected activity quickly.

**Technical Considerations:**
- Reuse transaction API; add `limit` parameter
- New widget component with minimal styling
- Update dashboard layout to include widget

### Subtask 2.1: Backend – Add Limit/Offset to Transaction Endpoint
**Description:** Extend `/api/transaction` to support `limit` and `offset` query parameters for pagination.

**Acceptance Criteria:**
- `GET /api/transaction?limit=15` returns at most 15 transactions, most recent first
- `GET /api/transaction?limit=15&offset=15` returns next page
- Default limit is 500 (current behavior if not specified)
- Offset defaults to 0
- Limit is capped at 500 (prevent abuse)
- Response includes total count for pagination UI

**Implementation Notes:**
- Modify `GetTransactions()` in `internal/server/transaction.go`
- Order by date DESC, then by posting ID DESC for tie-breaking
- Add `total_count` to response JSON
- Validate limit > 0 and limit ≤ 500

**Testing Strategy:**
- Unit: test limit/offset logic
- Integration: verify ordering is correct (most recent first), verify total count
- Boundary: test limit=0, limit=1000, offset=huge number

---

### Subtask 2.2: Frontend – Create Recent Transactions Widget
**Description:** Build a new dashboard widget that displays recent 10–15 transactions in a compact table or card list.

**Acceptance Criteria:**
- Widget shows date, account(s), description/payee, amount
- Displays 15 most recent transactions by default
- Clicking a transaction row navigates to transaction details or drill-down
- Widget has a "View All" link to transaction page
- Responsive: on mobile, show 8 transactions with smaller fonts
- Widget title shows "Recent Activity" or "Latest Transactions"

**Implementation Notes:**
- Create `src/lib/components/RecentTransactionsWidget.svelte`
- Fetch from `/api/transaction?limit=15` in `+page.ts` (dashboard load function)
- Reuse transaction table styling; make columns narrow
- Add "View All Transactions" button at bottom of widget
- Handle empty state: "No transactions yet"

**Testing Strategy:**
- Render with 0, 1, 5, 15+ transactions
- Verify responsive layout on mobile (tablet, phone widths)
- Check link navigation works

---

### Subtask 2.3: Frontend – Integrate Widget into Dashboard Layout
**Description:** Add the recent transactions widget to the dashboard grid layout.

**Acceptance Criteria:**
- Widget appears on main dashboard page (likely right side or bottom)
- Layout remains responsive when widget is added
- Widget doesn't push other widgets off-screen on mobile
- Dashboard still loads in < 2 seconds

**Implementation Notes:**
- Add to `src/routes/(app)/+page.svelte`
- Place after main networth/budget widgets or in side column
- Adjust grid layout if needed; consider CSS grid adjustments
- Ensure dashboard load function fetches transaction data

**Testing Strategy:**
- Visual regression on multiple screen sizes
- Performance: dashboard load time benchmark
- Mobile: test layout shift on all common mobile widths

---

## Epic 3: Simple Account Notes / Metadata

**Goal:** Allow users to attach notes to accounts (e.g., "Emergency fund", "Company 401k") without editing the ledger file.

**Core Value Prop:** Contextualize account purpose; helps new users understand account organization; aids in account reconciliation notes.

**Technical Considerations:**
- Create `account_metadata` DB table (account_name, notes, created_at, updated_at)
- New API endpoints for CRUD
- Add notes field to account detail pages

### Subtask 3.1: Backend – Create Account Metadata Table
**Description:** Add database schema for storing account notes and metadata.

**Acceptance Criteria:**
- Table `account_metadata` with columns: id, account_name (unique), notes, created_at, updated_at
- account_name is indexed (fast lookup)
- Supports up to 5000 character notes
- GORM migration script creates table on startup

**Implementation Notes:**
- Create GORM model: `internal/model/account_metadata.go`
- Define struct with gorm.Model, AccountName (string, index), Notes (string, type:text)
- Add migration in `internal/model/migrations.go` or auto-migration in DB init
- Ensure account_name matches ledger account names exactly (case-sensitive)

**Testing Strategy:**
- Unit: test migration creates table schema correctly
- Integration: verify table exists, can insert/update rows
- Constraint: test unique account_name constraint

---

### Subtask 3.2: Backend – CRUD Endpoints for Account Notes
**Description:** Implement `/api/accounts/{name}/notes` endpoints for GET, POST, PUT.

**Acceptance Criteria:**
- `GET /api/accounts/Assets:Checking/notes` returns `{ "account": "Assets:Checking", "notes": "..." }` or 404
- `POST /api/accounts/Assets:Checking/notes` with body `{ "notes": "Emergency fund" }` creates/updates
- `PUT /api/accounts/Assets:Checking/notes` updates existing (or creates if not exists)
- Account name in URL is validated against known accounts from config
- Returns 400 if account doesn't exist
- Response includes created_at, updated_at timestamps
- Rate-limited via WriteGroup (6 req/min)

**Implementation Notes:**
- Add handlers in new file `internal/server/account_metadata.go`
- Use `url.PathUnescape()` to handle encoded account names
- Validate account exists: cross-check against `accounting.AllAccounts(db)`
- Write operations guarded by `writeGroup.POST/PUT` in `server.go`
- Return error using `RespondError` pattern from `apierror.go`

**Testing Strategy:**
- Unit: test URL decoding, account validation
- Integration: create, read, update notes; verify DB persistence
- Error cases: invalid account, missing body, malformed JSON
- Concurrency: rapid updates don't corrupt data

---

### Subtask 3.3: Frontend – Add Notes Field to Account Detail Page
**Description:** Display and edit account notes on account detail pages.

**Acceptance Criteria:**
- Account detail page shows "Account Notes" section
- Notes are editable inline or via modal
- Save button persists notes to backend
- Unsaved changes warning if user tries to leave page
- On load, fetch and display existing notes
- Empty state: "No notes yet. Add one to track this account's purpose."
- Mobile responsive

**Implementation Notes:**
- Add to existing account detail page or create new section
- Fetch notes from `/api/accounts/{name}/notes` on page load
- POST/PUT to same endpoint on save
- Use existing modal or inline edit pattern
- Handle loading/error states with toast notifications

**Testing Strategy:**
- Create notes, reload page, verify persistence
- Update notes, verify changes saved
- Leave page with unsaved changes, verify warning
- Check responsive layout on mobile

---

### Subtask 3.4: Frontend – Show Notes in Account List / Widget
**Description:** Display account notes (truncated) in account lists, asset balance views, and drill-down pages.

**Acceptance Criteria:**
- Account list shows first 50 characters of notes in parentheses or as tooltip
- Tooltip on hover shows full notes text
- Asset balance table row shows account note as secondary text
- Account drill-down page (from Epic 1) shows full notes at top
- Mobile: notes display as subtitle below account name

**Implementation Notes:**
- Modify account display components to include notes
- Fetch notes when loading account lists (batch call to `/api/accounts/notes/list` if needed, or include in existing responses)
- Truncate notes with ellipsis if > 50 chars
- Use title attribute or tooltip library for full text on hover

**Testing Strategy:**
- Verify truncation logic
- Check tooltip displays full text
- Mobile: verify layout doesn't break with long notes

---

## Epic 4: Month-over-Month Spending Trends

**Goal:** Show spending trend comparisons for each budget category or expense type (e.g., "Groceries: $450 (last month: $420, +7%)").

**Core Value Prop:** Users see spending patterns immediately; helps identify budget-busting categories early in the month.

**Technical Considerations:**
- Extend expense API to return historical month data
- Calculate month-over-month deltas
- Add trend indicators to UI (up/down arrows)

### Subtask 4.1: Backend – Calculate Monthly Expense Trends
**Description:** Extend expense calculation to include current month vs. previous month for each category.

**Acceptance Criteria:**
- `/api/expense` response now includes `{ category: "Groceries", current_month: 450.00, previous_month: 420.00, variance: 30.00, variance_pct: 7.14 }`
- Variance is (current - previous) / previous * 100, rounded to 2 decimals
- Works across multiple currencies (report_currency query param)
- Previous month is relative to today, not calendar month start
- Handles missing data gracefully (e.g., no expenses in previous month = variance = null)

**Implementation Notes:**
- Modify `GetExpense()` in `internal/server/expense.go`
- Calculate month ranges: today-30 to today (current), today-60 to today-30 (previous)
- Query postings for both ranges, group by account/category
- Return both current and previous in response, let frontend calculate if preferred
- Handle multi-currency edge cases

**Testing Strategy:**
- Unit: test variance calculation (edge cases: 0%, negative %, very large %)
- Integration: verify month ranges are correct, data matches manual calculation
- Regression: ensure existing expense data still works

---

### Subtask 4.2: Frontend – Display Trend Indicators on Expense Cards
**Description:** Add trend indicators (↑ red, ↓ green) and variance % to expense breakdown cards.

**Acceptance Criteria:**
- Expense card shows current month amount prominently
- Below it: previous month amount in lighter text
- Variance % shown with color: red if up, green if down
- Up/down arrow icon next to variance %
- Format: "Groceries • +7.14%" (in red) or "Utilities • -12.5%" (in green)
- Tooltip on hover shows "Previous month: $420, Current: $450"
- Mobile: compact format to save space

**Implementation Notes:**
- Modify expense display component(s) to render trend UI
- Use existing color system (red=overspend, green=savings)
- Add arrow SVG icons or use icon font
- Calculate percentage in frontend from API response

**Testing Strategy:**
- Render with various variance %s (positive, negative, 0, null)
- Verify color coding is correct
- Check tooltip content
- Mobile responsive

---

### Subtask 4.3: Frontend – Monthly Trend Chart Extension
**Description:** Add optional sparkline or mini-chart showing 6-month trend for each category.

**Acceptance Criteria:**
- Clicking on expense card or "view trend" link shows 6-month history chart
- Chart shows monthly bars or line for that category
- X-axis: month labels (Jan, Feb, Mar, etc.)
- Y-axis: amount in report currency
- Average line shown (dashed)
- Current month highlighted
- Responsive and mobile-friendly

**Implementation Notes:**
- Use D3 (existing in codebase) or simple SVG bars
- Fetch `/api/expense?months=6` or extend endpoint to return historical
- Create reusable TrendChart component
- Optional: link from card to modal with chart

**Testing Strategy:**
- Render chart with 3, 6, 12 months of data
- Verify average calculation
- Check responsive layout

---

## Epic 5: Budget Variance Analysis by Category

**Goal:** Dashboard widget showing budget status: "X categories on track, Y over budget" with drill-down to see which categories exceeded allocations.

**Core Value Prop:** Actionable budget visibility; users know immediately which categories need attention without opening budget page.

**Technical Considerations:**
- Extend budget API to include variance
- Create summary widget for dashboard
- Reuse existing budget comparison logic

### Subtask 5.1: Backend – Add Budget Variance to Budget Endpoint
**Description:** Extend `/api/budget` to include variance (actual vs. budgeted) for current month.

**Acceptance Criteria:**
- Response includes `{ category: "Dining", budgeted: 200.00, actual: 245.00, variance: -45.00, variance_pct: -22.5, on_track: false }`
- variance is (actual - budgeted), negative means overspend
- on_track is true if actual <= budgeted
- Works for current month and query param `?month=2026-05`
- Handles missing budget (variance = null)
- Calculates actual from latest postings

**Implementation Notes:**
- Modify `GetBudget()` in `internal/server/budget.go`
- Calculate current month expenses (ledger entries)
- Compare against configured budget
- on_track = actual <= budgeted (with tolerance if needed)

**Testing Strategy:**
- Unit: variance calculation, on_track logic
- Integration: verify against configured budgets and actual postings
- Boundary: 0% variance, exactly at budget, over/under

---

### Subtask 5.2: Backend – Budget Summary Endpoint
**Description:** New endpoint `/api/budget/summary` returns high-level budget status.

**Acceptance Criteria:**
- Response: `{ on_track_count: 5, over_budget_count: 2, total_budgeted: 5000.00, total_actual: 5300.00, variance_pct: 6.0 }`
- on_track_count = # of categories where actual <= budgeted
- over_budget_count = # where actual > budgeted
- total_* sums all budgeted/actual amounts
- variance_pct = (total_actual - total_budgeted) / total_budgeted * 100
- Query param `?month=2026-05` for specific month (default: current)

**Implementation Notes:**
- Create new handler in `budget.go`
- Reuse budget variance logic from Epic 5.1
- Register route in `server.go`

**Testing Strategy:**
- Integration: verify counts, totals, percentages match manual calculation

---

### Subtask 5.3: Frontend – Budget Status Widget
**Description:** Create dashboard widget showing budget summary with drill-down.

**Acceptance Criteria:**
- Widget title: "Budget Status (May 2026)" with month picker
- Shows: "5 on track, 2 over budget"
- Progress bar showing total budgeted vs. actual (red if over)
- "View Details" button expands to show list of categories
- List shows category name, budgeted, actual, variance % (sorted by worst variance)
- Categories over budget highlighted in red
- Mobile: compact layout, list scrollable

**Implementation Notes:**
- Create `src/lib/components/BudgetStatusWidget.svelte`
- Fetch `/api/budget/summary?month={currentMonth}` on load
- Clicking category name navigates to budget detail page
- Month picker updates query

**Testing Strategy:**
- Render with various on_track/over_budget counts
- Verify list sorting (worst variance first)
- Month picker changes month correctly
- Mobile layout

---

## Epic 6: Account Balance Snapshot as of Date

**Goal:** Allow users to view historical account balances as of any date (e.g., "view balance as of 2026-03-01").

**Core Value Prop:** Historical reconciliation; helps explain balance changes over time; useful for statement matching.

**Technical Considerations:**
- Extend balance calculation to accept `as_of_date` parameter
- Reuse existing balance query logic with date filtering
- Create date picker UI on balance pages

### Subtask 6.1: Backend – Add Date Filter to Balance Endpoints
**Description:** Extend asset balance and account balance endpoints to accept `as_of_date` query parameter.

**Acceptance Criteria:**
- `GET /api/assets/balance?as_of_date=2026-03-01` returns balances as of 2026-03-01 EOD
- `GET /api/account/Assets:Checking/balance?as_of_date=2026-03-01` returns specific account balance
- Default: as_of_date = today if not specified
- Validates date format (YYYY-MM-DD)
- Returns 400 if date is in future or invalid format
- Balance calculation excludes postings after as_of_date

**Implementation Notes:**
- Modify balance calculation queries to add `WHERE date <= as_of_date`
- Update `accounting` package balance functions to accept optional date
- Ensure date comparison works across timezones (use UTC)
- Add new endpoint or extend existing with optional parameter

**Testing Strategy:**
- Unit: test date parsing, boundary (exact date, day after, day before)
- Integration: verify balance matches manual calculation for historic dates
- Edge cases: as_of_date before first posting, as_of_date = today

---

### Subtask 6.2: Frontend – Add Date Picker to Balance Pages
**Description:** Add date picker to asset balance and account detail pages to view historic balances.

**Acceptance Criteria:**
- Balance page has "View as of" date picker in header
- Date picker defaults to today
- Clicking a past date updates balance display
- Balance recalculates without page reload
- Display shows "as of May 1, 2026" clearly
- Mobile responsive date picker

**Implementation Notes:**
- Add date picker UI (use existing date picker component or add one)
- Add parameter to balance API call
- Update balance display when date changes
- Store selected date in component state (not URL for now)

**Testing Strategy:**
- Pick various past dates, verify balance updates
- Navigate away and back, date resets to today (expected behavior)
- Mobile date picker usability

---

### Subtask 6.3: Frontend – Historical Balance Trend View
**Description:** Optional: Show balance trend line for selected account over a date range.

**Acceptance Criteria:**
- "View Trend" button on account detail opens 6/12-month view
- Chart shows daily or weekly balance for the account
- User can select start/end date for trend window
- Current balance marked on chart
- Responsive and zooming-friendly

**Implementation Notes:**
- Create TrendChart component (reuse from Epic 4 if possible)
- Fetch daily/weekly balance snapshots
- Consider performance: may need backend aggregation for large date ranges

**Testing Strategy:**
- Chart renders correctly for various date ranges
- Performance acceptable for 1+ year of daily data

---

## Epic 7: Duplicate Transaction Detection

**Goal:** Detect and flag likely duplicate transactions (same amount, account, date within 1-2 days).

**Core Value Prop:** Catch data entry errors early; prevent overstating expenses and income.

**Technical Considerations:**
- Run detection on transaction list or during sync
- Create flagging system (flag in DB, return in API response)
- UI alerts and bulk delete capability

### Subtask 7.1: Backend – Duplicate Detection Algorithm
**Description:** Implement algorithm to identify potential duplicate postings.

**Acceptance Criteria:**
- Identifies postings with: same amount, same account, within 1-2 day date window
- Confidence score: 100% if all match, decreases if only 2 of 3 match
- Does NOT flag legitimate recurring transactions (user can suppress)
- Runs in < 500ms for 10K postings
- Ignores archived/deleted postings

**Implementation Notes:**
- Create `internal/server/duplicate_detection.go`
- Algorithm: group postings by account, then by amount, then check date distance
- Use hash/map for O(n) performance
- Return list of potential duplicate pairs with confidence score

**Testing Strategy:**
- Unit: test algorithm with known duplicates and non-duplicates
- Performance: benchmark with 10K+ postings
- Edge cases: same amount but different accounts, different amounts same date

---

### Subtask 7.2: Backend – Duplicate Detection Endpoint
**Description:** New endpoint `/api/transactions/duplicates` returns flagged duplicates.

**Acceptance Criteria:**
- Response: `[ { id1: 123, id2: 124, amount: 50.00, account: "Assets:Checking", date_diff_days: 1, confidence: 0.95 }, ... ]`
- Sorted by confidence descending (highest first)
- Optional query params: `account=Assets:Checking`, `date_range_days=2`
- Returns empty array if none found
- Response time < 1s

**Implementation Notes:**
- Create handler in `transaction.go` or new file
- Use algorithm from 7.1
- Add configuration option in paisa.yaml for date_range_days (default: 2)

**Testing Strategy:**
- Integration: find known duplicates in test data

---

### Subtask 7.3: Frontend – Duplicate Alert Widget
**Description:** Add warning badge to dashboard or transaction list showing duplicate count.

**Acceptance Criteria:**
- Dashboard shows red badge: "⚠️ 2 potential duplicates detected"
- Clicking badge navigates to duplicate list page
- Duplicate list shows side-by-side comparison of potential duplicates
- "Mark as OK" button to suppress false positives
- "Delete" button to remove one of the duplicates
- Mobile responsive

**Implementation Notes:**
- Fetch `/api/transactions/duplicates` on dashboard load
- Show badge if count > 0
- Create duplicate comparison page with two-column layout
- Store suppressions in DB (duplicate_suppressions table)

**Testing Strategy:**
- Render duplicate list with various confidence scores
- Delete functionality works
- Suppressions persist

---

### Subtask 7.4: Backend – Suppression & Batch Delete
**Description:** Allow users to suppress false-positive duplicates and bulk delete actual duplicates.

**Acceptance Criteria:**
- `POST /api/transactions/duplicates/{id1}/{id2}/suppress` marks as not-duplicate (stored in DB)
- `DELETE /api/transactions/{id}/delete` soft-deletes transaction
- Suppressed pairs don't appear in future detection results
- Deletions are logged for audit

**Implementation Notes:**
- Create `duplicate_suppressions` table
- Add soft-delete logic if not already present
- Rate-limit delete endpoint (WriteGroup)

**Testing Strategy:**
- Suppress, reload, verify doesn't show again
- Delete, verify journalremains unchanged but posting flagged

---

## Epic 8: Spending by Payee Breakdown

**Goal:** Show spending grouped by payee (e.g., "Amazon: $2,100 (42 transactions)") with drill-down to transactions.

**Core Value Prop:** Understand where money goes; identify biggest spending vendors; potential negotiation/switching opportunities.

**Technical Considerations:**
- Parse payee from transaction description
- Extend transaction/expense API with payee grouping
- Add payee management (alias mapping)

### Subtask 8.1: Backend – Payee Extraction and Grouping
**Description:** Extract and group transactions by payee; support payee aliases.

**Acceptance Criteria:**
- Payee extracted from transaction description/narration (first word or configurable pattern)
- Transactions grouped by payee with total spent, transaction count
- Support payee aliases (e.g., "Amazon Inc." → "Amazon", "Whole Foods" → "Amazon")
- Aliases configurable via paisa.yaml or API
- Returns top 10 payees by default, configurable via query param

**Implementation Notes:**
- Create payee extraction logic in `internal/accounting/payee.go`
- Parse payee from posting description (first word, or regex if specified)
- Create `payee_aliases` table or embed in config
- Group postings by payee, sum amounts

**Testing Strategy:**
- Unit: test payee extraction from various description formats
- Test alias mapping
- Verify grouping accuracy

---

### Subtask 8.2: Backend – Payee Breakdown Endpoint
**Description:** New endpoint `/api/expense/by-payee` returns spending grouped by payee.

**Acceptance Criteria:**
- Response: `[ { payee: "Amazon", total: 2100.00, transaction_count: 42, avg_transaction: 50.00 }, ... ]`
- Sorted by total descending
- Optional query params: `limit=20`, `account=Expenses:*`, `date_range=2026-05-01:2026-05-31`
- Supports multi-currency (report_currency)

**Implementation Notes:**
- Create handler in `internal/server/expense.go` or new file
- Reuse payee grouping from 8.1
- Apply date/account filters

**Testing Strategy:**
- Integration: verify payees grouped correctly, totals accurate

---

### Subtask 8.3: Frontend – Payee Breakdown Widget
**Description:** Add widget or page showing top payees with click-through to transactions.

**Acceptance Criteria:**
- Widget shows top 10 payees as list/table
- Format: "Amazon | $2,100 (42 txns)"
- Clicking payee name filters transaction list to that payee
- "View All Payees" link shows full list
- Mobile: compact layout, scrollable
- Option to view as bar chart (top 10 payees vs. spending)

**Implementation Notes:**
- Create `src/lib/components/TopPayeesWidget.svelte`
- Fetch `/api/expense/by-payee?limit=10`
- Clicking payee passes filter to transaction page
- Create optional chart view using D3

**Testing Strategy:**
- Render with various payee counts
- Click-through filters transactions correctly
- Chart displays correctly

---

### Subtask 8.4: Backend – Payee Alias Management
**Description:** CRUD endpoints for payee aliases to unify similar payees.

**Acceptance Criteria:**
- `GET /api/payee-aliases` returns all configured aliases
- `POST /api/payee-aliases` creates new alias mapping
- `PUT /api/payee-aliases/{id}` updates alias
- `DELETE /api/payee-aliases/{id}` removes alias
- Alias mapping is case-insensitive
- Changes apply immediately to grouping logic

**Implementation Notes:**
- Create `payee_aliases` DB table
- Create CRUD handlers in new file
- Reload alias cache after changes

**Testing Strategy:**
- CRUD operations work
- Alias mapping affects grouping results

---

## Epic 9: Quick-Add Transaction Button

**Goal:** Provide floating action button or prominent button to quickly add transactions without navigating away.

**Core Value Prop:** Reduce friction for mobile/quick entries; increase data capture during day.

**Technical Considerations:**
- Reuse existing `/api/transaction/add` endpoint
- Modal or side-drawer form
- Support pre-filling based on context

### Subtask 9.1: Frontend – Quick-Add Modal Component
**Description:** Create reusable modal for quick transaction entry.

**Acceptance Criteria:**
- Modal has fields: Date, Amount, From Account (dropdown), To Account (dropdown), Description
- Date defaults to today
- Amount field accepts decimal, currency symbols, math expressions (e.g., "100 * 1.08")
- Accounts are searchable/autocomplete dropdowns
- Description field has autocomplete from recent transactions
- "Add & Close" and "Add & New" buttons
- Form validation: amount > 0, accounts selected, date valid
- Mobile responsive (full-screen on mobile, modal on desktop)

**Implementation Notes:**
- Create `src/lib/components/QuickAddTransactionModal.svelte`
- Use existing form inputs/selects
- Reuse account autocomplete logic
- Pre-populate from context if possible (e.g., current account)

**Testing Strategy:**
- Submit valid/invalid transactions
- Test autocomplete suggestions
- Test math expression evaluation
- Mobile responsiveness

---

### Subtask 9.2: Frontend – Floating Action Button
**Description:** Add FAB to dashboard or main layout to open quick-add modal.

**Acceptance Criteria:**
- FAB positioned fixed on bottom-right (mobile) or visible on desktop
- FAB icon is + or "Add Transaction"
- Keyboard shortcut: Cmd+Shift+A (Mac) or Ctrl+Shift+A (Windows/Linux)
- FAB is visible on all pages in (app) layout
- FAB doesn't cover other UI elements (z-index, positioning)
- Mobile: FAB visible in safe area

**Implementation Notes:**
- Add to `src/routes/(app)/+layout.svelte`
- Use sticky positioning or fixed
- Wire to open/close quick-add modal via store
- Add keyboard listener for shortcut

**Testing Strategy:**
- FAB visible on mobile and desktop
- Click opens modal
- Keyboard shortcut works
- Doesn't overlap content

---

### Subtask 9.3: Backend – Enhance Transaction Add Endpoint
**Description:** Ensure `/api/transaction/add` supports all quick-add fields and returns clear errors.

**Acceptance Criteria:**
- Accepts: date, amount, from_account, to_account, description
- Validates all fields before posting
- Creates balanced posting pair automatically
- Returns transaction ID on success
- Returns detailed validation errors on failure
- Appends to configured `add_journal_path` file

**Implementation Notes:**
- Verify endpoint exists and supports all fields (from handoff, likely already exists)
- Enhance error messages
- Ensure it uses `add_journal_path` config

**Testing Strategy:**
- Integration: create transaction via endpoint, verify journal updated

---

### Subtask 9.4: Frontend – Recent Suggestions for Quick-Add
**Description:** Pre-fill quick-add form with suggestions based on recent transactions.

**Acceptance Criteria:**
- "From Account" dropdown shows 5 most recently used accounts at top
- Description field autocomplete suggests recent descriptions (grouped, deduplicated)
- Clicking suggestion pre-fills form
- Accounts sorted by frequency + recency

**Implementation Notes:**
- Track recent account usage in store or via API
- Fetch recent descriptions from `/api/transaction` and deduplicate
- Sort by frequency and recency

**Testing Strategy:**
- Verify suggestions are accurate and fresh

---

## Epic 10: Balance Projection (Simple 3-Month)

**Goal:** Project account balances 1–3 months ahead based on recurring transactions.

**Core Value Prop:** Anticipate cash flow needs; avoid surprises; simple forward-looking planning.

**Technical Considerations:**
- Detect recurring transactions
- Calculate average/trend from last 3 months
- Simple linear projection

### Subtask 10.1: Backend – Recurring Transaction Detection
**Description:** Identify recurring transactions from historical data.

**Acceptance Criteria:**
- Detects transactions that occur monthly, bi-weekly, weekly, bi-monthly
- Confidence score: high if pattern clear over 3+ occurrences, medium otherwise
- Groups similar transactions by account, amount (±10% variance allowed)
- Returns pattern: frequency, average amount, last occurrence
- Handles payroll (fixed amount, regular date), utilities (similar amounts, regular date), subscriptions

**Implementation Notes:**
- Create `internal/accounting/recurring.go`
- Analyze last 90 days of transactions
- Group by account and description pattern
- Detect frequency using date diff analysis
- Filter out one-off large transactions

**Testing Strategy:**
- Unit: test frequency detection (weekly, monthly, etc.)
- Test variance tolerance (±10%)
- Test with synthetic recurring patterns

---

### Subtask 10.2: Backend – Balance Projection Endpoint
**Description:** New endpoint `/api/balance/projection` returns projected balances 1–3 months ahead.

**Acceptance Criteria:**
- Response: `{ account: "Assets:Checking", current: 5000.00, projections: [ { month: "2026-06", projected: 5150.00 }, { month: "2026-07", projected: 5200.00 }, ... ] }`
- Projections include recurring transactions identified in 10.1
- Month defaults to "current month" and 2 months ahead
- Query param: `?months=6` to project 6 months
- Includes confidence score for each projection

**Implementation Notes:**
- Create handler in new file `internal/server/projection.go`
- Use recurring transaction detection from 10.1
- Simple linear projection: current + (sum of recurring * months)
- Add confidence score: high if recurring pattern clear, low otherwise

**Testing Strategy:**
- Integration: verify projection matches expected values with known recurring transactions

---

### Subtask 10.3: Frontend – Balance Projection Widget
**Description:** Dashboard widget showing 3-month balance projection.

**Acceptance Criteria:**
- Widget title: "Balance Outlook (Assets:Checking)"
- Shows current balance and projected balances for next 3 months
- Display format: "June 2026: $5,150 (↑$150)"
- Optional chart view: line chart with current + 3 projected points
- Click account name to see details and recurring transactions used in projection
- Confidence indicator (e.g., "High confidence" if based on clear patterns)

**Implementation Notes:**
- Create `src/lib/components/BalanceProjectionWidget.svelte`
- Fetch `/api/balance/projection?months=3` for relevant accounts
- Show top accounts (by balance)
- Optional: dropdown to select different account

**Testing Strategy:**
- Render with various projection scenarios
- Chart displays correctly
- Click-through to details works

---

### Subtask 10.4: Frontend – Projection Details & Assumptions
**Description:** Detail page showing projection assumptions and recurring transactions used.

**Acceptance Criteria:**
- Shows account name and current balance prominently
- Lists recurring transactions identified: name, frequency, amount, confidence
- Shows projection month-by-month with running total
- User can toggle recurring transactions on/off to see impact
- Shows "What-if" scenarios: +10% spending, +5% income, etc.
- Projections update dynamically when assumptions changed

**Implementation Notes:**
- Create detail page for projection
- Fetch recurring transaction list
- Allow toggling each recurring transaction
- Recalculate projection client-side when toggled

**Testing Strategy:**
- Verify projections update when toggling recurring transactions
- Check what-if scenarios

---

## Epic 11: Year-over-Year Comparison Charts

**Goal:** Overlay and compare spending/income across years (e.g., "Spending 2025 vs 2026").

**Core Value Prop:** Identify long-term spending trends; see if habits improving/degrading; spot seasonal patterns.

**Technical Considerations:**
- Extend expense/income endpoints to return multi-year data
- Chart multiple series on same axis
- Align months across years for comparison

### Subtask 11.1: Backend – Multi-Year Expense/Income Data
**Description:** Extend `/api/expense` and `/api/income` to return data for multiple years.

**Acceptance Criteria:**
- Query param: `?years=2` returns current year + prior year (2025, 2026)
- Response includes separate data series per year
- Format: `{ 2025: { month: { "2025-01": 100, "2025-02": 120, ... }, total: 1200 }, 2026: { ... } }`
- Accounts for Feb 29 in leap years (or aggregate by calendar week)
- Filtered by account/category as usual

**Implementation Notes:**
- Modify `GetExpense()` and `GetIncome()` to accept `years` query param
- Return data in multi-year format
- Ensure date alignment (e.g., Jan 2025 vs Jan 2026)

**Testing Strategy:**
- Verify data for multiple years, totals accurate
- Test leap year handling

---

### Subtask 11.2: Frontend – YoY Comparison Chart
**Description:** Create chart component that plots multiple years side-by-side or overlay.

**Acceptance Criteria:**
- Chart shows 2 series (current year and prior year) overlaid
- X-axis: months (Jan–Dec)
- Y-axis: expense/income amount
- Each series has distinct color
- Legend shows year labels
- Hover shows values for both years
- Optional: bar chart (grouped) or line chart (overlay)
- Responsive and legend responsive on mobile

**Implementation Notes:**
- Use D3 for charting (existing in codebase)
- Create reusable YoYChart component
- Support both line and bar chart types

**Testing Strategy:**
- Chart renders with 2–3 years of data
- Hover data display accurate
- Mobile responsive

---

### Subtask 11.3: Frontend – YoY Analysis Page
**Description:** Create analysis page for YoY comparison with insights.

**Acceptance Criteria:**
- Page shows multiple YoY charts: total spending, total income, by category
- Category comparison: toggle between line and bar chart
- Insights section: "Spending up 12% YoY", "Top expense category: Groceries (+5% YoY)", etc.
- Can select different year ranges (e.g., 2023 vs 2024 vs 2025)
- Downloadable as PDF or CSV

**Implementation Notes:**
- Create `src/routes/(app)/analysis/yoy/+page.svelte`
- Compute insights from data (percentage changes, top categories)
- Add PDF export (optional, lower priority)

**Testing Strategy:**
- Verify insights calculations
- Charts render correctly for multiple year ranges

---

## Epic 12: Account Reconciliation Status Tracker

**Goal:** Track which accounts have been reconciled and when; optional reminders for overdue reconciliations.

**Core Value Prop:** Organized users stay on top of account validation; helps prevent undetected errors.

**Technical Considerations:**
- Add reconciliation metadata to accounts
- Configurable reconciliation frequency per account
- Optional notifications

### Subtask 12.1: Backend – Account Reconciliation Metadata
**Description:** Store reconciliation status and frequency for accounts.

**Acceptance Criteria:**
- New table: `account_reconciliation` with columns: account_name, last_reconciled_date, frequency_days (monthly=30, quarterly=90, weekly=7)
- Can update via PATCH `/api/accounts/{name}/reconciliation`
- Returns: `{ account: "Assets:Checking", last_reconciled: "2026-04-30", frequency_days: 30, days_since: 4, is_overdue: false }`
- Defaults: frequency_days=30 (monthly), last_reconciled=null initially

**Implementation Notes:**
- Create GORM model and migration
- Create CRUD endpoints
- Calculate days_since and is_overdue on the fly

**Testing Strategy:**
- CRUD operations work
- Overdue calculation correct

---

### Subtask 12.2: Frontend – Reconciliation Status Badge
**Description:** Add reconciliation status badge to account list and detail pages.

**Acceptance Criteria:**
- Asset balance page: show badge next to account name "Last reconciled: 4 days ago"
- Badge color: green if recent, amber if approaching due, red if overdue
- Clicking badge opens reconciliation modal
- Modal shows: last reconciled date, frequency, option to mark as reconciled now
- "Mark Reconciled" button updates backend and closes modal

**Implementation Notes:**
- Fetch reconciliation status from `/api/accounts/{name}/reconciliation`
- Display badge with appropriate color
- Create reconciliation modal

**Testing Strategy:**
- Badge displays correct status
- Clicking marks account as reconciled
- Status refreshes after marking

---

### Subtask 12.3: Frontend – Reconciliation Dashboard Widget
**Description:** Dashboard widget showing reconciliation summary.

**Acceptance Criteria:**
- Widget title: "Account Reconciliation"
- Shows count: "X accounts up-to-date, Y overdue"
- List of overdue accounts with "Reconcile" button
- Click account name to drill down to account detail page
- Mobile responsive

**Implementation Notes:**
- Fetch all accounts and their reconciliation status
- Filter overdue ones
- Display list

**Testing Strategy:**
- Widget renders correctly with various reconciliation states

---

## Epic 13: Export Current View to CSV

**Goal:** Allow users to export visible data (transactions, budget, allocation, etc.) to CSV.

**Core Value Prop:** Take analysis outside Paisa; share data; deeper analysis in spreadsheets.

**Technical Considerations:**
- Reuse backend data endpoints
- CSV serialization
- Handle multi-currency data
- Naming: include date range, report type

### Subtask 13.1: Backend – CSV Export Endpoint
**Description:** Generic endpoint to export various data types to CSV.

**Acceptance Criteria:**
- `/api/export/csv` accepts POST with query filter
- Supports export types: transactions, budget, allocation, expenses, income
- Includes headers and appropriate columns for each type
- Handles special characters, quotes, newlines correctly
- File name includes type and date range (e.g., `transactions_2026-04-01_to_2026-05-04.csv`)
- Returns file with appropriate Content-Disposition header

**Implementation Notes:**
- Create handler in new file `internal/server/export.go`
- Use Go's `encoding/csv` library
- Build CSV from existing API responses
- Return file with correct MIME type and filename

**Testing Strategy:**
- Unit: test CSV generation for various data types
- Integration: export real data, verify opens in Excel/Google Sheets

---

### Subtask 13.2: Frontend – Export Buttons
**Description:** Add export button to key pages (transactions, budget, allocation, expenses).

**Acceptance Criteria:**
- Export button visible on transaction page, budget page, expense analysis page
- Button opens menu: "Export as CSV", "Export as Excel" (if supported)
- Clicking export: POST to `/api/export/csv` with current filters/view
- File downloads automatically
- Button shows loading state while exporting
- Toast notification on success/error

**Implementation Notes:**
- Create reusable ExportButton component
- POST request with filters and data type
- Use existing download utility

**Testing Strategy:**
- Click export button, verify file downloads
- Verify file content matches displayed data
- Test with various filters applied

---

### Subtask 13.3: Frontend – Excel Export (Optional)
**Description:** Support exporting to Excel format with formatting.

**Acceptance Criteria:**
- Export menu includes "Export as Excel"
- Excel file includes formatting: bold headers, column widths, number formatting
- Multiple sheets if needed (e.g., one sheet per month for transactions)
- File naming consistent with CSV

**Implementation Notes:**
- Use existing XLSX library or add dependency (e.g., `exceljs`)
- Create separate export handler or extend CSV handler

**Testing Strategy:**
- Export to Excel, open in Excel, verify formatting

---

## Epic 14: Transaction Tags (Simple Key-Value)

**Goal:** Attach custom tags/labels to transactions (e.g., "travel", "medical", "work-reimbursable") without editing ledger.

**Core Value Prop:** Add custom metadata to transactions; filter/group by custom criteria; organize without ledger changes.

**Technical Considerations:**
- New `transaction_tags` table
- Tag autocomplete
- Filter/group by tags
- Batch tagging

### Subtask 14.1: Backend – Transaction Tags Table & API
**Description:** Store transaction tags in DB; create CRUD endpoints.

**Acceptance Criteria:**
- Table: `transaction_tags` with columns: transaction_id (FK), tag_name, created_at
- Composite index on (transaction_id, tag_name) to prevent duplicates
- GET `/api/transactions/{id}/tags` returns array of tag names
- POST `/api/transactions/{id}/tags` with body `{ "tag": "travel" }` adds tag
- DELETE `/api/transactions/{id}/tags/{tag}` removes tag
- GET `/api/tags/autocomplete?q=tr` returns matching tags (e.g., "travel", "tri-state")

**Implementation Notes:**
- Create GORM model and migration
- Create handlers in `internal/server/transaction.go` or new file
- Autocomplete: query existing tags, order by frequency
- Rate limit write endpoints

**Testing Strategy:**
- CRUD operations work
- Autocomplete returns suggestions
- Prevent duplicate tags on same transaction

---

### Subtask 14.2: Frontend – Tag Display on Transaction Row
**Description:** Show tags on transaction list with ability to add/remove.

**Acceptance Criteria:**
- Transaction row shows tags as small badges/chips
- Clicking "+" or "Add Tag" opens dropdown with autocomplete
- User can type new tag or select from suggestions
- Enter key or click to add tag
- Clicking X on tag removes it (with confirmation)
- Tags are styled with neutral background
- Mobile: tags wrap to next line if needed

**Implementation Notes:**
- Create TagBadge and TagInput components
- Fetch tag autocomplete from `/api/tags/autocomplete`
- POST/DELETE to `/api/transactions/{id}/tags` on add/remove
- Show loading state during save

**Testing Strategy:**
- Add tag, verify saved and displayed
- Remove tag, verify deleted
- Autocomplete works
- Mobile wrapping

---

### Subtask 14.3: Frontend – Filter/Group by Tags
**Description:** Filter transaction list and expense analysis by tags.

**Acceptance Criteria:**
- Transaction list has tag filter: click tag name to filter
- Multiple tags: OR logic (transactions with any of selected tags)
- Expense analysis can group by tag (similar to category grouping)
- Allocation page can show breakdown by tag
- Filter UI shows selected tags with X to remove

**Implementation Notes:**
- Add filter parameter to transaction API calls
- Extend grouping logic to support tags
- Update UI to show tag filter UI

**Testing Strategy:**
- Filter transactions by single/multiple tags
- Verify only matching transactions shown
- Expense/allocation grouping by tags

---

### Subtask 14.4: Frontend – Bulk Tagging
**Description:** Ability to bulk-add tags to multiple transactions.

**Acceptance Criteria:**
- Transaction list has checkbox selection
- "Bulk Actions" menu appears when transactions selected
- "Add Tag" action opens dialog to enter tag(s)
- Applies tag to all selected transactions
- Success toast shows # of transactions tagged
- Undo option available (for next X seconds)

**Implementation Notes:**
- Add checkbox column to transaction table
- Create bulk action UI (menu or toolbar)
- Batch API call or loop through transactions

**Testing Strategy:**
- Select multiple transactions, bulk add tag
- Verify all tagged
- Test undo

---

# Implementation Order Recommendation

**Phase 1 (Quick wins, low complexity):**
1. Epic 1: Account-Level Transaction Drill-Down
2. Epic 2: Recent Transactions Widget on Dashboard
3. Epic 3: Simple Account Notes / Metadata

**Phase 2 (Medium complexity, high value):**
4. Epic 5: Budget Variance Analysis by Category
5. Epic 4: Month-over-Month Spending Trends
6. Epic 8: Spending by Payee Breakdown

**Phase 3 (Medium-high complexity, planning features):**
7. Epic 9: Quick-Add Transaction Button
8. Epic 10: Balance Projection (Simple 3-Month)
9. Epic 6: Account Balance Snapshot as of Date

**Phase 4 (Analytics and advanced):**
10. Epic 11: Year-over-Year Comparison Charts
11. Epic 7: Duplicate Transaction Detection
12. Epic 12: Account Reconciliation Status Tracker

**Phase 5 (Extensibility):**
13. Epic 13: Export Current View to CSV
14. Epic 14: Transaction Tags (Simple Key-Value)

---

# Notes for Development

- **Testing:** Every subtask should have unit tests (for backend logic) and integration/UI tests. Regression tests should be updated.
- **Documentation:** Update CHANGELOG.md after completing each epic.
- **Code Review:** Follow existing patterns in paisa (Go error handling, TypeScript types, Svelte component structure).
- **Performance:** Monitor API response times; aim for < 200ms for typical requests.
- **Mobile:** All UI features should work on mobile (375px width minimum).
- **Accessibility:** Ensure color-coded information (red/green) has text labels too.
