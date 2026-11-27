<script lang="ts">
  import dayjs from "dayjs";
  import { onMount } from "svelte";
  import { AlertTriangle, ArrowUpRight } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import {
    apiGet,
    formatMoney,
    formatShortDate,
    type AssetBreakdown,
    type DashboardResponse,
    type Transaction
  } from "$lib/api";

  let dashboard = $state<DashboardResponse | null>(null);
  let healthIssues = $state(0);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedTransaction = $state<Transaction | null>(null);

  const currentMonth = dayjs().format("YYYY-MM");

  onMount(async () => {
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
  });

  const networth = $derived(dashboard?.networth?.networth);
  const xirr = $derived(dashboard?.networth?.xirr ?? 0);
  const checkingAccounts = $derived(
    Object.values(dashboard?.checkingBalances?.asset_breakdowns || {}).slice(0, 4) as AssetBreakdown[]
  );
  const monthExpenses = $derived(dashboard?.expenses?.[currentMonth] || []);
  const totalExpenses = $derived(monthExpenses.reduce((sum, p) => sum + (p.amount || 0), 0));
  const recentTransactions = $derived((dashboard?.transactions || []).slice(0, 8));

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
  {:else}
    <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
      <!-- Left Column -->
      <div class="lg:col-span-6 space-y-5">
        <!-- Net Worth Card -->
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

        <!-- Checking Balances -->
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
      </div>

      <!-- Right Column -->
      <div class="lg:col-span-6 space-y-5">
        <!-- Monthly Expenses -->
        <section class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-5 space-y-4">
          <div class="flex items-center justify-between">
            <div>
              <span class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
                Expenses ({dayjs().format("MMM YYYY")})
              </span>
              <div class="text-xl font-semibold tabular-nums text-[var(--loss)] mt-0.5">
                {formatMoney(totalExpenses)}
              </div>
            </div>
            <a
              href="/cash-flow"
              class="inline-flex items-center gap-1 text-xs text-[var(--muted)] hover:text-[var(--text)]"
            >
              <span>Breakdown</span>
              <ArrowUpRight class="w-3.5 h-3.5" />
            </a>
          </div>

          {#if topCategories.length === 0}
            <p class="text-sm text-[var(--muted)]">No expenses recorded this month.</p>
          {:else}
            <div class="space-y-2.5">
              {#each topCategories as cat}
                {@const pct = totalExpenses > 0 ? Math.min(100, (cat.amount / totalExpenses) * 100) : 0}
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

        <!-- Recent Transactions -->
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
      </div>
    </div>
  {/if}
</div>

<TransactionDrawer bind:transaction={selectedTransaction} />
