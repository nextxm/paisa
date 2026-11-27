<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import { Bookmark, Download, Pencil, Search, X } from "lucide-svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import {
    apiGet,
    formatMoney,
    formatShortDate,
    runtime,
    triggerSync,
    type Transaction
  } from "$lib/api";

  interface SavedSearch {
    name: string;
    query: string;
    fromDate: string;
    toDate: string;
    accountFilter: string;
    tagFilter: string;
  }

  let transactions = $state<Transaction[]>([]);
  let query = $state("");
  let fromDate = $state("");
  let toDate = $state("");
  let accountFilter = $state($page.url.searchParams.get("account") || "");
  let tagFilter = $state("");
  let savedSearches = $state<SavedSearch[]>([]);

  let bulkOpen = $state(false);
  let bulkAction = $state<"rename_account" | "rename_payee">("rename_payee");
  let bulkFrom = $state("");
  let bulkTo = $state("");
  let bulkRunning = $state(false);

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

  onMount(() => {
    try {
      const raw = localStorage.getItem("paisa-v2-saved-searches");
      if (raw) savedSearches = JSON.parse(raw);
    } catch {}
    void loadTransactions();
  });

  const allTags = $derived.by(() => {
    const set = new Set<string>();
    for (const tx of transactions) {
      for (const t of tx.tags || []) set.add(t);
    }
    return Array.from(set).sort();
  });

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return transactions.filter((tx) => {
      if (fromDate && tx.date < fromDate) return false;
      if (toDate && tx.date > toDate) return false;
      if (accountFilter && !tx.postings.some((p) => p.account === accountFilter)) return false;
      if (tagFilter && !(tx.tags || []).includes(tagFilter)) return false;
      if (!q) return true;
      if (tx.payee?.toLowerCase().includes(q)) return true;
      if (tx.narration?.toLowerCase().includes(q)) return true;
      if (tx.date?.includes(q)) return true;
      return tx.postings.some((p) => p.account.toLowerCase().includes(q));
    });
  });

  const visible = $derived(filtered.slice(0, 250));

  function saveCurrentSearch() {
    const label = query || accountFilter || tagFilter || "Filtered View";
    const entry: SavedSearch = {
      name: label,
      query,
      fromDate,
      toDate,
      accountFilter,
      tagFilter
    };
    savedSearches = [entry, ...savedSearches.filter((s) => s.name !== label)].slice(0, 8);
    try {
      localStorage.setItem("paisa-v2-saved-searches", JSON.stringify(savedSearches));
    } catch {}
  }

  function applySavedSearch(s: SavedSearch) {
    query = s.query;
    fromDate = s.fromDate;
    toDate = s.toDate;
    accountFilter = s.accountFilter;
    tagFilter = s.tagFilter;
  }

  function removeSavedSearch(name: string) {
    savedSearches = savedSearches.filter((s) => s.name !== name);
    try {
      localStorage.setItem("paisa-v2-saved-searches", JSON.stringify(savedSearches));
    } catch {}
  }

  async function applyBulkReplace() {
    if (!bulkFrom.trim() || !bulkTo.trim()) return;
    bulkRunning = true;
    try {
      const filesRes = await apiGet<{ files?: Array<{ file_name: string; content: string }> }>(
        "/api/editor/files"
      );
      for (const f of filesRes.files || []) {
        if (!f.content.includes(bulkFrom)) continue;
        const updated = f.content.split(bulkFrom).join(bulkTo);
        await fetch("/api/editor/save", {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ file_name: f.file_name, content: updated })
        });
      }
      await triggerSync({ journal: true });
      bulkOpen = false;
      await loadTransactions();
    } finally {
      bulkRunning = false;
    }
  }

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

  function exportJSON() {
    const blob = new Blob([JSON.stringify(filtered, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = "paisa-transactions.json";
    link.click();
    URL.revokeObjectURL(url);
  }
</script>

<div class="space-y-4">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <h1 class="text-xl font-semibold tracking-tight">Transactions</h1>
      <p class="text-xs text-[var(--muted)] mt-0.5 tabular-nums">
        {filtered.length.toLocaleString()} matching transactions
      </p>
    </div>

    <div class="flex flex-wrap items-center gap-2">
      {#if !runtime.readonly}
        <button
          type="button"
          class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() => (bulkOpen = !bulkOpen)}
        >
          <Pencil class="w-3.5 h-3.5" />
          <span>Bulk Edit</span>
        </button>
      {/if}
      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
        onclick={exportCSV}
      >
        <Download class="w-3.5 h-3.5" />
        <span>CSV</span>
      </button>
      <button
        type="button"
        class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium bg-[var(--surface)] border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
        onclick={exportJSON}
      >
        <Download class="w-3.5 h-3.5" />
        <span>JSON</span>
      </button>
    </div>
  </div>

  {#if bulkOpen}
    <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3 text-xs">
      <div class="font-semibold">Bulk Replace Across Ledger</div>
      <div class="flex flex-wrap items-center gap-2">
        <select
          bind:value={bulkAction}
          class="px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        >
          <option value="rename_payee">Rename Payee / Text</option>
          <option value="rename_account">Rename Account</option>
        </select>
        <input
          type="text"
          bind:value={bulkFrom}
          placeholder="Find exact text…"
          class="flex-1 min-w-40 px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        />
        <input
          type="text"
          bind:value={bulkTo}
          placeholder="Replace with…"
          class="flex-1 min-w-40 px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
        />
        <button
          type="button"
          disabled={bulkRunning}
          class="px-3.5 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] font-medium cursor-pointer"
          onclick={applyBulkReplace}
        >
          {bulkRunning ? "Applying…" : "Apply"}
        </button>
      </div>
    </div>
  {/if}

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
      class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none max-w-48"
    >
      <option value="">All accounts</option>
      {#each runtime.accounts as acc}
        <option value={acc}>{acc}</option>
      {/each}
    </select>

    {#if allTags.length > 0}
      <select
        bind:value={tagFilter}
        aria-label="Filter by tag"
        class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
      >
        <option value="">All tags</option>
        {#each allTags as t}
          <option value={t}>#{t}</option>
        {/each}
      </select>
    {/if}

    {#if query || fromDate || toDate || accountFilter || tagFilter}
      <button
        type="button"
        class="inline-flex items-center gap-1 px-2.5 py-1.5 rounded-lg text-xs border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
        onclick={saveCurrentSearch}
      >
        <Bookmark class="w-3 h-3" />
        <span>Save</span>
      </button>
      <button
        type="button"
        class="px-2.5 py-1.5 rounded-lg text-xs text-[var(--muted)] hover:text-[var(--text)] cursor-pointer"
        onclick={() => {
          query = "";
          fromDate = "";
          toDate = "";
          accountFilter = "";
          tagFilter = "";
        }}
      >
        Clear
      </button>
    {/if}
  </div>

  {#if savedSearches.length > 0}
    <div class="flex flex-wrap items-center gap-1.5 text-xs">
      {#each savedSearches as s}
        <span class="inline-flex items-center gap-1 px-2.5 py-1 rounded-full bg-[var(--surface)] border border-[var(--border)]">
          <button type="button" class="cursor-pointer hover:text-[var(--accent)]" onclick={() => applySavedSearch(s)}>
            {s.name}
          </button>
          <button
            type="button"
            class="text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
            onclick={() => removeSavedSearch(s.name)}
            aria-label="Remove saved search"
          >
            <X class="w-3 h-3" />
          </button>
        </span>
      {/each}
    </div>
  {/if}

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
