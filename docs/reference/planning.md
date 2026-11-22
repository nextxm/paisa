---
description: "Comprehensive financial planning in Paisa: Financial DNA profiling, Monte Carlo life projections, milestone life goals, what-if scenario testing, and tax-aware drawdown strategies"
---

# Financial Planning & Life Projections

Paisa features a **Planning** suite that turns your historical double-entry accounting records into deterministic projections, stochastic Monte Carlo life simulations, life-goal tracking, scenario stress-testing, and tax-aware drawdown strategies.

In the default **Grouped** navigation layout, Planning contains **Projections & Life Plan**, **Financial Goals**, and (when `default_currency` is `INR`) **Tax**.

The planning modules include:

1. **Life Plan & Financial DNA (`/planning/life`)** — Personal financial profiling, goal markers, and Monte Carlo net worth simulation.
2. **Net Worth Projections (`/planning/projection`)** — Multi-scenario deterministic curves (Conservative, Expected, Optimistic) with precision inputs, Financial DNA calibration, year-by-year breakdown tables, and CSV export.
3. **What-If Scenarios (`/planning/life/whatif`)** — Comparative Monte Carlo stress testing against your baseline plan.
4. **Tax-Aware Drawdown Strategy (`/planning/life/drawdown`)** — Priority-bucketed withdrawal ordering and capital-gains tax optimization over FIFO lots.
5. **Milestone Life Goals (`/planning/life/goals`)** — One-off `milestone` and `recurring` goals integrated directly into life simulations.
6. **Retirement & Savings Goals (`/planning/goals`)** — Config-backed target trackers (see [Goals](./goals/index.md)).

---

## 1. Financial DNA Extraction & Monte Carlo Simulation

Navigating to **Planning → Life Plan & DNA** (`/planning/life`; Classic: **Planning → Life Plan**) extracts your **Financial DNA** (`GET /api/projection/dna`) from historical ledger data:

- **Annual Income & Income Growth Rate**: Derived from historical `Income:*` postings.
- **Annual Expenses & Expense Growth Rate**: Trailing living costs (excluding `Expenses:Tax`).
- **Savings Rate & Monthly Contribution**: Derived from actual historical net investments vs. net income.
- **Historical Return & Volatility ($\sigma$)**: Calculated across your portfolio price history.

### Interactive Simulation Parameters

Adjust slider controls to run stochastic simulations (`POST /api/projection/simulate`):

- **Projection Years**: `5` to `50` years (default `30`).
- **Expected Return (%)**: `0` to `30%`.
- **Volatility (Risk %)**: `0` to `40%`.
- **Monthly Contribution**: Ongoing monthly savings additions.
- **Contribution Growth (%)**: `0` to `20%` annual increment rate.
- **Inflation Rate (%)**: `0` to `15%`.
- **Safe Withdrawal Rate (SWR %)**: `2` to `8%`.
- **Iterations**: `100` to `10,000` stochastic Monte Carlo paths (step `100`, default `1,000`).

Simulation results are cached in memory by normalized configuration and automatically invalidated whenever a journal or price sync completes.

### Monte Carlo Fan Chart, Goals & FIRE Timeline

The chart plots percentile bands (**P10**, **P25**, **P50 (Median)**, **P75**, and **P90**) alongside configured **Milestone Life Goals** as timeline markers and probability cards showing each goal's funding likelihood.

---

## 2. Deterministic Net Worth Projections

Under **Planning → Projections & Life Plan → Net Worth Projections** (`/planning/projection`), Paisa computes **Conservative** (6% default), **Expected** (9% default), and **Optimistic** (12% default) trajectories alongside **Current Net Worth**, **Monthly Contribution**, **Annual Expenses**, **Target Corpus**, **Years to FIRE**, and **FIRE Progress**:

- Every parameter includes both a slider and a precision numeric input box for exact values or horizons beyond the slider range.
- **Your Financial DNA** displays extracted income/expense growth, savings rate, and historical return/volatility, with an **Apply to Sliders** button to calibrate the projection from your ledger history.
- **Year-by-Year Projection Tables** list **Year**, **Starting Corpus**, **Annual Contribution**, **Return**, **Ending Corpus**, and **Inflated Expenses** for all three scenarios, with an **Export CSV** button for each scenario.

See [Assets – Projection (FIRE)](./assets.md#projection-fire) for the exact formulas.

---

## 3. Milestone & Recurring Life Goals

Under **Planning → Financial Goals → Milestone Life Goals** (`/planning/life/goals`), you can create and prioritize life goals stored under `goals.life[]` in `paisa.yaml`:

- **Goal Types** (two supported types):
  - **`milestone`**: One-time lump-sum outflow requiring **Target Amount** and **Target Month** (`YYYY-MM`).
  - **`recurring`**: Repeated outflow requiring **Start Month** (`YYYY-MM`), **End Month** (`YYYY-MM`), **Frequency** (`monthly`, `quarterly`, or `yearly`), and **Allocation Per Occurrence** (`monthly_allocation`).
- **Additional Parameters**: **Name**, **Icon**, **Inflation Override (%)** (optional per-goal override), **Priority** (assigned from list order; higher-priority goals are funded first), and **Funded By** accounts.

```yaml
goals:
  life:
    - name: House Down Payment
      icon: mdi:home
      type: milestone
      target_amount: 2000000
      target_date: "2029-06"
      priority: 2
      funded_by:
        - Assets:Debt:*
    - name: College Tuition
      icon: mdi:school
      type: recurring
      start_date: "2032-06"
      end_date: "2036-05"
      frequency: yearly
      inflation_rate: 8
      priority: 1
      monthly_allocation: 400000
      funded_by:
        - Assets:Equity:*
```

---

## 4. What-If Scenario Stress Testing

The **What-If Scenarios** workspace (`/planning/life/whatif`, backed by `POST /api/projection/whatif`) runs your baseline plan side-by-side against up to **five** alternate scenarios using the same goal-aware Monte Carlo engine.

- **Prebuilt Templates**: **Increase SIP by 10k**, **Job loss for 6 months**, and **Early retirement**, plus **Add Custom Scenario**.
- **Summary Cards & Overlay Chart**: Displays **Baseline FIRE (%)**, **Target Corpus**, **Median FIRE Year** (`fire_year_p50`), and **Active Goals**, alongside interactive scenario toggles overlaid on the Baseline P25–P75 fan band and a comparison table.

---

## 5. Tax-Aware Drawdown Strategy

The **Tax-Aware Drawdown** module (`/planning/life/drawdown`, backed by `POST /api/projection/drawdown`) ranks which asset lots to liquidate first to meet a target withdrawal amount while minimizing estimated capital gains tax.

Only asset accounts holding commodities configured with a supported `tax_category` (`equity`, `equity65`, `equity35`, `debt`, `unlisted_equity`) appear in the eligible assets pool.

### Priority Buckets, Drag-and-Drop & Auto-Grouping

- **Strategy Buckets**: Order priority buckets (`Priority 1`, `Priority 2`, …). Each bucket supports:
  - **Explicit Assigned Assets**: Drag asset chips from **Unassigned Assets** directly into buckets (or click **Clear** to revert to rule matching).
  - **Account Glob**: Pattern filter such as `Assets:Equity:*` (used when explicit accounts are not set on the bucket).
  - **Match Tax Category**: Filter lots matching a specific commodity tax category.
  - **Apply Tax As** (`override_tax_category`): Override how matched lots in this bucket are taxed during drawdown analysis without changing global `paisa.yaml` commodity definitions.
  - **Min Holding Period (Months)**: Require lots to meet a minimum age before becoming eligible for the bucket.
- **One-Click Auto-Group**: Seed buckets automatically with live bucket-count previews via **Auto-group: Account Family**, **Auto-group: Tax Category**, or **Auto-group: Current Rules**.

### Lot Selection & Projection Impact

For a target withdrawal amount:

1. Paisa inspects open FIFO lots across eligible asset accounts and evaluates buckets in priority order.
2. Within each bucket, eligible lots are sorted by lowest effective tax rate, then lowest total tax amount, then longest holding duration.
3. Withdrawals cascade from higher-priority buckets into subsequent buckets until the requested amount is covered (reporting any uncovered shortfall if insufficient eligible lots exist).
4. The UI shows covered vs. uncovered amounts, total estimated tax, post-tax net proceeds, optional Monte Carlo **Projection Impact**, and per-lot sale recommendations (units, purchase date, holding days, tax breakdown, and effective rate).

## Related Guides

- [Goals (Retirement & Savings)](./goals/index.md)
- [Tax Harvesting, Capital Gains & Schedule AL](./tax/index.md) *(INR journals)*
