# Month-on-Month (MoM) Analysis Page – Design Plan

## Strategic Objectives

1. **Primary Goal**: Show expense **trajectory and volatility** over selected months
2. **Secondary Goal**: Identify **dimension-level trends** (which categories/payees/accounts drive changes)
3. **Tertiary Goal**: Enable **multi-currency drill-down** and quick insights
4. **Design Philosophy**: **Charts first, tables supporting** — opposite of YoY's basic approach

---

## Key Visualizations Needed

### 1. **Expense Trajectory Chart** (Hero Chart - Primary Focus)
- **Type**: Line/Area chart with multiple traces
- **X-Axis**: Month (in chronological order)
- **Y-Axis**: Cumulative expense amount
- **Traces**:
  - Current month values
  - 3-month moving average (smoother trend line)
  - Previous month comparison (optional overlay)
- **Interaction**: 
  - Hover to see exact values + MoM % change
  - Click to filter breakdown by that month
- **Purpose**: Visual understanding of expense volatility and trend direction at a glance

### 2. **Variance Waterfall Chart** (For Latest Period)
- **Type**: Waterfall showing "Previous Month → Current Month" bridge
- **Segments**: Top 5-7 dimension movers (categories/payees), with up/down bars
- **Purpose**: Quickly see what categories drove the change month-over-month
- **Alternative**: Grouped bar chart (Previous Month | Current Month) for top dimensions

### 3. **Dimension Composition Over Time** (Category/Payee/Account Breakdown)
- **Type**: Stacked area chart OR small multiples (grid of mini line charts per dimension)
- **X-Axis**: Months
- **Y-Axis**: Percentage or absolute amount of expense
- **Interaction**: 
  - Click dimension in legend to highlight/dim
  - Hover to see exact contribution
- **Purpose**: See how composition shifts (e.g., "Food became 35% of budget vs 28% last month")

### 4. **Volatility/Momentum Gauge** (Supporting KPI)
- **Type**: Single-value card or mini gauge
- **Metric**: Coefficient of variation (StdDev / Mean) over selected months
- **Color**: Red (high volatility) → Yellow → Green (low, stable)
- **Purpose**: Quick sense of expense predictability

### 5. **Top Movers Detail** (Supporting List)
- **Type**: Dense table, 2-3 rows max visible
- **Columns**: Dimension, Prev Month, This Month, Change, % Change, Trend Arrow
- **Sorted**: By absolute change or % change
- **Purpose**: For users who need exact numbers

---

## Layout Structure

```
┌─────────────────────────────────────────────────────────────────────────┐
│  HEADER: Controls (Month Picker | Window | Dimension | Currency)       │
│  (Compact, single row, wrapped if needed)                              │
└─────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│ SECTION 1: HERO CHART                                                      │
│ ┌──────────────────────────────────────────────────────────────────────┐  │
│ │  Expense Trajectory (Line/Area Chart)                               │  │
│ │  - Monthly values with 3-month MA overlay                           │  │
│ │  - Hover: exact values + MoM %                                      │  │
│ │  [CHART HEIGHT: 300-400px, FULL WIDTH]                             │  │
│ └──────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│ SECTION 2: QUICK INSIGHTS (3-Column Summary)                              │
│ ┌──────────────────┬──────────────────┬──────────────────────────────┐    │
│ │ Latest Month     │ MoM Change       │ Volatility / Momentum        │    │
│ │ $5,200           │ ↑ +12.5% (+$600) │ High ⚠️ (CV: 0.42)           │    │
│ │ vs 3M Avg: 4.9K  │ vs 3M Avg: 4.8K  │ Trending: Up ↗️              │    │
│ └──────────────────┴──────────────────┴──────────────────────────────┘    │
└────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────┬────────────────────────────────────────────┐
│ SECTION 3A: Dimension Movers │ SECTION 3B: Composition Chart            │
│ (Variance Waterfall or List) │ (Stacked Area or Mini-Multiples)         │
│ ┌─────────────────────────┐ │ ┌─────────────────────────────────────┐  │
│ │  Top Movers             │ │ │ Category Distribution Over Time      │  │
│ │  Food      ↑ +$340      │ │ │ [STACKED AREA CHART, 250px height]  │  │
│ │  Transport ↑ +$120      │ │ │ (Food, Transport, Utilities, etc.)  │  │
│ │  Utilities ↓ -$85       │ │ │                                     │  │
│ │  Housing   ↔ +$10       │ │ │                                     │  │
│ │  Other     ↓ -$215      │ │ │                                     │  │
│ └─────────────────────────┘ │ │                                     │  │
│ (Dense, 5-7 items)          │ └─────────────────────────────────────┘  │
│                             │  (Color per dimension, legend toggle)     │
└─────────────────────────────┴────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────────────────────────┐
│ SECTION 4: DETAIL TABLES (Collapsible or Below-Fold)                      │
│ ┌──────────────────────────────────────────────────────────────────────┐  │
│ │  Monthly Timeline (Dense Table)                                      │  │
│ │  Month    | Total  | MoM Chg | 3M Avg | Trend                       │  │
│ │  Jan 2026 | 4,800  | ↑ 8%    | 4,900  | ↗ ↗ ↗                      │  │
│ │  Dec 2025 | 4,450  | ↓ -5%   | 4,700  | ↘ ↘ ↗                      │  │
│ │  ...                                                                  │  │
│ └──────────────────────────────────────────────────────────────────────┘  │
│                                                                             │
│ ┌──────────────────────────────────────────────────────────────────────┐  │
│ │  Breakdown by Category/Payee/Account (Dynamic based on selection)    │  │
│ │  Name      | Latest | Avg  | Prev | MoM  | Share | Trend           │  │
│ │  Food      | 1,200  | 1,100| 1,100| ↑ 9% | 23%   | ↗ ↗ ↘          │  │
│ │  Transport | 800    | 780  | 750  | ↑ 7% | 15%   | → → ↗          │  │
│ │  ...                                                                  │  │
│ └──────────────────────────────────────────────────────────────────────┘  │
└────────────────────────────────────────────────────────────────────────────┘
```

---

## Implementation Roadmap

### Phase 1: Fix Layout & Controls (Quick Wins)
- [x] Debug currency dropdown visibility (check availableCurrencies)
- [ ] Compact control panel (single row, smaller fonts)
- [ ] Reduce summary card padding (use Bulma spacing utilities: `p-2` or `p-3`)
- [ ] Make summary cards 3 columns instead of 2 (`column is-4` instead of `is-half`)

### Phase 2: Add Hero Chart (Primary Value)
- [ ] Create `ExpenseTrajectoryChart.svelte` (reuse D3/Recharts pattern from YoY if available)
  - Line chart with area fill
  - Dual traces: actual values + 3-month moving average
  - Hover tooltips with exact amounts + MoM %
  - ~300-400px height, full width
- [ ] Integrate into +page.svelte above tables

### Phase 3: Add Variance/Movers Visualization
- [ ] Create `VarianceWaterfall.svelte` OR `TopMoversChart.svelte`
  - Option A: Waterfall chart (Prev Month → Current Month with segments)
  - Option B: Grouped bar chart (Previous | Current for top 5-7 categories)
  - With labels showing $ amount and % change
- [ ] Display in 2-column grid with Composition chart

### Phase 4: Add Composition Chart (Dimension Breakdown)
- [ ] Create `DimensionComposition.svelte` (Stacked Area Chart)
  - Months on X-axis, expense % or $ on Y-axis
  - One colored segment per top dimension
  - Legend with toggle to highlight/dim specific dimensions
  - Hover: exact values per dimension per month
- [ ] Display in 2-column grid with Variance chart

### Phase 5: Polish & Interaction
- [ ] Make charts clickable/filterable (click a month to highlight, click a dimension to filter)
- [ ] Add tooltips explaining metrics (volatility, 3M avg, etc.)
- [ ] Responsive breakpoints (stack 2-column charts vertically on mobile)
- [ ] Optional: Export/print friendly mode

---

## Chart Library Recommendation

### Current Paisa Setup
- **Recharts** (if used in YoY) — simple, React-like API, but not Svelte-native
- **D3.js** (if custom in other pages) — powerful, steeper learning curve
- **Chart.js** — simple, good for basic charts
- **Lightweight alternative**: Use SVG + `<svg>` directly with computed values (most Svelte-like)

**Suggestion**: 
- If Recharts/D3 already in use elsewhere: **reuse that pattern** for consistency
- If neither: **consider Recharts for simplicity** or **native SVG with computed transforms** for control

---

## Key Implementation Notes

### Data Mapping
- `monthlyPoints`: Already computed by `buildMonthlyPoints()` — use for hero chart
- `entityTrends`: Already computed by `buildDimensionTrends()` — use for composition/variance
- `insights`: Already computed by `calculateMoMInsights()` — use for volatility/momentum KPIs

### State Dependencies
- Charts reactive to: `reportCurrency`, `monthWindow`, `dimension`, `endMonth`, `topN`
- Derived updates trigger `loadExpenseData()`, which feeds fresh `monthlyPoints` and `entityTrends`

### Performance
- All calculations already in-memory; no additional API calls needed
- Memoize chart rendering if data is large (use Svelte `$effect`)

### Accessibility
- Add `role="img"` to charts with `aria-label` describing the trend
- Provide text alternative: "Expenses increased 12% month-over-month; largest driver is Food (+$340)"

---

## Success Criteria

✅ **Currency dropdown visible** (if multi-currency configured)  
✅ **Hero trajectory chart** showing expense trend with hover details  
✅ **Variance/movers visualization** (waterfall or grouped bar)  
✅ **Composition chart** (stacked area showing dimension breakdown)  
✅ **Compact layout** (no large empty spaces, efficient use of width)  
✅ **Detail tables** provide drill-down (not primary focus)  
✅ **Zero type errors** (npm run check passes)  
✅ **Responsive** (works on desktop and mobile)  

---

## Comparison: YoY vs MoM Design

| Aspect | YoY (Basic) | MoM (Advanced) |
|--------|-----------|---------|
| Primary View | Year summary table | Expense trajectory chart |
| Interaction | Minimal | Rich: hover, click, filter |
| Visuals | Text-heavy | Chart-driven |
| Drill-down | Limited | Deep: dimension, period, currency |
| Use Case | High-level review | Detailed trend analysis |

**MoM should be a clear step up in sophistication and visual richness.**
