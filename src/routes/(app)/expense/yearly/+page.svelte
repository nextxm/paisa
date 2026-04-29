<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import { ajax, formatCurrency, formatPercentage, type Legend, type Posting } from "$lib/utils";
  import {
    renderYearlyExpensesTimeline,
    renderCurrentExpensesBreakdown,
    renderCalendar
  } from "$lib/expense/yearly";
  import { dateMin, dateMax, year } from "../../../../store";
  import { writable } from "svelte/store";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import COLORS from "$lib/colors";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let groups = writable([]);
  let z: d3.ScaleOrdinal<string, string, never> = $state(null),
    renderer: (ps: Posting[]) => void = $state(),
    expenses: Posting[] = $state(null),
    grouped_expenses: Record<string, Posting[]> = $state(null),
    grouped_incomes: Record<string, Posting[]> = $state(null),
    grouped_investments: Record<string, Posting[]> = $state(null),
    grouped_taxes: Record<string, Posting[]> = $state(null);

  let currentYearExpenses: Posting[] = $state([]);

  let legends: Legend[] = $state([]);

  let netValue = $derived(
    sum(grouped_incomes?.[$year] || [], -1) - sum(grouped_taxes?.[$year] || [])
  );
  let grossIncomeValue = $derived(sum(grouped_incomes?.[$year] || [], -1));

  let income = $derived(sumCurrency(grouped_incomes?.[$year] || [], -1));
  let tax = $derived(sumCurrency(grouped_taxes?.[$year] || []));
  let expense = $derived(sumCurrency(grouped_expenses?.[$year] || []));
  let investment = $derived(sumCurrency(grouped_investments?.[$year] || []));

  let netIncome = $derived(
    !_.isEmpty(grouped_incomes?.[$year]) ? formatCurrency(netValue) + " net income" : ""
  );
  let taxRate = $derived(
    !_.isEmpty(grouped_incomes?.[$year])
      ? formatPercentage(sum(grouped_taxes?.[$year] || []) / grossIncomeValue) + " of income"
      : ""
  );
  let expenseRate = $derived(
    !_.isEmpty(grouped_incomes?.[$year])
      ? formatPercentage(sum(grouped_expenses?.[$year] || []) / netValue) + " of net income"
      : ""
  );
  let savingRate = $derived(
    !_.isEmpty(grouped_incomes?.[$year])
      ? formatPercentage(sum(grouped_investments?.[$year] || []) / netValue) + " of net income"
      : ""
  );

  $effect(() => {
    if (grouped_expenses && z && renderer) {
      currentYearExpenses = grouped_expenses[$year] || [];
      renderCalendar(currentYearExpenses, z, $groups);
      renderer(currentYearExpenses);
    }
  });

  onMount(async () => {
    ({
      expenses: expenses,
      year_wise: {
        expenses: grouped_expenses,
        incomes: grouped_incomes,
        investments: grouped_investments,
        taxes: grouped_taxes
      }
    } = await ajax("/api/expense"));

    const [start, end] = d3.extent(_.map(expenses, (e) => e.date));
    if (start) {
      dateMin.set(start);
      dateMax.set(end);
    }

    ({ z, legends } = renderYearlyExpensesTimeline(expenses, groups, year));

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
                  title="Gross Income"
                  value={income}
                  color={COLORS.gainText}
                  subtitle={netIncome}
                />
                <LevelItem title="Tax" value={tax} color={COLORS.lossText} subtitle={taxRate} />
              </nav>
            </div>
          </div>
          <div class="column is-full">
            <div>
              <nav class="level grid-2">
                <LevelItem
                  title="Net Investment"
                  value={investment}
                  color={COLORS.secondary}
                  subtitle={savingRate}
                />

                <LevelItem
                  title="Expenses"
                  value={expense}
                  color={COLORS.lossText}
                  subtitle={expenseRate}
                />
              </nav>
            </div>
          </div>
        </div>
      </div>
      <div class="column is-3">
        <div class="px-3 box">
          <div id="d3-current-year-expense-calendar" class="d3-calendar">
            <div class="months"></div>
          </div>
        </div>
      </div>
      <div class="column is-full-tablet is-half-fullhd">
        <div class="px-3 box" style="height: 100%">
          <ZeroState item={currentYearExpenses}>
            <strong>Hurray!</strong> You have no expenses this year.
          </ZeroState>
          <svg id="d3-current-year-breakdown" width="100%" />
        </div>
      </div>
      <div class="column is-12">
        <div class="box">
          <ZeroState item={expenses}>
            <strong>Oops!</strong> You have no expenses.
          </ZeroState>

          <LegendCard {legends} clazz="ml-4" />
          <svg id="d3-yearly-expense-timeline" width="100%" height="500" />
        </div>
      </div>
    </div>
    <BoxLabel text="Yearly Expenses" />
  </div>
</section>
