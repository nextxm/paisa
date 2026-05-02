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

  const summary = $derived.by(() => {
    const incomes: Record<string, number> = {};
    const expenses: Record<string, number> = {};
    const taxes: Record<string, number> = {};
    const savings: Record<string, number> = {};

    aggregatedIncomes.forEach((i) => {
      incomes[i.commodity] = (incomes[i.commodity] || 0) + i.amount;
      savings[i.commodity] = (savings[i.commodity] || 0) + i.amount;
    });

    aggregatedExpenses.forEach((i) => {
      expenses[i.commodity] = (expenses[i.commodity] || 0) + i.amount;
      savings[i.commodity] = (savings[i.commodity] || 0) - i.amount;
    });

    aggregatedTaxes.forEach((i) => {
      taxes[i.commodity] = (taxes[i.commodity] || 0) + i.amount;
      savings[i.commodity] = (savings[i.commodity] || 0) - i.amount;
    });

    const toList = (record: Record<string, number>) =>
      _.sortBy(
        Object.entries(record).map(([commodity, amount]) => ({ commodity, amount })),
        "commodity"
      );

    return {
      incomes: toList(incomes),
      expenses: toList(expenses),
      taxes: toList(taxes),
      savings: toList(savings)
    };
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
            <div class="is-flex is-justify-content-space-between is-align-items-center mb-6 px-2">
              <h2 class="title is-4 mb-0">Monthly Breakdown</h2>
              <MonthPicker bind:value={selectedMonth} min={minDate} max={maxDate} />
            </div>

            <div class="columns is-multiline">
              <div class="column is-12 mb-4">
                <div class="columns is-mobile is-multiline">
                  <div class="column is-3-desktop is-6-mobile">
                    <div class="summary-card is-success-card">
                      <div class="is-flex is-justify-content-space-between is-align-items-start">
                        <div>
                          <p class="is-size-7 summary-label is-uppercase">Income</p>
                          <div class="mt-1">
                            {#each summary.incomes as item}
                              <p class="title is-4 mb-1">
                                {formatCurrency(item.amount)}
                                <span class="is-size-6">{item.commodity}</span>
                              </p>
                            {/each}
                            {#if summary.incomes.length === 0}
                              <p class="title is-4">0.00</p>
                            {/if}
                          </div>
                        </div>
                        <span class="icon is-medium">
                          <i class="fas fa-arrow-trend-up fa-lg"></i>
                        </span>
                      </div>
                    </div>
                  </div>
                  <div class="column is-3-desktop is-6-mobile">
                    <div class="summary-card is-danger-card">
                      <div class="is-flex is-justify-content-space-between is-align-items-start">
                        <div>
                          <p class="is-size-7 summary-label is-uppercase">Expenses</p>
                          <div class="mt-1">
                            {#each summary.expenses as item}
                              <p class="title is-4 mb-1">
                                {formatCurrency(item.amount)}
                                <span class="is-size-6">{item.commodity}</span>
                              </p>
                            {/each}
                            {#if summary.expenses.length === 0}
                              <p class="title is-4">0.00</p>
                            {/if}
                          </div>
                        </div>
                        <span class="icon is-medium">
                          <i class="fas fa-arrow-trend-down fa-lg"></i>
                        </span>
                      </div>
                    </div>
                  </div>
                  <div class="column is-3-desktop is-6-mobile">
                    <div class="summary-card is-warning-card">
                      <div class="is-flex is-justify-content-space-between is-align-items-start">
                        <div>
                          <p class="is-size-7 summary-label is-uppercase">Taxes</p>
                          <div class="mt-1">
                            {#each summary.taxes as item}
                              <p class="title is-4 mb-1">
                                {formatCurrency(item.amount)}
                                <span class="is-size-6">{item.commodity}</span>
                              </p>
                            {/each}
                            {#if summary.taxes.length === 0}
                              <p class="title is-4">0.00</p>
                            {/if}
                          </div>
                        </div>
                        <span class="icon is-medium">
                          <i class="fas fa-receipt fa-lg"></i>
                        </span>
                      </div>
                    </div>
                  </div>
                  <div class="column is-3-desktop is-6-mobile">
                    <div class="summary-card is-info-card">
                      <div class="is-flex is-justify-content-space-between is-align-items-start">
                        <div>
                          <p class="is-size-7 summary-label is-uppercase">Surplus</p>
                          <div class="mt-1">
                            {#each summary.savings as item}
                              <p class="title is-4 mb-1 {item.amount < 0 && 'is-deficit'}">
                                {formatCurrency(item.amount)}
                                <span class="is-size-6">{item.commodity}</span>
                              </p>
                            {/each}
                            {#if summary.savings.length === 0}
                              <p class="title is-4">0.00</p>
                            {/if}
                          </div>
                        </div>
                        <span class="icon is-medium">
                          <i class="fas fa-piggy-bank fa-lg"></i>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="column is-6">
                <div class="card-section">
                  <div class="level is-mobile mb-4">
                    <div class="level-left">
                      <h3 class="subtitle is-5 has-text-weight-bold mb-0">Inflows</h3>
                    </div>
                  </div>

                  <div class="mb-5">
                    <h4 class="is-size-7 has-text-grey has-text-weight-bold mb-2 is-uppercase">
                      Income
                    </h4>
                    {#if aggregatedIncomes.length > 0}
                      <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                        <tbody>
                          {#each aggregatedIncomes as item}
                            <tr>
                              <td class="pl-0">{restName(item.account)}</td>
                              <td class="has-text-right tabular-nums pr-0">
                                <span class="has-text-weight-medium">
                                  {formatCurrency(item.amount)}
                                </span>
                                <span class="is-size-7 has-text-grey">{item.commodity}</span>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    {:else}
                      <p class="has-text-grey is-size-7">No income this month</p>
                    {/if}
                  </div>

                  {#if aggregatedLiabilities.length > 0}
                    <div class="mb-5">
                      <h4 class="is-size-7 has-text-grey has-text-weight-bold mb-2 is-uppercase">
                        Liabilities Inflow
                      </h4>
                      <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                        <tbody>
                          {#each aggregatedLiabilities as item}
                            <tr>
                              <td class="pl-0">{restName(item.account)}</td>
                              <td class="has-text-right tabular-nums pr-0">
                                <span class="has-text-weight-medium">
                                  {formatCurrency(item.amount)}
                                </span>
                                <span class="is-size-7 has-text-grey">{item.commodity}</span>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}
                </div>
              </div>

              <div class="column is-6">
                <div class="card-section border-left-desktop">
                  <div class="level is-mobile mb-4">
                    <div class="level-left">
                      <h3 class="subtitle is-5 has-text-weight-bold mb-0">Outflows</h3>
                    </div>
                  </div>

                  <div class="mb-5">
                    <h4 class="is-size-7 has-text-grey has-text-weight-bold mb-2 is-uppercase">
                      Expenses
                    </h4>
                    {#if aggregatedExpenses.length > 0}
                      <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                        <tbody>
                          {#each aggregatedExpenses as item}
                            <tr>
                              <td class="pl-0">{restName(item.account)}</td>
                              <td class="has-text-right tabular-nums pr-0">
                                <span class="has-text-weight-medium">
                                  {formatCurrency(item.amount)}
                                </span>
                                <span class="is-size-7 has-text-grey">{item.commodity}</span>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    {:else}
                      <p class="has-text-grey is-size-7">No expenses this month</p>
                    {/if}
                  </div>

                  {#if aggregatedTaxes.length > 0}
                    <div class="mb-5">
                      <h4 class="is-size-7 has-text-grey has-text-weight-bold mb-2 is-uppercase">
                        Taxes
                      </h4>
                      <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                        <tbody>
                          {#each aggregatedTaxes as item}
                            <tr>
                              <td class="pl-0">{restName(item.account)}</td>
                              <td class="has-text-right tabular-nums pr-0">
                                <span class="has-text-weight-medium">
                                  {formatCurrency(item.amount)}
                                </span>
                                <span class="is-size-7 has-text-grey">{item.commodity}</span>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if}

                  <!-- {#if aggregatedInvestments.length > 0}
                    <div class="mb-5">
                      <div class="is-flex is-align-items-center mb-2">
                        <h4 class="is-size-7 has-text-grey has-text-weight-bold is-uppercase">
                          Investments
                        </h4>
                        <span
                          class="icon is-small has-text-grey-light ml-2"
                          data-tippy-content="Asset transfers (e.g. Stocks, Savings) excluding Checking accounts"
                        >
                          <i class="fas fa-circle-info"></i>
                        </span>
                      </div>
                      <table class="table is-fullwidth is-narrow is-hoverable is-borderless">
                        <tbody>
                          {#each aggregatedInvestments as item}
                            <tr>
                              <td class="pl-0">{restName(item.account)}</td>
                              <td class="has-text-right tabular-nums pr-0">
                                <span class="has-text-weight-medium">
                                  {formatCurrency(item.amount)}
                                </span>
                                <span class="is-size-7 has-text-grey">{item.commodity}</span>
                              </td>
                            </tr>
                          {/each}
                        </tbody>
                      </table>
                    </div>
                  {/if} -->
                </div>
              </div>

              <div class="column is-12 mt-4">
                <div class="net-surplus-footer">
                  <div class="level is-mobile">
                    <div class="level-left">
                      <div>
                        <h3 class="title is-5 mb-0">Monthly Surplus</h3>
                        <p class="is-size-7 has-text-grey">
                          Overall cash flow remaining after expenses and taxes
                        </p>
                      </div>
                    </div>

                    <div class="level-right has-text-right">
                      <div>
                        {#each netBreakdown as item}
                          <div
                            class="title is-4 mb-1 {item.amount >= 0
                              ? 'has-text-success-dark'
                              : 'has-text-danger-dark'}"
                          >
                            {formatCurrency(item.amount)}
                            <span class="is-size-6">{item.commodity}</span>
                          </div>
                        {/each}
                        {#if netBreakdown.length === 0}
                          <div class="title is-4 mb-0">0.00</div>
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
      border-left: 1px solid var(--bulma-border, #dbdbdb);
      padding-left: 2rem;
    }
  }

  .summary-label {
    font-weight: 700;
    color: var(--paisa-summary-label-color, #666);
  }

  .summary-card {
    padding: 1.25rem;
    border-radius: 8px;
    height: 100%;
    transition: transform 0.2s;
    background-color: var(--bulma-scheme-main-ter, #f5f5f5);
    border: 1px solid var(--bulma-border, #dbdbdb);

    .title {
      color: var(--bulma-text, #363636);
    }

    .icon {
      color: var(--bulma-text-grey-light, #b5b5b5);
    }

    &.is-success-card {
      background-color: var(--paisa-success-bg, rgba(72, 199, 142, 0.1));
      border-color: var(--bulma-success, #48c78e);
      .title,
      .summary-label,
      .icon {
        color: var(--bulma-success-dark, #257953);
      }
    }

    &.is-danger-card {
      background-color: var(--paisa-danger-bg, rgba(241, 70, 104, 0.1));
      border-color: var(--bulma-danger, #f14668);
      .title,
      .summary-label,
      .icon {
        color: var(--bulma-danger-dark, #cc0f35);
      }
    }

    &.is-warning-card {
      background-color: var(--paisa-warning-bg, rgba(255, 221, 87, 0.1));
      border-color: var(--bulma-warning, #ffdd57);
      .title,
      .summary-label,
      .icon {
        color: var(--bulma-warning-dark, #947600);
      }
    }

    &.is-info-card {
      background-color: var(--paisa-info-bg, rgba(62, 142, 208, 0.1));
      border-color: var(--bulma-info, #3e8ed0);
      .title,
      .summary-label,
      .icon {
        color: var(--bulma-info-dark, #205d8a);
      }
      .title.is-deficit {
        color: var(--bulma-danger-dark, #cc0f35);
      }
    }

    &:hover {
      transform: translateY(-2px);
    }
  }

  .card-section {
    padding: 1rem 0;
  }

  .net-surplus-footer {
    background-color: var(--bulma-scheme-main-bis, #fafafa);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--bulma-border, #dbdbdb);
  }

  .table.is-borderless {
    background-color: transparent;
    td {
      border: none;
      padding-top: 0.5rem;
      padding-bottom: 0.5rem;
    }
    tr {
      border-bottom: 1px solid var(--bulma-border-light, #f0f0f0);
      &:last-child {
        border-bottom: none;
      }
    }
  }

  :global(html[data-theme="dark"]) {
    .summary-card {
      &.is-success-card {
        background-color: rgba(72, 199, 142, 0.05);
      }
      &.is-danger-card {
        background-color: rgba(241, 70, 104, 0.05);
      }
      &.is-warning-card {
        background-color: rgba(255, 221, 87, 0.05);
      }
      &.is-info-card {
        background-color: rgba(62, 142, 208, 0.05);
      }
    }
  }

  .tabular-nums {
    font-variant-numeric: tabular-nums;
  }
</style>
