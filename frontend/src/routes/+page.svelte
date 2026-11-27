<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { AlertTriangle, ArrowUpRight, SlidersHorizontal, X } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import {
    apiGet,
    formatMoney,
    formatShortDate,
    triggerSync,
    type AssetBreakdown,
    type DashboardResponse,
    type Transaction
  } from "$lib/api";

  type WidgetKey =
    | "networth"
    | "checking"
    | "investmentIncome"
    | "expenses"
    | "recentTransactions"
    | "budget"
    | "goals"
    | "recurring";

  const widgetLabels: Record<WidgetKey, string> = {
    networth: "Net Worth",
    checking: "Checking Balances",
    investmentIncome: "Investment Income (TTM)",
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
    expenses: true,
    recentTransactions: true,
    budget: false,
    goals: false,
    recurring: false
  };

  let visibleWidgets = $state<Record<WidgetKey, boolean>>({ ...defaultVisible });
  let showCustomize = $state(false);
  let dashboard = $state<DashboardResponse | null>(null);
  let healthIssues = $state(0);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedTransaction = $state<Transaction | null>(null);
  let selectedMonth = $state(dayjs().format("YYYY-MM"));

  const last3Months = [2, 1, 0].map((offset) => dayjs().subtract(offset, "month").format("YYYY-MM"));

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
    } catch {}
    await loadDashboard();
  });

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
    Object.values(dashboard?.checkingBalances?.asset_breakdowns || {}).slice(0, 4) as AssetBreakdown[]
  );
  const ttmDividend = $derived(dashboard?.investmentIncome?.ttm_dividend || 0);
  const ttmInterest = $derived(dashboard?.investmentIncome?.ttm_interest || 0);
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
      <h1 class="text-xl font-semibold tracking-tight">Home</h1>
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
      <a href="/ledger" class="text-xs font-medium text-[var(--accent)] hover:underline shrink-0">
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
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-8 text-center space-y-3">
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
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <!-- Left Column -->
      <div class="lg:col-span-6 space-y-5">
        {#if visibleWidgets.networth}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Net Worth
              </span>
              <a
                href="/wealth"
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
        {/if}

        {#if visibleWidgets.checking}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Checking Balances
              </span>
              <a
                href="/wealth"
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
                    <span class="text-sm font-semibold tabular-nums shrink-0">
                      {formatMoney(account.marketAmount)}
                    </span>
                  </div>
                {/each}
              </div>
            {/if}
          </section>
        {/if}

        {#if visibleWidgets.investmentIncome}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Investment Income (TTM)
              </span>
              <a
                href="/wealth"
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
        {/if}

        {#if visibleWidgets.budget && budgetAccounts.length > 0}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Budget ({dayjs(`${selectedMonth}-01`).format("MMM YYYY")})
              </span>
              <a href="/cash-flow" class="text-xs text-[var(--muted)] hover:text-[var(--text)]">
                Manage →
              </a>
            </div>
            <div class="space-y-2.5">
              {#each budgetAccounts as b}
                {@const pct = b.forecast > 0 ? Math.min(100, (b.actual / b.forecast) * 100) : 0}
                <div class="space-y-1 text-xs">
                  <div class="flex justify-between">
                    <span class="truncate">{b.account.replace(/^Expenses:/, "")}</span>
                    <span class="tabular-nums {b.available >= 0 ? 'text-[var(--gain)]' : 'text-[var(--loss)]'}">
                      {formatMoney(b.available)}
                    </span>
                  </div>
                  <div class="h-1.5 rounded-full bg-[var(--surface-elevated)] overflow-hidden">
                    <div
                      class="h-full rounded-full {b.available >= 0 ? 'bg-[var(--gain)]' : 'bg-[var(--loss)]'}"
                      style="width: {pct}%"
                    ></div>
                  </div>
                </div>
              {/each}
            </div>
          </section>
        {/if}

        {#if visibleWidgets.goals && goalSummaries.length > 0}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Goals
              </span>
              <a href="/planning" class="text-xs text-[var(--muted)] hover:text-[var(--text)]">
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
        {/if}
      </div>

      <!-- Right Column -->
      <div class="lg:col-span-6 space-y-5">
        {#if visibleWidgets.expenses}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
            <div class="flex flex-wrap items-center justify-between gap-2">
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
        {/if}

        {#if visibleWidgets.recurring && recurringItems.length > 0}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Recurring
              </span>
              <a href="/cash-flow" class="text-xs text-[var(--muted)] hover:text-[var(--text)]">
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
        {/if}

        {#if visibleWidgets.recentTransactions}
          <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-3">
            <div class="flex items-center justify-between">
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Recent Transactions
              </span>
              <a
                href="/transactions"
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
                  {@const primaryPosting = tx.postings[0]}
                  <button
                    type="button"
                    class="w-full py-2.5 flex items-center justify-between gap-3 text-left hover:bg-[var(--surface-elevated)]/50 px-1.5 -mx-1.5 rounded-lg transition-colors cursor-pointer"
                    onclick={() => (selectedTransaction = tx)}
                  >
                    <div class="min-w-0">
                      <div class="text-sm font-medium truncate">
                        {tx.payee || tx.narration || "Transaction"}
                      </div>
                      <div class="text-xs text-[var(--muted)] truncate">
                        {formatShortDate(tx.date)} · {primaryPosting?.account || ""}
                      </div>
                    </div>
                    {#if primaryPosting}
                      <span
                        class="text-sm font-semibold tabular-nums shrink-0 {primaryPosting.amount < 0
                          ? 'text-[var(--loss)]'
                          : ''}"
                      >
                        {formatMoney(primaryPosting.amount)}
                      </span>
                    {/if}
                  </button>
                {/each}
              </div>
            {/if}
          </section>
        {/if}
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
        <h2 class="text-base font-semibold">Customize Home</h2>
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
