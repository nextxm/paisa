---
description: "Track and analyze spending across Monthly Expenses, Yearly Expenses, Spending Heatmap, MoM, YoY, and Expense Breakdown in Paisa"
---

# Expenses

Paisa provides multiple dedicated views to track day-to-day spending, spot seasonality, compare month-over-month and year-over-year changes, and trace where outflows go.

In the default **Grouped** navigation layout, expense tools live under **Cash Flow → Expenses & Budget**. In the **Classic** layout, they appear under the top-level **Expenses** menu.

## Monthly Expenses

Route: `/expense/monthly`

The **Monthly Expenses** page combines monthly summary metrics, interactive visualizations, spending trends, and the transaction list for the selected month:

- **Summary KPIs**: Displays **Gross Income**, **Tax**, **Net Investment**, and **Expenses** for the selected month (along with net income, tax rate, savings rate, and expense rate percentages).
- **Daily Expense Calendar**: Shows daily spending totals and per-category donut rings across the selected calendar month. Hover over any day to inspect individual postings.
- **Category Breakdown**: Horizontal bar chart ranking level-2 expense categories (`Expenses:<Category>`) for the selected month.
- **Spending Trends**: Compares category-level spending between two periods and renders a **6-month sparkline** per category (with the latest bar highlighted and a dashed average line). Use the toggle in the top-right corner of the trends box to switch between:
  - **Last 30 Days** *(default)* — compares the rolling last 30 days against the preceding 30-day window.
  - **Calendar Month** — compares the currently selected calendar month (e.g., from the **Month Picker** or by clicking a bar in the monthly timeline) against the immediately preceding calendar month. The active current month is marked **(MTD)**, and each sparkline anchors at the selected month. Your selected comparison mode is saved in browser storage (`localStorage`).
- **Monthly Expenses Timeline**: Stacked monthly bar chart filtered by the navbar date range selector. Clicking any bar selects that month across the calendar, breakdown, expense list, and Calendar Month trends; clicking a legend item filters to a single category.
- **Responsive Mobile Layout**: On mobile viewports, the summary KPIs and analytical widgets (calendar, category breakdown, spending trends, and timeline) appear first at the top of the page, followed by the individual list of **Expenses** postings for the selected month.

`Expenses:Tax` postings are excluded from expense trend calculations so tax payments do not distort living-cost trends.

## Yearly Expenses

Route: `/expense/yearly`

The **Yearly Expenses** view summarizes spending across an entire financial year (selectable via the **Financial Year Picker** in the sub-navigation bar):

- Displays annual **Gross Income**, **Tax**, **Net Investment**, and **Expenses** KPIs.
- Provides a 12-month mini-calendar grid, annual category breakdown bar chart, and stacked multi-year timeline.

## Expense Breakdown (Sankey)

Route: `/expense/sankey`

**Expense Breakdown** renders a Sankey flow diagram restricted exclusively to links that terminate in `Expenses:*` accounts over the selected navbar date range, along with **Total Inflow** and **Total Outflow** summary figures.

To analyze full-path flows across Income, Assets, Liabilities, and Expenses (including transfers and account-depth aggregation), see [Cash Flow – Money Flow (Sankey)](./cash-flow.md#money-flow-sankey).

## Spending Heatmap & Seasonality

Route: `/expense/heatmap`

The **Heatmap** page visualizes daily and seasonal spending patterns using `GET /api/expense/daily`:

- **Filters**: Choose a calendar **Year**, filter by **Category** (or **All categories**), and switch between **Day-wise** and **Week-wise** intensity grids.
- **KPI Cards**: Shows **Total spend**, **Active spend days** (with average spend per calendar day), **Peak day**, and **Strongest pattern** (peak month and weekday average).
- **Seasonality Charts**:
  - **Average spend by weekday** — highlights day-of-week habits (e.g., weekend spikes).
  - **Seasonality by month** — compares average monthly totals across January–December.

## Month-over-Month (MoM)

Route: `/expense/mom`

The **Month-over-Month (MoM)** diagnostic view analyzes short- and medium-term spending trajectory and drivers:

- **Controls**: Select an **End month**, rolling **Window** (**6**, **12**, or **24 months**), **Breakdown** dimension (**Category**, **Payee**, or **Account**), **Rows** (**Top 5**, **Top 8**, or **Top 12**), and **Currency** mode (**Default** converted currency vs. **Actual** transaction currency with an optional single-currency filter when multiple currencies appear).
- **Charts & Insights**:
  - **Expense Trajectory**: Monthly spending curve overlaid with a 3-month rolling average.
  - **Summary Cards**: **Latest Month** vs. previous month, **Range Highlights** (High, Low, Window Average), and **Volatility & Trend** (Coefficient of Variation = StdDev / Mean, flagged as Low `<15%`, Moderate `15–30%`, or High `>30%`).
  - **Top Movers Variance** and **Distribution Composition** charts for the selected dimension.
  - **MoM Delta** diverging bar chart plus **Monthly Timeline** and **Breakdown** tables with per-row sparklines.

## Year-over-Year (YoY)

Route: `/expense/yoy`

The **Year-over-Year (YoY)** view compares calendar years side-by-side (legacy `/analysis/yoy` redirects here):

- **Controls**: Compare **2 to 5 years** ending at a chosen **Until year**, and export the comparison table via **Download CSV**.
- **KPI Strip**: Summarizes **Spending**, **Income**, **Net savings**, **Savings rate** (with top expense category), **Biggest mover**, and **Efficiency snapshot**.
- **Visualizations**: **Spending YoY**, **Income YoY**, and **Category YoY** (with line/bar toggle), alongside **Category Movers** and **Monthly Net Profile** tables.

## Related

- Track planned vs. actual monthly allocations in [Budget](./budget.md).
- Inspect full cash movements in [Cash Flow](./cash-flow.md).
