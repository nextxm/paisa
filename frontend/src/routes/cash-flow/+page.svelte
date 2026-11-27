<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import BarTrendChart, { type MonthlyBarPoint } from "$lib/components/BarTrendChart.svelte";
  import { apiGet, formatMoney, type Posting } from "$lib/api";

  type StudioTab = "cashflow" | "statement" | "sankey" | "expenses" | "budget" | "recurring";
  type CashFlowGranularity = "monthly" | "yearly";
  type ExpenseView = "breakdown" | "heatmap" | "trends";
  type SankeyPeriod = "month" | "quarter" | "year";

  interface CashFlowRow {
    date: string;
    income: number;
    expenses: number;
    investment: number;
    tax: number;
    liabilities: number;
    balance: number;
  }

  interface IncomeStatement {
    startingBalance: number;
    endingBalance: number;
    income?: Record<string, number>;
    expenses?: Record<string, number>;
    tax?: Record<string, number>;
  }

  interface SankeyLink {
    source: string;
    target: string;
    value: number;
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
  let cfGranularity = $state<CashFlowGranularity>("monthly");
  let expenseView = $state<ExpenseView>("breakdown");
  let sankeyPeriod = $state<SankeyPeriod>("month");

  let monthsWindow = $state<number>(12);
  let selectedMonth = $state(dayjs().format("YYYY-MM"));
  let selectedYear = $state("");
  let groupDepth = $state<1 | 2>(1);

  let loading = $state(true);
  let error = $state<string | null>(null);

  let cashFlows = $state<CashFlowRow[]>([]);
  let monthWiseExpenses = $state<Record<string, Posting[]>>({});
  let yearlyStatements = $state<Record<string, IncomeStatement>>({});
  let sankeyLinks = $state<SankeyLink[]>([]);
  let sankeyLoading = $state(false);
  let budgetsByMonth = $state<Record<string, BudgetMonth>>({});
  let availableForBudgeting = $state(0);
  let recurringItems = $state<RecurringItem[]>([]);

  onMount(async () => {
    try {
      const [cfRes, expRes, stmtRes, budgetRes, recRes] = await Promise.all([
        apiGet<{ cash_flows?: CashFlowRow[] }>("/api/cash_flow"),
        apiGet<{ month_wise?: { expenses?: Record<string, Posting[]> } }>("/api/expense"),
        apiGet<{ yearly?: Record<string, IncomeStatement> }>("/api/income_statement").catch(
          () => ({ yearly: {} })
        ),
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
      yearlyStatements = stmtRes.yearly || {};
      budgetsByMonth = budgetRes.budgetsByMonth || {};
      availableForBudgeting = budgetRes.availableForBudgeting || 0;
      recurringItems = recRes.transaction_sequences || [];

      const availableMonths = Object.keys(monthWiseExpenses).sort();
      if (availableMonths.length > 0 && !monthWiseExpenses[selectedMonth]) {
        selectedMonth = availableMonths[availableMonths.length - 1];
      }

      const stmtYears = Object.keys(yearlyStatements).sort().reverse();
      if (stmtYears.length > 0) {
        selectedYear = stmtYears[0];
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load cash flow data";
    } finally {
      loading = false;
    }
  });

  async function loadSankey() {
    sankeyLoading = true;
    try {
      const res = await apiGet<{ links?: SankeyLink[] }>(
        `/api/sankey?period=${encodeURIComponent(sankeyPeriod)}`
      );
      sankeyLinks = (res.links || []).slice().sort((a, b) => b.value - a.value);
    } catch {
      sankeyLinks = [];
    } finally {
      sankeyLoading = false;
    }
  }

  $effect(() => {
    if (activeTab === "sankey") {
      const _p = sankeyPeriod;
      void loadSankey();
    }
  });

  const windowedFlows = $derived(
    monthsWindow > 0 ? cashFlows.slice(-monthsWindow) : cashFlows
  );

  const yearlyFlows = $derived.by(() => {
    const byYear = new Map<string, CashFlowRow>();
    for (const row of cashFlows) {
      const y = dayjs(row.date).format("YYYY");
      const existing = byYear.get(y) || {
        date: `${y}-01-01`,
        income: 0,
        expenses: 0,
        investment: 0,
        tax: 0,
        liabilities: 0,
        balance: 0
      };
      existing.income += row.income;
      existing.expenses += row.expenses;
      existing.investment += row.investment;
      existing.tax += row.tax;
      existing.balance = row.balance;
      byYear.set(y, existing);
    }
    return Array.from(byYear.values());
  });

  const activeFlowRows = $derived(cfGranularity === "yearly" ? yearlyFlows : windowedFlows);

  const chartPoints = $derived<MonthlyBarPoint[]>(
    activeFlowRows.map((c) => ({
      date: c.date,
      income: c.income,
      expenses: c.expenses,
      investment: c.investment,
      balance: c.balance
    }))
  );

  const avgIncome = $derived(
    activeFlowRows.length > 0
      ? activeFlowRows.reduce((s, c) => s + c.income, 0) / activeFlowRows.length
      : 0
  );
  const avgExpenses = $derived(
    activeFlowRows.length > 0
      ? activeFlowRows.reduce((s, c) => s + c.expenses, 0) / activeFlowRows.length
      : 0
  );
  const avgSavingsRate = $derived(
    avgIncome > 0 ? ((avgIncome - avgExpenses) / avgIncome) * 100 : 0
  );

  const statementYears = $derived(Object.keys(yearlyStatements).sort().reverse());
  const currentStatement = $derived(selectedYear ? yearlyStatements[selectedYear] : undefined);

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
        groupDepth === 1 ? parts.slice(0, 2).join(":") : parts.slice(0, 3).join(":");
      map.set(key, (map.get(key) || 0) + (p.amount || 0));
    }
    return Array.from(map.entries())
      .map(([name, amount]) => ({ name, amount }))
      .sort((a, b) => b.amount - a.amount);
  });

  const dailyExpenseMap = $derived.by(() => {
    const map = new Map<number, number>();
    for (const p of selectedMonthPostings) {
      const day = dayjs(p.date).date();
      map.set(day, (map.get(day) || 0) + (p.amount || 0));
    }
    return map;
  });

  const daysInSelectedMonth = $derived(dayjs(`${selectedMonth}-01`).daysInMonth());
  const maxDailySpend = $derived(Math.max(1, ...Array.from(dailyExpenseMap.values())));

  const momComparison = $derived.by(() => {
    const prevMonth = dayjs(`${selectedMonth}-01`).subtract(1, "month").format("YYYY-MM");
    const yoyMonth = dayjs(`${selectedMonth}-01`).subtract(1, "year").format("YYYY-MM");
    const sumMonth = (m: string) =>
      (monthWiseExpenses[m] || []).reduce((s, p) => s + (p.amount || 0), 0);
    const prevTotal = sumMonth(prevMonth);
    const yoyTotal = sumMonth(yoyMonth);
    return {
      prevMonth,
      prevTotal,
      momDeltaPct: prevTotal > 0 ? ((selectedMonthTotal - prevTotal) / prevTotal) * 100 : null,
      yoyMonth,
      yoyTotal,
      yoyDeltaPct: yoyTotal > 0 ? ((selectedMonthTotal - yoyTotal) / yoyTotal) * 100 : null
    };
  });

  const currentBudgetAccounts = $derived(budgetsByMonth[selectedMonth]?.accounts || []);
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Cash Flow &amp; Spending</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Cash flow, income statements, money flow, expenses, budgets, and recurring commitments
      </p>
    </div>

    {#if activeTab === "cashflow"}
      <div class="flex flex-wrap items-center gap-2">
        <div
          class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
        >
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {cfGranularity === 'monthly'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (cfGranularity = "monthly")}
          >
            Monthly
          </button>
          <button
            type="button"
            class="px-2.5 py-1 rounded-md cursor-pointer {cfGranularity === 'yearly'
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (cfGranularity = "yearly")}
          >
            Yearly
          </button>
        </div>

        {#if cfGranularity === "monthly"}
          <div
            class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
          >
            {#each [6, 12, 24, 0] as w}
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {monthsWindow === w
                  ? 'bg-[var(--surface-elevated)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (monthsWindow = w)}
              >
                {w === 0 ? "All" : `${w}M`}
              </button>
            {/each}
          </div>
        {/if}
      </div>
    {:else if activeTab === "statement" && statementYears.length > 0}
      <select
        bind:value={selectedYear}
        aria-label="Select financial year"
        class="px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] outline-none"
      >
        {#each statementYears as y}
          <option value={y}>FY {y}</option>
        {/each}
      </select>
    {:else if activeTab === "sankey"}
      <div
        class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface)] p-0.5 text-xs"
      >
        {#each ["month", "quarter", "year"] as p}
          <button
            type="button"
            class="px-2.5 py-1 rounded-md capitalize cursor-pointer {sankeyPeriod === p
              ? 'bg-[var(--surface-elevated)] font-semibold'
              : 'text-[var(--muted)]'}"
            onclick={() => (sankeyPeriod = p as SankeyPeriod)}
          >
            {p}
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
      { id: "statement", label: "Income Statement" },
      { id: "sankey", label: "Money Flow" },
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
    <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg {cfGranularity === "yearly" ? "Yearly" : "Monthly"} Income
        </div>
        <div class="text-2xl font-semibold tabular-nums text-[var(--gain)] mt-1">
          {formatMoney(avgIncome)}
        </div>
      </div>
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Avg {cfGranularity === "yearly" ? "Yearly" : "Monthly"} Expenses
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

    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">Period</th>
              <th class="py-2.5 px-4 text-right font-medium">Income</th>
              <th class="py-2.5 px-4 text-right font-medium">Expenses</th>
              <th class="py-2.5 px-4 text-right font-medium">Investments</th>
              <th class="py-2.5 px-4 text-right font-medium">Net</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each [...activeFlowRows].reverse() as row}
              {@const net = row.income - row.expenses - row.tax}
              <tr class="hover:bg-[var(--surface-elevated)]/50">
                <td class="py-2.5 px-4 font-medium">
                  {cfGranularity === "yearly"
                    ? dayjs(row.date).format("YYYY")
                    : dayjs(row.date).format("MMM YYYY")}
                </td>
                <td class="py-2.5 px-4 text-right text-[var(--gain)]">{formatMoney(row.income)}</td>
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
  {:else if activeTab === "statement"}
    {#if !currentStatement}
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center text-sm text-[var(--muted)]"
      >
        No income statement data for this financial year.
      </div>
    {:else}
      {@const diff =
        (currentStatement.endingBalance || 0) - (currentStatement.startingBalance || 0)}
      <div class="grid grid-cols-1 sm:grid-cols-3 gap-4">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Starting Net Worth
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(currentStatement.startingBalance || 0)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Ending Net Worth
          </div>
          <div class="text-2xl font-semibold tabular-nums mt-1">
            {formatMoney(currentStatement.endingBalance || 0)}
          </div>
        </div>
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
          <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Net Change
          </div>
          <div
            class="text-2xl font-semibold tabular-nums mt-1 {diff >= 0
              ? 'text-[var(--gain)]'
              : 'text-[var(--loss)]'}"
          >
            {formatMoney(diff)}
          </div>
        </div>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-5">
        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
          <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">Income</h3>
          <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
            {#each Object.entries(currentStatement.income || {}) as [acct, amt]}
              <div class="py-2 flex justify-between gap-2">
                <span class="truncate">{acct}</span>
                <span class="font-medium text-[var(--gain)]">{formatMoney(Math.abs(amt))}</span>
              </div>
            {/each}
          </div>
        </div>

        <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
          <h3 class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            Expenses &amp; Taxes
          </h3>
          <div class="divide-y divide-[var(--border)] text-sm tabular-nums">
            {#each Object.entries(currentStatement.expenses || {}) as [acct, amt]}
              <div class="py-2 flex justify-between gap-2">
                <span class="truncate">{acct}</span>
                <span class="font-medium text-[var(--loss)]">{formatMoney(Math.abs(amt))}</span>
              </div>
            {/each}
            {#each Object.entries(currentStatement.tax || {}) as [acct, amt]}
              <div class="py-2 flex justify-between gap-2">
                <span class="truncate">{acct}</span>
                <span class="font-medium text-[var(--warning)]">{formatMoney(Math.abs(amt))}</span>
              </div>
            {/each}
          </div>
        </div>
      </div>
    {/if}
  {:else if activeTab === "sankey"}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if sankeyLoading}
        <div class="h-48 animate-pulse"></div>
      {:else if sankeyLinks.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">
          No money flow links found for this {sankeyPeriod}.
        </div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead
              class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
            >
              <tr>
                <th class="py-2.5 px-4 font-medium">From</th>
                <th class="py-2.5 px-4 font-medium">To</th>
                <th class="py-2.5 px-4 text-right font-medium">Flow</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each sankeyLinks.slice(0, 40) as link}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-4 text-[var(--muted)]">{link.source}</td>
                  <td class="py-2 px-4 font-medium">{link.target}</td>
                  <td class="py-2 px-4 text-right font-semibold">{formatMoney(link.value)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
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

        <div class="flex flex-wrap items-center gap-2">
          <div
            class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
          >
            {#each [
              { id: "breakdown", label: "Breakdown" },
              { id: "heatmap", label: "Heatmap" },
              { id: "trends", label: "MoM / YoY" }
            ] as v}
              <button
                type="button"
                class="px-2.5 py-1 rounded-md cursor-pointer {expenseView === v.id
                  ? 'bg-[var(--surface)] font-semibold'
                  : 'text-[var(--muted)]'}"
                onclick={() => (expenseView = v.id as ExpenseView)}
              >
                {v.label}
              </button>
            {/each}
          </div>

          {#if expenseView === "breakdown"}
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
          {/if}
        </div>
      </div>

      {#if expenseView === "breakdown"}
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
      {:else if expenseView === "heatmap"}
        <div class="grid grid-cols-7 gap-2 pt-1">
          {#each Array.from({ length: daysInSelectedMonth }, (_, i) => i + 1) as day}
            {@const spent = dailyExpenseMap.get(day) || 0}
            {@const intensity = spent > 0 ? Math.max(0.18, spent / maxDailySpend) : 0}
            <div
              class="rounded-lg border border-[var(--border)] p-2 text-xs tabular-nums"
              style={spent > 0
                ? `background-color: color-mix(in srgb, var(--accent) ${Math.round(intensity * 45)}%, var(--surface))`
                : ""}
              title="Day {day}: {formatMoney(spent)}"
            >
              <div class="text-[var(--muted)] text-[10px]">Day {day}</div>
              <div class="font-medium mt-0.5 truncate">{spent > 0 ? formatMoney(spent) : "—"}</div>
            </div>
          {/each}
        </div>
      {:else}
        <div class="grid grid-cols-1 sm:grid-cols-2 gap-4 pt-1">
          <div class="rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] p-4">
            <div class="text-xs text-[var(--muted)]">
              vs Previous Month ({dayjs(`${momComparison.prevMonth}-01`).format("MMM YYYY")})
            </div>
            <div class="text-lg font-semibold tabular-nums mt-1">
              {formatMoney(momComparison.prevTotal)}
              {#if momComparison.momDeltaPct !== null}
                <span
                  class="text-xs ml-2 {momComparison.momDeltaPct <= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {momComparison.momDeltaPct >= 0 ? "+" : ""}{momComparison.momDeltaPct.toFixed(1)}%
                </span>
              {/if}
            </div>
          </div>

          <div class="rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] p-4">
            <div class="text-xs text-[var(--muted)]">
              vs Same Month Last Year ({dayjs(`${momComparison.yoyMonth}-01`).format("MMM YYYY")})
            </div>
            <div class="text-lg font-semibold tabular-nums mt-1">
              {formatMoney(momComparison.yoyTotal)}
              {#if momComparison.yoyDeltaPct !== null}
                <span
                  class="text-xs ml-2 {momComparison.yoyDeltaPct <= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {momComparison.yoyDeltaPct >= 0 ? "+" : ""}{momComparison.yoyDeltaPct.toFixed(1)}%
                </span>
              {/if}
            </div>
          </div>
        </div>
      {/if}
    </div>
  {:else if activeTab === "budget"}
    <div class="space-y-4">
      <div
        class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 flex items-center justify-between"
      >
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
                <span class="text-sm font-medium truncate">
                  {b.account.replace(/^Expenses:/, "")}
                </span>
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
