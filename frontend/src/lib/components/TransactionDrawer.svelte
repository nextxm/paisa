<script lang="ts">
  import { Plus, Trash2, X } from "lucide-svelte";
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

  let tab = $state<"details" | "structured" | "raw">("details");
  let rawText = $state("");
  let editDate = $state("");
  let editPayee = $state("");
  let editNarration = $state("");
  let editPostings = $state<Array<{ account: string; amount: string; commodity: string }>>([]);

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
      editDate = transaction.date || "";
      editPayee = transaction.payee || "";
      editNarration = transaction.narration || "";
      editPostings = (transaction.postings || []).map((p) => ({
        account: p.account,
        amount: String(p.amount),
        commodity: p.commodity || runtime.defaultCurrency || currency
      }));
    }
  });

  async function ensureRawLoaded() {
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

  function serializeStructured(): string {
    const header = `${editDate} ${editPayee || editNarration || "Transaction"}`.trim();
    const lines = editPostings
      .filter((p) => p.account.trim())
      .map((p) => `  ${p.account.trim()}  ${p.amount.trim()} ${p.commodity.trim()}`.trimEnd());
    return [header, ...lines].join("\n");
  }

  async function savePatch(action: "replace" | "delete") {
    if (!transaction?.fileName) return;
    saving = true;
    error = null;
    const content =
      action === "delete"
        ? ""
        : tab === "structured"
          ? serializeStructured()
          : rawText;

    try {
      const res = await fetch("/api/transaction/patch", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          file_name: transaction.fileName,
          begin_line: transaction.beginLine,
          end_line: transaction.endLine,
          action,
          content
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
            class="py-2.5 border-b-2 cursor-pointer {tab === 'structured'
              ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
              : 'border-transparent text-[var(--muted)]'}"
            onclick={() => (tab = "structured")}
          >
            Edit Fields
          </button>
          <button
            type="button"
            class="py-2.5 border-b-2 cursor-pointer {tab === 'raw'
              ? 'border-[var(--accent)] font-semibold text-[var(--text)]'
              : 'border-transparent text-[var(--muted)]'}"
            onclick={() => {
              tab = "raw";
              void ensureRawLoaded();
            }}
          >
            Raw Ledger
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
        {:else if tab === "structured"}
          {#if error}
            <div
              class="rounded-lg bg-[var(--loss)]/10 border border-[var(--loss)]/30 px-3 py-2 text-xs text-[var(--loss)]"
            >
              {error}
            </div>
          {/if}

          <div class="space-y-3 text-xs">
            <div class="grid grid-cols-2 gap-2">
              <div>
                <label class="block text-[var(--muted)] mb-1" for="ed-date">Date</label>
                <input
                  id="ed-date"
                  type="date"
                  bind:value={editDate}
                  class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
                />
              </div>
              <div>
                <label class="block text-[var(--muted)] mb-1" for="ed-payee">Payee</label>
                <input
                  id="ed-payee"
                  type="text"
                  bind:value={editPayee}
                  class="w-full px-2.5 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
                />
              </div>
            </div>

            <div class="space-y-2">
              <div class="text-[var(--muted)] font-medium">Postings</div>
              {#each editPostings as p, i}
                <div class="flex items-center gap-1.5">
                  <input
                    list="drawer-accounts"
                    bind:value={p.account}
                    class="flex-1 px-2 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)]"
                  />
                  <input
                    type="text"
                    bind:value={p.amount}
                    class="w-20 px-2 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums text-right"
                  />
                  <button
                    type="button"
                    class="p-1 text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
                    onclick={() => (editPostings = editPostings.filter((_, idx) => idx !== i))}
                    aria-label="Remove posting"
                  >
                    <Trash2 class="w-3.5 h-3.5" />
                  </button>
                </div>
              {/each}
              <button
                type="button"
                class="inline-flex items-center gap-1 text-[var(--accent)] cursor-pointer"
                onclick={() =>
                  (editPostings = [
                    ...editPostings,
                    { account: "", amount: "0", commodity: runtime.defaultCurrency }
                  ])}
              >
                <Plus class="w-3.5 h-3.5" />
                <span>Add posting</span>
              </button>
              <datalist id="drawer-accounts">
                {#each runtime.accounts as acc}
                  <option value={acc}></option>
                {/each}
              </datalist>
            </div>

            <div class="flex items-center justify-between pt-2">
              <button
                type="button"
                disabled={saving}
                class="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-lg font-medium text-[var(--loss)] border border-[var(--loss)]/30 hover:bg-[var(--loss)]/10 cursor-pointer"
                onclick={() => savePatch("delete")}
              >
                <Trash2 class="w-3.5 h-3.5" />
                <span>Delete</span>
              </button>
              <button
                type="button"
                disabled={saving}
                class="px-4 py-1.5 rounded-lg font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 cursor-pointer"
                onclick={() => savePatch("replace")}
              >
                {saving ? "Saving…" : "Save Changes"}
              </button>
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
                  onclick={() => savePatch("delete")}
                >
                  <Trash2 class="w-3.5 h-3.5" />
                  <span>Delete</span>
                </button>
                <button
                  type="button"
                  disabled={saving}
                  class="px-4 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
                  onclick={() => savePatch("replace")}
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
