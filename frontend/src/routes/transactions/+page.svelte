<script lang="ts">
  import { onMount } from "svelte";
  import { Search } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import { apiGet, formatMoney, formatShortDate, type Transaction } from "$lib/api";

  let transactions = $state<Transaction[]>([]);
  let query = $state("");
  let loading = $state(true);
  let error = $state<string | null>(null);
  let selectedTransaction = $state<Transaction | null>(null);

  onMount(async () => {
    try {
      const res = await apiGet<{ transactions?: Transaction[] }>("/api/transaction");
      transactions = res.transactions || [];
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load transactions";
    } finally {
      loading = false;
    }
  });

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return transactions;
    return transactions.filter((tx) => {
      if (tx.payee?.toLowerCase().includes(q)) return true;
      if (tx.narration?.toLowerCase().includes(q)) return true;
      if (tx.date?.includes(q)) return true;
      return tx.postings.some((p) => p.account.toLowerCase().includes(q));
    });
  });

  const visible = $derived(filtered.slice(0, 200));
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Transactions</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5 tabular-nums">
        {filtered.length.toLocaleString()} matching transactions
      </p>
    </div>

    <div class="relative w-full sm:w-80">
      <Search class="w-4 h-4 text-[var(--muted)] absolute left-3 top-1/2 -translate-y-1/2" />
      <input
        type="search"
        bind:value={query}
        placeholder="Filter by payee, account, or date…"
        class="w-full pl-9 pr-3 py-1.5 rounded-lg text-sm bg-[var(--surface)] border border-[var(--border)] outline-none focus:border-[var(--accent)]"
      />
    </div>
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

<TransactionDrawer bind:transaction={selectedTransaction} />
