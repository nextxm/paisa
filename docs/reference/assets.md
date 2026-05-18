---
description: "Managing and analyzing your assets in Paisa"
---

# Assets

The Assets section provides tools to view your balances, analyze your portfolio, and track your gains over time.

## Balance

The Balance page provides an overview of all your asset accounts and their current valuations. 

### Flat View and Exports
By default, accounts are displayed in a hierarchical tree. You can toggle the **Flat Accounts** view to see a non-rollup, flat list of your accounts. 

The current view (whether hierarchical or flat) can be exported to CSV or Excel for external analysis and record-keeping. The exported data will respect your selected display mode.

## Gain

The Gain page helps you track the performance of your investments. It calculates key metrics such as Internal Rate of Return (XIRR), total investment, and absolute return. The calculations use historical market prices to provide accurate valuations, even when dealing with multiple currencies.

## Analysis

The Analysis page unwraps commodity containers (like mutual funds) and categorizes the underlying securities by industry and rating. See the [Analysis](analysis.md) documentation for more details.

## Projection (FIRE)

The Projection page maps out your long-term net worth trajectory and calculates key Financial Independence, Retire Early (FIRE) metrics. It simulates your future net worth across three scenarios (Conservative, Expected, and Optimistic) using your current net worth, historical savings/investment behavior, and customizable compound growth assumptions.

### Key Metrics & Calculations

#### 1. Target Corpus (FIRE Number)
The **Target Corpus** is the total asset valuation required to sustain your current living expenses indefinitely, computed using the Safe Withdrawal Rate (SWR):

$$\text{Target Corpus} = \frac{\text{Annual Expenses}}{\text{SWR} / 100}$$

*   **Annual Expenses**: Derived from the cost sum of all transaction postings matching the `Expenses:*` prefix (excluding tax-related postings under `Expenses:Tax` so tax liabilities do not distort long-term sustainable living costs) over the trailing 12 months (TTM). This is annualized proportionally based on the number of active months that contain transaction data:
    $$\text{Annual Expenses} = \frac{\text{Total Expenses in TTM}}{\text{Active Months}} \times 12$$
    *   *Active Months* represents the duration of data presence, computed as the difference in months between the earliest expense posting date within the TTM window and the current month (capped at a minimum of 1 and a maximum of 12).
*   **SWR (Safe Withdrawal Rate)**: The withdrawal rate percentage (defaults to **4.0%**, representing the standard 25x expenses rule), which is fully adjustable in the UI.

#### 2. Years to FIRE
The estimated time (in years) until your projected net worth in the **Expected** scenario meets or exceeds your **Target Corpus**:

$$\text{Years to FIRE} = \frac{\text{Months to Target}}{12}$$

*   **Start Date**: Today's date.
*   **Target Crossing**: The first month in the expected projection scenario where the balance is greater than or equal to the target corpus. If the target is not crossed within the selected projection window (default: 15 years), the UI displays "Not in projection window" or "N/A".

#### 3. FIRE Progress
Your current progress toward achieving financial independence:

$$\text{FIRE Progress} = \min\left(100\%,\, \frac{\text{Current Net Worth}}{\text{Target Corpus}} \times 100\%\right)$$

---

### Net Worth Projection Engine

Paisa models three projection curves over your chosen timeframe (1 to 40 years, default 15):

*   **Conservative**: Defaults to **6.0% CAGR**
*   **Expected**: Defaults to **9.0% CAGR** (used to calculate Years to FIRE)
*   **Optimistic**: Defaults to **12.0% CAGR**

For each month $i$, the compound balance is calculated as:

$$\text{Balance}_i = \text{Balance}_{i-1} \times (1 + r) + \text{Monthly Contribution}$$

where the monthly growth rate $r$ is derived from the annual CAGR:

$$r = (1 + \text{CAGR})^{1/12} - 1$$

---

### Deriving Historical Inputs

Paisa automatically analyzes your last 12 months of journal history to pre-populate projection inputs, ensuring realistic, data-driven defaults:

1.  **Current Net Worth**: The valuation sum of all postings in `Assets:*` (excluding checking accounts), `Income:CapitalGains:*`, and `Liabilities:*` up to today, with historical market prices applied.
2.  **Net Investment**: The total cost sum of postings to asset accounts (`Assets:*`), excluding checking accounts (`Assets:Checking`) and any transactions tied to liabilities (`Liabilities:*`). Non-cash adjustment transactions such as stock splits are filtered out.
3.  **Monthly Contribution**:
    *   Paisa derives your regular monthly contribution by analyzing your historical savings rate and actual net investments.
    *   First, your **Net Income** is computed as the negative sum of all income postings (matching the `Income:*` prefix) over the past 12 months.
    *   If your **Net Income** is greater than zero, Paisa calculates your **Savings Rate** as:
        $$\text{Savings Rate} = \frac{\text{Net Investment}}{\text{Net Income}} \times 100\%$$
    *   The **Monthly Contribution** is then calculated by multiplying your average monthly income by this savings rate, which provides a smoothed, representative estimation of your monthly investments:
        $$\text{Monthly Contribution} = \text{Average Monthly Income} \times \frac{\text{Savings Rate}}{100}$$
    *   If your **Net Income** is zero or negative (representing cases with no formal income data, or net expenses exceeding income), the savings rate calculation is skipped. Instead, the monthly contribution defaults directly to the average monthly net investment:
        $$\text{Monthly Contribution} = \frac{\text{Net Investment}}{\text{Active Months}}$$
