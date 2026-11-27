<script lang="ts">
  import { onMount } from "svelte";
  import { Download, Search } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import { apiGet, formatMoney, formatShortDate, runtime, type Transaction } from "$lib/api";

  let transactions = $state<Transaction[]>([]);
  let query = $state("");
  let fromDate = $state("");
  let toDate = $state("");
  let accountFilter = $state("");
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedTransaction = $state<Transaction | null>(null);

  async function loadTransactions() {
    loading = true;
    error = null;
    try {
      const res = await apiGet<{ transactions?: Transaction[] }>("/api/transaction");
      transactions = res.transactions || [];
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load transactions";
    } finally {
      loading = false;
    }
  }

  onMount(loadTransactions);

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return transactions.filter((tx) => {
      if (fromDate && tx.date < fromDate) return false;
      if (toDate && tx.date > toDate) return false;
      if (accountFilter && !tx.postings.some((p) => p.account === accountFilter)) return false;
      if (!q) return true;
      if (tx.payee?.toLowerCase().includes(q)) return true;
      if (tx.narration?.toLowerCase().includes(q)) return true;
      if (tx.date?.includes(q)) return true;
      return tx.postings.some((p) => p.account.toLowerCase().includes(q));
    });
  });

  const visible = $derived(filtered.slice(0, 250));

  function exportCSV() {
    const rows = [
      ["Date", "Payee", "Narration", "Account", "Amount", "Commodity"].join(","),
      ...filtered.map((tx) => {
        const p = tx.postings[0];
        const esc = (s: string) => `"${(s || "").replaceAll('"', '""')}"`;
        return [
          tx.date,
          esc(tx.payee || ""),
          esc(tx.narration || ""),
          esc(p?.account || ""),
          p?.amount ?? 0,
          p?.commodity || runtime.defaultCurrency
        ].join(",");
      })
    ].join("\n");

    const blob = new Blob([rows], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "paisa-transactions.csv";
    link.click();
    URL.revokeObjectURL(url);
  }
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Transactions</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5 tabular-nums">
        {filtered.length.toLocaleString()} matching transactions
      </p>
    </div>

    <button
      type="button"
      class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
      onclick={exportCSV}
    >
      <Download class="w-3.5 h-3.5" />
      <span>Export CSV</span>
    </button>
  </div>

  <!-- Filter Bar -->
  <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-3 flex flex-wrap items-center gap-2.5">
    <div class="relative flex-1 min-w-48">
      <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
      <input
        type="search"
        bind:value={query}
        placeholder="Search payee, narration, or account…"
        class="w-full pl-9 pr-3 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
      />
    </div>

    <input
      type="date"
      bind:value={fromDate}
      aria-label="From date"
      class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
    />
    <input
      type="date"
      bind:value={toDate}
      aria-label="To date"
      class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
    />
    <select
      bind:value={accountFilter}
      aria-label="Filter by account"
      class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none max-w-52"
    >
      <option value="">All accounts</option>
      {#each runtime.accounts as acc}
        <option value={acc}>{acc}</option>
      {/each}
    </select>

    {#if query || fromDate || toDate || accountFilter}
      <button
        type="button"
        class="px-2.5 py-1.5 rounded-lg text-xs text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
        onclick={() => {
          query = "";
          fromDate = "";
          toDate = "";
          accountFilter = "";
        }}
      >
        Clear
      </button>
    {/if}
  </div>

  {#if loading}
    <div
      class="h-64 rounded-xl bg-[var(--surface)] border border-[var(--border)] animate-pulse"
    ></div>
  {:else if error}
    <div class="rounded-xl border border-[var(--loss)]/30 bg-[var(--loss)]/10 p-4 text-sm">
      <p class="font-medium text-[var(--loss)]">Could not load transactions</p>
      <p class="text-xs text-[var(--muted)] mt-1">{error}</p>
    </div>
  {:else}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      <div class="overflow-x-auto">
        <table class="w-full text-sm text-left">
          <thead
            class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50"
          >
            <tr>
              <th class="py-2.5 px-4 font-medium">Date</th>
              <th class="py-2.5 px-4 font-medium">Payee / Narration</th>
              <th class="py-2.5 px-4 font-medium">Account</th>
              <th class="py-2.5 px-4 text-right font-medium">Amount</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-[var(--border)] tabular-nums">
            {#each visible as tx (tx.id)}
              {@const primary = tx.postings[0]}
              <tr
                class="hover:bg-[var(--surface-elevated)]/50 transition-colors cursor-pointer"
                onclick={() => (selectedTransaction = tx)}
              >
                <td class="py-2.5 px-4 text-xs text-[var(--muted)] whitespace-nowrap">
                  {formatShortDate(tx.date)}
                </td>
                <td class="py-2.5 px-4 font-medium max-w-xs truncate">
                  {tx.payee || tx.narration || "—"}
                </td>
                <td class="py-2.5 px-4 text-xs text-[var(--muted)] max-w-xs truncate">
                  {primary?.account || ""}
                </td>
                <td
                  class="py-2.5 px-4 text-right font-medium whitespace-nowrap {(primary?.amount ||
                    0) < 0
                    ? 'text-[var(--loss)]'
                    : ''}"
                >
                  {primary ? formatMoney(primary.amount) : "—"}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}
</div>

<TransactionDrawer bind:transaction={selectedTransaction} onSaved={loadTransactions} />
