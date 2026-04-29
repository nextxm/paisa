<script lang="ts">
  import _ from "lodash";
  import { renderMonthlyFlow } from "$lib/cash_flow";
  import {
    ajax,
    formatCurrency,
    restName,
    type CashFlow,
    type Legend,
    type Posting
  } from "$lib/utils";
  import { onMount } from "svelte";
  import { dateRange, setAllowedDateRange } from "../../../../store";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import MonthPicker from "$lib/components/MonthPicker.svelte";
  import dayjs from "dayjs";

  let legends: Legend[] = $state([]);
  let cashFlows: CashFlow[] = $state([]);
  let renderer: (cashflows: CashFlow[]) => void = $state();

  let expenseData: {
    month_wise: {
      incomes: Record<string, Posting[]>;
      expenses: Record<string, Posting[]>;
      investments: Record<string, Posting[]>;
      taxes: Record<string, Posting[]>;
      liabilities: Record<string, Posting[]>;
    };
  } = $state({
    month_wise: { incomes: {}, expenses: {}, investments: {}, taxes: {}, liabilities: {} }
  });

  let selectedMonth = $state("");

  const minDate = $derived(_.first(cashFlows)?.date || dayjs());
  const maxDate = $derived(_.last(cashFlows)?.date || dayjs());

  $effect(() => {
    if (!_.isEmpty(cashFlows) && renderer) {
      renderer(
        _.filter(
          cashFlows,
          (c) => c.date.isSameOrBefore($dateRange.to) && c.date.isSameOrAfter($dateRange.from)
        )
      );
    }
  });

  $effect(() => {
    if (!_.isEmpty(cashFlows) && !selectedMonth) {
      selectedMonth = _.last(cashFlows).date.format("YYYY-MM");
    }
  });

  interface AggregatedItem {
    account: string;
    commodity: string;
    amount: number;
  }

  function aggregate(postings: Posting[], negate = false): AggregatedItem[] {
    const result: Record<string, Record<string, number>> = {};
    for (const p of postings) {
      if (!result[p.account]) result[p.account] = {};
      if (!result[p.account][p.commodity]) result[p.account][p.commodity] = 0;
      result[p.account][p.commodity] += negate ? -p.quantity : p.quantity;
    }

    const flattened: AggregatedItem[] = [];
    for (const account in result) {
      for (const commodity in result[account]) {
        flattened.push({ account, commodity, amount: result[account][commodity] });
      }
    }
    return _.sortBy(flattened, "account");
  }

  const aggregatedIncomes = $derived(
    aggregate(expenseData.month_wise.incomes[selectedMonth] || [], true)
  );
  const aggregatedExpenses = $derived(
    aggregate(expenseData.month_wise.expenses[selectedMonth] || [])
  );
  const aggregatedInvestments = $derived(
    aggregate(expenseData.month_wise.investments[selectedMonth] || [])
  );
  const aggregatedTaxes = $derived(aggregate(expenseData.month_wise.taxes[selectedMonth] || []));
  const aggregatedLiabilities = $derived(
    aggregate(expenseData.month_wise.liabilities[selectedMonth] || [], true)
  );

  const netBreakdown = $derived.by(() => {
    const net: Record<string, number> = {};
    aggregatedIncomes.forEach((i) => (net[i.commodity] = (net[i.commodity] || 0) + i.amount));
    aggregatedExpenses.forEach((i) => (net[i.commodity] = (net[i.commodity] || 0) - i.amount));
    aggregatedTaxes.forEach((i) => (net[i.commodity] = (net[i.commodity] || 0) - i.amount));

    return _.sortBy(
      Object.entries(net).map(([commodity, amount]) => ({ commodity, amount })),
      "commodity"
    );
  });

  onMount(async () => {
    ({ cash_flows: cashFlows } = await ajax("/api/cash_flow"));
    setAllowedDateRange(_.map(cashFlows, (c) => c.date));
    ({ renderer, legends } = renderMonthlyFlow("#d3-monthly-cash-flow", {
      rotate: true,
      balance: _.last(cashFlows)?.balance || 0
    }));

    expenseData = (await ajax("/api/expense")) as any;
  });
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="columns flex-wrap">
      <div class="column is-12">
        <div class="box">
          <ZeroState item={cashFlows}>
            <strong>Oops!</strong> You have not made any transactions.
          </ZeroState>

          <LegendCard {legends} clazz="ml-5 mb-2" />
          <svg
            class:is-not-visible={_.isEmpty(cashFlows)}
            id="d3-monthly-cash-flow"
            width="100%"
            height="500"
          />
        </div>
      </div>

      {#if !_.isEmpty(cashFlows)}
        <div class="column is-12">
          <div class="box">
            <div class="level is-mobile mb-5">
              <div class="level-left">
                <h2 class="title is-4 mb-0">Monthly Breakdown</h2>
              </div>
              <div class="level-right">
                <MonthPicker bind:value={selectedMonth} min={minDate} max={maxDate} />
              </div>
            </div>

            <div class="columns is-multiline">
              <div class="column is-6">
                <h3 class="subtitle is-5 has-text-success mb-3">Income</h3>
                {#if aggregatedIncomes.length > 0}
                  <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                    <tbody>
                      {#each aggregatedIncomes as item}
                        <tr>
                          <td>{restName(item.account)}</td>
                          <td class="has-text-right tabular-nums">
                            {formatCurrency(item.amount)}
                            <span class="is-size-7 has-text-grey">{item.commodity}</span>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                {:else}
                  <p class="has-text-grey is-size-7 ml-2">No income this month</p>
                {/if}
              </div>

              <div class="column is-6 border-left-desktop">
                <h3 class="subtitle is-5 has-text-danger mb-3">Expenses</h3>
                {#if aggregatedExpenses.length > 0}
                  <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                    <tbody>
                      {#each aggregatedExpenses as item}
                        <tr>
                          <td>{restName(item.account)}</td>
                          <td class="has-text-right tabular-nums">
                            {formatCurrency(item.amount)}
                            <span class="is-size-7 has-text-grey">{item.commodity}</span>
                          </td>
                        </tr>
                      {/each}
                    </tbody>
                  </table>
                {:else}
                  <p class="has-text-grey is-size-7 ml-2">No expenses this month</p>
                {/if}
              </div>

              {#if aggregatedInvestments.length > 0 || aggregatedTaxes.length > 0 || aggregatedLiabilities.length > 0}
                <div class="column is-12 pb-0">
                  <hr class="my-4" />
                </div>

                <div class="column is-4">
                  <h3 class="subtitle is-5 has-text-info mb-3">Investments</h3>
                  {#if aggregatedInvestments.length > 0}
                    <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                      <tbody>
                        {#each aggregatedInvestments as item}
                          <tr>
                            <td>{restName(item.account)}</td>
                            <td class="has-text-right tabular-nums">
                              {formatCurrency(item.amount)}
                              <span class="is-size-7 has-text-grey">{item.commodity}</span>
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  {:else}
                    <p class="has-text-grey is-size-7 ml-2">No investments this month</p>
                  {/if}
                </div>

                <div class="column is-4 border-left-desktop">
                  <h3 class="subtitle is-5 has-text-warning-dark mb-3">Taxes</h3>
                  {#if aggregatedTaxes.length > 0}
                    <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                      <tbody>
                        {#each aggregatedTaxes as item}
                          <tr>
                            <td>{restName(item.account)}</td>
                            <td class="has-text-right tabular-nums">
                              {formatCurrency(item.amount)}
                              <span class="is-size-7 has-text-grey">{item.commodity}</span>
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  {:else}
                    <p class="has-text-grey is-size-7 ml-2">No taxes this month</p>
                  {/if}
                </div>

                <div class="column is-4 border-left-desktop">
                  <h3 class="subtitle is-5 has-text-grey-dark mb-3">Liabilities</h3>
                  {#if aggregatedLiabilities.length > 0}
                    <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                      <tbody>
                        {#each aggregatedLiabilities as item}
                          <tr>
                            <td>{restName(item.account)}</td>
                            <td class="has-text-right tabular-nums">
                              {formatCurrency(item.amount)}
                              <span class="is-size-7 has-text-grey">{item.commodity}</span>
                            </td>
                          </tr>
                        {/each}
                      </tbody>
                    </table>
                  {:else}
                    <p class="has-text-grey is-size-7 ml-2">No liability changes this month</p>
                  {/if}
                </div>
              {/if}

              <div class="column is-12">
                <div class="notification is-light py-4">
                  <div class="level is-mobile">
                    <div class="level-left">
                      <div>
                        <h3 class="title is-5 mb-0">Net Surplus</h3>
                        <p class="is-size-7 has-text-grey">Income - Taxes - Expenses</p>
                      </div>
                    </div>
                    <div class="level-right has-text-right">
                      <div>
                        {#each netBreakdown as item}
                          <div
                            class="title is-5 mb-1 {item.amount >= 0
                              ? 'has-text-success'
                              : 'has-text-danger'}"
                          >
                            {formatCurrency(item.amount)}
                            <span class="is-size-6">{item.commodity}</span>
                          </div>
                        {/each}
                        {#if netBreakdown.length === 0}
                          <div class="title is-5 mb-0">0.00</div>
                        {/if}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      {/if}
    </div>
  </div>
</section>

<style lang="scss">
  @media screen and (min-width: 1024px) {
    .border-left-desktop {
      border-left: 1px solid #f0f0f0;
    }
  }

  .table.is-borderless {
    td {
      border: none;
    }
  }

  .tabular-nums {
    font-variant-numeric: tabular-nums;
  }
</style>
