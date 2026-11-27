<script lang="ts">
  import dayjs from "dayjs";
  import { X } from "lucide-svelte";
  import { runtime, triggerSync } from "$lib/api";

  let {
    open = $bindable(false),
    onSaved
  }: {
    open: boolean;
    onSaved?: () => void;
  } = $props();

  let date = $state(dayjs().format("YYYY-MM-DD"));
  let payee = $state("");
  let narration = $state("");
  let fromAccount = $state("");
  let toAccount = $state("");
  let amount = $state("");
  let commodity = $state(runtime.defaultCurrency || "INR");
  let saving = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    if (open) {
      commodity = runtime.defaultCurrency || "INR";
      error = null;
    }
  });

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!fromAccount.trim() || !toAccount.trim() || !amount.trim()) {
      error = "From account, To account, and Amount are required.";
      return;
    }

    saving = true;
    error = null;
    try {
      const res = await fetch("/api/transaction/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          date,
          payee: payee.trim(),
          narration: narration.trim(),
          fromAccount: fromAccount.trim(),
          toAccount: toAccount.trim(),
          amount: amount.trim(),
          commodity: commodity.trim() || runtime.defaultCurrency
        })
      });
      const data = (await res.json()) as { success?: boolean; message?: string; error?: string };
      if (!res.ok || data.success === false) {
        error = data.message || data.error || "Failed to add transaction";
        return;
      }

      await triggerSync({ journal: true });
      payee = "";
      narration = "";
      amount = "";
      open = false;
      onSaved?.();
    } catch (err) {
      error = err instanceof Error ? err.message : "Failed to add transaction";
    } finally {
      saving = false;
    }
  }
</script>

{#if open}
  <div class="fixed inset-0 z-50 flex items-center justify-center p-4">
    <button
      type="button"
      class="fixed inset-0 bg-black/40 backdrop-blur-[1px]"
      aria-label="Close Quick Add"
      onclick={() => (open = false)}
    ></button>

    <div
      class="relative z-10 w-full max-w-lg rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden"
    >
      <div class="flex items-center justify-between px-5 py-4 border-b border-[var(--border)]">
        <h2 class="text-base font-semibold">Quick Add Transaction</h2>
        <button
          type="button"
          class="p-1.5 rounded-lg text-[var(--muted)] hover:bg-[var(--surface-elevated)] cursor-pointer"
          onclick={() => (open = false)}
          aria-label="Close"
        >
          <X class="w-4 h-4" />
        </button>
      </div>

      <form class="p-5 space-y-4 text-sm" onsubmit={submit}>
        {#if error}
          <div class="rounded-lg bg-[var(--loss)]/10 border border-[var(--loss)]/30 px-3 py-2 text-xs text-[var(--loss)]">
            {error}
          </div>
        {/if}

        <div class="grid grid-cols-2 gap-3">
          <div>
            <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-date">
              Date
            </label>
            <input
              id="qa-date"
              type="date"
              bind:value={date}
              class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
              required
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-payee">
              Payee
            </label>
            <input
              id="qa-payee"
              type="text"
              bind:value={payee}
              placeholder="Blue Tokai"
              class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            />
          </div>
        </div>

        <div>
          <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-from">
            From Account (Credit)
          </label>
          <input
            id="qa-from"
            list="qa-accounts"
            bind:value={fromAccount}
            placeholder="Assets:Checking:HDFC"
            class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            required
          />
        </div>

        <div>
          <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-to">
            To Account (Debit)
          </label>
          <input
            id="qa-to"
            list="qa-accounts"
            bind:value={toAccount}
            placeholder="Expenses:Food:Dining"
            class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            required
          />
        </div>

        <datalist id="qa-accounts">
          {#each runtime.accounts as acc}
            <option value={acc}></option>
          {/each}
        </datalist>

        <div class="grid grid-cols-3 gap-3">
          <div class="col-span-2">
            <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-amount">
              Amount
            </label>
            <input
              id="qa-amount"
              type="number"
              step="any"
              bind:value={amount}
              placeholder="480"
              class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none tabular-nums"
              required
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-curr">
              Currency
            </label>
            <input
              id="qa-curr"
              type="text"
              bind:value={commodity}
              class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
            />
          </div>
        </div>

        <div>
          <label class="block text-xs font-medium text-[var(--muted)] mb-1" for="qa-narration">
            Note / Narration
          </label>
          <input
            id="qa-narration"
            type="text"
            bind:value={narration}
            placeholder="Optional note"
            class="w-full px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          />
        </div>

        <div class="pt-2 flex justify-end gap-2">
          <button
            type="button"
            class="px-3.5 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
            onclick={() => (open = false)}
          >
            Cancel
          </button>
          <button
            type="submit"
            disabled={saving}
            class="px-4 py-1.5 rounded-lg text-xs font-medium bg-[var(--accent)] text-[var(--accent-fg)] hover:opacity-90 disabled:opacity-50 cursor-pointer"
          >
            {saving ? "Saving…" : "Save Transaction"}
          </button>
        </div>
      </form>
    </div>
  </div>
{/if}
