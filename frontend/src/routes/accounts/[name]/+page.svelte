<script lang="ts">
  import { page } from "$app/stores";
  import { onMount } from "svelte";
  import { ArrowLeft, StickyNote, Trash2 } from "lucide-svelte";
  import { apiGet, appPath, authHeaders, formatMoney, formatShortDate, runtime, type Transaction } from "$lib/api";

  const accountName = $derived(decodeURIComponent($page.params.name || ""));

  let loading = $state(true);
  let error = $state<string | null>(null);
  let transactions = $state<Transaction[]>([]);
  let noteText = $state("");
  let noteSaving = $state(false);
  let noteStatus = $state<string | null>(null);

  async function loadAccountData() {
    if (!accountName) return;
    loading = true;
    error = null;
    try {
      const [txRes, noteRes] = await Promise.all([
        apiGet<{ transactions?: Transaction[] }>(
          `/api/transaction?account=${encodeURIComponent(accountName)}`
        ),
        apiGet<{ account_note?: { note?: string } | null }>(
          `/api/account_notes/${encodeURIComponent(accountName)}`
        ).catch(() => ({ account_note: null }))
      ]);
      transactions = (txRes.transactions || []).slice().reverse();
      noteText = noteRes.account_note?.note || "";
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load account";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadAccountData();
  });

  $effect(() => {
    const _acc = accountName;
    if (_acc) void loadAccountData();
  });

  async function saveNote() {
    noteSaving = true;
    noteStatus = null;
    try {
      const res = await fetch("/api/account_notes/upsert", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ account: accountName, note: noteText })
      });
      noteStatus = res.ok ? "Note saved." : "Failed to save note.";
    } finally {
      noteSaving = false;
    }
  }

  async function deleteNote() {
    noteSaving = true;
    noteStatus = null;
    try {
      const res = await fetch("/api/account_notes/delete", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...authHeaders() },
        body: JSON.stringify({ account: accountName })
      });
      if (res.ok) {
        noteText = "";
        noteStatus = "Note deleted.";
      }
    } finally {
      noteSaving = false;
    }
  }

  const netFlow = $derived.by(() => {
    let total = 0;
    for (const tx of transactions) {
      for (const p of tx.postings || []) {
        if (p.account === accountName) {
          total += p.amount || 0;
        }
      }
    }
    return total;
  });
</script>

<div class="space-y-5">
  <div class="flex flex-wrap items-center justify-between gap-3">
    <div>
      <a
        href={appPath("/transactions")}
        class="inline-flex items-center gap-1.5 text-xs text-[var(--muted)] hover:text-[var(--text)] mb-1"
      >
        <ArrowLeft class="w-3.5 h-3.5" />
        All Transactions
      </a>
      <h1 class="text-xl font-semibold tracking-tight break-all">{accountName}</h1>
    </div>
    <a
      href={appPath(`/transactions?account=${encodeURIComponent(accountName)}`)}
      class="px-3 py-1.5 rounded-lg border border-[var(--border)] bg-[var(--surface)] text-xs font-medium hover:border-[var(--accent)]"
    >
      Filter in Transactions
    </a>
  </div>

  <div class="grid grid-cols-1 lg:grid-cols-12 gap-5">
    <div class="lg:col-span-4 space-y-4">
      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4">
        <div class="text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
          Transactions ({transactions.length})
        </div>
        <div class="text-2xl font-semibold tabular-nums mt-1">{formatMoney(netFlow)}</div>
        <p class="text-xs text-[var(--muted)] mt-0.5">Cumulative posting total</p>
      </div>

      <div class="rounded-xl bg-[var(--surface)] border border-[var(--border)] p-4 space-y-3">
        <div class="flex items-center justify-between">
          <span class="inline-flex items-center gap-1.5 text-xs font-semibold uppercase tracking-wider text-[var(--muted)]">
            <StickyNote class="w-3.5 h-3.5" />
            Account Note
          </span>
          {#if !runtime.readonly && noteText}
            <button
              type="button"
              onclick={deleteNote}
              class="text-xs text-[var(--loss)] hover:underline inline-flex items-center gap-1 cursor-pointer"
            >
              <Trash2 class="w-3 h-3" />
              Delete
            </button>
          {/if}
        </div>
        <textarea
          bind:value={noteText}
          readonly={runtime.readonly}
          rows="4"
          placeholder="Add notes, IFSC/account metadata, or reminders…"
          class="w-full p-2.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] text-xs outline-none focus:border-[var(--accent)]"
        ></textarea>
        {#if !runtime.readonly}
          <div class="flex items-center justify-between gap-2">
            <span class="text-xs text-[var(--muted)]">{noteStatus || ""}</span>
            <button
              type="button"
              disabled={noteSaving}
              onclick={saveNote}
              class="px-3 py-1.5 rounded-lg bg-[var(--accent)] text-[var(--accent-fg)] text-xs font-medium cursor-pointer disabled:opacity-50"
            >
              {noteSaving ? "Saving…" : "Save Note"}
            </button>
          </div>
        {/if}
      </div>
    </div>

    <div class="lg:col-span-8 rounded-xl bg-[var(--surface)] border border-[var(--border)] overflow-hidden">
      {#if loading}
        <div class="h-48 animate-pulse"></div>
      {:else if error}
        <div class="p-6 text-sm text-[var(--loss)]">{error}</div>
      {:else if transactions.length === 0}
        <div class="p-8 text-center text-sm text-[var(--muted)]">No transactions for this account.</div>
      {:else}
        <div class="overflow-x-auto">
          <table class="w-full text-sm text-left">
            <thead class="text-xs uppercase tracking-wider text-[var(--muted)] border-b border-[var(--border)] bg-[var(--surface-elevated)]/50">
              <tr>
                <th class="py-2.5 px-4 font-medium">Date</th>
                <th class="py-2.5 px-4 font-medium">Payee / Narration</th>
                <th class="py-2.5 px-4 text-right font-medium">Amount</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-[var(--border)] tabular-nums">
              {#each transactions.slice(0, 100) as tx}
                {@const amt = (tx.postings || []).find((p) => p.account === accountName)?.amount || 0}
                <tr class="hover:bg-[var(--surface-elevated)]/50">
                  <td class="py-2 px-4 text-xs text-[var(--muted)] whitespace-nowrap">
                    {formatShortDate(tx.date)}
                  </td>
                  <td class="py-2 px-4">
                    <div class="font-medium">{tx.payee || "—"}</div>
                    {#if tx.narration}
                      <div class="text-xs text-[var(--muted)]">{tx.narration}</div>
                    {/if}
                  </td>
                  <td
                    class="py-2 px-4 text-right font-semibold {amt >= 0
                      ? 'text-[var(--gain)]'
                      : 'text-[var(--loss)]'}"
                  >
                    {formatMoney(amt)}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </div>
  </div>
</div>
