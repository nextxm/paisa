<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import _ from "lodash";
  import {
    ajax,
    secondName,
    type Posting,
    formatCurrency,
    formatPercentage,
    type Legend
  } from "$lib/utils";
  import {
    renderMonthlyExpensesTimeline,
    renderCurrentExpensesBreakdown,
    renderCalendar
  } from "$lib/expense/monthly";
  import { dateRange, month, setAllowedDateRange } from "../../../../store";
  import { writable } from "svelte/store";
  import PostingCard from "$lib/components/PostingCard.svelte";
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
    <div class="columns is-flex-wrap-wrap">
      <div class="column is-3">
        <div class="columns is-flex-wrap-wrap">
          <div class="column is-full">
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
          </div>
          <div class="column is-full">
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
          <div class="column is-full">
            {#each current_month_expenses as expense}
              <PostingCard posting={expense} color={z(secondName(expense.account))} icon={true} />
            {/each}
          </div>
        </div>
      </div>
      <div class="column is-9">
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
    </div>
  </div>
</section>
