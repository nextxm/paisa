<script lang="ts">
  import * as cashFlow from "$lib/cash_flow";
  import COLORS from "$lib/colors";
  import LastNMonths from "$lib/components/LastNMonths.svelte";
  import * as expense from "$lib/expense/monthly";
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
  import {
    DASHBOARD_WIDGET_DEFINITIONS,
    DASHBOARD_LAYOUT_STORAGE_KEY,
    loadDashboardLayout,
    persistDashboardLayout,
    reconcileDashboardLayout,
    reorderDashboardWidget,
    stepMoveDashboardWidget,
    type DashboardWidgetDefinition,
    type DashboardWidgetId,
    type DashboardWidgetLayout
  } from "$lib/dashboard_layout";

  let { data }: { data: PageData } = $props();

  let cashflowLegends: Legend[] = $state([]);
  let month = $state(now().format("YYYY-MM"));
  let goalSummaries: GoalSummary[] = $state(
    _.sortBy(data.dashboard.goalSummaries, (g) => -g.priority)
  );
  let transactionSequences: TransactionSequence[] = $state(
    _.take(sortTrantionSequence(enrichTrantionSequence(data.dashboard.transactionSequences)), 16)
  );
  let cashFlows: CashFlow[] = $state(data.dashboard.cashFlows);
  let expenses: { [key: string]: Posting[] } = $state(data.dashboard.expenses);
  let xirr = $state(data.dashboard.networth.xirr);
  let networth: Networth = $state(data.dashboard.networth.networth);
  let renderer: (data: Posting[]) => void = $state();
  let selectedExpenses = $derived(expenses[month] || []);
  let totalExpense = $derived(_.sumBy(selectedExpenses, (p) => p.amount));
  let transactions: Transaction[] = $state(data.dashboard.transactions);
  let budgetsByMonth: Record<string, Budget> = $state(data.dashboard.budget.budgetsByMonth);
  let currentBudget = $derived(budgetsByMonth[month]);
  let isEmpty = $state(_.isEmpty(data.dashboard.transactions));
  let checkingBalances: Record<string, AssetBreakdown> = $state(
    data.dashboard.checkingBalances.asset_breakdowns
  );
  let investmentIncomeDividendTTM = $state(data.income.ttm_dividend || 0);
  let investmentIncomeInterestTTM = $state(data.income.ttm_interest || 0);
  let investmentIncomeLoading = $state(false);
  let widgetSettingsOpen = $state(false);
  let dragWidgetId: DashboardWidgetId | null = $state(null);
  let dashboardLayout: DashboardWidgetLayout[] = $state([]);

  function isWidgetAvailable(id: DashboardWidgetId) {
    if (id == "checking-balances") {
      return !_.isEmpty(checkingBalances);
    }

    if (id == "budget") {
      return !!currentBudget;
    }

    if (id == "goals") {
      return !_.isEmpty(goalSummaries);
    }

    if (id == "recurring") {
      return !_.isEmpty(transactionSequences);
    }

    if (id == "recent-transactions") {
      return !_.isEmpty(transactions);
    }

    return true;
  }

  let availableWidgetDefinitions = $derived(
    DASHBOARD_WIDGET_DEFINITIONS.filter((widget) => isWidgetAvailable(widget.id))
  );

  let sortedWidgetDefinitions = $derived(
    [...availableWidgetDefinitions].sort((a, b) => a.defaultPosition - b.defaultPosition)
  );

  function widgetTitle(id: DashboardWidgetId) {
    return _.find(availableWidgetDefinitions, { id })?.title || id;
  }

  function widgetDefinition(id: DashboardWidgetId): DashboardWidgetDefinition | undefined {
    return _.find(availableWidgetDefinitions, { id });
  }

  function widgetLayout(id: DashboardWidgetId): DashboardWidgetLayout | undefined {
    return _.find(dashboardLayout, { id });
  }

  function widgetVisible(id: DashboardWidgetId) {
    return widgetLayout(id)?.visible ?? true;
  }

  function widgetLimit(id: DashboardWidgetId, fallback: number) {
    const limit = widgetLayout(id)?.config?.limit;
    return _.isNumber(limit) ? limit : fallback;
  }

  function leftWidgetIds() {
    const byId = _.keyBy(availableWidgetDefinitions, "id");
    return dashboardLayout
      .filter((layout) => layout.visible && byId[layout.id]?.column == "left")
      .map((layout) => layout.id);
  }

  function rightWidgetIds() {
    const byId = _.keyBy(availableWidgetDefinitions, "id");
    return dashboardLayout
      .filter((layout) => layout.visible && byId[layout.id]?.column == "right")
      .map((layout) => layout.id);
  }

  function orderedSettingsWidgets() {
    const known = new Set(dashboardLayout.map((layout) => layout.id));
    const ordered = dashboardLayout
      .map((layout) => widgetDefinition(layout.id))
      .filter((widget): widget is DashboardWidgetDefinition => !!widget);
    const missing = sortedWidgetDefinitions.filter((widget) => !known.has(widget.id));
    return [...ordered, ...missing];
  }

  function toggleWidget(id: DashboardWidgetId, visible: boolean) {
    dashboardLayout = dashboardLayout.map((layout) =>
      layout.id == id
        ? {
            ...layout,
            visible
          }
        : layout
    );
  }

  function setWidgetLimit(id: DashboardWidgetId, value: number) {
    const definition = widgetDefinition(id);
    const limitSetting = definition?.settings?.find((setting) => setting.key == "limit");
    if (!limitSetting || !Number.isFinite(value)) {
      return;
    }

    const clamped = Math.min(limitSetting.max, Math.max(limitSetting.min, Math.round(value)));
    dashboardLayout = dashboardLayout.map((layout) =>
      layout.id == id
        ? {
            ...layout,
            config: {
              ...layout.config,
              limit: clamped
            }
          }
        : layout
    );
  }

  function moveWidget(id: DashboardWidgetId, direction: -1 | 1) {
    dashboardLayout = stepMoveDashboardWidget(dashboardLayout, id, direction);
  }

  function dragStartWidget(id: DashboardWidgetId) {
    dragWidgetId = id;
  }

  function dragEndWidget() {
    dragWidgetId = null;
  }

  function dragOverWidget(event: DragEvent) {
    event.preventDefault();
  }

  function dropWidget(targetId: DashboardWidgetId) {
    if (!dragWidgetId || dragWidgetId == targetId) {
      return;
    }

    dashboardLayout = reorderDashboardWidget(dashboardLayout, dragWidgetId, targetId);
    dragWidgetId = null;
  }

  $effect(() => {
    if (renderer) {
      renderer(selectedExpenses);
    }
  });

  async function initDemo() {
    await ajax("/api/init", { method: "POST" });
    refresh();
  }

  onMount(() => {
    dashboardLayout = loadDashboardLayout(
      sortedWidgetDefinitions,
      DASHBOARD_LAYOUT_STORAGE_KEY,
      localStorage
    );

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

  $effect(() => {
    const reconciled = reconcileDashboardLayout(sortedWidgetDefinitions, dashboardLayout);
    if (!_.isEqual(reconciled, dashboardLayout)) {
      dashboardLayout = reconciled;
    }
  });

  $effect(() => {
    if (!_.isEmpty(dashboardLayout)) {
      persistDashboardLayout(dashboardLayout, DASHBOARD_LAYOUT_STORAGE_KEY, localStorage);
    }
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
    <div class="is-flex is-justify-content-flex-end mb-3">
      <button
        class="button is-light is-small invertable"
        type="button"
        onclick={() => (widgetSettingsOpen = true)}
      >
        <span class="icon is-small"><i class="fas fa-gear" aria-hidden="true"></i></span>
        <span>Widgets</span>
      </button>
    </div>
    <div class="tile is-ancestor is-align-items-start">
      <div class="tile is-4 is-vertical">
        {#each leftWidgetIds() as widgetId (widgetId)}
          {#if widgetId == "assets-overview"}
            <div class="tile is-parent">
              <div class="tile is-child">
                <div class="content">
                  <p class="subtitle">
                    <a class="secondary-link has-text-grey" href="/assets/networth">Assets</a>
                  </p>
                  <div class="content">
                    <div>
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
            </div>
          {/if}

          {#if widgetId == "checking-balances"}
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
                      {#each _.values(checkingBalances) as assetBreakdown}
                        <div class="is-flex-grow-1">
                          <BalanceCard {assetBreakdown} />
                        </div>
                      {/each}
                    </div>
                  </div>
                </div>
              </article>
            </div>
          {/if}

          {#if widgetId == "investment-income"}
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
          {/if}

          {#if widgetId == "cash-flow"}
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
          {/if}

          {#if widgetId == "budget" && currentBudget}
            <div class="tile is-parent">
              <div class="tile is-child">
                <div class="content">
                  <p class="subtitle">
                    <a class="secondary-link has-text-grey" href="/expense/budget">Budget</a>
                  </p>
                  <div class="content">
                    <div>
                      {#each _.take(currentBudget.accounts, widgetLimit("budget", 10)) as accountBudget (accountBudget)}
                        <BudgetCard compact {accountBudget} />
                      {/each}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          {/if}

          {#if widgetId == "goals"}
            <div class="tile">
              <div class="tile is-parent is-12">
                <article class="tile is-child">
                  <div class="content">
                    <p class="subtitle">
                      <a class="secondary-link has-text-grey" href="/planning/goals">Goals</a>
                    </p>
                    <div class="content">
                      {#each _.take(goalSummaries, widgetLimit("goals", 8)) as goal}
                        <GoalSummaryCard {goal} small />
                      {/each}
                    </div>
                  </div>
                </article>
              </div>
            </div>
          {/if}
        {/each}
      </div>
      <div class="tile is-vertical">
        {#each rightWidgetIds() as widgetId (widgetId)}
          {#if widgetId == "expenses"}
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
          {/if}

          {#if widgetId == "recurring"}
            <div class="tile is-parent is-12">
              <article class="tile is-child">
                <div class="content">
                  <p class="subtitle">
                    <a class="secondary-link has-text-grey" href="/cash_flow/recurring">Recurring</a
                    >
                  </p>
                  <div class="content box">
                    <div
                      class="grid grid-rows-1 overflow-hidden"
                      style="grid-auto-rows: 0px; grid-template-columns: repeat(auto-fit, minmax(130px, 150px));"
                    >
                      {#each _.take(transactionSequences, widgetLimit("recurring", 16)) as ts (ts)}
                        <UpcomingCard transactionSequece={ts} />
                      {/each}
                    </div>
                  </div>
                </div>
              </article>
            </div>
          {/if}

          {#if widgetId == "recent-transactions"}
            <div class="tile is-parent is-12">
              <article class="tile is-child">
                <RecentTransactionsWidget
                  {transactions}
                  limit={widgetLimit("recent-transactions", 15)}
                />
              </article>
            </div>
          {/if}
        {/each}
      </div>
    </div>
  </div>
</section>

{#if widgetSettingsOpen}
  <div class="modal is-active">
    <button
      class="modal-background"
      type="button"
      aria-label="Close dashboard widget settings"
      onclick={() => (widgetSettingsOpen = false)}
    ></button>
    <div class="modal-card dashboard-settings-modal">
      <header class="modal-card-head">
        <p class="modal-card-title">Dashboard widgets</p>
        <button
          class="delete"
          aria-label="close"
          type="button"
          onclick={() => (widgetSettingsOpen = false)}
        ></button>
      </header>
      <section class="modal-card-body">
        <p class="mb-3 is-size-7 has-text-grey">
          Drag to reorder, toggle visibility, and adjust widget limits.
        </p>
        {#each orderedSettingsWidgets() as widget (widget.id)}
          <div
            class="box mb-2 widget-picker-item"
            role="listitem"
            draggable
            ondragstart={() => dragStartWidget(widget.id)}
            ondragend={dragEndWidget}
            ondragover={dragOverWidget}
            ondrop={() => dropWidget(widget.id)}
          >
            <div class="is-flex is-justify-content-space-between is-align-items-center">
              <div class="is-flex is-align-items-center gap-2">
                <span class="has-text-grey is-size-6" title="Drag to reorder">⋮⋮</span>
                <span class="has-text-weight-medium">{widgetTitle(widget.id)}</span>
              </div>
              <label class="checkbox">
                <input
                  type="checkbox"
                  checked={widgetVisible(widget.id)}
                  onchange={(event) =>
                    toggleWidget(widget.id, (event.currentTarget as HTMLInputElement).checked)}
                />
                <span class="ml-1">Show</span>
              </label>
            </div>
            <div class="is-flex is-justify-content-space-between is-align-items-center mt-2">
              <div class="buttons has-addons are-small mb-0">
                <button
                  class="button"
                  type="button"
                  onclick={() => moveWidget(widget.id, -1)}
                  aria-label={`Move ${widget.title} up`}
                >
                  ↑
                </button>
                <button
                  class="button"
                  type="button"
                  onclick={() => moveWidget(widget.id, 1)}
                  aria-label={`Move ${widget.title} down`}
                >
                  ↓
                </button>
              </div>
              {#if widget.settings?.length}
                {#each widget.settings as setting}
                  <label class="is-flex is-align-items-center is-size-7">
                    <span class="mr-2">{setting.label}</span>
                    <input
                      class="input is-small widget-number-setting"
                      type="number"
                      min={setting.min}
                      max={setting.max}
                      step={setting.step}
                      value={widgetLimit(widget.id, setting.defaultValue)}
                      onchange={(event) =>
                        setWidgetLimit(
                          widget.id,
                          Number((event.currentTarget as HTMLInputElement).value)
                        )}
                    />
                  </label>
                {/each}
              {/if}
            </div>
          </div>
        {/each}
      </section>
      <footer class="modal-card-foot is-justify-content-flex-end">
        <button class="button is-link" type="button" onclick={() => (widgetSettingsOpen = false)}>
          Done
        </button>
      </footer>
    </div>
  </div>
{/if}

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

  .dashboard-settings-modal {
    max-width: 720px;
    width: calc(100% - 2rem);
  }

  .widget-picker-item {
    cursor: grab;
  }

  .widget-number-setting {
    width: 96px;
  }
</style>
