---
description: "Comprehensive financial planning in Paisa: Financial DNA profiling, Monte Carlo projection simulation, milestone life goals, what-if scenario testing, and tax-aware drawdown strategies"
---

# Financial Planning & Life Projections

Paisa features an advanced **Planning** suite that transforms your historical double-entry accounting records into forward-looking projections, life milestone simulations, scenario stress-testing, and retirement drawdown strategies.

The Planning suite is accessible via the top navigation menu under **Planning** and comprises five integrated modules:

1. **Life Plan & Financial DNA (`/planning/life`)** — Personal financial profiling and Monte Carlo net worth simulation.
2. **Net Worth Projections (`/planning/projection`)** — Multi-scenario trajectory curves (Conservative, Expected, Optimistic).
3. **Milestone Life Goals (`/planning/life/goals`)** — Targeted life events and education/asset milestones.
4. **What-If Scenarios (`/planning/life/whatif`)** — Comparative stress testing against unexpected life events or budget shifts.
5. **Tax-Aware Drawdown Strategy (`/planning/life/drawdown`)** — Phase-based withdrawal planning and tax liability optimization.

---

## 1. Financial DNA Extraction & Monte Carlo Simulation

Navigating to **Planning → Life Plan** automatically extracts your **Financial DNA** from historical ledger data:
* **Baseline Savings & Investments**: Derived from actual historical monthly cash flows.
* **Effective Historical Return & Volatility**: Calculated across your commodity and asset portfolios.
* **Spending Baseline**: Trailing twelve-month average living costs, excluding taxes.

### Interactive Simulation Parameters

Adjust interactive slider controls to simulate net worth trajectories under uncertainty:
* **Expected Annual Return (%)**: Projected nominal portfolio CAGR.
* **Return Volatility / $\sigma$ (%)**: Annualized return standard deviation.
* **Monthly Contribution**: Ongoing monthly savings additions.
* **Contribution Growth Rate (%)**: Annual percentage increase in savings (e.g. annual salary increments).
* **Inflation Rate (%)**: Projected cost-of-living inflation rate.
* **Safe Withdrawal Rate (SWR %)**: Target withdrawal percentage for retirement sustainability.
* **Projection Horizon**: 1 to 40 years.
* **Simulation Iterations**: Runs up to 1,000+ stochastic Monte Carlo paths.

### Monte Carlo Fan Chart & Percentile Bands
The engine renders an interactive fan chart charting probability bands:
* **P90 (90th percentile)**: Strong bull market trajectory.
* **P75 / P50 (Median)**: Most probable baseline corridor.
* **P25 / P10**: Bear market / stress-tested trajectory to verify retirement safety.

---

## 2. Milestone Life Goals

Under **Planning → Life Plan → Goals** (`/planning/life/goals`), you can model specific financial milestones across your timeline:

* **Goal Types**:
  * `milestone`: One-off lump-sum target (e.g. buying a house, down payment, vehicle purchase).
  * `education`: Recurring milestone with target duration and inflation adjustment.
  * `legacy` / `custom`: Long-term preservation targets.
* **Configuration Parameters**:
  * **Target Amount**: Nominal requirement in base currency.
  * **Target Date / Timeframe**: Expected milestone maturity date.
  * **Specific Inflation Rate**: Optional override for education or healthcare costs that inflate faster than general CPI.
  * **Funded By Accounts**: Specific asset accounts earmarked to fund the milestone.
  * **Monthly Allocation**: Budgeted monthly contribution toward this specific goal.

---

## 3. What-If Scenario Stress Testing

The **What-If Scenario** engine (`/planning/life/whatif`) allows you to test how life events impact your long-term solvency by running side-by-side simulations against your baseline plan:

### Pre-configured & Custom Templates
* **Increase Monthly SIP**: Models the long-term compounding impact of increasing monthly contributions by a fixed increment (e.g., +₹10,000/month).
* **Job Loss / Income Shock**: Models temporary pauses in contributions (e.g. 6 months without income) coupled with moderate market downturns.
* **Early Retirement**: Tests retiring several years early with lower contribution horizons, a stricter safe withdrawal rate (e.g., 3.5%), and higher inflation assumptions.
* **Custom Scenarios**: Define arbitrary adjustments to expected returns, inflation, volatility, and monthly cash injections.

### Comparison Visualizations
The UI presents comparative metric cards and side-by-side fan charts displaying:
* Delta in ending net worth across scenarios.
* Probability of meeting target corpus across scenarios.
* Earliest projected financial independence age.

---

## 4. Tax-Aware Drawdown Strategy

The **Tax-Aware Drawdown** module (`/planning/life/drawdown`) guides your asset de-accumulation strategy during retirement to minimize capital gains taxes and prevent early portfolio exhaustion.

### Priority Bucketing & Drag-and-Drop Ordering
Configure an ordered hierarchy of drawdown buckets (e.g., *Cash & Liquid First* $\to$ *Equity Holdings* $\to$ *Debt Funds* $\to$ *Fallback Assets*):
* **Explicit Account Assignment**: Drag and drop accounts between buckets to specify exact liquidation order.
* **Pattern / Glob Matching**: Match asset classes using patterns (e.g., `Assets:Equity:*`).
* **Tax Category Overrides**: Set tax treatments per bucket (e.g., `equity65`, `equity35`, `debt`, `unlisted_equity`).
* **Holding Period Constraints**: Specify minimum holding periods in months to ensure lots qualify for Long-Term Capital Gains (LTCG) instead of Short-Term Capital Gains (STCG).

### Water-filling Liquidation Engine
When you simulate withdrawing a lump sum (e.g. ₹5,00,000 annual living expenses):
1. The engine checks available balances in the highest-priority bucket.
2. It prioritizes units with zero exit load and favorable capital gains brackets.
3. Once a bucket is exhausted, liquidation cascades smoothly into subsequent buckets.
4. The strategy output itemizes estimated gross withdrawal, net realized amount, estimated tax liability, and post-drawdown asset distribution.
