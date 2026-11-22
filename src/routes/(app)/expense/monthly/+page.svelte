<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import _ from "lodash";
  import {
    ajax,
    secondName,
    now,
    type Posting,
    type ExpenseTrend,
    formatCurrency,
    formatPercentage,
    type Legend
  } from "$lib/utils";
  import {
    renderMonthlyExpensesTimeline,
    renderCurrentExpensesBreakdown,
    renderCalendar
  } from "$lib/expense/monthly";
  import {
    buildCategorySparklineData,
    computeCalendarMonthTrends,
    previousCalendarMonth
  } from "$lib/expense/trends";
  import { dateRange, month, setAllowedDateRange } from "../../../../store";
  import { expenseTrendMode } from "../../../../persisted_store";
  import { writable } from "svelte/store";
  import PostingCard from "$lib/components/PostingCard.svelte";
  import ExpenseTrendCard from "$lib/components/ExpenseTrendCard.svelte";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import COLORS from "$lib/colors";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import dayjs from "dayjs";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let groups = writable([]);
  let z: d3.ScaleOrdinal<string, string, never> = $state(null),
    renderer: (ps: Posting[]) => void = $state(),
    expenses: Posting[] = $state(null),
    grouped_expenses: Record<string, Posting[]> = $state(null),
    grouped_incomes: Record<string, Posting[]> = $state(null),
    grouped_investments: Record<string, Posting[]> = $state(null),
    grouped_taxes: Record<string, Posting[]> = $state(null),
    trends: ExpenseTrend[] = $state([]),
    destroy: () => void = $state();

  let legends: Legend[] = $state([]);

  let current_month_expenses = $derived(
    _.chain((grouped_expenses && grouped_expenses[$month]) || [])
      .filter((e) => _.includes($groups, secondName(e.account)))
      .sortBy((e) => e.date)
      .reverse()
      .value()
  );
  let income = $derived(sumCurrency(grouped_incomes?.[$month] || [], -1));
  let tax = $derived(sumCurrency(grouped_taxes?.[$month] || []));
  let expense = $derived(sumCurrency(grouped_expenses?.[$month] || []));
  let saving = $derived(sumCurrency(grouped_investments?.[$month] || []));

  let netValue = $derived(
    sum(grouped_incomes?.[$month] || [], -1) - sum(grouped_taxes?.[$month] || [])
  );
  let grossIncomeValue = $derived(sum(grouped_incomes?.[$month] || [], -1));

  let netIncome = $derived(
    !_.isEmpty(grouped_incomes?.[$month]) ? formatCurrency(netValue) + " net income" : ""
  );
  let taxRate = $derived(
    !_.isEmpty(grouped_incomes?.[$month])
      ? formatPercentage(sum(grouped_taxes?.[$month] || []) / grossIncomeValue) + " on income"
      : ""
  );
  let expenseRate = $derived(
    !_.isEmpty(grouped_incomes?.[$month])
      ? formatPercentage(sum(grouped_expenses?.[$month] || []) / netValue) + " of net income"
      : ""
  );
  let savingRate = $derived(
    !_.isEmpty(grouped_incomes?.[$month])
      ? formatPercentage(sum(grouped_investments?.[$month] || []) / netValue) + " of net income"
      : ""
  );

  let prevMonth = $derived(previousCalendarMonth($month));
  let currentMonthLabel = $derived(dayjs(`${$month}-01`).format("MMM YYYY"));
  let prevMonthLabel = $derived(dayjs(`${prevMonth}-01`).format("MMM YYYY"));
  let prevMonthShort = $derived(dayjs(`${prevMonth}-01`).format("MMM"));
  let isCurrentCalendarMonth = $derived($month === now().format("YYYY-MM"));
  let calendarTrends = $derived(computeCalendarMonthTrends(grouped_expenses, $month));
  let activeTrends = $derived($expenseTrendMode === "calendarMonth" ? calendarTrends : trends);
  let hasTrendSection = $derived(trends.length > 0 || !_.isEmpty(grouped_expenses));
  let trendHeading = $derived(
    $expenseTrendMode === "calendarMonth"
      ? `${currentMonthLabel}${isCurrentCalendarMonth ? " (MTD)" : ""} vs. ${prevMonthLabel}`
      : "30-Day Spending Trends (vs. previous 30 days)"
  );
  let trendBoxLabel = $derived(
    $expenseTrendMode === "calendarMonth" ? "Calendar Month Trends" : "Rolling 30-Day Trends"
  );
  let sparkAnchorMonth = $derived($expenseTrendMode === "calendarMonth" ? $month : undefined);

  $effect(() => {
    if (grouped_expenses && z && renderer) {
      renderCalendar($month, grouped_expenses[$month], z, $groups);
      renderer(grouped_expenses[$month] || []);
    }
  });

  onDestroy(async () => {
    if (destroy) {
      destroy();
    }
  });

  onMount(async () => {
    ({
      expenses: expenses,
      trends: trends,
      month_wise: {
        expenses: grouped_expenses,
        incomes: grouped_incomes,
        investments: grouped_investments,
        taxes: grouped_taxes
      }
    } = await ajax("/api/expense"));

    setAllowedDateRange(_.map(expenses, (e) => e.date));
    ({ z, destroy, legends } = renderMonthlyExpensesTimeline(expenses, groups, month, dateRange));
    renderer = renderCurrentExpensesBreakdown(z);
  });

  function sum(postings: Posting[], sign = 1) {
    return sign * _.sumBy(postings, (p) => p.amount);
  }

  function sumCurrency(postings: Posting[], sign = 1) {
    return formatCurrency(sign * _.sumBy(postings, (p) => p.amount));
  }
</script>

<section class="section tab-expense">
  <div class="container is-fluid">
    <div class="monthly-expense-grid">
      <div class="monthly-expense-summary">
        <div class="monthly-expense-summary-stack">
          <div>
            <nav class="level grid-2">
              <LevelItem
                narrow
                title="Gross Income"
                value={income}
                color={COLORS.gainText}
                subtitle={netIncome}
              />
              <LevelItem
                narrow
                title="Tax"
                value={tax}
                subtitle={taxRate}
                color={COLORS.lossText}
              />
            </nav>
          </div>
          <div>
            <nav class="level grid-2">
              <LevelItem
                narrow
                title="Net Investment"
                value={saving}
                subtitle={savingRate}
                color={COLORS.secondary}
              />
              <LevelItem
                narrow
                title="Expenses"
                value={expense}
                color={COLORS.lossText}
                subtitle={expenseRate}
              />
            </nav>
          </div>
        </div>
      </div>

      <div class="monthly-expense-widgets">
        <div class="columns is-flex-wrap-wrap">
          <div class="column is-4">
            <div class="p-3 box">
              <div id="d3-current-month-expense-calendar" class="d3-calendar">
                <div class="weekdays">
                  {#each dayjs.weekdaysShort(true) as day}
                    <div>{day}</div>
                  {/each}
                </div>
                <div class="days"></div>
              </div>
            </div>
          </div>
          <div class="column is-8">
            <div class="px-3 box" style="height: 100%">
              <ZeroState item={grouped_expenses?.[$month]}>
                <strong>Hurray!</strong> You have no expenses this month.
              </ZeroState>
              <svg id="d3-current-month-breakdown" width="100%" />
            </div>
          </div>
          {#if hasTrendSection}
            <div class="column is-full">
              <div class="box">
                <div
                  class="is-flex is-flex-wrap-wrap is-justify-content-space-between is-align-items-center mb-3"
                  style="gap: 0.5rem;"
                >
                  <p class="has-text-weight-semibold has-text-grey mb-0">{trendHeading}</p>
                  <div
                    class="du-tabs du-tabs-boxed du-tabs-sm"
                    role="tablist"
                    aria-label="Spending comparison period"
                  >
                    <button
                      type="button"
                      role="tab"
                      aria-selected={$expenseTrendMode === "rolling30"}
                      class="du-tab {$expenseTrendMode === 'rolling30' ? 'du-tab-active' : ''}"
                      onclick={() => expenseTrendMode.set("rolling30")}
                    >
                      Last 30 Days
                    </button>
                    <button
                      type="button"
                      role="tab"
                      aria-selected={$expenseTrendMode === "calendarMonth"}
                      class="du-tab {$expenseTrendMode === 'calendarMonth' ? 'du-tab-active' : ''}"
                      onclick={() => expenseTrendMode.set("calendarMonth")}
                    >
                      Calendar Month
                    </button>
                  </div>
                </div>
                {#if activeTrends.length > 0}
                  <div class="columns is-multiline is-mobile">
                    {#each activeTrends as trend}
                      {@const sparkData = buildCategorySparklineData(
                        grouped_expenses,
                        trend.category,
                        sparkAnchorMonth
                      )}
                      {@const cardColor = z ? z(trend.category) : COLORS.lossText}
                      <div class="column is-half-mobile is-one-third-tablet is-one-quarter-desktop">
                        <ExpenseTrendCard
                          {trend}
                          color={cardColor}
                          {sparkData}
                          currentLabel={$expenseTrendMode === "calendarMonth"
                            ? currentMonthLabel
                            : "Current 30 days"}
                          previousLabel={$expenseTrendMode === "calendarMonth"
                            ? prevMonthLabel
                            : "Previous 30 days"}
                          previousShortLabel={$expenseTrendMode === "calendarMonth"
                            ? prevMonthShort
                            : "prev"}
                        />
                      </div>
                    {/each}
                  </div>
                {:else}
                  <p class="has-text-grey is-size-7 mb-0">
                    No expenses recorded in {$expenseTrendMode === "calendarMonth"
                      ? `${currentMonthLabel} or ${prevMonthLabel}`
                      : "the last 60 days"}.
                  </p>
                {/if}
              </div>
              <BoxLabel text={trendBoxLabel} />
            </div>
          {/if}
          <div class="column is-full">
            <div class="box">
              <ZeroState item={expenses}>
                <strong>Oops!</strong> You have no expenses.
              </ZeroState>
              <LegendCard {legends} clazz="ml-4 overflow-x-auto" />
              <svg id="d3-monthly-expense-timeline" width="100%" height="400" />
            </div>
          </div>
        </div>
        <BoxLabel text="Monthly Expenses" />
      </div>

      <div class="monthly-expense-list">
        {#if current_month_expenses.length > 0}
          <p class="monthly-expense-list-heading has-text-weight-semibold has-text-grey mb-2">
            Expenses ({current_month_expenses.length})
          </p>
        {/if}
        {#each current_month_expenses as expense}
          <PostingCard
            posting={expense}
            color={z ? z(secondName(expense.account)) : COLORS.lossText}
            icon={true}
          />
        {/each}
      </div>
    </div>
  </div>
</section>

<style lang="scss">
  .monthly-expense-grid {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: 1.25rem;
  }

  .monthly-expense-summary,
  .monthly-expense-widgets,
  .monthly-expense-list {
    min-width: 0;
  }

  .monthly-expense-summary-stack {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  @media screen and (min-width: 769px) {
    .monthly-expense-grid {
      grid-template-columns: minmax(0, 1fr) minmax(0, 3fr);
      grid-template-rows: auto 1fr;
      column-gap: 1.5rem;
      row-gap: 1.5rem;
      align-items: start;
    }

    .monthly-expense-summary {
      grid-column: 1;
      grid-row: 1;
    }

    .monthly-expense-list {
      grid-column: 1;
      grid-row: 2;
    }

    .monthly-expense-widgets {
      grid-column: 2;
      grid-row: 1 / span 2;
    }

    .monthly-expense-list-heading {
      display: none;
    }
  }
</style>
