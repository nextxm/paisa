<script lang="ts">
  import { Trash2, X } from "lucide-svelte";
  import { formatMoney, formatShortDate, runtime, triggerSync, type Transaction } from "$lib/api";

  let {
    transaction = $bindable<Transaction | null>(null),
    currency = "INR",
    onSaved
  }: {
    transaction: Transaction | null;
    currency?: string;
    onSaved?: () => void;
  } = $props();

  let tab = $state<"details" | "edit">("details");
  let rawText = $state("");
  let loadingRaw = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);

  const canEdit = $derived(
    Boolean(
      !runtime.readonly &&
        transaction?.fileName &&
        transaction?.beginLine !== undefined &&
        transaction?.endLine !== undefined
    )
  );

  $effect(() => {
    if (transaction) {
      tab = "details";
      rawText = "";
      error = null;
    }
  });

  async function openEdit() {
    tab = "edit";
    if (!transaction?.fileName || rawText) return;
    loadingRaw = true;
    error = null;
    try {
      const params = new URLSearchParams({
        file_name: transaction.fileName,
        begin_line: String(transaction.beginLine ?? 0),
        end_line: String(transaction.endLine ?? 0)
      });
      const res = await fetch(`/api/transaction/raw?${params.toString()}`);
      if (res.ok) {
        const data = (await res.json()) as { raw?: string; content?: string };
        rawText = data.raw ?? data.content ?? "";
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to load raw transaction";
    } finally {
      loadingRaw = false;
    }
  }

  async function saveRaw(action: "replace" | "delete") {
    if (!transaction?.fileName) return;
    saving = true;
    error = null;
    try {
      const res = await fetch("/api/transaction/patch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          file_name: transaction.fileName,
          begin_line: transaction.beginLine,
          end_line: transaction.endLine,
          action,
          content: action === "replace" ? rawText : ""
        })
      });
      const data = (await res.json()) as { saved?: boolean; message?: string };
      if (!res.ok || data.saved === false) {
        error = data.message || "Failed to update transaction";
        return;
      }
      await triggerSync({ journal: true });
      transaction = null;
      onSaved?.();
    } catch (e) {
      error = e instanceof Error ? e.message : "Failed to update transaction";
    } finally {
      saving = false;
    }
  }

  function close() {
    transaction = null;
  }
</script>

{#if transaction}
  <div class="fixed inset-0 z-50 flex justify-end">
    <button
      type="button"
      class="fixed inset-0 bg-black/40 backdrop-blur-[1px]"
      aria-label="Close transaction drawer"
      onclick={close}
    ></button>

    <aside
      class="relative z-10 w-full max-w-md bg-[var(--surface)] border-l border-[var(--border)] h-full flex flex-col shadow-xl"
      aria-label="Transaction details"
    >
      <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--border)]">
        <div class="min-w-0">
          <div class="text-xs font-medium text-[var(--muted)]">
            {formatShortDate(transaction.date)}
          </div>
          <h2 class="text-base font-semibold mt-0.5 truncate">
            {transaction.payee || transaction.narration || "Transaction"}
          </h2>
        </div>
        <button
          type="button"
          class="p-1.5 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] hover:text-[var(--text)] cursor-pointer"
          onclick={close}
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      {#if canEdit}
        <div class="px-5 border-b border-[var(--border)] flex items-center gap-5 text-xs">
          <button
            type="button"
            class="py-2.5 border-b-2 cursor-pointer {tab === 'details'
              ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
              : 'border-transparent text-[var(--muted)]'}"
            onclick={() => (tab = "details")}
          >
            Details
          </button>
          <button
            type="button"
            class="py-2.5 border-b-2 cursor-pointer {tab === 'edit'
              ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
              : 'border-transparent text-[var(--muted)]'}"
            onclick={openEdit}
          >
            Edit Entry
          </button>
        </div>
      {/if}

      <div class="flex-1 overflow-y-auto p-5 space-y-5">
        {#if tab === "details"}
          {#if transaction.narration && transaction.narration !== transaction.payee}
            <div>
              <div class="text-xs font-medium uppercase tracking-wider text-[var(--muted)] mb-1">
                Narration
              </div>
              <p class="text-sm">{transaction.narration}</p>
            </div>
          {/if}

          {#if transaction.tags && transaction.tags.length > 0}
            <div>
              <div class="text-xs font-medium uppercase tracking-wider text-[var(--muted)] mb-1.5">
                Tags
              </div>
              <div class="flex flex-wrap gap-1.5">
                {#each transaction.tags as tag}
                  <span
                    class="px-2 py-0.5 rounded-md text-xs bg-[var(--surface-elevated)] border border-[var(--border)]"
                  >
                    #{tag}
                  </span>
                {/each}
              </div>
            </div>
          {/if}

          <div>
            <div class="text-xs font-medium uppercase tracking-wider text-[var(--muted)] mb-2">
              Postings
            </div>
            <div class="rounded-xl border border-[var(--border)] divide-y divide-[var(--border)]">
              {#each transaction.postings as posting}
                <div class="px-3.5 py-2.5 flex items-center justify-between gap-3 text-sm">
                  <div class="min-w-0">
                    <div class="font-medium truncate">{posting.account}</div>
                    {#if posting.note}
                      <div class="text-xs text-[var(--muted)] truncate">{posting.note}</div>
                    {/if}
                  </div>
                  <span
                    class="tabular-nums font-medium shrink-0 {posting.amount < 0
                      ? 'text-[var(--loss)]'
                      : 'text-[var(--gain)]'}"
                  >
                    {formatMoney(posting.amount, posting.commodity || currency)}
                  </span>
                </div>
              {/each}
            </div>
          </div>
        {:else}
          {#if error}
            <div
              class="rounded-lg bg-[var(--loss)]/10 border border-[var(--loss)]/30 px-3 py-2 text-xs text-[var(--loss)]"
            >
              {error}
            </div>
          {/if}

          {#if loadingRaw}
            <div class="h-40 rounded-lg bg-[var(--surface-elevated)] animate-pulse"></div>
          {:else}
            <div class="space-y-3">
              <div class="text-xs text-[var(--muted)]">
                {transaction.fileName}:{transaction.beginLine}–{transaction.endLine}
              </div>
              <textarea
                bind:value={rawText}
                rows="10"
                class="w-full p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] font-mono text-xs outline-none focus:border-[var(--accent)]"
              ></textarea>
              <div class="flex items-center justify-between gap-2 pt-1">
                <button
                  type="button"
                  disabled={saving}
                  class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-medium text-[var(--loss)] border border-[var(--loss)]/30 hover:bg-[var(--loss)]/10 cursor-pointer"
                  onclick={() => saveRaw("delete")}
                >
                  <Trash2 class="w-3.5 h-3.5" />
                  <span>Delete</span>
                </button>
                <button
                  type="button"
                  disabled={saving}
                  class="px-4 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
                  onclick={() => saveRaw("replace")}
                >
                  {saving ? "Saving…" : "Save Changes"}
                </button>
              </div>
            </div>
          {/if}
        {/if}
      </div>
    </aside>
  </div>
{/if}
