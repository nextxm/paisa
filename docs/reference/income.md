---
description: "Track salary, passive income, dividends, interest, distributions, and trailing yields in Paisa"
---

# Income

The **Income** section tracks your earnings over time and separates passive investment payouts (dividends, interest, and distributions) from earned income.

In the default **Grouped** navigation layout, these views are under **Cash Flow → Income & Returns**. In the **Classic** layout, they appear under the top-level **Income** menu.

## Income Timeline

Route: `/income`

The **Income Timeline** page shows:

- **Summary KPIs**: **Gross Income**, **Net Tax** (`Expenses:Tax:*`), and **TTM Investment Income** (trailing 12 months).
- **Monthly Income Timeline**: Stacked monthly bar chart across all `Income:*` accounts.
- **Financial Year Income Timeline**: Side-by-side yearly totals for total income, net income (after taxes), and net taxes.

## Investment Income

Route: `/income/investment`

The **Investment Income** page (backed by `GET /api/income/investment`) analyzes passive returns by holding and income category:

- **Financial Year Picker**: View trailing 12-month (TTM) metrics as of the current date or the end of a historical financial year.
- **KPI Strip**: Displays **TTM Investment Income**, **Dividend (TTM)**, **Interest (TTM)**, and **Distribution (TTM)**.
- **Per-Holding Breakdown Table**: Lists each holding and payout type along with **Total Income**, **TTM Income**, **Current Balance**, **TTM Yield** (`TTM Income / Current Balance`), and a **Dividend Growth** sparkline.
- **Monthly Timeline**: Visualizes investment income payouts over time.

A compact **Investment Income** widget (showing **TTM Dividend** and **TTM Interest**) is also available on the [Dashboard](./dashboard.md).

## Account Naming Conventions

Paisa classifies investment income automatically from standard account prefixes (see [Accounts](./accounts.md)):

- `Income:Dividend:*` — stock and fund dividends
- `Income:Interest:*` — savings, fixed deposit, EPF, or bond interest
- `Income:Distribution:*` or `Income:Distributions:*` — REIT/fund distributions

Investment income also feeds into **Assets → Gain** (`/assets/gain`) to compute **Income Received**, **Total Return** (`Price Appreciation + Income Received`), and per-account **TTM Yield**.
