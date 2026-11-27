<script lang="ts">
  import dayjs from "dayjs";
  import { Plus, Sparkles, Trash2, X } from "lucide-svelte";
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

  let parserText = $state("");
  let parsing = $state(false);

  let showMultiCurrency = $state(false);
  let toAmount = $state("");
  let toCommodity = $state("");
  let exchangeRate = $state("");

  let splitMode = $state(false);
  let splits = $state<Array<{ account: string; amount: string; commodity: string }>>([
    { account: "", amount: "", commodity: runtime.defaultCurrency || "INR" },
    { account: "", amount: "", commodity: runtime.defaultCurrency || "INR" }
  ]);

  let saving = $state(false);
  let error = $state<string | null>(null);

  $effect(() => {
    if (open) {
      commodity = runtime.defaultCurrency || "INR";
      error = null;
    }
  });

  async function parseNaturalText() {
    if (!parserText.trim()) return;
    parsing = true;
    error = null;
    try {
      const res = await fetch("/api/parser/parse", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ text: parserText.trim() })
      });
      if (!res.ok) {
        error = "Could not parse text";
        return;
      }
      const data = (await res.json()) as {
        date?: string;
        payee?: string;
        narration?: string;
        amount?: number | string;
        commodity?: string;
        from_account?: string;
        to_account?: string;
      };
      if (data.date && dayjs(data.date).isValid()) {
        date = dayjs(data.date).format("YYYY-MM-DD");
      }
      if (data.payee) payee = data.payee;
      if (data.narration) narration = data.narration;
      if (data.amount !== undefined) amount = String(data.amount);
      if (data.commodity) commodity = data.commodity;
      if (data.from_account) fromAccount = data.from_account;
      if (data.to_account) toAccount = data.to_account;
    } catch {
      error = "Failed to parse natural text";
    } finally {
      parsing = false;
    }
  }

  function addSplit() {
    splits = [...splits, { account: "", amount: "", commodity: commodity || runtime.defaultCurrency }];
  }

  function removeSplit(index: number) {
    if (splits.length <= 2) return;
    splits = splits.filter((_, i) => i !== index);
  }

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    if (!fromAccount.trim()) {
      error = "From account is required.";
      return;
    }

    let payload: Record<string, unknown>;
    if (splitMode) {
      const validSplits = splits.filter((s) => s.account.trim() && s.amount.trim());
      if (validSplits.length < 2) {
        error = "Split mode requires at least 2 split rows.";
        return;
      }
      const total = validSplits.reduce((s, row) => s + (parseFloat(row.amount) || 0), 0);
      payload = {
        date,
        payee: payee.trim(),
        narration: narration.trim(),
        fromAccount: fromAccount.trim(),
        toAccount: validSplits[0].account.trim(),
        amount: String(total),
        commodity: commodity.trim() || runtime.defaultCurrency,
        postings: [
          { account: fromAccount.trim(), amount: String(-total), commodity },
          ...validSplits.map((s) => ({
            account: s.account.trim(),
            amount: s.amount.trim(),
            commodity: s.commodity.trim() || commodity
          }))
        ]
      };
    } else {
      if (!toAccount.trim() || !amount.trim()) {
        error = "From account, To account, and Amount are required.";
        return;
      }
      payload = {
        date,
        payee: payee.trim(),
        narration: narration.trim(),
        fromAccount: fromAccount.trim(),
        toAccount: toAccount.trim(),
        amount: amount.trim(),
        commodity: commodity.trim() || runtime.defaultCurrency,
        ...(showMultiCurrency
          ? {
              toAmount: toAmount.trim() || undefined,
              toCommodity: toCommodity.trim() || undefined,
              exchangeRate: exchangeRate.trim() || undefined
            }
          : {})
      };
    }

    saving = true;
    error = null;
    try {
      const res = await fetch("/api/transaction/add", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload)
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
      parserText = "";
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
      class="relative z-10 w-full max-w-lg rounded-xl bg-[var(--surface)] border border-[var(--border)] shadow-xl overflow-hidden max-h-[90vh] flex flex-col"
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

      <form class="p-5 space-y-4 text-sm overflow-y-auto" onsubmit={submit}>
        {#if error}
          <div class="rounded-lg bg-[var(--loss)]/10 border border-[var(--loss)]/30 px-3 py-2 text-xs text-[var(--loss)]">
            {error}
          </div>
        {/if}

        <!-- Natural language quick parse -->
        <div class="flex gap-2">
          <input
            type="text"
            bind:value={parserText}
            placeholder="e.g. Spent 450 on coffee at Blue Tokai via HDFC"
            class="flex-1 px-3 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] outline-none"
          />
          <button
            type="button"
            disabled={parsing}
            class="inline-flex items-center gap-1 px-3 py-1.5 rounded-lg text-xs font-medium border border-[var(--border)] hover:bg-[var(--surface-elevated)] cursor-pointer"
            onclick={parseNaturalText}
          >
            <Sparkles class="w-3.5 h-3.5 text-[var(--accent)]" />
            <span>{parsing ? "Parsing…" : "Parse"}</span>
          </button>
        </div>

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

        <div class="flex items-center gap-4 text-xs">
          <label class="flex items-center gap-1.5 cursor-pointer">
            <input type="checkbox" bind:checked={splitMode} />
            <span>Split postings</span>
          </label>
          {#if !splitMode}
            <label class="flex items-center gap-1.5 cursor-pointer">
              <input type="checkbox" bind:checked={showMultiCurrency} />
              <span>Multi-currency</span>
            </label>
          {/if}
        </div>

        {#if !splitMode}
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

          {#if showMultiCurrency}
            <div class="grid grid-cols-3 gap-2 pt-1">
              <input
                type="number"
                step="any"
                bind:value={toAmount}
                placeholder="To Amount"
                class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
              />
              <input
                type="text"
                bind:value={toCommodity}
                placeholder="To Currency"
                class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)]"
              />
              <input
                type="number"
                step="any"
                bind:value={exchangeRate}
                placeholder="FX Rate"
                class="px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
              />
            </div>
          {/if}
        {:else}
          <div class="space-y-2">
            {#each splits as row, idx}
              <div class="flex items-center gap-2">
                <input
                  list="qa-accounts"
                  bind:value={row.account}
                  placeholder="Account"
                  class="flex-1 px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)]"
                />
                <input
                  type="number"
                  step="any"
                  bind:value={row.amount}
                  placeholder="Amount"
                  class="w-24 px-2.5 py-1.5 rounded-lg text-xs bg-[var(--surface-elevated)] border border-[var(--border)] tabular-nums"
                />
                <button
                  type="button"
                  class="p-1 text-[var(--muted)] hover:text-[var(--loss)] cursor-pointer"
                  onclick={() => removeSplit(idx)}
                  aria-label="Remove split"
                >
                  <Trash2 class="w-3.5 h-3.5" />
                </button>
              </div>
            {/each}
            <button
              type="button"
              class="inline-flex items-center gap-1 text-xs text-[var(--accent)] cursor-pointer"
              onclick={addSplit}
            >
              <Plus class="w-3.5 h-3.5" />
              <span>Add split row</span>
            </button>
          </div>
        {/if}

        <datalist id="qa-accounts">
          {#each runtime.accounts as acc}
            <option value={acc}></option>
          {/each}
        </datalist>

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
