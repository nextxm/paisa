<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { AlertTriangle, ArrowUpRight, GripVertical, SlidersHorizontal, X } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import BarTrendChart from "$lib/components/BarTrendChart.svelte";
  import {
    apiGet,
    appPath,
    formatMoney,
    formatShortDate,
    runtime,
    triggerSync,
    type AssetBreakdown,
    type DashboardResponse,
    type Transaction
  } from "$lib/api";

  type WidgetKey =
    | "networth"
    | "checking"
    | "investmentIncome"
    | "cashFlow"
    | "expenses"
    | "recentTransactions"
    | "budget"
    | "goals"
    | "recurring";

  const widgetLabels: Record<WidgetKey, string> = {
    networth: "Net Worth",
    checking: "Checking Balances",
    investmentIncome: "Investment Income (TTM)",
    cashFlow: "Cash Flow",
    expenses: "Monthly Expenses",
    recentTransactions: "Recent Transactions",
    budget: "Budget",
    goals: "Goals",
    recurring: "Recurring"
  };

  const defaultVisible: Record<WidgetKey, boolean> = {
    networth: true,
    checking: true,
    investmentIncome: true,
    cashFlow: true,
    expenses: true,
    recentTransactions: true,
    budget: false,
    goals: false,
    recurring: false
  };

  const defaultLayout: { left: WidgetKey[]; right: WidgetKey[] } = {
    left: ["networth", "checking", "investmentIncome", "cashFlow", "budget", "goals"],
    right: ["expenses", "recurring", "recentTransactions"]
  };

  let visibleWidgets = $state<Record<WidgetKey, boolean>>({ ...defaultVisible });
  let widgetLayout = $state<{ left: WidgetKey[]; right: WidgetKey[] }>({
    left: [...defaultLayout.left],
    right: [...defaultLayout.right]
  });
  let draggingKey = $state<WidgetKey | null>(null);
  let dragOverKey = $state<WidgetKey | null>(null);
  let showCustomize = $state(false);
  let dashboard = $state<DashboardResponse | null>(null);
  let healthIssues = $state(0);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedTransaction = $state<Transaction | null>(null);
  let selectedMonth = $state(dayjs().format("YYYY-MM"));

  const last3Months = [2, 1, 0].map((offset) =>
    dayjs().subtract(offset, "month").format("YYYY-MM")
  );

  async function loadDashboard() {
    loading = true;
    error = null;
    try {
      const [dash, doctor] = await Promise.all([
        apiGet<DashboardResponse>("/api/dashboard"),
        apiGet<{ summary?: { fix_count?: number; review_count?: number } }>(
          "/api/doctor/findings"
        ).catch(() => ({ summary: undefined }))
      ]);
      dashboard = dash;
      healthIssues = (doctor.summary?.fix_count || 0) + (doctor.summary?.review_count || 0);
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load dashboard";
    } finally {
      loading = false;
    }
  }

  onMount(async () => {
    try {
      const saved = localStorage.getItem("paisa-v2-dashboard-widgets");
      if (saved) {
        visibleWidgets = { ...defaultVisible, ...JSON.parse(saved) };
      }
      const savedLayout = localStorage.getItem("paisa-v2-dashboard-layout");
      if (savedLayout) {
        const parsed = JSON.parse(savedLayout) as { left?: WidgetKey[]; right?: WidgetKey[] };
        if (Array.isArray(parsed.left) && Array.isArray(parsed.right)) {
          const allKnown = new Set<WidgetKey>([...defaultLayout.left, ...defaultLayout.right]);
          const seen = new Set<WidgetKey>();
          const clean = (arr: WidgetKey[]) =>
            arr.filter((k) => {
              if (!allKnown.has(k) || seen.has(k)) return false;
              seen.add(k);
              return true;
            });
          const left = clean(parsed.left);
          const right = clean(parsed.right);
          for (const k of allKnown) {
            if (!seen.has(k)) left.push(k);
          }
          widgetLayout = { left, right };
        }
      }
    } catch {}
    await loadDashboard();
  });

  function saveLayout(next: { left: WidgetKey[]; right: WidgetKey[] }) {
    widgetLayout = next;
    try {
      localStorage.setItem("paisa-v2-dashboard-layout", JSON.stringify(next));
    } catch {}
  }

  function moveWidget(source: WidgetKey, targetCol: "left" | "right", targetKey?: WidgetKey) {
    const nextLeft = widgetLayout.left.filter((k) => k !== source);
    const nextRight = widgetLayout.right.filter((k) => k !== source);
    const dest = targetCol === "left" ? nextLeft : nextRight;
    if (targetKey) {
      const idx = dest.indexOf(targetKey);
      if (idx >= 0) dest.splice(idx, 0, source);
      else dest.push(source);
    } else {
      dest.push(source);
    }
    saveLayout({ left: nextLeft, right: nextRight });
  }

  function isWidgetVisible(key: WidgetKey): boolean {
    if (!visibleWidgets[key]) return false;
    if (key === "cashFlow") return homeCashFlows.length > 0;
    if (key === "budget") return budgetAccounts.length > 0;
    if (key === "goals") return goalSummaries.length > 0;
    if (key === "recurring") return recurringItems.length > 0;
    return true;
  }

  function toggleWidget(key: WidgetKey) {
    visibleWidgets = { ...visibleWidgets, [key]: !visibleWidgets[key] };
    try {
      localStorage.setItem("paisa-v2-dashboard-widgets", JSON.stringify(visibleWidgets));
    } catch {}
  }

  function applyPreset(preset: "default" | "all") {
    visibleWidgets =
      preset === "all"
        ? {
            networth: true,
            checking: true,
            investmentIncome: true,
            cashFlow: true,
            expenses: true,
            recentTransactions: true,
            budget: true,
            goals: true,
            recurring: true
          }
        : { ...defaultVisible };
    try {
      localStorage.setItem("paisa-v2-dashboard-widgets", JSON.stringify(visibleWidgets));
    } catch {}
  }

  async function initDemo() {
    await fetch("/api/init", { method: "POST" });
    await triggerSync({ journal: true });
    await loadDashboard();
  }

  const networth = $derived(dashboard?.networth?.networth);
  const xirr = $derived(dashboard?.networth?.xirr ?? 0);
  const checkingAccounts = $derived(
    Object.values(dashboard?.checkingBalances?.asset_breakdowns || {}).slice(
      0,
      4
    ) as AssetBreakdown[]
  );
  const ttmDividend = $derived(dashboard?.investmentIncome?.ttm_dividend || 0);
  const ttmInterest = $derived(dashboard?.investmentIncome?.ttm_interest || 0);
  const homeCashFlows = $derived(
    (dashboard?.cashFlows || []).slice(-6).map((c) => ({
      date: c.date,
      income: Math.abs(c.income || 0),
      expenses: Math.abs(c.expenses || 0),
      investment: Math.abs(c.investment || 0),
      balance: c.balance || 0
    }))
  );
  const monthExpenses = $derived(dashboard?.expenses?.[selectedMonth] || []);
  const totalExpenses = $derived(monthExpenses.reduce((sum, p) => sum + (p.amount || 0), 0));
  const recentTransactions = $derived((dashboard?.transactions || []).slice(0, 8));
  const budgetAccounts = $derived(
    dashboard?.budget?.budgetsByMonth?.[selectedMonth]?.accounts?.slice(0, 5) || []
  );
  const goalSummaries = $derived((dashboard?.goalSummaries || []).slice(0, 4));
  const recurringItems = $derived((dashboard?.transactionSequences || []).slice(0, 5));
  const isEmpty = $derived(dashboard !== null && (dashboard.transactions || []).length === 0);

  const topCategories = $derived.by(() => {
    const byCategory = new Map<string, number>();
    for (const p of monthExpenses) {
      const top = p.account.split(":").slice(0, 2).join(":") || p.account;
      byCategory.set(top, (byCategory.get(top) || 0) + (p.amount || 0));
    }
    return Array.from(byCategory.entries())
      .map(([name, amount]) => ({ name, amount }))
      .sort((a, b) => b.amount - a.amount)
      .slice(0, 4);
  });
</script>

<div class="space-y-6">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Dashboard</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5">
        Financial snapshot as of {dayjs().format("DD MMM YYYY")}
      </p>
    </div>

    <button
      type="button"
      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
      onclick={() => (showCustomize = true)}
    >
      <SlidersHorizontal class="w-3.5 h-3.5" />
      <span>Customize</span>
    </button>
  </div>

  {#if healthIssues > 0}
    <div
      class="rounded-xl border border-[var(--warning)]/30 bg-[var(--warning)]/10 px-4 py-3 flex items-center justify-between gap-3 text-sm"
    >
      <div class="flex items-center gap-2.5">
        <AlertTriangle class="w-4 h-4 text-[var(--warning)] shrink-0" />
        <span>
          <strong>{healthIssues}</strong> data health item{healthIssues === 1 ? "" : "s"} need attention
          in your ledger.
        </span>
      </div>
      <a
        href={appPath("/ledger")}
        class="text-xs font-medium text-[var(--accent)] hover:underline shrink-0"
      >
        Review →
      </a>
    </div>
  {/if}

  {#if loading}
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      {#each [1, 2, 3, 4] as _}
        <div
          class="h-44 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
        ></div>
      {/each}
    </div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Could not load dashboard</p>
      <p class="text-xs text-[var(--muted)] mt-1">
        Make sure the Go backend is running on port 7500 ({error}).
      </p>
    </div>
  {:else if isEmpty}
    <div
      class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center space-y-3"
    >
      <h2 class="text-base font-semibold">Welcome to Paisa</h2>
      <p class="text-sm text-[var(--muted)] max-w-md mx-auto">
        No transactions found yet. Add your first transaction or load the sample demo ledger.
      </p>
      <div class="pt-1">
        <button
          type="button"
          class="px-4 py-2 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] cursor-pointer"
          onclick={initDemo}
        >
          Load Demo Setup
        </button>
      </div>
    </div>
  {:else}
    {#snippet widgetCard(key: WidgetKey, col: "left" | "right")}
      <div
        role="region"
        aria-label={widgetLabels[key]}
        draggable="true"
        ondragstart={(e) => {
          draggingKey = key;
          e.dataTransfer?.setData("text/plain", key);
        }}
        ondragend={() => {
          draggingKey = null;
          dragOverKey = null;
        }}
        ondragover={(e) => {
          e.preventDefault();
          if (draggingKey && draggingKey !== key) dragOverKey = key;
        }}
        ondragleave={() => {
          if (dragOverKey === key) dragOverKey = null;
        }}
        ondrop={(e) => {
          e.preventDefault();
          e.stopPropagation();
          if (draggingKey && draggingKey !== key) {
            moveWidget(draggingKey, col, key);
          }
          draggingKey = null;
          dragOverKey = null;
        }}
        class="relative group transition-all {draggingKey === key
          ? 'opacity-50'
          : ''} {dragOverKey === key ? 'ring-2 ring-[var(--accent)] rounded-xl' : ''}"
      >
        <div
          class="absolute top-3.5 right-3.5 z-10 p-1 rounded text-[var(--muted)] opacity-40 group-hover:opacity-100 hover:bg-[var(--surface-elevated)] cursor-grab active:cursor-grabbing"
          title="Drag to arrange widget"
        >
          <GripVertical class="w-3.5 h-3.5" />
        </div>

        {#if key === "networth"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Net Worth
              </span>
              <a
                href={appPath("/wealth")}
                class="inline-flex items-center gap-1 text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                <span>Wealth</span>
                <ArrowUpRight class="w-3.5 h-3.5" />
              </a>
            </div>
            <div class="grid grid-cols-2 sm:grid-cols-4 gap-4">
              <div class="sm:col-span-2">
                <div class="text-xs text-[var(--muted)]">Total Net Worth</div>
                <div class="text-2xl font-semibold tabular-nums mt-0.5">
                  {formatMoney(networth?.balanceAmount || 0)}
                </div>
              </div>
              <div>
                <div class="text-xs text-[var(--muted)]">Gain</div>
                <div
                  class="text-base font-semibold tabular-nums mt-1 {(networth?.gainAmount || 0) >= 0
                    ? 'text-[var(--gain)]'
                    : 'text-[var(--loss)]'}"
                >
                  {formatMoney(networth?.gainAmount || 0)}
                </div>
              </div>
              <div>
                <div class="text-xs text-[var(--muted)]">XIRR</div>
                <div class="text-base font-semibold tabular-nums mt-1">{xirr.toFixed(1)}%</div>
              </div>
            </div>
          </section>
        {:else if key === "checking"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Checking Balances
              </span>
              <a
                href={appPath("/wealth")}
                class="inline-flex items-center gap-1 text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                <span>All accounts</span>
                <ArrowUpRight class="w-3.5 h-3.5" />
              </a>
            </div>
            {#if checkingAccounts.length === 0}
              <p class="text-sm text-[var(--muted)]">No checking accounts found.</p>
            {:else}
              <div class="grid grid-cols-1 sm:grid-cols-2 gap-3">
                {#each checkingAccounts as account}
                  <div
                    class="rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] px-3.5 py-2.5 flex items-center justify-between gap-2"
                  >
                    <span class="text-xs font-medium truncate" title={account.group}>
                      {account.group.split(":").slice(1).join(":") || account.group}
                    </span>
                    <div class="text-sm font-semibold tabular-nums shrink-0 text-right">
                      {#if account.originalBalances && account.originalBalances.length > 0}
                        {#each account.originalBalances as ob}
                          <div>{formatMoney(ob.amount, ob.currency)}</div>
                        {/each}
                      {:else}
                        {formatMoney(account.marketAmount)}
                      {/if}
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </section>
        {:else if key === "investmentIncome"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Investment Income (TTM)
              </span>
              <a
                href={appPath("/wealth")}
                class="inline-flex items-center gap-1 text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                <span>Details</span>
                <ArrowUpRight class="w-3.5 h-3.5" />
              </a>
            </div>
            <div class="grid grid-cols-2 gap-4">
              <div>
                <div class="text-xs text-[var(--muted)]">TTM Dividend</div>
                <div class="text-lg font-semibold tabular-nums text-[var(--gain)] mt-0.5">
                  {formatMoney(ttmDividend)}
                </div>
              </div>
              <div>
                <div class="text-xs text-[var(--muted)]">TTM Interest</div>
                <div class="text-lg font-semibold tabular-nums text-[var(--gain)] mt-0.5">
                  {formatMoney(ttmInterest)}
                </div>
              </div>
            </div>
          </section>
        {:else if key === "cashFlow"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Cash Flow
              </span>
              <a
                href={appPath("/cash-flow")}
                class="text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                Details →
              </a>
            </div>
            {#if homeCashFlows.length > 0}
              {@const latestCf = homeCashFlows[homeCashFlows.length - 1]}
              {@const latestNet = latestCf.income - latestCf.expenses}
              <div class="grid grid-cols-3 gap-3 text-xs tabular-nums pb-1">
                <div>
                  <div class="text-[var(--muted)]">Last Month In</div>
                  <div class="font-semibold text-[var(--gain)] mt-0.5">
                    {formatMoney(latestCf.income)}
                  </div>
                </div>
                <div>
                  <div class="text-[var(--muted)]">Last Month Out</div>
                  <div class="font-semibold text-[var(--loss)] mt-0.5">
                    {formatMoney(latestCf.expenses)}
                  </div>
                </div>
                <div>
                  <div class="text-[var(--muted)]">Net Flow</div>
                  <div
                    class="font-semibold {latestNet >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'} mt-0.5"
                  >
                    {formatMoney(latestNet)}
                  </div>
                </div>
              </div>
            {/if}
            <BarTrendChart data={homeCashFlows} />
          </section>
        {:else if key === "budget"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Budget ({dayjs(`${selectedMonth}-01`).format("MMM YYYY")})
              </span>
              <a
                href={appPath("/cash-flow")}
                class="text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                Manage →
              </a>
            </div>
            <div class="space-y-2.5">
              {#each budgetAccounts as b}
                {@const pct = b.forecast > 0 ? Math.min(100, (b.actual / b.forecast) * 100) : 0}
                <div class="space-y-1 text-xs">
                  <div class="flex justify-between">
                    <span class="truncate">{b.account.replace(/^Expenses:/, "")}</span>
                    <span
                      class="tabular-nums {b.available >= 0
                        ? 'text-[var(--gain)]'
                        : 'text-[var(--loss)]'}"
                    >
                      {formatMoney(b.available)}
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
                </div>
              {/each}
            </div>
          </section>
        {:else if key === "goals"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Goals
              </span>
              <a
                href={appPath("/planning")}
                class="text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                View all →
              </a>
            </div>
            <div class="space-y-3">
              {#each goalSummaries as g}
                {@const pct = g.target > 0 ? Math.min(100, (g.current / g.target) * 100) : 0}
                <div class="space-y-1 text-xs">
                  <div class="flex justify-between">
                    <span class="font-medium">{g.name}</span>
                    <span class="tabular-nums">{pct.toFixed(0)}%</span>
                  </div>
                  <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                    <div class="h-full rounded-full bg-[var(--accent)]" style="width: {pct}%"></div>
                  </div>
                </div>
              {/each}
            </div>
          </section>
        {:else if key === "expenses"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4"
          >
            <div class="flex flex-wrap items-center justify-between gap-2 pr-7">
              <div>
                <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                  Expenses ({dayjs(`${selectedMonth}-01`).format("MMM YYYY")})
                </span>
                <div class="text-xl font-semibold tabular-nums text-[var(--loss)] mt-0.5">
                  {formatMoney(totalExpenses)}
                </div>
              </div>
              <div
                class="inline-flex rounded-lg border border-[var(--border)] bg-[var(--surface-elevated)] p-0.5 text-xs"
              >
                {#each last3Months as m}
                  <button
                    type="button"
                    class="px-2 py-1 rounded-md cursor-pointer {selectedMonth === m
                      ? 'bg-[var(--surface)] font-semibold'
                      : 'text-[var(--muted)]'}"
                    onclick={() => (selectedMonth = m)}
                  >
                    {dayjs(`${m}-01`).format("MMM")}
                  </button>
                {/each}
              </div>
            </div>
            {#if topCategories.length === 0}
              <p class="text-sm text-[var(--muted)]">No expenses recorded this month.</p>
            {:else}
              <div class="space-y-2.5">
                {#each topCategories as cat}
                  {@const pct =
                    totalExpenses > 0 ? Math.min(100, (cat.amount / totalExpenses) * 100) : 0}
                  <div class="space-y-1">
                    <div class="flex justify-between text-xs">
                      <span class="truncate">{cat.name.replace(/^Expenses:/, "")}</span>
                      <span class="tabular-nums font-medium">{formatMoney(cat.amount)}</span>
                    </div>
                    <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                      <div
                        class="h-full rounded-full bg-[var(--accent)]"
                        style="width: {pct}%"
                      ></div>
                    </div>
                  </div>
                {/each}
              </div>
            {/if}
          </section>
        {:else if key === "recurring"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Recurring
              </span>
              <a
                href={appPath("/cash-flow")}
                class="text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                View all →
              </a>
            </div>
            <div class="divide-y divide-[var(--border)] text-xs">
              {#each recurringItems as item}
                <div class="py-2 flex items-center justify-between gap-2">
                  <span class="font-medium truncate">
                    {item.name || item.payee || item.narration || item.account}
                  </span>
                  <span class="text-[var(--muted)]">{item.next_date || item.frequency || ""}</span>
                </div>
              {/each}
            </div>
          </section>
        {:else if key === "recentTransactions"}
          <section
            class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3"
          >
            <div class="flex items-center justify-between pr-7">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Recent Transactions
              </span>
              <a
                href={appPath("/transactions")}
                class="inline-flex items-center gap-1 text-xs text-[var(--muted)] hover:text-[var(--text)]"
              >
                <span>View all</span>
                <ArrowUpRight class="w-3.5 h-3.5" />
              </a>
            </div>
            {#if recentTransactions.length === 0}
              <p class="text-sm text-[var(--muted)]">No recent transactions.</p>
            {:else}
              <div class="divide-y divide-[var(--border)]">
                {#each recentTransactions as tx (tx.id)}
                  <button
                    type="button"
                    class="w-full py-2.5 text-left hover:bg-[var(--surface-elevated)]/50 px-1.5 -mx-1.5 rounded-lg transition-colors cursor-pointer space-y-1.5"
                    onclick={() => (selectedTransaction = tx)}
                  >
                    <div class="flex items-baseline justify-between gap-2">
                      <span class="text-sm font-medium truncate">
                        {tx.payee || tx.narration || "Transaction"}
                      </span>
                      <span class="text-xs text-[var(--muted)] shrink-0 tabular-nums">
                        {formatShortDate(tx.date)}
                      </span>
                    </div>
                    <div class="space-y-0.5">
                      {#each tx.postings as posting}
                        <div class="flex items-baseline justify-between gap-2 text-xs">
                          <span class="text-[var(--muted)] truncate" title={posting.account}>
                            {posting.account}
                          </span>
                          <span
                            class="tabular-nums font-medium shrink-0 {posting.amount < 0
                              ? 'text-[var(--loss)]'
                              : ''}"
                          >
                            {#if posting.commodity && posting.commodity !== runtime.defaultCurrency && posting.quantity !== undefined}
                              <span>{formatMoney(posting.quantity, posting.commodity)}</span>
                              <span class="text-[var(--muted)] font-normal ml-1">
                                ({formatMoney(posting.amount)})
                              </span>
                            {:else}
                              {formatMoney(posting.amount)}
                            {/if}
                          </span>
                        </div>
                      {/each}
                    </div>
                  </button>
                {/each}
              </div>
            {/if}
          </section>
        {/if}
      </div>
    {/snippet}

    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <div
        role="list"
        class="lg:col-span-6 space-y-5 min-h-[120px]"
        ondragover={(e) => e.preventDefault()}
        ondrop={(e) => {
          e.preventDefault();
          if (draggingKey) moveWidget(draggingKey, "left");
          draggingKey = null;
          dragOverKey = null;
        }}
      >
        {#each widgetLayout.left.filter(isWidgetVisible) as key (key)}
          {@render widgetCard(key, "left")}
        {/each}
      </div>

      <div
        role="list"
        class="lg:col-span-6 space-y-5 min-h-[120px]"
        ondragover={(e) => e.preventDefault()}
        ondrop={(e) => {
          e.preventDefault();
          if (draggingKey) moveWidget(draggingKey, "right");
          draggingKey = null;
          dragOverKey = null;
        }}
      >
        {#each widgetLayout.right.filter(isWidgetVisible) as key (key)}
          {@render widgetCard(key, "right")}
        {/each}
      </div>
    </div>
  {/if}
</div>

{#if showCustomize}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <button
      type="button"
      class="fixed inset-0 bg-black/40"
      aria-label="Close customize modal"
      onclick={() => (showCustomize = false)}
    ></button>
    <div
      class="relative z-10 w-full max-w-md rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl p-5 space-y-4"
    >
      <div class="flex items-center justify-between">
        <h2 class="text-base font-semibold">Customize Dashboard</h2>
        <button
          type="button"
          class="p-1 rounded-md text-[var(--muted)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() => (showCustomize = false)}
        >
          <X class="w-4 h-4" />
        </button>
      </div>
      <div class="flex gap-2 text-xs">
        <button
          type="button"
          class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() => applyPreset("default")}
        >
          Default (5)
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() => applyPreset("all")}
        >
          Show All
        </button>
        <button
          type="button"
          class="px-2.5 py-1 rounded-md border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() =>
            saveLayout({
              left: [...defaultLayout.left],
              right: [...defaultLayout.right]
            })}
        >
          Reset Order
        </button>
      </div>
      <div class="space-y-2 text-sm">
        {#each Object.entries(widgetLabels) as [key, label]}
          <label class="flex items-center justify-between py-1 cursor-pointer">
            <span>{label}</span>
            <input
              type="checkbox"
              checked={visibleWidgets[key as WidgetKey]}
              onchange={() => toggleWidget(key as WidgetKey)}
            />
          </label>
        {/each}
      </div>
    </div>
  </div>
{/if}

<TransactionDrawer bind:transaction={selectedTransaction} onSaved={loadDashboard} />
