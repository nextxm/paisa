<script lang="ts">
  import _ from "lodash";
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import MonthPicker from "$lib/components/MonthPicker.svelte";
  import SparklineChart from "$lib/components/SparklineChart.svelte";
  import ExpenseTrajectoryChart from "$lib/components/ExpenseTrajectoryChart.svelte";
  import DimensionVarianceChart from "$lib/components/DimensionVarianceChart.svelte";
  import DimensionCompositionChart from "$lib/components/DimensionCompositionChart.svelte";
  import MoMDeltaChart from "$lib/components/MoMDeltaChart.svelte";
  import {
    ajax,
    formatCurrency,
    formatPercentage,
    type Posting,
    type ExpenseTrend
  } from "$lib/utils";
  import {
    availableExpenseMonths,
    buildDimensionTrends,
    buildMonthlyPoints,
    calculateMoMInsights,
    monthRange,
    type MoMDimension
  } from "$lib/mom_utils";

  const YEARS_TO_FETCH = 5;

  let expenses: Posting[] = $state([]);
  let trends: ExpenseTrend[] = $state([]);

  let monthWindow = $state(12);
  let topN = $state(8);
  let dimension: MoMDimension = $state("category");
  let endMonth = $state(dayjs().format("YYYY-MM"));

  let currencyMode: "default" | "actual" = $state("default");
  let defaultCurrency = $state("");
  let selectedCurrencyInActualMode = $state("");

  let availableMonths = $derived(availableExpenseMonths(expenses));
  let minMonth = $derived(
    availableMonths.length > 0
      ? dayjs(`${availableMonths[0]}-01`)
      : dayjs().add(-1, "year").startOf("month")
  );
  let maxMonth = $derived(
    availableMonths.length > 0
      ? dayjs(`${availableMonths[availableMonths.length - 1]}-01`)
      : dayjs().startOf("month")
  );

  let selectedMonths = $derived(monthRange(endMonth, monthWindow));

  let actualCurrencies = $derived.by(() => {
    const monthSet = new Set(selectedMonths);
    return _.chain(expenses)
      .filter(
        (posting) =>
          posting.account.startsWith("Expenses:") &&
          !posting.account.startsWith("Expenses:Tax") &&
          monthSet.has(posting.date.format("YYYY-MM"))
      )
      .map((posting) => posting.commodity || defaultCurrency)
      .uniq()
      .sort()
      .value();
  });

  let monthlyPoints = $derived(buildMonthlyPoints(expenses, selectedMonths, currencyMode));

  let entityTrends = $derived(
    buildDimensionTrends(expenses, selectedMonths, dimension, topN, currencyMode)
  );

  /**
   * When a currency is selected in actual mode, filter both monthlyPoints and entityTrends
   * to show only that currency's data (remove the currency suffix from keys).
   */
  let displayMonthlyPoints = $derived.by(() => {
    if (!selectedCurrencyInActualMode || currencyMode === "default") {
      return monthlyPoints;
    }
    return monthlyPoints
      .filter((p) => p.currency === selectedCurrencyInActualMode)
      .map((p) => ({
        ...p,
        label: undefined
      }));
  });

  let displayEntityTrends = $derived.by(() => {
    if (!selectedCurrencyInActualMode || currencyMode === "default") {
      return entityTrends;
    }
    const baseKeyPattern = / \([A-Z]{3}\)$/;
    return entityTrends
      .filter((t) => t.currency === selectedCurrencyInActualMode)
      .map((t) => ({
        ...t,
        key: t.key.replace(baseKeyPattern, "")
      }));
  });

  let insights = $derived(calculateMoMInsights(displayMonthlyPoints, displayEntityTrends));
  let showSummaryCards = $derived(
    currencyMode === "default" ||
      actualCurrencies.length <= 1 ||
      selectedCurrencyInActualMode !== ""
  );
  let singleActualCurrency = $derived(actualCurrencies.length === 1 ? actualCurrencies[0] : "");

  let topMovers = $derived(
    _.chain(displayEntityTrends)
      .orderBy([(entry) => Math.abs(entry.change), (entry) => entry.current], ["desc", "desc"])
      .take(Math.min(6, topN))
      .value()
  );

  function formatMetric(value: number, currency = "") {
    if (currencyMode === "actual") {
      const resolvedCurrency = currency || selectedCurrencyInActualMode || singleActualCurrency;
      return resolvedCurrency
        ? `${formatCurrency(value)} ${resolvedCurrency}`
        : formatCurrency(value);
    }
    return defaultCurrency ? `${formatCurrency(value)} ${defaultCurrency}` : formatCurrency(value);
  }

  function formatPointLabel(point: { month: string; label?: string }) {
    return point.label || dayjs(`${point.month}-01`).format("MMM YYYY");
  }

  function trendText(changePct: number | null) {
    if (changePct == null) {
      return "n/a";
    }

    const direction = changePct > 0 ? "up" : changePct < 0 ? "down" : "flat";
    return `${direction} ${formatPercentage(Math.abs(changePct))}`;
  }

  function getAverageForEntity(series: Record<string, number>) {
    const values = Object.values(series).filter((v) => v > 0);
    return values.length > 0 ? _.sum(values) / values.length : 0;
  }

  /**
   * When in actual currency mode with multiple currencies, groups entity trends
   * by their base dimension (e.g., "Food" groups "Food (CAD)", "Food (USD)").
   * Returns rows with grouping context for visual display.
   */
  let groupedEntityTrends = $derived.by(() => {
    if (
      currencyMode === "default" ||
      actualCurrencies.length <= 1 ||
      selectedCurrencyInActualMode
    ) {
      return displayEntityTrends.map((row) => ({ ...row, isGrouped: false, isGroupHeader: false }));
    }

    // Group by base key (without currency suffix)
    const baseKeyPattern = / \([A-Z]{3}\)$/;
    const groups: Record<string, typeof displayEntityTrends> = {};
    for (const row of displayEntityTrends) {
      const baseKey = row.key.replace(baseKeyPattern, "");
      groups[baseKey] = groups[baseKey] || [];
      groups[baseKey].push(row);
    }

    const result: Array<
      (typeof displayEntityTrends)[0] & { isGrouped: boolean; isGroupHeader: boolean }
    > = [];
    for (const baseKey of Object.keys(groups).sort()) {
      const currencyRows = groups[baseKey];
      if (currencyRows.length > 1) {
        for (const row of currencyRows) {
          result.push({ ...row, isGrouped: true, isGroupHeader: false });
        }
      } else {
        result.push({ ...currencyRows[0], isGrouped: false, isGroupHeader: false });
      }
    }

    return result;
  });

  async function loadExpenseData() {
    const params = new URLSearchParams();
    params.set("years", YEARS_TO_FETCH.toString());

    const data = await ajax(`/api/expense?${params.toString()}`);
    expenses = data.expenses || [];
    trends = data.trends || [];

    const months = availableExpenseMonths(data.expenses || []);
    if (months.length > 0) {
      endMonth = _.last(months) || endMonth;
    }
  }

  onMount(async () => {
    // Get default currency from config
    defaultCurrency = globalThis.USER_CONFIG?.default_currency || "";
    await loadExpenseData();
  });

  $effect(() => {
    if (selectedCurrencyInActualMode && !actualCurrencies.includes(selectedCurrencyInActualMode)) {
      selectedCurrencyInActualMode = "";
    }
  });
</script>

<section class="section py-4">
  <div class="container is-fluid">
    <!-- Compact Control Panel -->
    <div class="box p-3 mb-4">
      <div class="mom-controls">
        <div class="field mb-0 mom-control-field">
          <p class="label is-size-7 mb-1">End month</p>
          <MonthPicker min={minMonth} max={maxMonth} bind:value={endMonth} />
        </div>

        <div class="field mb-0 mom-control-field">
          <label class="label is-size-7 mb-1" for="mom-window">Window</label>
          <div class="control">
            <div class="select is-small">
              <select id="mom-window" bind:value={monthWindow}>
                <option value={6}>Last 6 months</option>
                <option value={12}>Last 12 months</option>
                <option value={24}>Last 24 months</option>
              </select>
            </div>
          </div>
        </div>

        <div class="field mb-0 mom-control-field">
          <label class="label is-size-7 mb-1" for="mom-dimension">Breakdown</label>
          <div class="control">
            <div class="select is-small">
              <select id="mom-dimension" bind:value={dimension}>
                <option value="category">Category</option>
                <option value="payee">Payee</option>
                <option value="account">Account</option>
              </select>
            </div>
          </div>
        </div>

        <div class="field mb-0 mom-control-field">
          <label class="label is-size-7 mb-1" for="mom-topn">Rows</label>
          <div class="control">
            <div class="select is-small">
              <select id="mom-topn" bind:value={topN}>
                <option value={5}>Top 5</option>
                <option value={8}>Top 8</option>
                <option value={12}>Top 12</option>
              </select>
            </div>
          </div>
        </div>

        <div class="control mom-control-field mom-toggle-field">
          <div class="field mb-0">
            <label class="label is-size-7 mb-1" for="mom-currency-mode">Currency</label>
            <div class="is-flex is-align-items-center">
              <span class="is-size-7 has-text-grey mr-2">Default</span>
              <input
                id="mom-currency-mode"
                type="checkbox"
                class="switch is-rounded is-small"
                checked={currencyMode === "actual"}
                onchange={(e) => {
                  currencyMode = (e.currentTarget as HTMLInputElement).checked
                    ? "actual"
                    : "default";
                  selectedCurrencyInActualMode = "";
                }}
              />
              <label class="ml-1" for="mom-currency-mode">Actual</label>
            </div>
          </div>
        </div>

        {#if currencyMode === "actual" && actualCurrencies.length > 1}
          <div class="field mb-0 mom-control-field">
            <label class="label is-size-7 mb-1" for="mom-currency-select">Currency</label>
            <div class="control">
              <div class="select is-small">
                <select id="mom-currency-select" bind:value={selectedCurrencyInActualMode}>
                  <option value="">All currencies</option>
                  {#each actualCurrencies as currency}
                    <option value={currency}>{currency}</option>
                  {/each}
                </select>
              </div>
            </div>
          </div>
        {/if}
      </div>
    </div>

    <!-- Hero Chart: Expense Trajectory -->
    <div class="box mb-4">
      <ExpenseTrajectoryChart id="mom-trajectory-chart" data={displayMonthlyPoints} />
    </div>

    <!-- Summary Cards (3 columns, compact) -->
    {#if showSummaryCards}
      <div class="columns is-multiline mb-3">
        <div class="column is-4">
          <div class="box p-3">
            <p class="heading is-size-7 mb-2">Latest Month</p>
            <p class="title is-5 mb-2">{dayjs(`${insights.latestMonth}-01`).format("MMM YYYY")}</p>
            <p class="title is-4 mb-2">{formatMetric(insights.latestTotal)}</p>
            <div class="is-size-7">
              <p class="mb-1">
                <span class="has-text-grey">Prev:</span>
                {formatMetric(insights.previousTotal)}
              </p>
              <p>{trendText(displayMonthlyPoints.at(-1)?.changePct ?? null)}</p>
            </div>
          </div>
        </div>

        <div class="column is-4">
          <div class="box p-3">
            <p class="heading is-size-7 mb-2">Range Highlights</p>
            <div class="mb-2">
              <span class="is-size-8 has-text-grey">High:</span>
              <p class="is-size-7 has-text-weight-semibold">
                {insights.highestMonth
                  ? `${dayjs(`${insights.highestMonth.month}-01`).format("MMM")} ${formatMetric(insights.highestMonth.total)}`
                  : "n/a"}
              </p>
            </div>
            <div class="mb-2">
              <span class="is-size-8 has-text-grey">Low:</span>
              <p class="is-size-7 has-text-weight-semibold">
                {insights.lowestMonth
                  ? `${dayjs(`${insights.lowestMonth.month}-01`).format("MMM")} ${formatMetric(insights.lowestMonth.total)}`
                  : "n/a"}
              </p>
            </div>
            <div>
              <span class="is-size-8 has-text-grey">Avg ({monthWindow}m):</span>
              <p class="is-size-7 has-text-weight-semibold">
                {formatMetric(insights.averageMonthlySpend)}
              </p>
            </div>
          </div>
        </div>

        <div class="column is-4">
          <div class="box p-3">
            <p class="heading is-size-7 mb-2">Volatility & Trend</p>
            <div class="mb-3">
              <span
                class="is-size-8 has-text-grey"
                title="Coefficient of Variation = StdDev / Mean. Low (<15%): stable spending. Medium (15-30%): moderate swings. High (>30%): erratic spending."
                >Volatility <i class="fas fa-circle-info" style="font-size: 0.65rem;"></i></span
              >
              <p class="is-size-6 has-text-weight-semibold">
                {#if insights.volatilityPct == null}
                  <span class="has-text-grey">n/a</span>
                {:else}
                  <span
                    class={insights.volatilityPct > 0.3
                      ? "has-text-danger"
                      : insights.volatilityPct > 0.15
                        ? "has-text-warning"
                        : "has-text-success"}
                    title={insights.volatilityPct > 0.3
                      ? "High volatility — spending is erratic"
                      : insights.volatilityPct > 0.15
                        ? "Moderate volatility — some month-to-month swings"
                        : "Low volatility — spending is consistent"}
                  >
                    {formatPercentage(insights.volatilityPct)}
                    {insights.volatilityPct > 0.3
                      ? " ⚠️"
                      : insights.volatilityPct <= 0.15
                        ? " ✓"
                        : ""}
                  </span>
                {/if}
              </p>
            </div>
            <div class="mb-2">
              <span class="is-size-8 has-text-grey">Trend:</span>
              <p class="is-size-7 has-text-weight-semibold">
                {#if displayMonthlyPoints.length >= 2}
                  {displayMonthlyPoints.at(-1)?.changePct !== undefined &&
                  (displayMonthlyPoints.at(-1)?.changePct ?? 0) > 0
                    ? "📈 Up this month"
                    : "📉 Down this month"}
                {:else}
                  <span class="has-text-grey">n/a</span>
                {/if}
              </p>
            </div>
            <div>
              <span
                class="is-size-8 has-text-grey"
                title="Average of all months in the selected window"
                >3M Avg <i class="fas fa-circle-info" style="font-size: 0.65rem;"></i></span
              >
              <p class="is-size-7 has-text-weight-semibold">
                {formatMetric(insights.averageMonthlySpend)}
              </p>
            </div>
          </div>
        </div>
      </div>
    {:else}
      <div class="notification is-light mb-3 py-3 px-4 is-size-7">
        Actual currency mode contains multiple commodities in the selected window. Monthly charts
        and tables are split by currency, so the single-value summary cards are hidden to avoid
        mixing currencies into one total. Select a currency from the dropdown above to view clear,
        single-currency summary cards.
      </div>
    {/if}

    <!-- Variance & Composition Charts (2-column grid) -->
    <div class="columns is-multiline mb-3">
      <div class="column is-6">
        <div class="box p-3">
          <DimensionVarianceChart
            id="mom-variance-chart"
            data={topMovers.map((m) => ({
              key: m.key,
              previous: m.previous,
              current: m.current,
              change: m.change,
              changePct: m.changePct
            }))}
            title={`Top Movers (by ${_.capitalize(dimension)})`}
          />
        </div>
      </div>

      <div class="column is-6">
        <div class="box p-3">
          <DimensionCompositionChart
            id="mom-composition-chart"
            data={displayEntityTrends}
            months={selectedMonths}
            title={`Distribution by ${_.capitalize(dimension)}`}
          />
        </div>
      </div>
    </div>

    <!-- Detail Tables: Timeline + Delta chart side-by-side -->
    <div class="columns is-multiline">
      <div class="column is-5">
        <div class="box p-3">
          <h3 class="heading is-size-7 mb-3">Monthly Timeline</h3>
          <div class="table-container">
            <table class="table is-fullwidth is-hoverable is-size-8" style="white-space: nowrap;">
              <thead>
                <tr>
                  <th class="is-size-8">Month</th>
                  <th class="has-text-right is-size-8">Total</th>
                  <th class="has-text-right is-size-8" title="Change vs previous month">MoM</th>
                  <th class="has-text-right is-size-8" title="3-month rolling average">3M Avg</th>
                </tr>
              </thead>
              <tbody>
                {#each displayMonthlyPoints as point}
                  <tr>
                    <td class="is-size-8">{formatPointLabel(point)}</td>
                    <td class="has-text-right has-text-weight-semibold is-size-8"
                      >{formatMetric(point.total, point.currency)}</td
                    >
                    <td class="has-text-right is-size-8">
                      {#if point.change == null}
                        <span class="has-text-grey">—</span>
                      {:else}
                        <span
                          class={point.change > 0
                            ? "has-text-danger"
                            : point.change < 0
                              ? "has-text-success"
                              : "has-text-grey"}
                          title={`${point.change > 0 ? "+" : ""}${formatMetric(point.change, point.currency)}`}
                        >
                          {point.changePct == null
                            ? "—"
                            : `${point.changePct > 0 ? "+" : ""}${formatPercentage(point.changePct)}`}
                        </span>
                      {/if}
                    </td>
                    <td class="has-text-right has-text-grey is-size-8"
                      >{formatMetric(point.movingAverage3, point.currency)}</td
                    >
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <div class="column is-7">
        <div class="box p-3">
          <MoMDeltaChart id="mom-delta-chart" data={displayMonthlyPoints} />
        </div>
      </div>

      <div class="column is-full">
        <div class="box p-3">
          <h3 class="heading is-size-7 mb-3">
            Breakdown by {_.capitalize(dimension)}
            <span class="is-size-8 has-text-grey has-text-weight-normal ml-2"
              >— hover rows for details</span
            >
          </h3>
          <div class="table-container">
            <table class="table is-fullwidth is-hoverable is-size-8">
              <thead>
                <tr>
                  <th class="is-size-8">{_.capitalize(dimension)}</th>
                  <th class="has-text-right is-size-8">Latest</th>
                  <th class="has-text-right is-size-8" title="Average across all months in window"
                    >Avg</th
                  >
                  <th class="has-text-right is-size-8">Prev</th>
                  <th class="has-text-right is-size-8" title="Change vs previous month">MoM Δ</th>
                  <th class="has-text-right is-size-8" title="Share of total spending this month"
                    >Share</th
                  >
                  <th class="is-size-8" title="Spending trend over selected window">Trend</th>
                </tr>
              </thead>
              <tbody>
                {#each groupedEntityTrends as row}
                  {@const avg = getAverageForEntity(row.series)}
                  <tr class={row.isGrouped ? "is-size-8 has-background-white-ter" : ""}>
                    <td
                      class="is-size-8"
                      style={row.isGrouped
                        ? "font-weight: 400; padding-left: 2rem;"
                        : "font-weight: 500;"}
                    >
                      {row.key}
                    </td>
                    <td class="has-text-right has-text-weight-semibold is-size-8"
                      >{formatMetric(row.current, row.currency)}</td
                    >
                    <td class="has-text-right has-text-grey is-size-8"
                      >{formatMetric(avg, row.currency)}</td
                    >
                    <td class="has-text-right has-text-grey is-size-8"
                      >{formatMetric(row.previous, row.currency)}</td
                    >
                    <td class="has-text-right is-size-8">
                      <span
                        class={row.change > 0
                          ? "has-text-danger"
                          : row.change < 0
                            ? "has-text-success"
                            : "has-text-grey"}
                        title={`${row.change > 0 ? "+" : ""}${formatMetric(row.change, row.currency)}`}
                      >
                        {row.changePct == null
                          ? "—"
                          : `${row.changePct > 0 ? "+" : ""}${formatPercentage(row.changePct)}`}
                      </span>
                    </td>
                    <td class="has-text-right has-text-link is-size-8"
                      >{formatPercentage(row.shareOfCurrent)}</td
                    >
                    <td class="is-size-8"><SparklineChart data={row.series} color="#3273dc" /></td>
                  </tr>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<style lang="scss">
  .mom-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: flex-end;
    column-gap: 1rem;
    row-gap: 0.75rem;
  }

  .mom-control-field {
    min-width: 10rem;
  }

  .mom-control-field .label {
    margin-bottom: 0.35rem !important;
    padding-left: 0.1rem;
  }

  .mom-control-field .select,
  .mom-control-field .select select {
    width: 100%;
  }

  .mom-toggle-field {
    min-width: auto;
    padding-inline: 0.35rem;
  }

  @media (max-width: 960px) {
    .mom-control-field {
      min-width: 9rem;
      flex: 1 1 11rem;
    }

    .mom-toggle-field {
      flex: 0 0 auto;
      padding-inline: 0;
    }
  }
</style>
