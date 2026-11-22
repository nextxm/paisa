---
description: "Managing and analyzing your net worth, balances, investments, gains, and projections in Paisa"
---

# Assets & Net Worth

The **Net Worth** section (listed as **Assets** in Classic navigation) provides tools to inspect asset balances, track net worth history and FX impact, analyze investment performance, unwrap mutual fund holdings, and target/rebalance your portfolio.

## Asset Balances

Route: `/assets/balance` (Grouped: `Net Worth → Net Worth → Asset Balances`; Classic: `Assets → Balance`)

The **Asset Balances** page provides a complete breakdown of your asset accounts, units, average cost, market prices, and current valuations:

- **View as of**: Pick any historical date (`as_of_date`) to inspect balances as of the end of that day.
- **Report Currency**: Switch between your default currency and other configured currencies when multiple currencies exist.
- **Flat Accounts**: Toggle between the hierarchical rollup tree and a flat list of accounts.
- **Filter Inactive & Filter 0**: Hide accounts marked inactive (`inactive_accounts` / `accounts[].inactive`) or zero-balance accounts.
- **Exports**: Export the current view (respecting hierarchy vs. flat mode) to **CSV** or **Excel**.

## Net Worth History

Route: `/assets/networth` (Grouped: `Net Worth → Net Worth → Net Worth History`; Classic: `Assets → Networth`)

Tracks your portfolio valuation over time with multi-currency attribution and optional forward projection:

- **Summary Metrics**: Displays **Net worth**, **Net Investment**, **Gain / Loss**, **Contribution**, **Investment Return**, **FX Impact**, and overall **XIRR**.
- **FX Impact Decomposition**: Separates cumulative foreign-exchange movements from local investment returns for holdings denominated in non-default currencies, with a **Show FX impact overlay** toggle on the chart.
- **Report Currency**: Re-value the timeline in another configured currency.
- **Project Forward**: Toggle **Project Forward** to overlay the conservative/expected/optimistic projection curves and milestone markers directly onto your historical net worth timeline.

## Holdings & Valuation

Route: `/assets/investment` (Grouped: `Net Worth → Investments & Portfolio → Holdings & Valuation`; Classic: `Assets → Investment`)

Visualizes monthly and financial-year investment flows and holdings valuations over time, including a **Report Currency** switcher when multiple currencies are configured.

## Performance & Gain (XIRR)

Route: `/assets/gain` (Grouped: `Net Worth → Investments & Portfolio → Performance & Gain (XIRR)`; Classic: `Assets → Gain`)

Calculates **Internal Rate of Return (XIRR)**, total investment, unrealized/realized price appreciation, and total return using historical market prices across currencies.

The gain breakdown table also integrates passive payouts from [Investment Income](./income.md#investment-income) (`Income:Dividend:*`, `Income:Interest:*`, `Income:Distribution:*`), showing per-account:

- **Price Appreciation**
- **Income Received**
- **Total Return** (`Price Appreciation + Income Received`)
- **TTM Yield** (trailing 12-month yield)

Click any account row to drill into `/assets/gain/[account]` for its detailed timeline and breakdown.

## Target Asset Allocation & Rebalancing

Route: `/assets/allocation` (Grouped: `Net Worth → Investments & Portfolio → Target Asset Allocation`; Classic: `Assets → Allocation`)

Configure target weights, drift bands, and calculate cash deployment or full buy/sell rebalance plans, plus inspect currency exposure. See [Target Asset Allocation & Rebalancing](./allocation-targets.md).

## Underlying Portfolio

Route: `/assets/portfolio` (Grouped: `Net Worth → Investments & Portfolio → Underlying Portfolio`; Classic: `Assets → Portfolio`) *(formerly `Analysis`; `/assets/analysis` redirects here)*

Unwraps commodity containers (such as mutual funds) and categorizes underlying securities by industry and rating. See [Underlying Portfolio](./analysis.md) for details.

## Projection (FIRE)

Route: `/planning/projection` (Grouped: `Planning → Projections & Life Plan → Net Worth Projections`; Classic: `Planning → Projection`)

The **Net Worth Projections** page maps out your long-term net worth trajectory and calculates key Financial Independence, Retire Early (FIRE) metrics. It simulates future net worth across three scenarios (**Conservative**, **Expected**, and **Optimistic**) using your current net worth, historical savings/investment behavior, and customizable compound growth assumptions. See [Financial Planning & Life Projections](./planning.md) for the broader planning suite (including Monte Carlo simulations, Milestone Life Goals, What-If scenarios, and Tax-Aware Drawdown).

### Key Metrics & Calculations

#### 1. Target Corpus (FIRE Number)
The **Target Corpus** is the total asset valuation required to sustain your current living expenses indefinitely, computed using the Safe Withdrawal Rate (SWR):

$$\text{Target Corpus} = \frac{\text{Annual Expenses}}{\text{SWR} / 100}$$

* **Annual Expenses**: Derived from the cost sum of all transaction postings matching the `Expenses:*` prefix (excluding `Expenses:Tax` so tax liabilities do not distort long-term sustainable living costs) over the trailing 12 months (TTM), annualized proportionally across active months:
  $$\text{Annual Expenses} = \frac{\text{Total Expenses in TTM}}{\text{Active Months}} \times 12$$
  * *Active Months* is the number of months between the earliest expense posting within the TTM window and the current month (clamped to `[1, 12]`).
* **SWR (Safe Withdrawal Rate)**: The withdrawal rate percentage (defaults to **4.0%**, representing the 25x expenses rule), adjustable in the UI via slider or numeric input.

#### 2. Years to FIRE
The estimated time (in years) until your projected net worth in the **Expected** scenario meets or exceeds your **Target Corpus**:

$$\text{Years to FIRE} = \frac{\text{Months to Target}}{12}$$

If the target is not crossed within the selected projection window (default: 15 years), the UI displays `"Not in projection window"`.

#### 3. FIRE Progress
Your current progress toward achieving financial independence:

$$\text{FIRE Progress} = \min\left(100\%,\, \frac{\text{Current Net Worth}}{\text{Target Corpus}} \times 100\%\right)$$

### Net Worth Projection Engine & Interactive Controls

Paisa models three deterministic compounding curves (defaults: **6.0% Conservative**, **9.0% Expected**, **12.0% Optimistic** CAGR) over your chosen timeframe (`1–40` via slider, or arbitrary precision values via the numeric input field next to every control):

$$\text{Balance}_i = \text{Balance}_{i-1} \times (1 + r) + \text{Monthly Contribution}, \quad r = (1 + \text{CAGR})^{1/12} - 1$$

The **Your Financial DNA** panel automatically extracts your historical **Annual Income**, **Annual Expenses**, **Savings Rate**, and **Historical Return / Volatility**, and lets you click **Apply to Sliders** to calibrate the projection controls from your ledger history.

### Year-by-Year Projection Breakdown

Below the chart, Paisa renders expandable year-by-year scenario tables for Conservative, Expected, and Optimistic trajectories, each with an **Export CSV** button:

| Year | Starting Corpus | Annual Contribution | Return | Ending Corpus | Inflated Expenses |
| --- | ---: | ---: | ---: | ---: | ---: |
| 1 | $C_0$ | $12 \times \text{PMT}$ | $\sum \text{monthly return}$ | $C_1$ | $E_0$ |
| 2 | $C_1$ | $12 \times \text{PMT}$ | $\sum \text{monthly return}$ | $C_2$ | $E_0 \times (1 + \text{inf})^{1}$ |
| 3 | $C_2$ | $12 \times \text{PMT}$ | $\sum \text{monthly return}$ | $C_3$ | $E_0 \times (1 + \text{inf})^{2}$ |

The projection curve compounds your starting corpus with ongoing monthly contributions without subtracting annual expenses; **Inflated Expenses** shows your inflation-adjusted annual cost of living and FIRE benchmark alongside each year.

### Deriving Historical Inputs

Paisa analyzes your last 12 months of journal history to pre-populate projection defaults (`GET /api/networth/projection`):

1. **Current Net Worth**: Valuation sum of postings in `Assets:*` (excluding checking accounts), `Income:CapitalGains:*`, and `Liabilities:*` up to today with historical market prices applied.
2. **Net Investment**: Cost sum of postings to `Assets:*`, excluding checking accounts and liability-linked transactions (non-cash adjustments such as stock splits are filtered out).
3. **Monthly Contribution**: Computed from your **Net Income** (`Income:*`) and **Savings Rate** ($\text{Net Investment} / \text{Net Income}$), falling back to $\text{Net Investment} / \text{Active Months}$ if net income is zero or negative.
