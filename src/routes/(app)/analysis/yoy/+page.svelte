<script lang="ts">
  import _ from "lodash";
  import dayjs from "dayjs";
  import Papa from "papaparse";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import YoYChart from "$lib/components/YoYChart.svelte";
  import { ajax, formatCurrency, formatPercentage, type Posting, type YoYSeries } from "$lib/utils";
  import {
    buildCategoryYoYSeries,
    buildYoYDashboardSummary,
    buildYoYExportRows,
    calculateYoYInsights,
    orderedYears,
    type YoYChartType
  } from "$lib/yoy_utils";

  const currentYear = dayjs().year();
  const availableUntilYears = _.range(currentYear, currentYear - 11, -1).map(String);

  let yearCount = $state(2);
  let untilYear = $state(String(currentYear));
  let expenseSeries: Record<string, YoYSeries> = $state({});
  let incomeSeries: Record<string, YoYSeries> = $state({});
  let expensePostings: Posting[] = $state([]);
  let category = $state("");
  let categoryChartType: YoYChartType = $state("line");

  let years = $derived(orderedYears(expenseSeries));
  let categories = $derived(
    _.sortBy(Object.keys(buildCategoryYoYSeries(expensePostings, years)), (value) => value)
  );
  let categorySeries = $derived(buildCategoryYoYSeries(expensePostings, years));
  let selectedCategorySeries = $derived((category ? categorySeries[category] : undefined) || {});
  let insights = $derived(calculateYoYInsights(expenseSeries, incomeSeries, categorySeries));
  let dashboard = $derived(buildYoYDashboardSummary(expenseSeries, incomeSeries, categorySeries));
  let topMovers = $derived(dashboard.topCategoryMovers.slice(0, 6));
  let topMover = $derived(topMovers[0] || null);
  let expenseLoadPct = $derived(
    dashboard.latestIncomeTotal === 0
      ? null
      : (dashboard.latestExpenseTotal / dashboard.latestIncomeTotal) * 100
  );
  let maxAbsNet = $derived(
    _.max(dashboard.monthlyNet.map((row) => Math.abs(row.net)).filter((value) => value > 0)) || 1
  );

  async function refresh(selectedYearCount: number, selectedUntilYear: string) {
    try {
      const [expenseData, incomeData] = await Promise.all([
        ajax(`/api/expense?years=${selectedYearCount}&until_year=${selectedUntilYear}`),
        ajax(`/api/income?years=${selectedYearCount}&until_year=${selectedUntilYear}`)
      ]);

      expenseSeries = expenseData.multi_year || {};
      incomeSeries = incomeData.multi_year || {};
      expensePostings = expenseData.expenses || [];

      const availableCategories = _.sortBy(
        Object.keys(buildCategoryYoYSeries(expenseData.expenses || [], orderedYears(expenseSeries))),
        (value) => value
      );
      if (!availableCategories.includes(category)) {
        category = availableCategories[0] || "";
      }
    } catch (e) {
      console.error("YoY refresh failed:", e);
    }
  }

  function downloadCSV() {
    const rows = buildYoYExportRows(expenseSeries, incomeSeries);
    const csv = Papa.unparse(rows);
    const link = document.createElement("a");
    const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
    link.href = window.URL.createObjectURL(blob);
    link.download = "paisa-yoy-analysis.csv";
    link.click();
  }

  function trendLabel(changePct: number | null, noun: string) {
    if (changePct == null) return `${noun}: n/a`;
    return `${noun}: ${changePct >= 0 ? "up" : "down"} ${formatPercentage(Math.abs(changePct) / 100)}`;
  }

  function signedPct(changePct: number | null) {
    if (changePct == null) return "n/a";
    return `${changePct > 0 ? "+" : ""}${formatPercentage(changePct / 100)}`;
  }

  function deltaLabel(current: number, previous: number) {
    const delta = current - previous;
    return `${delta >= 0 ? "+" : "-"}${formatCurrency(Math.abs(delta))}`;
  }

  function pctTone(changePct: number | null, favorableWhenPositive = true) {
    if (changePct == null) return "has-text-grey";
    const isPositive = changePct >= 0;
    const favorable = favorableWhenPositive ? isPositive : !isPositive;
    return favorable ? "has-text-success" : "has-text-danger";
  }

  $effect(() => {
    void refresh(yearCount, untilYear);
  });
</script>

<section class="section yoy-section">
  <div class="container is-fluid yoy-shell">
    <div class="box yoy-control-box mb-4">
      <div class="yoy-toolbar is-flex is-flex-wrap-wrap">
        <div class="yoy-filter-row">
          <div class="field">
            <label class="label mb-1" for="yoy-years">Years to compare</label>
            <div class="control">
              <div class="select is-small">
                <select id="yoy-years" bind:value={yearCount}>
                  <option value={2}>2 years</option>
                  <option value={3}>3 years</option>
                  <option value={4}>4 years</option>
                  <option value={5}>5 years</option>
                </select>
              </div>
            </div>
          </div>
          <div class="field">
            <label class="label mb-1" for="yoy-until-year">Until year</label>
            <div class="control">
              <div class="select is-small">
                <select id="yoy-until-year" bind:value={untilYear}>
                  {#each availableUntilYears as year}
                    <option value={year}>{year}</option>
                  {/each}
                </select>
              </div>
            </div>
          </div>
        </div>

        <button class="button is-small is-link is-light yoy-download-btn" onclick={downloadCSV}>
          <span class="icon is-small"><i class="fas fa-download"></i></span>
          <span>Download CSV</span>
        </button>
      </div>

      <div class="yoy-hero mt-3">
        <div>
          <p class="heading mb-2 yoy-hero-heading">Year over year diagnostic</p>
          <p class="title is-4 mb-1 yoy-hero-title">Spending, income and savings momentum</p>
          <p class="is-size-7 mb-0 yoy-hero-sub">
            {dashboard.previousYear && dashboard.latestYear
              ? `${dashboard.previousYear} to ${dashboard.latestYear}`
              : "Select at least two years to unlock full insights"}
          </p>
        </div>
        <div class="yoy-hero-chips">
          <span class="tag is-dark is-light">{yearCount} year window</span>
          <span class="tag is-info is-light">through {untilYear}</span>
          {#if dashboard.savingsRatePct != null}
            <span class="tag is-success is-light"
              >Savings rate {formatPercentage(dashboard.savingsRatePct / 100)}</span
            >
          {/if}
        </div>
      </div>
    </div>

    <div class="columns is-multiline yoy-stat-grid mb-2">
      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Spending ({dashboard.latestYear || "n/a"})</p>
            <p class="title is-6 yoy-stat-value mb-0">
              {formatCurrency(dashboard.latestExpenseTotal)}
            </p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">
            Prev {dashboard.previousYear || "n/a"}: {formatCurrency(dashboard.previousExpenseTotal)}
          </p>
          <p
            class={`is-size-7 has-text-weight-semibold ${pctTone(dashboard.expenseChangePct, false)}`}
          >
            {signedPct(dashboard.expenseChangePct)} ({deltaLabel(
              dashboard.latestExpenseTotal,
              dashboard.previousExpenseTotal
            )})
          </p>
        </div>
      </div>

      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Income ({dashboard.latestYear || "n/a"})</p>
            <p class="title is-6 yoy-stat-value mb-0">
              {formatCurrency(dashboard.latestIncomeTotal)}
            </p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">
            Prev {dashboard.previousYear || "n/a"}: {formatCurrency(dashboard.previousIncomeTotal)}
          </p>
          <p
            class={`is-size-7 has-text-weight-semibold ${pctTone(dashboard.incomeChangePct, true)}`}
          >
            {signedPct(dashboard.incomeChangePct)} ({deltaLabel(
              dashboard.latestIncomeTotal,
              dashboard.previousIncomeTotal
            )})
          </p>
        </div>
      </div>

      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Net savings ({dashboard.latestYear || "n/a"})</p>
            <p class="title is-6 yoy-stat-value mb-0">{formatCurrency(dashboard.latestNetTotal)}</p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">
            Prev {dashboard.previousYear || "n/a"}: {formatCurrency(dashboard.previousNetTotal)}
          </p>
          <p class={`is-size-7 has-text-weight-semibold ${pctTone(dashboard.netChangePct, true)}`}>
            {signedPct(dashboard.netChangePct)} ({deltaLabel(
              dashboard.latestNetTotal,
              dashboard.previousNetTotal
            )})
          </p>
        </div>
      </div>

      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Savings rate</p>
            <p class="title is-6 yoy-stat-value mb-0">
              {dashboard.savingsRatePct == null
                ? "n/a"
                : formatPercentage(dashboard.savingsRatePct / 100)}
            </p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">
            Prev: {dashboard.previousSavingsRatePct == null
              ? "n/a"
              : formatPercentage(dashboard.previousSavingsRatePct / 100)}
          </p>
          <p class="is-size-7 has-text-weight-semibold has-text-link">
            Top category: {insights.topExpenseCategory?.name || "n/a"}
          </p>
        </div>
      </div>

      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Biggest mover</p>
            <p class="title is-6 yoy-stat-value mb-0">{topMover?.name || "n/a"}</p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">
            Latest: {formatCurrency(topMover?.latestYearTotal || 0)}
          </p>
          <p
            class={`is-size-7 has-text-weight-semibold ${pctTone(topMover?.changePct ?? null, false)}`}
          >
            {signedPct(topMover?.changePct ?? null)}
          </p>
        </div>
      </div>

      <div class="column is-2-desktop is-4-tablet is-6-mobile">
        <div class="box yoy-stat-card">
          <div class="yoy-stat-top">
            <p class="heading is-size-7 mb-0">Efficiency snapshot</p>
            <p class="title is-6 yoy-stat-value mb-0">
              {expenseLoadPct == null ? "n/a" : formatPercentage(expenseLoadPct / 100)}
            </p>
          </div>
          <p class="is-size-7 mb-1 has-text-grey">Expense as % of income</p>
          <p class="is-size-7 has-text-weight-semibold has-text-link">
            Best month: {dashboard.bestNetMonth
              ? `${dashboard.bestNetMonth.month} (${formatCurrency(dashboard.bestNetMonth.net)})`
              : "n/a"}
          </p>
        </div>
      </div>
    </div>

    <div class="columns is-multiline">
      <div class="column is-full">
        <div class="box yoy-chart-box">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
            <p class="heading is-size-7 mb-0">Spending trajectory by month</p>
            <span class="tag is-light">Year-over-Year Spending</span>
          </div>
          <YoYChart id="d3-yoy-expense" series={expenseSeries} chartType="line" />
        </div>
        <BoxLabel text="Year-over-Year Spending" />
      </div>

      <div class="column is-full">
        <div class="box yoy-chart-box">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
            <p class="heading is-size-7 mb-0">Income trajectory by month</p>
            <span class="tag is-light">Year-over-Year Income</span>
          </div>
          <YoYChart id="d3-yoy-income" series={incomeSeries} chartType="line" />
        </div>
        <BoxLabel text="Year-over-Year Income" />
      </div>

      <div class="column is-full">
        <div class="box yoy-chart-box">
          <div
            class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-2 mb-3"
          >
            <div class="field mb-0">
              <div class="control">
                <div class="select is-small">
                  <select bind:value={category}>
                    {#each categories as value}
                      <option {value}>{value}</option>
                    {/each}
                  </select>
                </div>
              </div>
            </div>
            <div class="buttons has-addons">
              <button
                class="button is-small"
                class:is-primary={categoryChartType === "line"}
                onclick={() => (categoryChartType = "line")}>Line</button
              >
              <button
                class="button is-small"
                class:is-primary={categoryChartType === "bar"}
                onclick={() => (categoryChartType = "bar")}>Bar</button
              >
            </div>
          </div>

          <YoYChart
            id="d3-yoy-category"
            series={selectedCategorySeries}
            chartType={categoryChartType}
          />
        </div>
        <BoxLabel text={`Year-over-Year by Category${category ? ` (${category})` : ""}`} />
      </div>

      <div class="column is-full">
        <div class="box yoy-chart-box">
          <div class="columns is-multiline">
            <div class="column is-7-desktop">
              <p class="heading is-size-7 mb-2">
                Category movers ({dashboard.latestYear || "n/a"})
              </p>
              <div class="table-container">
                <table
                  class="table is-fullwidth is-hoverable is-size-8"
                  style="white-space: nowrap;"
                >
                  <thead>
                    <tr>
                      <th class="is-size-8">Category</th>
                      <th class="has-text-right is-size-8">Latest</th>
                      <th class="has-text-right is-size-8">YoY</th>
                      <th class="has-text-right is-size-8">Share</th>
                    </tr>
                  </thead>
                  <tbody>
                    {#if topMovers.length === 0}
                      <tr>
                        <td colspan="4" class="has-text-grey is-size-8"
                          >No category data available.</td
                        >
                      </tr>
                    {:else}
                      {#each topMovers as mover}
                        <tr>
                          <td class="has-text-weight-semibold is-size-8">{mover.name}</td>
                          <td class="has-text-right is-size-8"
                            >{formatCurrency(mover.latestYearTotal)}</td
                          >
                          <td class="has-text-right is-size-8">
                            <span class={pctTone(mover.changePct, false)}>
                              {signedPct(mover.changePct)}
                            </span>
                          </td>
                          <td class="has-text-right is-size-8">
                            {mover.shareOfExpensePct == null
                              ? "n/a"
                              : formatPercentage(mover.shareOfExpensePct / 100)}
                          </td>
                        </tr>
                      {/each}
                    {/if}
                  </tbody>
                </table>
              </div>
            </div>

            <div class="column is-5-desktop">
              <p class="heading is-size-7 mb-2">
                Monthly net profile ({dashboard.latestYear || "n/a"})
              </p>
              <div class="yoy-monthly-net-list">
                {#if dashboard.monthlyNet.length === 0}
                  <p class="is-size-7 has-text-grey">No monthly data available.</p>
                {:else}
                  {#each dashboard.monthlyNet as row}
                    <div class="yoy-monthly-net-row">
                      <div class="is-flex is-justify-content-space-between is-size-8 mb-1">
                        <span class="has-text-weight-semibold">{row.month}</span>
                        <span class={row.net >= 0 ? "has-text-success" : "has-text-danger"}
                          >{formatCurrency(row.net)}</span
                        >
                      </div>
                      <div class="yoy-net-track">
                        <div
                          class={`yoy-net-fill ${row.net >= 0 ? "is-positive" : "is-negative"}`}
                          style={`width:${Math.max((Math.abs(row.net) / maxAbsNet) * 100, 4)}%`}
                        ></div>
                      </div>
                    </div>
                  {/each}
                {/if}
              </div>

              <div class="notification is-light is-size-7 mt-3 mb-0">
                <p class="mb-1">
                  <strong>{trendLabel(insights.spendingChangePct, "Spending")}</strong>
                </p>
                <p class="mb-1">
                  <strong>{trendLabel(insights.incomeChangePct, "Income")}</strong>
                </p>
                <p class="mb-1">
                  <strong>
                    Highest expense month:
                    {dashboard.highestExpenseMonth
                      ? `${dashboard.highestExpenseMonth.month} (${formatCurrency(dashboard.highestExpenseMonth.expense)})`
                      : "n/a"}
                  </strong>
                </p>
                <p>
                  <strong>
                    Best net month:
                    {dashboard.bestNetMonth
                      ? `${dashboard.bestNetMonth.month} (${formatCurrency(dashboard.bestNetMonth.net)})`
                      : "n/a"}
                  </strong>
                </p>
              </div>
            </div>
          </div>
        </div>
        <BoxLabel text="YoY Insights" />
      </div>
    </div>
  </div>
</section>

<style>
  section.yoy-section {
    padding: 1.5rem 1.5rem 3rem;
  }

  .container.yoy-shell {
    --yoy-ink: #1a2634;
    --yoy-soft: #546a84;
    --yoy-surface: #ffffff;
    --yoy-surface-alt: #f8fafc;
    --yoy-border: rgba(180, 195, 215, 0.4);
    --yoy-accent: #1e7a6f;
    --yoy-accent-2: #1f5d9a;
    --yoy-card-bg: #ffffff;
    --yoy-hero-text: #f4fbff;
    --yoy-tag-bg: #f1f5f9;
    --yoy-tag-text: #475569;
    --yoy-input-bg: #ffffff;
    --yoy-input-text: #1a2634;
    --yoy-row-bg: #ffffff;
    --yoy-track-bg: #f1f5f9;
    max-width: 1620px;
    margin: 0 auto;
    padding: 0 0.75rem;
  }

  :global(html[data-theme="dark"]) .container.yoy-shell {
    --yoy-ink: #e6edf7;
    --yoy-soft: #9dafc5;
    --yoy-surface: rgba(20, 30, 48, 0.72);
    --yoy-surface-alt: rgba(27, 39, 58, 0.88);
    --yoy-border: rgba(141, 163, 191, 0.24);
    --yoy-card-bg: linear-gradient(180deg, rgba(20, 30, 48, 0.76), rgba(15, 24, 40, 0.94));
    --yoy-hero-text: #f4fbff;
    --yoy-tag-bg: rgba(40, 57, 80, 0.78);
    --yoy-tag-text: #cdddf3;
    --yoy-input-bg: rgba(11, 20, 34, 0.75);
    --yoy-input-text: #dbe7f8;
    --yoy-row-bg: rgba(18, 28, 45, 0.76);
    --yoy-track-bg: rgba(118, 141, 171, 0.24);
  }

  .yoy-control-box,
  .yoy-chart-box,
  .yoy-stat-card {
    border: 1px solid var(--yoy-border);
    box-shadow: 0 14px 28px rgba(6, 13, 24, 0.24);
    border-radius: 14px;
    background: var(--yoy-surface);
    backdrop-filter: blur(6px);
  }

  .box.yoy-control-box {
    padding: 1.5rem 1.75rem;
  }

  .yoy-toolbar {
    gap: 1.75rem;
    align-items: flex-end;
    margin-bottom: 0.75rem;
    padding: 0.25rem 0.5rem;
  }

  .yoy-filter-row {
    display: grid;
    grid-template-columns: repeat(2, minmax(11rem, max-content));
    gap: 1.2rem;
    align-items: end;
  }

  .yoy-filter-row .field {
    display: grid;
    grid-template-rows: 0.95rem 2rem;
    row-gap: 0.35rem;
    align-items: end;
    margin-bottom: 0;
  }

  .yoy-filter-row .control,
  .yoy-filter-row .select {
    display: flex;
    align-items: center;
  }

  .yoy-control-box .field {
    margin-bottom: 0;
  }

  .yoy-control-box .label {
    color: var(--yoy-soft);
    font-weight: 600;
    letter-spacing: 0.02em;
    font-size: 0.72rem;
    line-height: 1;
    white-space: nowrap;
    min-height: 0.95rem;
    margin-bottom: 0 !important;
    display: flex;
    align-items: flex-end;
  }

  .yoy-control-box .select.is-small select {
    height: 2rem;
    min-height: 2rem;
    line-height: 1.1;
  }

  .yoy-filter-row .select.is-small select {
    min-width: 10rem;
  }

  .yoy-filter-row .control,
  .yoy-filter-row .select,
  .yoy-filter-row .select select {
    height: 2rem;
  }

  .yoy-control-box .select select,
  .yoy-chart-box .select select,
  .yoy-chart-box .button {
    background: var(--yoy-input-bg) !important;
    border-color: var(--yoy-border) !important;
    color: var(--yoy-input-text) !important;
  }

  .yoy-download-btn {
    border-color: var(--yoy-border);
    background: var(--yoy-input-bg);
    color: var(--yoy-input-text);
    height: 2rem;
    align-self: flex-end;
  }

  .yoy-download-btn:hover {
    background: rgba(19, 30, 48, 0.96);
  }

  .yoy-hero.yoy-hero {
    display: flex;
    justify-content: space-between;
    gap: 1.5rem;
    align-items: center;
    flex-wrap: wrap;
    border-radius: 14px;
    padding: 1.5rem 1.75rem;
    background:
      radial-gradient(circle at 8% 0%, rgba(255, 255, 255, 0.24), transparent 32%),
      linear-gradient(135deg, var(--yoy-accent-2), var(--yoy-accent));
    border: 1px solid rgba(153, 208, 219, 0.28);
    margin-top: 1.25rem !important;
  }

  .yoy-hero-heading,
  .yoy-hero-title,
  .yoy-hero-sub {
    color: var(--yoy-hero-text) !important;
  }

  .yoy-hero .heading,
  .yoy-hero .title,
  .yoy-hero .is-size-7 {
    color: #f4fbff !important;
  }

  .yoy-hero-chips {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
  }

  .yoy-hero-chips .tag {
    border: 1px solid rgba(255, 255, 255, 0.3);
    background: rgba(8, 19, 31, 0.28);
    color: #f4fbff;
  }

  .yoy-stat-grid {
    margin-bottom: 1.25rem !important;
    margin-top: 0.75rem;
  }

  .yoy-stat-grid > .column {
    padding: 0.6rem;
  }

  .box.yoy-stat-card {
    background: var(--yoy-card-bg);
    min-height: 124px;
    padding: 1.25rem 1.5rem !important;
    text-align: left;
    display: flex;
    flex-direction: column;
    gap: 0.2rem;
    justify-content: flex-start;
  }

  .yoy-stat-top {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: 0.2rem;
    margin-bottom: 0.1rem;
  }

  .yoy-stat-card .heading {
    color: var(--yoy-soft);
    margin-bottom: 0 !important;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .yoy-stat-value {
    color: var(--yoy-ink) !important;
    margin-bottom: 0 !important;
    line-height: 1.15;
    font-size: clamp(1.02rem, 1.45vw, 1.24rem) !important;
    letter-spacing: 0.01em;
    font-weight: 600;
  }

  .yoy-stat-card .is-size-7 {
    color: var(--yoy-soft);
    margin-bottom: 0 !important;
    line-height: 1.3;
  }

  .yoy-stat-card .has-text-link {
    display: block;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }

  .box.yoy-chart-box {
    padding: 1.5rem 1.75rem;
    overflow: hidden;
  }

  .yoy-chart-box .columns {
    margin: 0;
  }

  .yoy-chart-box .columns > .column {
    padding: 0.6rem 0.65rem;
  }

  .yoy-chart-box .heading {
    color: var(--yoy-soft);
    padding-left: 0.05rem;
  }

  .yoy-chart-box .tag.is-light {
    background: var(--yoy-tag-bg);
    color: var(--yoy-tag-text);
  }

  .yoy-chart-box .table {
    background: transparent;
    color: var(--yoy-ink);
  }

  .yoy-chart-box .table-container {
    padding: 0.15rem 0.2rem 0.1rem;
  }

  .yoy-chart-box .table thead th {
    border-color: var(--yoy-border);
    color: var(--yoy-soft);
    padding-left: 0.72rem;
    padding-right: 0.72rem;
  }

  .yoy-chart-box .table td {
    border-color: var(--yoy-border);
    color: var(--yoy-ink);
    padding-top: 0.55rem;
    padding-bottom: 0.55rem;
    padding-left: 0.72rem;
    padding-right: 0.72rem;
  }

  .yoy-chart-box .table.is-hoverable tbody tr:hover {
    background: rgba(45, 64, 92, 0.45);
  }

  .yoy-chart-box .notification.is-light {
    background: var(--yoy-surface) !important;
    color: var(--yoy-ink) !important;
    border: 1px solid var(--yoy-border) !important;
    padding: 0.8rem 0.9rem;
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.04);
  }

  .yoy-monthly-net-list {
    display: grid;
    gap: 0.45rem;
    max-height: 298px;
    overflow: auto;
    padding: 0.1rem 0.35rem 0.1rem 0.1rem;
  }

  .yoy-monthly-net-row {
    padding: 0.42rem 0.55rem;
    border: 1px solid var(--yoy-border);
    border-radius: 8px;
    background-color: var(--yoy-row-bg);
  }

  .yoy-net-track {
    height: 7px;
    border-radius: 999px;
    background: var(--yoy-track-bg);
    overflow: hidden;
  }

  .yoy-net-fill {
    height: 100%;
    border-radius: 999px;
  }

  .yoy-net-fill.is-positive {
    background: linear-gradient(90deg, #2d8671, #51b891);
  }

  .yoy-net-fill.is-negative {
    background: linear-gradient(90deg, #b84a5b, #de7355);
  }

  @media (max-width: 768px) {
    .yoy-section {
      padding-top: 0.5rem;
    }

    .yoy-control-box {
      padding: 1.25rem;
    }

    .yoy-toolbar {
      padding: 0 0.5rem;
    }

    .yoy-hero {
      padding: 1.25rem;
    }

    .yoy-filter-row {
      display: flex;
      flex-wrap: wrap;
    }

    .yoy-toolbar {
      align-items: flex-start !important;
      gap: 0.65rem;
    }

    .yoy-hero {
      padding: 0.8rem;
    }

    .yoy-stat-card {
      min-height: 0;
      padding: 1rem 1.15rem;
      gap: 0.2rem;
    }

    .yoy-stat-top {
      gap: 0.14rem;
      margin-bottom: 0.05rem;
    }

    .yoy-stat-value {
      font-size: 1.06rem !important;
    }
  }

  @media (min-width: 769px) and (max-width: 1023px) {
    .yoy-stat-grid > .column {
      padding-top: 0.25rem;
      padding-bottom: 0.25rem;
    }

    .yoy-stat-card {
      min-height: 106px;
      padding: 0.82rem 0.92rem !important;
      gap: 0.16rem;
    }

    .yoy-stat-top {
      gap: 0.15rem;
      margin-bottom: 0.05rem;
    }

    .yoy-stat-value {
      font-size: 1.05rem !important;
    }

    .yoy-chart-box {
      padding: 0.8rem;
    }

    .yoy-chart-box .columns > .column {
      padding: 0.5rem 0.42rem;
    }

    .yoy-chart-box .table-container {
      padding: 0.1rem 0;
    }

    .yoy-chart-box .table thead th,
    .yoy-chart-box .table td {
      padding-left: 0.5rem;
      padding-right: 0.5rem;
    }
  }
</style>
