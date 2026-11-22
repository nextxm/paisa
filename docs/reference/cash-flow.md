---
description: "Understanding Monthly Cash Flow, Yearly Cash Flow, Income Statement, and Money Flow (Sankey) in Paisa"
---

# Cash Flow

The **Cash Flow** section provides a comprehensive view of your inflows, outflows, taxes, and net savings over time, helping you monitor your savings rate and trace how money moves across your accounts.

In the default **Grouped** navigation layout, these tools live under **Cash Flow → Cash Flow** and **Cash Flow → Flows & Recurring** (and in **Classic** navigation under the top-level **Cash Flow** menu).

## Monthly Cash Flow

Route: `/cash_flow/monthly`

The **Monthly Cash Flow** view combines a multi-year interactive timeline with a currency-accurate monthly ledger breakdown:

- **Monthly Cash Flow Chart**: Compares monthly Income, Expenses, Investment/Net Cashflow, and Checking Balance across the navbar date range.
- **Monthly Breakdown**: Use the **Month Picker** to inspect any month. Summary cards display **Income**, **Expenses**, **Taxes**, and **Net Flow** using original ledger quantities broken down by native currency (avoiding mark-to-market distortion in multi-currency setups).
- **Inflows & Outflows Tables**: Lists per-account Income, Liabilities Inflow, Expenses, and Taxes alongside a **Monthly Net Flow** summary footer.

## Yearly Cash Flow

Route: `/cash_flow/yearly`

The **Yearly Cash Flow** page renders a Sankey flow diagram for the selected financial year using the **Financial Year Picker**:

- **Flow Settings** (slider icon in the sub-navigation bar): Adjust **Expenses** and **Income** account depth aggregation independently, and toggle **Show transfers** to include or exclude internal asset transfers (`Assets → Assets`).

## Income Statement

Route: `/cash_flow/income_statement`

The **Income Statement** provides a financial-year breakdown of revenues, expenses, taxes, and net change. It accounts for liabilities and computes starting and ending net worth directly from the balance sheet (assets at market value plus liabilities at book value), mirroring the [Net Worth](./assets.md#net-worth-history) calculation so liability-funded transactions and foreign-currency mark-to-market movements reconcile cleanly.

## Money Flow (Sankey)

Route: `/cash_flow/sankey`

**Money Flow** visualizes end-to-end money paths across Income, Assets, Liabilities, and Expenses for a selected period:

- **Period Selector**: Switch between **Month**, **Quarter**, or **Year** windows and step forward or backward by period.
- **Account Depth & Transfer Filtering**: Collapse sub-accounts to a chosen depth or toggle **Hide asset transfers** to focus strictly on income-to-expense and savings flows.
- **FX Warnings**: Alerts you if any multi-currency flows cannot be converted due to missing exchange rates.

For an outflow-only Sankey focused strictly on `Expenses:*` accounts over the navbar date range, see [Expenses – Expense Breakdown](./expenses.md#expense-breakdown-sankey).

## Related Views

- [Expenses](./expenses.md) — Monthly Expenses (including **Last 30 Days** vs. **Calendar Month** spending trends), Yearly Expenses, Heatmap, MoM, and YoY diagnostics.
- [Income](./income.md) — Income Timeline and TTM Investment Income (dividends, interest, distributions).
- [Recurring](./recurring.md) — Upcoming recurring bills and scheduled transactions.
- [Dashboard](./dashboard.md) — 3-month Cash Flow overview widget.
