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
                        <div class="summary-content">
                          <p class="is-size-7 summary-label is-uppercase">Income</p>
                          <div class="mt-1 summary-values">
                            {#each summary.incomes as item}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value"
                                  >{formatCurrency(item.amount)}</span
                                >
                                <span class="summary-amount-commodity">{item.commodity}</span>
                              </div>
                            {/each}
                            {#if summary.incomes.length === 0}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value">0.00</span>
                              </div>
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
                        <div class="summary-content">
                          <p class="is-size-7 summary-label is-uppercase">Expenses</p>
                          <div class="mt-1 summary-values">
                            {#each summary.expenses as item}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value"
                                  >{formatCurrency(item.amount)}</span
                                >
                                <span class="summary-amount-commodity">{item.commodity}</span>
                              </div>
                            {/each}
                            {#if summary.expenses.length === 0}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value">0.00</span>
                              </div>
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
                        <div class="summary-content">
                          <p class="is-size-7 summary-label is-uppercase">Taxes</p>
                          <div class="mt-1 summary-values">
                            {#each summary.taxes as item}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value"
                                  >{formatCurrency(item.amount)}</span
                                >
                                <span class="summary-amount-commodity">{item.commodity}</span>
                              </div>
                            {/each}
                            {#if summary.taxes.length === 0}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value">0.00</span>
                              </div>
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
                        <div class="summary-content">
                          <p class="is-size-7 summary-label is-uppercase">Net Flow</p>
                          <div class="mt-1 summary-values">
                            {#each summary.savings as item}
                              <div class="summary-amount-row">
                                <span
                                  class="summary-amount-value"
                                  class:is-deficit={item.amount < 0}
                                  >{formatCurrency(item.amount)}</span
                                >
                                <span class="summary-amount-commodity">{item.commodity}</span>
                              </div>
                            {/each}
                            {#if summary.savings.length === 0}
                              <div class="summary-amount-row">
                                <span class="summary-amount-value">0.00</span>
                              </div>
                            {/if}
                          </div>
                        </div>
                        <span class="icon is-medium">
                          <i class="fas fa-right-left fa-lg"></i>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
              </div>

              <div class="column is-6">
                <div class="card-section flow-column">
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
                      <table
                        class="table is-fullwidth is-narrow is-hoverable is-borderless cashflow-breakdown-table"
                      >
                        <tbody>
                          {#each aggregatedIncomes as item}
                            <tr>
                              <td class="cashflow-row-label">{restName(item.account)}</td>
                              <td class="cashflow-row-value has-text-right tabular-nums">
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
                      <table
                        class="table is-fullwidth is-narrow is-hoverable is-borderless cashflow-breakdown-table"
                      >
                        <tbody>
                          {#each aggregatedLiabilities as item}
                            <tr>
                              <td class="cashflow-row-label">{restName(item.account)}</td>
                              <td class="cashflow-row-value has-text-right tabular-nums">
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
                <div class="card-section flow-column border-left-desktop">
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
                      <table
                        class="table is-fullwidth is-narrow is-hoverable is-borderless cashflow-breakdown-table"
                      >
                        <tbody>
                          {#each aggregatedExpenses as item}
                            <tr>
                              <td class="cashflow-row-label">{restName(item.account)}</td>
                              <td class="cashflow-row-value has-text-right tabular-nums">
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
                      <table
                        class="table is-fullwidth is-narrow is-hoverable is-borderless cashflow-breakdown-table"
                      >
                        <tbody>
                          {#each aggregatedTaxes as item}
                            <tr>
                              <td class="cashflow-row-label">{restName(item.account)}</td>
                              <td class="cashflow-row-value has-text-right tabular-nums">
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

              <div class="column is-12 mt-4">
                <div class="net-surplus-footer">
                  <div class="level is-mobile">
                    <div class="level-left">
                      <div>
                        <h3 class="title is-5 mb-0">Monthly Net Flow</h3>
                        <p class="is-size-7 has-text-grey">
                          Total income remaining after all expenses and taxes
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
      margin-left: 1rem;
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

    .summary-amount-value,
    .summary-amount-commodity {
      color: var(--bulma-text, #363636);
    }

    .icon {
      color: var(--bulma-text-grey-light, #b5b5b5);
    }

    &.is-success-card {
      background-color: var(--paisa-success-bg, rgba(72, 199, 142, 0.1));
      border-color: var(--bulma-success, #48c78e);
      .summary-amount-value,
      .summary-amount-commodity,
      .summary-label,
      .icon {
        color: var(--bulma-success-dark, #257953);
      }
    }

    &.is-danger-card {
      background-color: var(--paisa-danger-bg, rgba(241, 70, 104, 0.1));
      border-color: var(--bulma-danger, #f14668);
      .summary-amount-value,
      .summary-amount-commodity,
      .summary-label,
      .icon {
        color: var(--bulma-danger-dark, #cc0f35);
      }
    }

    &.is-warning-card {
      background-color: var(--paisa-warning-bg, rgba(255, 221, 87, 0.1));
      border-color: var(--bulma-warning, #ffdd57);
      .summary-amount-value,
      .summary-amount-commodity,
      .summary-label,
      .icon {
        color: var(--bulma-warning-dark, #947600);
      }
    }

    &.is-info-card {
      background-color: var(--paisa-info-bg, rgba(62, 142, 208, 0.1));
      border-color: var(--bulma-info, #3e8ed0);
      .summary-amount-value,
      .summary-amount-commodity,
      .summary-label,
      .icon {
        color: var(--bulma-info-dark, #205d8a);
      }
      .summary-amount-value.is-deficit {
        color: var(--bulma-danger-dark, #cc0f35);
      }
    }

    &:hover {
      transform: translateY(-2px);
    }
  }

  .summary-values {
    display: grid;
    gap: 0.15rem;
    min-height: 5.75rem;
  }

  .summary-content {
    flex: 1;
    min-width: 0;
    padding-right: 0.35rem;
  }

  .summary-amount-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    align-items: baseline;
    column-gap: 0.45rem;
    line-height: 1.15;
  }

  .summary-amount-value {
    text-align: right;
    font-size: 1.95rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
    letter-spacing: 0.01em;
  }

  .summary-amount-commodity {
    font-size: 1rem;
    font-weight: 600;
    min-width: 3ch;
    text-transform: uppercase;
    opacity: 0.9;
  }

  .card-section {
    padding-top: 1rem;
    padding-bottom: 1rem;
  }

  .flow-column {
    .subtitle {
      letter-spacing: 0.01em;
    }
  }

  .net-surplus-footer {
    background-color: var(--bulma-scheme-main-bis, #f8fafc);
    padding: 1.5rem;
    border-radius: 8px;
    border: 1px solid var(--bulma-border, #dbdbdb);

    .title {
      color: var(--bulma-text, #363636);
    }
  }

  .table.is-borderless {
    background-color: transparent;
    td {
      border: none !important;
      padding-top: 0.4rem;
      padding-bottom: 0.4rem;
      color: var(--bulma-text, inherit);
    }
    tr {
      border: none !important;
    }
  }

  .cashflow-breakdown-table {
    table-layout: fixed;

    tbody tr {
      border-radius: 6px;
      transition: background-color 0.15s ease;
    }

    tbody tr:hover {
      background-color: var(--bulma-scheme-main-ter, rgba(0, 0, 0, 0.03));
    }
  }

  .cashflow-row-label {
    width: 66%;
    padding-right: 1.25rem;
    word-break: break-word;
  }

  .cashflow-row-value {
    width: 34%;
    white-space: nowrap;

    .is-size-7 {
      margin-left: 0.35rem;
    }
  }

  :global(html[data-theme="dark"]) {
    .summary-card {
      background-color: rgba(255, 255, 255, 0.02);
      border-color: rgba(255, 255, 255, 0.1);

      .summary-label {
        color: var(--bulma-text-grey, #7a7a7a);
      }
      .icon {
        color: var(--bulma-text-grey, #7a7a7a);
      }

      &.is-success-card {
        background-color: rgba(72, 199, 142, 0.08);
        border-color: rgba(72, 199, 142, 0.3);
        .summary-amount-value,
        .summary-amount-commodity,
        .summary-label,
        .icon {
          color: #82e0aa;
        }
      }
      &.is-danger-card {
        background-color: rgba(241, 70, 104, 0.08);
        border-color: rgba(241, 70, 104, 0.3);
        .summary-amount-value,
        .summary-amount-commodity,
        .summary-label,
        .icon {
          color: #f5b7b1;
        }
      }
      &.is-warning-card {
        background-color: rgba(255, 221, 87, 0.08);
        border-color: rgba(255, 221, 87, 0.3);
        .summary-amount-value,
        .summary-amount-commodity,
        .summary-label,
        .icon {
          color: #f9e79f;
        }
      }
      &.is-info-card {
        background-color: rgba(62, 142, 208, 0.08);
        border-color: rgba(62, 142, 208, 0.3);
        .summary-amount-value,
        .summary-amount-commodity,
        .summary-label,
        .icon {
          color: #aed6f1;
        }
        .summary-amount-value.is-deficit {
          color: #f5b7b1;
        }
      }
    }

    .net-surplus-footer {
      background-color: rgba(255, 255, 255, 0.03);
      border-color: rgba(255, 255, 255, 0.1);

      .title.has-text-success-dark {
        color: #82e0aa !important;
      }
      .title.has-text-danger-dark {
        color: #f5b7b1 !important;
      }
    }

    .table.is-borderless tr {
      border: none !important;
    }
  }

  .tabular-nums {
    font-variant-numeric: tabular-nums;
  }

  @media screen and (max-width: 768px) {
    .summary-values {
      min-height: auto;
    }

    .summary-amount-value {
      font-size: 1.55rem;
    }

    .summary-amount-commodity {
      font-size: 0.95rem;
    }
  }
</style>
