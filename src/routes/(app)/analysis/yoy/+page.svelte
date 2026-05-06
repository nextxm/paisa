<script lang="ts">
  import _ from "lodash";
  import Papa from "papaparse";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import YoYChart from "$lib/components/YoYChart.svelte";
  import { ajax, formatCurrency, formatPercentage, type Posting, type YoYSeries } from "$lib/utils";
  import {
    buildCategoryYoYSeries,
    buildYoYExportRows,
    calculateYoYInsights,
    orderedYears,
    type YoYChartType
  } from "$lib/yoy_utils";

  let yearCount = $state(2);
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

  async function refresh() {
    const [expenseData, incomeData] = await Promise.all([
      ajax(`/api/expense?years=${yearCount}`),
      ajax(`/api/income?years=${yearCount}`)
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

  $effect(() => {
    void refresh();
  });
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="box">
      <div
        class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-2"
      >
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

        <button class="button is-small is-link is-light" onclick={downloadCSV}>
          <span class="icon is-small"><i class="fas fa-download"></i></span>
          <span>Download CSV</span>
        </button>
      </div>
    </div>

    <div class="columns is-multiline">
      <div class="column is-full">
        <div class="box">
          <YoYChart id="d3-yoy-expense" series={expenseSeries} chartType="line" />
        </div>
        <BoxLabel text="Year-over-Year Spending" />
      </div>

      <div class="column is-full">
        <div class="box">
          <YoYChart id="d3-yoy-income" series={incomeSeries} chartType="line" />
        </div>
        <BoxLabel text="Year-over-Year Income" />
      </div>

      <div class="column is-full">
        <div class="box">
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
        <div class="box">
          <p class="mb-2"><strong>{trendLabel(insights.spendingChangePct, "Spending")}</strong></p>
          <p class="mb-2"><strong>{trendLabel(insights.incomeChangePct, "Income")}</strong></p>
          {#if insights.topExpenseCategory}
            <p>
              <strong>
                Top expense category: {insights.topExpenseCategory.name}
                ({formatCurrency(insights.topExpenseCategory.latestYearTotal)}
                {insights.topExpenseCategory.changePct == null
                  ? ", n/a"
                  : `, ${insights.topExpenseCategory.changePct >= 0 ? "+" : ""}${formatPercentage(insights.topExpenseCategory.changePct / 100)}`})
              </strong>
            </p>
          {/if}
        </div>
        <BoxLabel text="YoY Insights" />
      </div>
    </div>
  </div>
</section>
