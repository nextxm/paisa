<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import BarTrendChart, { type MonthlyBarPoint } from "$lib/components/BarTrendChart.svelte";
  import { apiGet, formatMoney, type Posting } from "$lib/api";

  type StudioTab = "cashflow" | "expenses" | "budget" | "recurring";

  interface CashFlowRow {
    date: string;
    income: number;
    expenses: number;
    investment: number;
    tax: number;
    liabilities: number;
    balance: number;
  }

  interface BudgetAccount {
    account: string;
    forecast: number;
    actual: number;
    rollover: number;
    available: number;
  }

  interface BudgetMonth {
    date: string;
    accounts: BudgetAccount[];
  }

  interface RecurringItem {
    name?: string;
    account?: string;
    payee?: string;
    narration?: string;
    amount?: number;
    next_date?: string;
    frequency?: string;
  }

  let activeTab = $state<StudioTab>("cashflow");
  let monthsWindow = $state<number>(12);
  let selectedMonth = $state(dayjs().format("YYYY-MM"));
  let groupDepth = $state<1 | 2>(1);

  let loading = $state(true);
  let error = $state<string | null>(null);

  let cashFlows = $state<CashFlowRow[]>([]);
  let monthWiseExpenses = $state<Record<string, Posting[]>>({});
  let budgetsByMonth = $state<Record<string, BudgetMonth>>({});
  let availableForBudgeting = $state(0);
  let recurringItems = $state<RecurringItem[]>([]);

  onMount(async () => {
    try {
      const [cfRes, expRes, budgetRes, recRes] = await Promise.all([
        apiGet<{ cash_flows?: CashFlowRow[] }>("/api/cash_flow"),
        apiGet<{ month_wise?: { expenses?: Record<string, Posting[]> } }>("/api/expense"),
        apiGet<{
          budgetsByMonth?: Record<string, BudgetMonth>;
          availableForBudgeting?: number;
        }>("/api/budget").catch(() => ({ budgetsByMonth: {}, availableForBudgeting: 0 })),
        apiGet<{ transaction_sequences?: RecurringItem[] }>("/api/recurring").catch(() => ({
          transaction_sequences: []
        }))
      ]);

      cashFlows = (cfRes.cash_flows || []).map((c) => ({
        ...c,
        income: Math.abs(c.income || 0),
        expenses: Math.abs(c.expenses || 0),
        investment: Math.abs(c.investment || 0),
        tax: Math.abs(c.tax || 0)
      }));
      monthWiseExpenses = expRes.month_wise?.expenses || {};
      budgetsByMonth = budgetRes.budgetsByMonth || {};
      availableForBudgeting = budgetRes.availableForBudgeting || 0;
      recurringItems = recRes.transaction_sequences || [];

      const availableMonths = Object.keys(monthWiseExpenses).sort();
      if (availableMonths.length > 0 && !monthWiseExpenses[selectedMonth]) {
        selectedMonth = availableMonths[availableMonths.length - 1];
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load cash flow data";
    } finally {
      loading = false;
    }
  });

  const windowedFlows = $derived(
    monthsWindow > 0 ? cashFlows.slice(-monthsWindow) : cashFlows
  );

  const chartPoints = $derived<MonthlyBarPoint[]>(
    windowedFlows.map((c) => ({
      date: c.date,
      income: c.income,
      expenses: c.expenses,
      investment: c.investment,
      balance: c.balance
    }))
  );

  const avgIncome = $derived(
    windowedFlows.length > 0
      ? windowedFlows.reduce((s, c) => s + c.income, 0) / windowedFlows.length
      : 0
  );
  const avgExpenses = $derived(
    windowedFlows.length > 0
      ? windowedFlows.reduce((s, c) => s + c.expenses, 0) / windowedFlows.length
      : 0
  );
  const avgSavingsRate = $derived(
    avgIncome > 0 ? ((avgIncome - avgExpenses) / avgIncome) * 100 : 0
  );

  const availableMonths = $derived(Object.keys(monthWiseExpenses).sort().reverse());
  const selectedMonthPostings = $derived(monthWiseExpenses[selectedMonth] || []);
  const selectedMonthTotal = $derived(
    selectedMonthPostings.reduce((sum, p) => sum + (p.amount || 0), 0)
  );

  const groupedExpenseCategories = $derived.by(() => {
    const map = new Map<string, number>();
    for (const p of selectedMonthPostings) {
      const parts = p.account.split(":");
      const key =
        groupDepth === 1
          ? parts.slice(0, 2).join(":")
          : parts.slice(0, 3).join(":");
      map.set(key, (map.get(key) || 0) + (p.amount || 0));
    }
    return Array.from(map.entries())
      .map(([name, amount]) => ({ name, amount }))
      .sort((a, b) => b.amount - a.amount);
  });

  const currentBudgetAccounts = $derived(budgetsByMonth[selectedMonth]?.accounts || []);
</script>

<div class="space-y-5">
  <!-- Header & Unified Studio Tabs -->
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Cash Flow &amp; Spending</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Income, expenses, budgets, and recurring commitments in one workspace
      </p>
    </div>

    {#if activeTab === "cashflow"}
      <div
        class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
      >
        {#each [6, 12, 24, 0] as w}
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer transition-colors {monthsWindow === w
              ? 'bg-[var(--surface-elevated)] font-semibold text-[var(--text)]'
              : 'text-[var(--muted)] hover:text-[var(--text)]'}"
            onclick={() => (monthsWindow = w)}
          >
            {w === 0 ? "All" : `${w}M`}
          </button>
        {/each}
      </div>
    {:else if activeTab === "expenses" || activeTab === "budget"}
      <select
        bind:value={selectedMonth}
        aria-label="Select month"
        class="px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] outline-none"
      >
        {#each availableMonths as m}
          <option value={m}>{dayjs(`${m}-01`).format("MMM YYYY")}</option>
        {/each}
      </select>
    {/if}
  </div>

  <!-- Underline Tabs -->
  <div class="border-b border-[var(--border)] flex items-center gap-6 text-sm overflow-x-auto">
    {#each [
      { id: "cashflow", label: "Cash Flow" },
      { id: "expenses", label: "Expenses" },
      { id: "budget", label: "Budget" },
      { id: "recurring", label: "Recurring" }
    ] as tab}
      <button
        type="button"
        class="pb-2.5 border-b-2 transition-colors whitespace-nowrap cursor-pointer {activeTab ===
        tab.id
          ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
          : 'border-transparent text-[var(--muted)] hover:text-[var(--text)]'}"
        onclick={() => (activeTab = tab.id as StudioTab)}
      >
        {tab.label}
      </button>
    {/each}
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Failed to load cash flow data</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else if activeTab === "cashflow"}
    <!-- KPIs -->
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg Monthly Income
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
          {formatMoney(avgIncome)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg Monthly Expenses
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-1">
          {formatMoney(avgExpenses)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg Savings Rate
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">{avgSavingsRate.toFixed(1)}%</div>
      </div>
    </div>

    <!-- Chart -->
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
      <div class="flex items-center justify-between text-xs text-[var(--muted)]">
        <span class="font-semibold uppercase tracking-wider">Income vs Expenses</span>
        <div class="flex items-center gap-4">
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--gain)] inline-block"></span>
            Income
          </span>
          <span class="inline-flex items-center gap-1.5">
            <span class="w-2.5 h-2.5 rounded-xs bg-[var(--loss)] inline-block"></span>
            Expenses
          </span>
        </div>
      </div>
      <BarTrendChart data={chartPoints} />
    </div>

    <!-- Monthly Table -->
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">Month</th>
              <th class="py-2.5 px-4 text-right font-medium">Income</th>
              <th class="py-2.5 px-4 text-right font-medium">Expenses</th>
              <th class="py-2.5 px-4 text-right font-medium">Investments</th>
              <th class="py-2.5 px-4 text-right font-medium">Net</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each [...windowedFlows].reverse() as row}
              {@const net = row.income - row.expenses - row.tax}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2.5 px-4 font-medium">{dayjs(row.date).format("MMM YYYY")}</td>
                <td class="py-2.5 px-4 text-right text-[var(--gain)]">
                  {formatMoney(row.income)}
                </td>
                <td class="py-2.5 px-4 text-right text-[var(--loss)]">
                  {formatMoney(row.expenses)}
                </td>
                <td class="py-2.5 px-4 text-right">{formatMoney(row.investment)}</td>
                <td
                  class="py-2.5 px-4 text-right font-semibold {net >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {formatMoney(net)}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {:else if activeTab === "expenses"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <div>
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Total Spent ({dayjs(`${selectedMonth}-01`).format("MMMM YYYY")})
          </div>
          <div class="text-2xl font-semibold tabular-nums text-[var(--loss)] mt-0.5">
            {formatMoney(selectedMonthTotal)}
          </div>
        </div>
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {groupDepth === 1
              ? 'bg-[var(--surface)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (groupDepth = 1)}
          >
            Top Level
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {groupDepth === 2
              ? 'bg-[var(--surface)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (groupDepth = 2)}
          >
            Detailed
          </button>
        </div>
      </div>

      {#if groupedExpenseCategories.length === 0}
        <p class="text-sm text-[var(--muted)]">No expenses recorded in this month.</p>
      {:else}
        <div class="space-y-3">
          {#each groupedExpenseCategories as cat}
            {@const pct =
              selectedMonthTotal > 0 ? Math.min(100, (cat.amount / selectedMonthTotal) * 100) : 0}
            <div class="space-y-1">
              <div class="flex justify-between text-xs">
                <span class="font-medium">{cat.name.replace(/^Expenses:/, "")}</span>
                <span class="tabular-nums">
                  <strong>{formatMoney(cat.amount)}</strong>
                  <span class="text-[var(--muted)] ml-1">({pct.toFixed(1)}%)</span>
                </span>
              </div>
              <div class="h-2 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                <div class="h-full rounded-full bg-[var(--accent)]" style="width: {pct}%"></div>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else if activeTab === "budget"}
    <div class="space-y-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex items-center justify-between">
        <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Available for Budgeting
        </span>
        <span class="text-lg font-semibold tabular-nums">
          {formatMoney(availableForBudgeting)}
        </span>
      </div>

      {#if currentBudgetAccounts.length === 0}
        <div
          class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
        >
          No budget entries configured for {dayjs(`${selectedMonth}-01`).format("MMM YYYY")}.
        </div>
      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
          {#each currentBudgetAccounts as b}
            {@const pct = b.forecast > 0 ? Math.min(100, (b.actual / b.forecast) * 100) : 0}
            <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-2.5">
              <div class="flex items-center justify-between gap-2">
                <span class="text-sm font-medium truncate">{b.account.replace(/^Expenses:/, "")}</span>
                <span
                  class="text-xs font-semibold tabular-nums {b.available >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {b.available >= 0 ? "Left" : "Over"}: {formatMoney(Math.abs(b.available))}
                </span>
              </div>
              <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                <div
                  class="h-full rounded-full {b.available >= 0
                    ? 'bg-[var(--gain)]'
                    : 'bg-[var(--loss)]'}"
                  style="width: {pct}%"
                ></div>
              </div>
              <div class="flex justify-between text-xs text-[var(--muted)] tabular-nums">
                <span>Spent: {formatMoney(b.actual)}</span>
                <span>Budget: {formatMoney(b.forecast)}</span>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {:else}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if recurringItems.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">
          No recurring transactions configured.
        </div>
      {:else}
        <div class="divide-y divide-[var(--border)]">
          {#each recurringItems as item}
            <div class="px-4 py-3 flex items-center justify-between gap-3 text-sm">
              <div class="min-w-0">
                <div class="font-medium truncate">
                  {item.name || item.payee || item.narration || item.account || "Recurring item"}
                </div>
                <div class="text-xs text-[var(--muted)]">
                  {item.frequency || "Recurring"}
                  {item.next_date ? ` · Next: ${item.next_date}` : ""}
                </div>
              </div>
              {#if item.amount !== undefined}
                <span class="font-semibold tabular-nums">{formatMoney(item.amount)}</span>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/if}
</div>
