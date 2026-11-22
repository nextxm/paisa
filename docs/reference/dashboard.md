---
description: "Overview of the Paisa dashboard, its customizable widgets, presets, and quick navigation tools"
---

# Dashboard

The **Dashboard** (`/`) is the landing page of Paisa. It provides an at-a-glance overview of your financial health through a customizable two-column collection of widgets.

When you launch Paisa with an empty journal, the Dashboard displays a first-run guide with a **Setup Demo** button that loads a sample journal and configuration so you can explore every view immediately.

## Dashboard Widgets

All nine built-in widgets can be reordered, toggled on or off, and customized:

### Left Column (Default)

- **Assets**: Displays total **Net worth**, **Net Investment**, unrealized **Gain / Loss**, and overall **XIRR** (links to [Net Worth History](./assets.md)).
- **Checking Balance**: Shows current balances for accounts matching `checking_accounts` in `paisa.yaml` (default `Assets:Checking`). Configurable item limit: `1–12` (default `6`).
- **Investment Income**: Displays trailing 12-month (**TTM Dividend** and **TTM Interest**) figures (links to [Investment Income](./income.md#investment-income)).
- **Cash Flow**: Renders a 3-month income vs. expense bar chart in your default currency (links to [Monthly Cash Flow](./cash-flow.md)).
- **Budget**: Displays compact progress bars for the current month's budget accounts (links to [Budget](./budget.md)). Configurable account limit: `1–20` (default `10`).
- **Goals**: Lists configured retirement and savings goals ordered by descending priority (links to [Goals](./goals/index.md)). Configurable item limit: `1–12` (default `6`).

### Right Column (Default)

- **Expenses**: Shows total expenses for the selected month along with an interactive **3-month switcher** and a category breakdown bar chart (links to [Monthly Expenses](./expenses.md#monthly-expenses)).
- **Recurring**: Displays upcoming and due recurring bill cards (links to [Recurring](./recurring.md)). Configurable item limit: `1–30` (default `16`).
- **Recent Transactions**: Displays compact cards for your latest journal transactions (links to [Transactions](./bulk-edit.md)). Configurable item limit: `1–30` (default `10`).

The backing `GET /api/transaction` endpoint also supports optional `limit` (up to `500`), `offset`, `q`, `account`, `commodity`, `amount_min`, `amount_max`, `date_from`, and `date_to` query parameters.

## Customizing the Dashboard

Click the **Customize** (gear) button at the top right of the Dashboard to open the layout editor:

- **Built-in Presets**: Switch quickly between **Overview** (all 9 widgets), **Cash Flow & Budget**, **Investor**, and **Minimal** presets, or click **Reset to default**.
- **Drag-and-Drop Reordering**: Drag widgets within the Left or Right column to change their display order.
- **Visibility & Item Limits**: Toggle any widget on/off and configure item counts (Checking Balance, Budget accounts, Goals, Recurring items, Recent Transactions).
- **Persistence**: Preferences are saved automatically in browser storage (`localStorage` key `dashboard-layout`).

## Account Drill-Down

Clicking on any account balance card on the Dashboard (such as in **Checking Balance**) opens `/accounts/[name]/transactions`, displaying a filtered transaction history and any saved account note for that account and its sub-accounts. See [Accounts](./accounts.md#account-level-transaction-history) for details.

## Global Navigation & Top Action Bar

Across the Dashboard and every page, the top bar provides quick productivity tools:

- **Navigation Layout Switcher**: Toggle between **Grouped** (default: `Cash Flow`, `Net Worth`, `Planning`, `Ledger`, `Tools`) and **Classic** (`Cash Flow`, `Expenses`, `Assets`, `Liabilities`, `Income`, `Planning`, `Ledger`, `More`) menus.
- **Command Palette (++ctrl+k++ / ++cmd+k++)**: Search and jump to any page, currency FX view, transaction search, or Quick Add action.
- **Quick Add (`+`)**: Create transactions manually or via [Natural Language Quick Add](./parser.md).
- **Sync Controls & Freshness Indicators**: Open **Sync History**, trigger **Sync Journal** (amber when external disk edits are detected), **Update Prices** (amber after 24h, red after 48h), and **Update Mutual Fund Portfolios**. See [Data Synchronization](./sync.md).
- **Theme & Privacy Mode**: Switch between light and dark themes or click the eye icon to obscure monetary figures on screen.

## Forward Projections

For FIRE target corpus, years to FIRE, and multi-scenario compounding projections, see [Financial Planning & Life Projections](./planning.md).
