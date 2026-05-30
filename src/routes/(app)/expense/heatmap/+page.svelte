<script lang="ts">
  import _ from "lodash";
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import ExpenseHeatmap from "$lib/components/ExpenseHeatmap.svelte";
  import ExpensePatternBarChart from "$lib/components/ExpensePatternBarChart.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import {
    buildSeasonalityPattern,
    buildWeekdayPattern,
    getDailyExpenseAmount
  } from "$lib/expense_heatmap";
  import { ajax, formatCurrency, type DailyExpenseResponse } from "$lib/utils";

  const currentYear = dayjs().year();
  const availableYears = _.range(currentYear, currentYear - 10, -1).map(String);

  let selectedYear = $state(String(currentYear));
  let selectedCategory = $state("");
  let viewMode = $state<"day" | "week">("day");
  let response = $state<DailyExpenseResponse | null>(null);
  let isLoading = $state(false);
  let fetchId = 0;

  async function fetchHeatmap(yearValue: string) {
    const id = ++fetchId;
    isLoading = true;

    const from = dayjs(`${yearValue}-01-01`);
    const to = dayjs(`${yearValue}-12-31`);

    try {
      const data = (await ajax(
        `/api/expense/daily?from=${encodeURIComponent(from.format("YYYY-MM-DD"))}&to=${encodeURIComponent(
          to.format("YYYY-MM-DD")
        )}&group_by=category`
      )) as DailyExpenseResponse;

      if (id !== fetchId) return;
      response = data;

      if (selectedCategory && !data.categories.includes(selectedCategory)) {
        selectedCategory = "";
      }
    } finally {
      if (id === fetchId) isLoading = false;
    }
  }

  onMount(() => {
    void fetchHeatmap(selectedYear);
  });

  let fromDate = $derived(response?.from_date || dayjs(`${selectedYear}-01-01`));
  let toDate = $derived(response?.to_date || dayjs(`${selectedYear}-12-31`));
  let days = $derived(response?.days || []);
  let categories = $derived(response?.categories || []);
  let filteredTotals = $derived(
    days.map((day) => ({ date: day.date, total: getDailyExpenseAmount(day, selectedCategory) }))
  );
  let totalSpend = $derived(_.sumBy(filteredTotals, (entry) => entry.total));
  let activeDays = $derived(filteredTotals.filter((entry) => entry.total > 0).length);
  let peakDay = $derived(
    _.maxBy(
      filteredTotals.filter((entry) => entry.total > 0),
      (entry) => entry.total
    ) || null
  );
  let weekdayPattern = $derived(buildWeekdayPattern(days, fromDate, toDate, selectedCategory));
  let seasonalityPattern = $derived(
    buildSeasonalityPattern(days, fromDate, toDate, selectedCategory)
  );
  let peakWeekday = $derived(_.maxBy(weekdayPattern, (entry) => entry.value) || null);
  let peakMonth = $derived(
    _.maxBy(
      seasonalityPattern.filter((entry) => entry.count > 0),
      (entry) => entry.value
    ) || null
  );
  let calendarDays = $derived(toDate.diff(fromDate, "day") + 1);
</script>

<section class="section expense-heatmap-page">
  <div class="container is-fluid">
    <div class="box mb-4 heatmap-toolbar">
      <div>
        <p class="heading mb-2">Spending Patterns &amp; Seasonality</p>
        <p class="title is-4 mb-1">See when spending spikes across the year</p>
        <p class="is-size-7 has-text-grey mb-0">
          Spot December splurges, weekday habits, and category-specific seasonality at a glance.
        </p>
      </div>

      <div class="heatmap-filters">
        <div class="field">
          <label class="label mb-1" for="expense-heatmap-year">Year</label>
          <div class="control">
            <div class="select is-small">
              <select
                id="expense-heatmap-year"
                bind:value={selectedYear}
                onchange={() => void fetchHeatmap(selectedYear)}
              >
                {#each availableYears as year}
                  <option value={year}>{year}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>

        <div class="field">
          <label class="label mb-1" for="expense-heatmap-category">Category</label>
          <div class="control">
            <div class="select is-small">
              <select id="expense-heatmap-category" bind:value={selectedCategory}>
                <option value="">All categories</option>
                {#each categories as category}
                  <option value={category}>{category}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>

        <div class="field">
          <label class="label mb-1" for="expense-heatmap-view-mode">View</label>
          <div class="control">
            <div class="select is-small">
              <select id="expense-heatmap-view-mode" bind:value={viewMode}>
                <option value="day">Day-wise</option>
                <option value="week">Week-wise</option>
              </select>
            </div>
          </div>
        </div>
      </div>
    </div>

    <div class="columns is-multiline mb-2">
      <div class="column is-3-desktop is-6-tablet">
        <div class="box heatmap-stat-card">
          <p class="heading is-size-7 mb-1">Total spend</p>
          <p class="title is-5 mb-1">{formatCurrency(totalSpend)}</p>
          <p class="is-size-7 has-text-grey">
            {selectedCategory || "All categories"} in {selectedYear}
          </p>
        </div>
      </div>
      <div class="column is-3-desktop is-6-tablet">
        <div class="box heatmap-stat-card">
          <p class="heading is-size-7 mb-1">Active spend days</p>
          <p class="title is-5 mb-1">{activeDays}</p>
          <p class="is-size-7 has-text-grey">
            {formatCurrency(totalSpend / Math.max(calendarDays, 1))} per calendar day
          </p>
        </div>
      </div>
      <div class="column is-3-desktop is-6-tablet">
        <div class="box heatmap-stat-card">
          <p class="heading is-size-7 mb-1">Peak day</p>
          <p class="title is-5 mb-1">{peakDay ? formatCurrency(peakDay.total) : "0"}</p>
          <p class="is-size-7 has-text-grey">
            {peakDay ? peakDay.date.format("MMM D") : "No spending recorded"}
          </p>
        </div>
      </div>
      <div class="column is-3-desktop is-6-tablet">
        <div class="box heatmap-stat-card">
          <p class="heading is-size-7 mb-1">Strongest pattern</p>
          <p class="title is-6 mb-1">{peakMonth?.label || "n/a"} / {peakWeekday?.label || "n/a"}</p>
          <p class="is-size-7 has-text-grey">
            {formatCurrency(peakMonth?.value || 0)} avg month • {formatCurrency(
              peakWeekday?.value || 0
            )} avg weekday
          </p>
        </div>
      </div>
    </div>

    <div class="columns is-multiline">
      <div class="column is-full">
        <div class="box">
          {#if isLoading}
            <p class="has-text-grey is-size-7">Loading heatmap…</p>
          {:else}
            <ExpenseHeatmap
              {days}
              from={fromDate}
              to={toDate}
              category={selectedCategory}
              {viewMode}
            />
          {/if}
        </div>
        <BoxLabel text={viewMode === "day" ? "Daily Spend Intensity" : "Weekly Spend Intensity"} />
      </div>

      <div class="column is-half">
        <div class="box">
          <ExpensePatternBarChart
            id="expense-heatmap-weekday-chart"
            points={weekdayPattern}
            title="Average spend by weekday"
            subtitle="Average spend for each weekday across the selected range"
          />
        </div>
      </div>

      <div class="column is-half">
        <div class="box">
          <ExpensePatternBarChart
            id="expense-heatmap-seasonality-chart"
            points={seasonalityPattern}
            title="Seasonality by month"
            subtitle="Average monthly total for each month-of-year in the selected range"
            color="#f97316"
          />
        </div>
      </div>
    </div>
  </div>
</section>

<style>
  .heatmap-toolbar {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    flex-wrap: wrap;
    align-items: flex-end;
  }

  .heatmap-filters {
    display: flex;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .heatmap-stat-card {
    height: 100%;
  }
</style>
