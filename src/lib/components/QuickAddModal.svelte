<script lang="ts">
  import { ajax } from "$lib/utils";
  import * as toast from "bulma-toast";
  import dayjs from "dayjs";
  import {
    applySuggestionSelection,
    buildQuickAddSubmitRequest,
    clearedParserState,
    parserFormOverrides
  } from "./quick_add_parser_utils";

  let { open = $bindable(false), accounts = [] }: { open: boolean; accounts: string[] } = $props();

  let date = $state(dayjs().format("YYYY-MM-DD"));
  let payee = $state("");
  let narration = $state("");
  let fromAccount = $state("");
  let toAccount = $state("");
  let amount = $state("");
  let commodity = $state(USER_CONFIG.default_currency || "INR");
  let isLoading = $state(false);
  let isParsing = $state(false);
  let parserText = $state("");
  let parserResult = $state<any>(null);
  let parserWarnings = $state<string[]>([]);
  let requiresConfirmation = $state(false);
  let selectedSuggestionIndex = $state<Record<string, number>>({});
  let parseStartedAt = $state<number | null>(null);

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

  function resetFormState() {
    payee = "";
    narration = "";
    fromAccount = "";
    toAccount = "";
    amount = "";
    clearParsedState();
  }

  async function submit() {
    if (!date || !fromAccount || !toAccount || !amount || !commodity) {
      toast.toast({ message: "Please fill all required fields", type: "is-danger" });
      return;
    }

    isLoading = true;
    try {
      const request = buildQuickAddSubmitRequest({
        parserText,
        values: {
          date,
          payee,
          narration,
          fromAccount,
          toAccount,
          amount,
          commodity
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
            message: "Transaction added, but journal has validation errors. Please sync and check the Editor.",
            type: "is-warning",
            duration: 10000
          });
        } else {
          toast.toast({
            message: "Transaction added. Please sync to see changes.",
            type: "is-success"
          });
        }
        open = false;
        resetFormState();
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
      {#if !USER_CONFIG.add_journal_path}
        <article class="message is-warning is-small">
          <div class="message-body">
            <code>add_journal_path</code> is not configured. Please set it in Settings first.
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
        <div class="control mt-2 is-flex" style="gap: 0.5rem;">
          <button
            class="button is-small is-info is-light {isParsing ? 'is-loading' : ''}"
            onclick={parseNaturalLanguage}
            disabled={!parserText.trim() || !USER_CONFIG.add_journal_path}
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

      <div class="field">
        <label class="label is-small" for="date-input">Date</label>
        <div class="control">
          <input id="date-input" class="input is-small" type="date" bind:value={date} required />
        </div>
      </div>

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

      <div class="columns is-mobile mb-0">
        <div class="column">
          <div class="field">
            <label class="label is-small" for="from-account-input">From Account</label>
            <div class="control select is-small is-fullwidth">
              <select id="from-account-input" bind:value={fromAccount} required>
                <option value="" disabled>Select account...</option>
                {#each accounts as account}
                  <option value={account}>{account}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>
        <div class="column">
          <div class="field">
            <label class="label is-small" for="to-account-input">To Account</label>
            <div class="control select is-small is-fullwidth">
              <select id="to-account-input" bind:value={toAccount} required>
                <option value="" disabled>Select account...</option>
                {#each accounts as account}
                  <option value={account}>{account}</option>
                {/each}
              </select>
            </div>
          </div>
        </div>
      </div>

      <div class="columns is-mobile">
        <div class="column">
          <div class="field">
            <label class="label is-small" for="amount-input">Amount</label>
            <div class="control">
              <input
                id="amount-input"
                class="input is-small"
                type="text"
                bind:value={amount}
                placeholder="0.00"
                required
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
    </section>
    <footer class="modal-card-foot is-justify-content-flex-end">
      <button class="button is-small" onclick={() => (open = false)}>Cancel</button>
      <button
        class="button is-primary is-small {isLoading ? 'is-loading' : ''}"
        onclick={submit}
        disabled={!USER_CONFIG.add_journal_path}
      >
        Add Transaction
      </button>
    </footer>
  </div>
</div>

<style lang="scss">
  .modal-card {
    max-width: 620px;
  }
</style>
