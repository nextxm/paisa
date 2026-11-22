<script lang="ts">
  import Select from "svelte-select";
  import { ajax } from "$lib/utils";
  import { fetchConfig } from "$lib/config_client";
  import { syncJournalAndRefresh } from "../../store";
  import * as toast from "bulma-toast";
  import dayjs from "dayjs";
  import {
    applySuggestionSelection,
    buildQuickAddSubmitRequest,
    clearedParserState,
    parserFormOverrides,
    type QuickAddPostingValue
  } from "./quick_add_parser_utils";

  let { open = $bindable(false), accounts = [] }: { open: boolean; accounts?: string[] } = $props();

  let loadedAccounts = $state<string[]>([]);
  const effectiveAccounts = $derived(accounts.length > 0 ? accounts : loadedAccounts);
  const selectItems = $derived(effectiveAccounts.map((acc) => ({ id: acc, name: acc })));
  const hasJournalTarget = $derived(
    Boolean(USER_CONFIG.add_journal_path || USER_CONFIG.journal_path)
  );

  let date = $state(dayjs().format("YYYY-MM-DD"));
  let payee = $state("");
  let narration = $state("");
  let fromAccount = $state("");
  let toAccount = $state("");
  let amount = $state("");
  let commodity = $state(USER_CONFIG.default_currency || "INR");

  let showMultiCurrency = $state(false);
  let toAmount = $state("");
  let toCommodity = $state("");
  let exchangeRate = $state("");

  let splitMode = $state(false);
  let splits = $state<{ account: string; amount: string; commodity: string }[]>([
    { account: "", amount: "", commodity: USER_CONFIG.default_currency || "INR" },
    { account: "", amount: "", commodity: USER_CONFIG.default_currency || "INR" }
  ]);

  let isLoading = $state(false);
  let isParsing = $state(false);
  let parserText = $state("");
  let parserResult = $state<any>(null);
  let parserWarnings = $state<string[]>([]);
  let requiresConfirmation = $state(false);
  let selectedSuggestionIndex = $state<Record<string, number>>({});
  let parseStartedAt = $state<number | null>(null);

  $effect(() => {
    if (open && effectiveAccounts.length === 0) {
      void fetchConfig({ background: true }).then((cfg) => {
        loadedAccounts = cfg.accounts || [];
      });
    }
  });

  function applyParsedResult(result: any) {
    const parsedDate = dayjs(result.date);
    if (parsedDate.isValid()) {
      date = parsedDate.format("YYYY-MM-DD");
    }

    const overrides = parserFormOverrides(result, commodity);
    payee = overrides.payee || "";
    fromAccount = overrides.fromAccount || "";
    toAccount = overrides.toAccount || "";
    amount = overrides.amount || "";
    commodity = overrides.commodity || commodity;
  }

  function clearParsedState() {
    const cleared = clearedParserState();
    parserText = cleared.parserText;
    parserResult = cleared.parserResult;
    parserWarnings = cleared.parserWarnings;
    requiresConfirmation = cleared.requiresConfirmation;
    selectedSuggestionIndex = cleared.selectedSuggestionIndex;
    parseStartedAt = cleared.parseStartedAt;
  }

  async function parseNaturalLanguage() {
    if (!parserText.trim()) {
      toast.toast({ message: "Enter transaction text to parse", type: "is-warning" });
      return;
    }

    isParsing = true;
    parserWarnings = [];
    parseStartedAt = Date.now();

    try {
      const response = await ajax("/api/parser/parse", {
        method: "POST",
        body: JSON.stringify({ text: parserText }),
        background: true
      });

      parserResult = response?.result || null;
      requiresConfirmation = !!response?.requires_confirmation;
      parserWarnings = parserResult?.warnings || [];
      selectedSuggestionIndex = {};

      if (!parserResult) {
        toast.toast({ message: "Parser did not return a result", type: "is-danger" });
        return;
      }

      applyParsedResult(parserResult);
    } catch (e: any) {
      toast.toast({ message: e.message || "Failed to parse transaction text", type: "is-danger" });
      parserResult = null;
      parserWarnings = [];
      requiresConfirmation = false;
    } finally {
      isParsing = false;
    }
  }

  function applySuggestion(field: string, account: string, index: number) {
    const selection = applySuggestionSelection(
      field,
      account,
      index,
      {
        date,
        payee,
        narration,
        fromAccount,
        toAccount,
        amount,
        commodity
      },
      selectedSuggestionIndex
    );

    fromAccount = selection.values.fromAccount;
    toAccount = selection.values.toAccount;
    selectedSuggestionIndex = selection.selectedSuggestionIndex;
  }

  function addSplitRow() {
    splits = [...splits, { account: "", amount: "", commodity: commodity || "INR" }];
  }

  function removeSplitRow(index: number) {
    if (splits.length <= 2) return;
    splits = splits.filter((_, i) => i !== index);
  }

  function toggleSplitMode() {
    splitMode = !splitMode;
    if (splitMode) {
      if (!splits[0].account && toAccount) {
        splits[0].account = toAccount;
      }
      if (!splits[0].amount && amount) {
        splits[0].amount = amount;
      }
      splits = splits.map((s) => ({ ...s, commodity: s.commodity || commodity }));
    }
  }

  const splitSum = $derived(
    splits.reduce((sum, s) => {
      const n = parseFloat(s.amount);
      return Number.isFinite(n) ? sum + n : sum;
    }, 0)
  );

  function resetFormState() {
    payee = "";
    narration = "";
    fromAccount = "";
    toAccount = "";
    amount = "";
    toAmount = "";
    toCommodity = "";
    exchangeRate = "";
    showMultiCurrency = false;
    splitMode = false;
    splits = [
      { account: "", amount: "", commodity: commodity || "INR" },
      { account: "", amount: "", commodity: commodity || "INR" }
    ];
    clearParsedState();
  }

  async function submit() {
    if (!date || !fromAccount || !commodity) {
      toast.toast({ message: "Please fill all required fields", type: "is-danger" });
      return;
    }

    let postingsPayload: QuickAddPostingValue[] | undefined;
    let effectiveAmount = amount;
    let effectiveToAccount = toAccount;

    if (splitMode) {
      const validSplits = splits.filter((s) => s.account.trim() && s.amount.trim());
      if (validSplits.length < 2) {
        toast.toast({ message: "Split mode requires at least 2 split rows", type: "is-danger" });
        return;
      }
      if (!effectiveAmount.trim()) {
        effectiveAmount = splitSum.toFixed(2);
      }
      effectiveToAccount = validSplits[0].account;
      const negAmount = effectiveAmount.trim().startsWith("-")
        ? effectiveAmount.trim()
        : `-${effectiveAmount.trim()}`;
      postingsPayload = [
        { account: fromAccount, amount: negAmount, commodity },
        ...validSplits.map((s) => ({
          account: s.account,
          amount: s.amount,
          commodity: s.commodity || commodity
        }))
      ];
    } else if (!toAccount || !amount) {
      toast.toast({ message: "Please fill all required fields", type: "is-danger" });
      return;
    }

    isLoading = true;
    try {
      const request = buildQuickAddSubmitRequest({
        parserText: splitMode ? "" : parserText,
        values: {
          date,
          payee,
          narration,
          fromAccount,
          toAccount: effectiveToAccount,
          amount: effectiveAmount,
          commodity,
          toAmount: showMultiCurrency ? toAmount : undefined,
          toCommodity: showMultiCurrency ? toCommodity : undefined,
          exchangeRate: showMultiCurrency ? exchangeRate : undefined,
          postings: postingsPayload
        },
        selectedSuggestionIndex,
        parseStartedAt,
        nowMs: Date.now()
      });

      const response = await ajax(request.endpoint, {
        method: "POST",
        body: JSON.stringify(request.payload)
      });

      if (response.success) {
        if (response.errors && response.errors.length > 0) {
          toast.toast({
            message:
              "Transaction added, but journal has validation errors. Please check the Editor.",
            type: "is-warning",
            duration: 10000
          });
        } else {
          toast.toast({
            message: "Transaction added. Syncing in background...",
            type: "is-success"
          });
        }
        open = false;
        resetFormState();
        void syncJournalAndRefresh();
      } else {
        toast.toast({
          message: response.error?.message || "Failed to add transaction",
          type: "is-danger"
        });
      }
    } catch (e: any) {
      toast.toast({ message: e.message || "An error occurred", type: "is-danger" });
    } finally {
      isLoading = false;
    }
  }
</script>

<div class="modal" class:is-active={open}>
  <div
    class="modal-background"
    role="button"
    tabindex="0"
    onclick={() => (open = false)}
    onkeydown={(e) => e.key === "Escape" && (open = false)}
  ></div>
  <div class="modal-card">
    <header class="modal-card-head">
      <p class="modal-card-title">Quick Add Transaction</p>
      <button class="delete" aria-label="close" onclick={() => (open = false)}></button>
    </header>
    <section class="modal-card-body">
      {#if !hasJournalTarget}
        <article class="message is-warning is-small">
          <div class="message-body">
            Neither <code>add_journal_path</code> nor <code>journal_path</code> is configured.
          </div>
        </article>
      {/if}

      <div class="field">
        <label class="label is-small" for="parser-text">Natural Language Input</label>
        <div class="control">
          <textarea
            id="parser-text"
            class="textarea is-small"
            rows="2"
            bind:value={parserText}
            placeholder="e.g. 20 Apr, bought 15$ groceries using bmo cc from no frills"
          ></textarea>
        </div>
        <div class="control mt-2 is-flex is-justify-content-space-between" style="gap: 0.5rem;">
          <div class="is-flex" style="gap: 0.5rem;">
            <button
              class="button is-small is-info is-light {isParsing ? 'is-loading' : ''}"
              onclick={parseNaturalLanguage}
              disabled={!parserText.trim() || !hasJournalTarget}
            >
              Parse Text
            </button>
            <button
              class="button is-small is-light"
              type="button"
              onclick={clearParsedState}
              disabled={!parserText && !parserResult}
            >
              Clear Parsed State
            </button>
          </div>
          <div class="is-flex" style="gap: 0.5rem;">
            <button
              type="button"
              class="button is-small {splitMode ? 'is-link' : 'is-light'}"
              onclick={toggleSplitMode}
            >
              Split
            </button>
            <button
              type="button"
              class="button is-small {showMultiCurrency ? 'is-link' : 'is-light'}"
              onclick={() => (showMultiCurrency = !showMultiCurrency)}
              disabled={splitMode}
            >
              Multi-Currency
            </button>
          </div>
        </div>
      </div>

      {#if parserResult}
        <article class="message is-small {requiresConfirmation ? 'is-warning' : 'is-success'}">
          <div class="message-body">
            Parser confidence: {(parserResult.confidence?.overall || 0).toFixed(2)}
            {#if requiresConfirmation}
              - review fields before creating.
            {/if}
          </div>
        </article>
      {/if}

      {#if parserWarnings.length > 0}
        <article class="message is-warning is-small">
          <div class="message-body">
            {#each parserWarnings as warning}
              <div>{warning}</div>
            {/each}
          </div>
        </article>
      {/if}

      {#if parserResult?.suggestions?.length > 0}
        {#each parserResult.suggestions as suggestionSet}
          <div class="field">
            <span class="label is-small">Suggestions for {suggestionSet.field}</span>
            <div class="buttons are-small">
              {#each suggestionSet.suggestions as suggestion, index}
                <button
                  class="button is-light {selectedSuggestionIndex[suggestionSet.field] === index
                    ? 'is-link'
                    : ''}"
                  onclick={() => applySuggestion(suggestionSet.field, suggestion.account, index)}
                  type="button"
                >
                  {suggestion.account} ({suggestion.score.toFixed(2)})
                </button>
              {/each}
            </div>
          </div>
        {/each}
      {/if}

      <div class="columns is-mobile mb-0">
        <div class="column is-4">
          <div class="field">
            <label class="label is-small" for="date-input">Date</label>
            <div class="control">
              <input
                id="date-input"
                class="input is-small"
                type="date"
                bind:value={date}
                required
              />
            </div>
          </div>
        </div>
        <div class="column is-4">
          <div class="field">
            <label class="label is-small" for="payee-input">Payee</label>
            <div class="control">
              <input
                id="payee-input"
                class="input is-small"
                type="text"
                bind:value={payee}
                placeholder="e.g. Amazon"
              />
            </div>
          </div>
        </div>
        <div class="column is-4">
          <div class="field">
            <label class="label is-small" for="narration-input">Narration</label>
            <div class="control">
              <input
                id="narration-input"
                class="input is-small"
                type="text"
                bind:value={narration}
                placeholder="e.g. New keyboard"
              />
            </div>
          </div>
        </div>
      </div>

      <div class="columns is-mobile mb-0">
        <div class="column">
          <div class="field">
            <span class="label is-small">From Account</span>
            <Select
              value={fromAccount ? { id: fromAccount, name: fromAccount } : undefined}
              items={selectItems}
              label="name"
              itemId="id"
              placeholder="Select or type account..."
              searchable={true}
              clearable={true}
              showChevron={true}
              on:change={(e) => (fromAccount = e.detail?.name || "")}
              on:clear={() => (fromAccount = "")}
              on:input={(e) => {
                if (typeof e.detail === "string" && e.detail.trim()) {
                  fromAccount = e.detail.trim();
                }
              }}
            />
          </div>
        </div>
        {#if !splitMode}
          <div class="column">
            <div class="field">
              <span class="label is-small">To Account</span>
              <Select
                value={toAccount ? { id: toAccount, name: toAccount } : undefined}
                items={selectItems}
                label="name"
                itemId="id"
                placeholder="Select or type account..."
                searchable={true}
                clearable={true}
                showChevron={true}
                on:change={(e) => (toAccount = e.detail?.name || "")}
                on:clear={() => (toAccount = "")}
                on:input={(e) => {
                  if (typeof e.detail === "string" && e.detail.trim()) {
                    toAccount = e.detail.trim();
                  }
                }}
              />
            </div>
          </div>
        {/if}
      </div>

      <div class="columns is-mobile mb-0">
        <div class="column">
          <div class="field">
            <label class="label is-small" for="amount-input">
              {splitMode ? "Total Amount (optional, auto-sums splits)" : "Amount"}
            </label>
            <div class="control">
              <input
                id="amount-input"
                class="input is-small"
                type="text"
                bind:value={amount}
                placeholder={splitMode ? splitSum.toFixed(2) : "0.00"}
                required={!splitMode}
              />
            </div>
          </div>
        </div>
        <div class="column">
          <div class="field">
            <label class="label is-small" for="currency-input">Currency</label>
            <div class="control">
              <input
                id="currency-input"
                class="input is-small"
                type="text"
                bind:value={commodity}
                required
              />
            </div>
          </div>
        </div>
      </div>

      {#if showMultiCurrency && !splitMode}
        <div class="box p-2 mt-2 mb-2">
          <p class="is-size-7 has-text-weight-semibold mb-2">Multi-Currency / FX</p>
          <div class="columns is-mobile mb-0">
            <div class="column">
              <label class="label is-small" for="to-amount-input">To Amount</label>
              <input
                id="to-amount-input"
                class="input is-small"
                type="text"
                bind:value={toAmount}
                placeholder="e.g. 45.00"
              />
            </div>
            <div class="column">
              <label class="label is-small" for="to-currency-input">To Currency</label>
              <input
                id="to-currency-input"
                class="input is-small"
                type="text"
                bind:value={toCommodity}
                placeholder="e.g. EUR"
              />
            </div>
            <div class="column">
              <label class="label is-small" for="fx-rate-input">Exchange Rate</label>
              <input
                id="fx-rate-input"
                class="input is-small"
                type="text"
                bind:value={exchangeRate}
                placeholder="e.g. 0.90"
              />
            </div>
          </div>
        </div>
      {/if}

      {#if splitMode}
        <div class="mt-2">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
            <span class="label is-small mb-0"
              >To Accounts (Splits: {splitSum.toFixed(2)} {commodity})</span
            >
            <button type="button" class="button is-small is-link is-light" onclick={addSplitRow}>
              + Add Split
            </button>
          </div>
          {#each splits as split, idx}
            <div class="columns is-mobile is-vcentered mb-1">
              <div class="column is-6 py-1">
                <Select
                  value={split.account ? { id: split.account, name: split.account } : undefined}
                  items={selectItems}
                  label="name"
                  itemId="id"
                  placeholder="Account #{idx + 1}"
                  searchable={true}
                  clearable={true}
                  showChevron={true}
                  on:change={(e) => (split.account = e.detail?.name || "")}
                  on:clear={() => (split.account = "")}
                />
              </div>
              <div class="column is-3 py-1">
                <input
                  class="input is-small"
                  type="text"
                  bind:value={split.amount}
                  placeholder="Amount"
                />
              </div>
              <div class="column is-2 py-1">
                <input
                  class="input is-small"
                  type="text"
                  bind:value={split.commodity}
                  placeholder="Curr"
                />
              </div>
              <div class="column is-1 py-1">
                {#if splits.length > 2}
                  <button
                    type="button"
                    class="delete is-small"
                    aria-label="Remove split"
                    onclick={() => removeSplitRow(idx)}
                  ></button>
                {/if}
              </div>
            </div>
          {/each}
        </div>
      {/if}
    </section>
    <footer class="modal-card-foot is-justify-content-flex-end">
      <button class="button is-small" onclick={() => (open = false)}>Cancel</button>
      <button
        class="button is-primary is-small {isLoading ? 'is-loading' : ''}"
        onclick={submit}
        disabled={!hasJournalTarget}
      >
        Add Transaction
      </button>
    </footer>
  </div>
</div>

<style lang="scss">
  .modal-card {
    max-width: 680px;
  }
</style>
