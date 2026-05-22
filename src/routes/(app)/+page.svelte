<script lang="ts">
  import * as cashFlow from "$lib/cash_flow";
  import COLORS from "$lib/colors";
  import LastNMonths from "$lib/components/LastNMonths.svelte";
  import Modal from "$lib/components/Modal.svelte";
  import { dndzone } from "svelte-dnd-action";
  import { flip } from "svelte/animate";
  import * as expense from "$lib/expense/monthly";
  import {
    getWidgetMeta,
    loadDashboardLayout,
    saveDashboardLayout,
    type DashboardLayout,
    type DashboardWidgetColumn,
    type DashboardWidgetConfigOption,
    type DashboardWidgetId,
    type DashboardWidgetSetting
  } from "$lib/dashboard_layout";
  import { enrichTrantionSequence, sortTrantionSequence } from "$lib/transaction_sequence";
  import {
    ajax,
    formatCurrency,
    formatFloat,
    type Budget,
    type CashFlow,
    type Networth,
    type Posting,
    type Transaction,
    type TransactionSequence,
    type Legend,
    now,
    type GoalSummary,
    type AssetBreakdown
  } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import type { PageData } from "./$types";

  import BudgetCard from "$lib/components/BudgetCard.svelte";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import { refresh } from "../../store";
  import UpcomingCard from "$lib/components/UpcomingCard.svelte";
  import GoalSummaryCard from "$lib/components/GoalSummaryCard.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import BalanceCard from "$lib/components/BalanceCard.svelte";
  import RecentTransactionsWidget from "$lib/components/RecentTransactionsWidget.svelte";

  let { data }: { data: PageData } = $props();

  let cashflowLegends: Legend[] = $state([]);
  let month = $state(now().format("YYYY-MM"));
  let showWidgetSettings = $state(false);
  let dashboardLayout: DashboardLayout = $state(loadDashboardLayout());
  let goalSummaries: GoalSummary[] = $derived(
    _.sortBy(data.dashboard.goalSummaries, (g) => -g.priority)
  );
  let transactionSequences: TransactionSequence[] = $derived(
    sortTrantionSequence(enrichTrantionSequence(data.dashboard.transactionSequences))
  );
  let cashFlows: CashFlow[] = $derived(data.dashboard.cashFlows);
  let expenses: { [key: string]: Posting[] } = $derived(data.dashboard.expenses);
  let xirr = $derived(data.dashboard.networth.xirr);
  let networth: Networth = $derived(data.dashboard.networth.networth);
  let renderer: ((data: Posting[]) => void) | undefined = $state();
  let selectedExpenses = $derived(expenses[month] || []);
  let totalExpense = $derived(_.sumBy(selectedExpenses, (p) => p.amount));
  let transactions: Transaction[] = $derived(data.dashboard.transactions);
  let budgetsByMonth: Record<string, Budget> = $derived(data.dashboard.budget.budgetsByMonth);
  let currentBudget = $derived(budgetsByMonth[month]);
  let isEmpty = $derived(_.isEmpty(data.dashboard.transactions));
  let checkingBalances: Record<string, AssetBreakdown> = $derived(
    data.dashboard.checkingBalances.asset_breakdowns
  );
  let investmentIncomeDividendTTM = $derived(data.income.ttm_dividend || 0);
  let investmentIncomeInterestTTM = $derived(data.income.ttm_interest || 0);
  let investmentIncomeLoading = $state(false);
  const widgetColumns: DashboardWidgetColumn[] = ["left", "right"];
  const leftWidgets = $derived(dashboardLayout.left);
  const rightWidgets = $derived(dashboardLayout.right);

  const goalSummariesForWidget = $derived(
    _.take(goalSummaries, getWidgetConfig("goals", "maxItems", goalSummaries.length))
  );
  const transactionSequencesForWidget = $derived(
    _.take(transactionSequences, getWidgetConfig("recurring", "maxItems", 16))
  );
  const recentTransactionsForWidget = $derived(
    _.take(transactions, getWidgetConfig("recentTransactions", "maxItems", 10))
  );
  const checkingBalancesForWidget = $derived(
    _.take(
      _.values(checkingBalances),
      getWidgetConfig("checkingBalances", "maxItems", _.values(checkingBalances).length)
    )
  );
  const budgetAccountsForWidget = $derived(
    currentBudget
      ? _.take(
          currentBudget.accounts,
          getWidgetConfig("budget", "maxAccounts", currentBudget.accounts.length)
        )
      : []
  );

  $effect(() => {
    if (renderer) {
      renderer(selectedExpenses);
    }
  });

  async function initDemo() {
    await ajax("/api/init", { method: "POST" });
    refresh();
  }

  function updateDashboardLayout(nextLayout: DashboardLayout) {
    dashboardLayout = nextLayout;
    saveDashboardLayout(nextLayout);
  }

  function updateColumn(column: DashboardWidgetColumn, items: DashboardWidgetSetting[]) {
    updateDashboardLayout({
      ...dashboardLayout,
      [column]: items
    });
  }

  function handleColumnConsider(
    column: DashboardWidgetColumn,
    event: CustomEvent<{ items: any[] }>
  ) {
    updateColumn(column, event.detail.items as DashboardWidgetSetting[]);
  }

  function handleColumnFinalize(
    column: DashboardWidgetColumn,
    event: CustomEvent<{ items: any[] }>
  ) {
    updateColumn(column, event.detail.items as DashboardWidgetSetting[]);
  }

  function setWidgetVisible(
    column: DashboardWidgetColumn,
    id: DashboardWidgetId,
    visible: boolean
  ) {
    updateColumn(
      column,
      dashboardLayout[column].map((widget) => (widget.id === id ? { ...widget, visible } : widget))
    );
  }

  function setWidgetConfig(
    column: DashboardWidgetColumn,
    id: DashboardWidgetId,
    option: DashboardWidgetConfigOption,
    value: string
  ) {
    const parsed = Number(value);
    if (Number.isNaN(parsed)) {
      return;
    }
    const boundedValue = Math.max(option.min, Math.min(option.max, parsed));
    updateColumn(
      column,
      dashboardLayout[column].map((widget) =>
        widget.id === id
          ? {
              ...widget,
              config: {
                ...widget.config,
                [option.key]: boundedValue
              }
            }
          : widget
      )
    );
  }

  function getWidgetConfig(id: DashboardWidgetId, key: string, fallback: number): number {
    const layoutEntry = [...dashboardLayout.left, ...dashboardLayout.right].find(
      (widget) => widget.id === id
    );
    if (!layoutEntry) {
      return fallback;
    }
    const value = Number(layoutEntry.config[key]);
    return Number.isNaN(value) ? fallback : value;
  }

  onMount(() => {
    const postings = _.chain(expenses).values().flatten().value();
    const z = expense.colorScale(postings);
    renderer = expense.renderCurrentExpensesBreakdown(z);

    const { renderer: cashflowRenderer, legends } = cashFlow.renderMonthlyFlow(
      "#d3-current-cash-flow",
      {
        rotate: false,
        balance: _.last(cashFlows)?.balance || 0
      }
    );
    cashflowRenderer(cashFlows);
    cashflowLegends = legends;
  });
</script>

<section class="section" class:is-hidden={!isEmpty}>
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <ZeroState item={!isEmpty}>
          <div class="has-text-left" style="max-width: 640px;">
            <p class="mb-2">
              Looks like you are new here, you can either get started or look at a demo setup
            </p>
            <div>
              <p class="is-size-4">I want to get started</p>
              <ol class="ml-5 mt-2 mb-4">
                <li>
                  Go to <a href="/more/config">configuration</a> page and set your default currency and
                  locale.
                </li>
                <li>
                  Go to <a href="/ledger/editor">editor</a> page and start adding transactions to your
                  journal.
                </li>
              </ol>
              <p class="is-size-4">I want to view a Demo</p>
              <p class="ml-3"></p>
              <ol class="ml-5 mt-2 mb-4">
                <li>
                  Click the button below to load a demo setup. This will load a demo journal with
                  relevant config.
                </li>
                <li>
                  Once you are done playing around, you can go to <a href="/ledger/editor">editor</a
                  > page and select all the content and delete them.
                </li>
                <li>
                  Go to <a href="/more/config">configuration</a> page and click the reset to defaults
                  button.
                </li>
              </ol>

              <button type="button" onclick={initDemo} class="button is-link">Setup Demo</button>
            </div>
          </div>
        </ZeroState>
      </div>
    </div>
  </div>
</section>

<section class="section tab-networth" class:is-hidden={isEmpty}>
  <div class="container is-fluid">
    <div class="is-flex is-justify-content-flex-end mb-2">
      <button
        class="button is-small is-light"
        type="button"
        aria-label="Customize dashboard widgets"
        onclick={() => (showWidgetSettings = true)}
      >
        <span class="icon is-small"><i class="fas fa-gear"></i></span>
        <span>Customize</span>
      </button>
    </div>
    <div class="tile is-ancestor is-align-items-start">
      <div class="tile is-4 is-vertical">
        {#each leftWidgets as widget (widget.id)}
          {#if widget.visible}
            {#if widget.id === "networth"}
              <div class="tile is-parent">
                <div class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/assets/networth">Assets</a>
                    </p>
                    <div class="content">
                      {#if networth}
                        <nav class="level grid-2">
                          <LevelItem
                            narrow
                            title="Net worth"
                            color={COLORS.primary}
                            value={formatCurrency(networth.balanceAmount)}
                          />

                          <LevelItem
                            narrow
                            title="Net Investment"
                            color={COLORS.secondary}
                            value={formatCurrency(networth.netInvestmentAmount)}
                          />
                        </nav>
                        <nav class="level grid-2">
                          <LevelItem
                            narrow
                            title="Gain / Loss"
                            color={networth.gainAmount >= 0 ? COLORS.gainText : COLORS.lossText}
                            value={formatCurrency(networth.gainAmount)}
                          />

                          <LevelItem narrow title="XIRR" value={formatFloat(xirr)} />
                        </nav>
                      {/if}
                    </div>
                  </div>
                </div>
              </div>
            {:else if widget.id === "checkingBalances" && !_.isEmpty(checkingBalancesForWidget)}
              <div class="tile is-parent">
                <article class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/assets/balance"
                        >Checking Balance</a
                      >
                    </p>
                    <div class="content">
                      <div class="masonry-grid masonry-grid-400">
                        {#each checkingBalancesForWidget as assetBreakdown}
                          <div class="is-flex-grow-1">
                            <BalanceCard {assetBreakdown} />
                          </div>
                        {/each}
                      </div>
                    </div>
                  </div>
                </article>
              </div>
            {:else if widget.id === "investmentIncome"}
              <div class="tile is-parent">
                <article class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/income/investment"
                        >Investment Income</a
                      >
                    </p>
                    <div class="content">
                      <nav class="level grid-2">
                        <LevelItem
                          narrow
                          title="TTM Dividend"
                          color={investmentIncomeLoading ? undefined : COLORS.gainText}
                          value={investmentIncomeLoading
                            ? "—"
                            : formatCurrency(investmentIncomeDividendTTM)}
                        />
                        <LevelItem
                          narrow
                          title="TTM Interest"
                          color={investmentIncomeLoading ? undefined : COLORS.gainText}
                          value={investmentIncomeLoading
                            ? "—"
                            : formatCurrency(investmentIncomeInterestTTM)}
                        />
                      </nav>
                    </div>
                  </div>
                </article>
              </div>
            {:else if widget.id === "cashFlow"}
              <div class="tile is-parent">
                <article class="tile is-child min-w-0">
                  <p class="subtitle">
                    <a class="secondary-link has-text-grey" href="/cash_flow/monthly">Cash Flow</a>
                  </p>
                  <div class="content box px-2 pb-0">
                    <ZeroState item={cashFlows}>
                      <strong>Oops!</strong> You have not made any transactions in the last 3 months.
                    </ZeroState>

                    <LegendCard legends={cashflowLegends} clazz="mb-2 overflow-x-auto" />

                    <svg
                      class:is-not-visible={_.isEmpty(cashFlows)}
                      id="d3-current-cash-flow"
                      height="250"
                      width="100%"
                    />
                  </div>
                </article>
              </div>
            {:else if widget.id === "budget" && currentBudget}
              <div class="tile is-parent">
                <div class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/expense/budget">Budget</a>
                    </p>
                    <div class="content">
                      {#each budgetAccountsForWidget as accountBudget (accountBudget)}
                        <BudgetCard compact {accountBudget} />
                      {/each}
                    </div>
                  </div>
                </div>
              </div>
            {:else if widget.id === "goals" && !_.isEmpty(goalSummariesForWidget)}
              <div class="tile">
                <div class="tile is-parent is-12">
                  <article class="tile is-child">
                    <div class="content">
                      <p class="subtitle">
                        <a class="secondary-link has-text-grey" href="/planning/goals">Goals</a>
                      </p>
                      <div class="content">
                        {#each goalSummariesForWidget as goal}
                          <GoalSummaryCard {goal} small />
                        {/each}
                      </div>
                    </div>
                  </article>
                </div>
              </div>
            {/if}
          {/if}
        {/each}
      </div>
      <div class="tile is-vertical">
        {#each rightWidgets as widget (widget.id)}
          {#if widget.visible}
            {#if widget.id === "expenses"}
              <div class="tile is-parent is-12">
                <article class="tile is-child">
                  <p class="subtitle is-flex is-justify-content-space-between is-align-items-end">
                    <span
                      ><a class="secondary-link has-text-grey" href="/expense/monthly">Expenses</a>
                      <span
                        class="is-size-5 has-text-weight-bold px-2"
                        style="color: {COLORS.expenses}"
                        >{formatCurrency(totalExpense)} {USER_CONFIG.default_currency}</span
                      ></span
                    >
                    <LastNMonths n={3} bind:value={month} />
                  </p>
                  <div class="content box px-3">
                    <ZeroState item={selectedExpenses}>
                      <strong>Hurray!</strong> You have no expenses this month.
                    </ZeroState>
                    <svg id="d3-current-month-breakdown" width="100%" />
                  </div>
                </article>
              </div>
            {:else if widget.id === "recurring" && !_.isEmpty(transactionSequencesForWidget)}
              <div class="tile is-parent is-12">
                <article class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/cash_flow/recurring"
                        >Recurring</a
                      >
                    </p>
                    <div class="content box">
                      <div
                        class="grid grid-rows-1 overflow-hidden"
                        style="grid-auto-rows: 0px; grid-template-columns: repeat(auto-fit, minmax(130px, 150px));"
                      >
                        {#each transactionSequencesForWidget as ts (ts)}
                          <UpcomingCard transactionSequece={ts} />
                        {/each}
                      </div>
                    </div>
                  </div>
                </article>
              </div>
            {:else if widget.id === "recentTransactions" && !_.isEmpty(recentTransactionsForWidget)}
              <div class="tile is-parent is-12">
                <article class="tile is-child">
                  <RecentTransactionsWidget transactions={recentTransactionsForWidget} />
                </article>
              </div>
            {/if}
          {/if}
        {/each}
      </div>
    </div>
  </div>
</section>

<Modal bind:active={showWidgetSettings} width="min(920px, 100vw)" footerClass="justify-end">
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">Customize Dashboard</p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="Close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}
  {#snippet body()}
    <p class="mb-3 has-text-grey is-size-7">
      Drag widgets to reorder them. Toggle visibility and adjust per-widget options below.
    </p>
    <div class="columns is-variable is-3">
      {#each widgetColumns as column}
        <div class="column is-6">
          <p class="is-size-7 has-text-weight-semibold mb-2">
            {column === "left" ? "Left column" : "Right column"}
          </p>
          <div
            class="widget-picker-column"
            use:dndzone={{
              items: dashboardLayout[column],
              dropTargetStyle: {},
              flipDurationMs: 200
            }}
            onconsider={(event) => handleColumnConsider(column, event)}
            onfinalize={(event) => handleColumnFinalize(column, event)}
          >
            {#each dashboardLayout[column] as widget (widget.id)}
              {@const meta = getWidgetMeta(widget.id)}
              <div class="widget-picker-item box p-3 mb-2" animate:flip={{ duration: 200 }}>
                {#if meta}
                  <div class="is-flex is-justify-content-space-between is-align-items-center">
                    <div class="is-flex is-align-items-center">
                      <span class="icon is-small has-text-grey mr-2"
                        ><i class="fas fa-grip-vertical"></i></span
                      >
                      <span class="has-text-weight-semibold">{meta.title}</span>
                    </div>
                    <label class="checkbox is-flex is-align-items-center">
                      <input
                        type="checkbox"
                        checked={widget.visible}
                        onchange={(event) =>
                          setWidgetVisible(
                            column,
                            widget.id,
                            (event.currentTarget as HTMLInputElement).checked
                          )}
                      />
                      <span class="ml-1">Visible</span>
                    </label>
                  </div>
                  {#if meta.configOptions && meta.configOptions.length > 0}
                    <div class="mt-2">
                      {#each meta.configOptions as option}
                        {@const optionInputId = `widget-config-${column}-${widget.id}-${option.key}`}
                        <label class="label is-size-7 mb-1" for={optionInputId}
                          >{option.label}</label
                        >
                        <input
                          id={optionInputId}
                          class="input is-small"
                          type="number"
                          min={option.min}
                          max={option.max}
                          step={option.step || 1}
                          value={widget.config[option.key] ?? option.defaultValue}
                          onchange={(event) =>
                            setWidgetConfig(
                              column,
                              widget.id,
                              option,
                              (event.currentTarget as HTMLInputElement).value
                            )}
                        />
                      {/each}
                    </div>
                  {/if}
                {/if}
              </div>
            {/each}
          </div>
        </div>
      {/each}
    </div>
  {/snippet}
</Modal>

<style lang="scss">
  .masonry-grid {
    display: grid;
    gap: 10px;
    align-items: stretch;
  }

  .masonry-grid-400 {
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  }

  p.subtitle {
    margin-bottom: 0.5rem !important;
  }

  p.subtitle a.secondary-link {
    text-transform: uppercase;
    font-size: 1rem;
  }

  .widget-picker-column :global(.dndDraggingSource) {
    opacity: 0.5;
  }

  .widget-picker-item {
    cursor: grab;
  }
</style>
