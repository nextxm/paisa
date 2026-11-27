<script lang="ts">
  import { X } from "lucide-svelte";
  import { formatMoney, formatShortDate, type Transaction } from "$lib/api";

  let {
    transaction = $bindable<Transaction | null>(null),
    currency = "INR"
  }: {
    transaction: Transaction | null;
    currency?: string;
  } = $props();

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
        <div>
          <div class="text-xs font-medium text-[var(--muted)]">
            {formatShortDate(transaction.date)}
          </div>
          <h2 class="text-base font-semibold mt-0.5">
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

      <div class="flex-1 overflow-y-auto p-5 space-y-5">
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
      </div>
    </aside>
  </div>
{/if}
