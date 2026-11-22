<script lang="ts">
  import Select from "svelte-select";
  import * as toast from "bulma-toast";
  import { ajax, postingUrl, type LedgerFileError, type Transaction } from "$lib/utils";
  import { fetchConfig } from "$lib/config_client";
  import { syncJournalAndRefresh } from "../../store";
  import {
    parseRawTransaction,
    serializeEditableTransaction,
    type EditablePosting,
    type EditableTransaction
  } from "$lib/transaction_patch";

  let {
    transaction = $bindable<Transaction | null>(null),
    accounts = [],
    onsaved
  }: {
    transaction?: Transaction | null;
    accounts?: string[];
    onsaved?: () => void | Promise<void>;
  } = $props();

  let loadedAccounts = $state<string[]>([]);
  let activeTab = $state<"form" | "raw">("form");
  let isLoadingRaw = $state(false);
  let isSaving = $state(false);
  let confirmDelete = $state(false);
  let rawText = $state("");
  let editable = $state<EditableTransaction | null>(null);
  let validationErrors = $state<LedgerFileError[]>([]);
  let loadError = $state("");

  const effectiveAccounts = $derived(accounts.length > 0 ? accounts : loadedAccounts);
  const selectItems = $derived(effectiveAccounts.map((acc) => ({ id: acc, name: acc })));

  function close() {
    transaction = null;
    confirmDelete = false;
    validationErrors = [];
    loadError = "";
  }

  $effect(() => {
    if (!transaction) {
      editable = null;
      rawText = "";
      confirmDelete = false;
      validationErrors = [];
      loadError = "";
      return;
    }

    const currentTx = transaction;
    confirmDelete = false;
    validationErrors = [];
    loadError = "";
    activeTab = "form";

    if (effectiveAccounts.length === 0) {
      void fetchConfig({ background: true }).then((cfg) => {
        loadedAccounts = cfg.accounts || [];
      });
    }

    isLoadingRaw = true;
    const params = new URLSearchParams({
      file_name: currentTx.fileName,
      begin_line: String(currentTx.beginLine),
      end_line: String(currentTx.endLine)
    });

    void ajax(`/api/transaction/raw?${params.toString()}`, { background: true })
      .then((res) => {
        if (transaction !== currentTx) return;
        rawText = res.raw || "";
        editable = parseRawTransaction(rawText, currentTx);
      })
      .catch((err: any) => {
        if (transaction !== currentTx) return;
        loadError = err?.message || "Failed to load transaction lines";
        editable = parseRawTransaction("", currentTx);
        rawText = serializeEditableTransaction(editable);
      })
      .finally(() => {
        if (transaction === currentTx) {
          isLoadingRaw = false;
        }
      });
  });

  function switchTab(tab: "form" | "raw") {
    if (tab === activeTab) return;
    if (tab === "raw" && editable) {
      rawText = serializeEditableTransaction(editable);
    } else if (tab === "form") {
      editable = parseRawTransaction(rawText, transaction ?? undefined);
    }
    activeTab = tab;
  }

  function addSplit() {
    if (!editable) return;
    const defaultCommodity =
      editable.postings.find((p) => p.commodity)?.commodity ||
      USER_CONFIG.default_currency ||
      "INR";
    editable.postings = [
      ...editable.postings,
      { account: "", amount: "", commodity: defaultCommodity, suffix: "" }
    ];
  }

  function removePosting(index: number) {
    if (!editable) return;
    editable.postings = editable.postings.filter((_, idx) => idx !== index);
  }

  function updatePostingAccount(index: number, value: string) {
    if (!editable) return;
    editable.postings[index].account = value;
  }

  const runningTotal = $derived.by(() => {
    if (!editable) return 0;
    return editable.postings.reduce((sum, p) => {
      if (p.isCommentOnly) return sum;
      const n = parseFloat(p.amount);
      return Number.isFinite(n) ? sum + n : sum;
    }, 0);
  });

  async function savePatch() {
    if (!transaction) return;
    const contentToSave =
      activeTab === "raw" ? rawText : editable ? serializeEditableTransaction(editable) : rawText;

    isSaving = true;
    validationErrors = [];
    try {
      const res = await ajax("/api/transaction/patch", {
        method: "POST",
        body: JSON.stringify({
          file_name: transaction.fileName,
          begin_line: transaction.beginLine,
          end_line: transaction.endLine,
          action: "replace",
          content: contentToSave
        }),
        background: true
      });

      if (!res.saved) {
        validationErrors = res.errors || [];
        toast.toast({
          message: res.message || "Validation failed",
          type: "is-danger",
          duration: 8000
        });
        return;
      }

      toast.toast({
        message: "Transaction updated. Syncing in background...",
        type: "is-success"
      });
      close();
      void syncJournalAndRefresh(onsaved);
    } catch (e: any) {
      toast.toast({
        message: e?.message || "Failed to update transaction",
        type: "is-danger"
      });
    } finally {
      isSaving = false;
    }
  }

  async function deleteTransaction() {
    if (!transaction) return;
    isSaving = true;
    validationErrors = [];
    try {
      const res = await ajax("/api/transaction/patch", {
        method: "POST",
        body: JSON.stringify({
          file_name: transaction.fileName,
          begin_line: transaction.beginLine,
          end_line: transaction.endLine,
          action: "delete"
        }),
        background: true
      });

      if (!res.saved) {
        validationErrors = res.errors || [];
        toast.toast({
          message: res.message || "Failed to delete transaction",
          type: "is-danger",
          duration: 8000
        });
        return;
      }

      toast.toast({
        message: "Transaction deleted. Syncing in background...",
        type: "is-success"
      });
      close();
      void syncJournalAndRefresh(onsaved);
    } catch (e: any) {
      toast.toast({
        message: e?.message || "Failed to delete transaction",
        type: "is-danger"
      });
    } finally {
      isSaving = false;
    }
  }

  function handleKeydown(e: KeyboardEvent) {
    if (transaction && e.key === "Escape") {
      close();
    }
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if transaction}
  <div class="drawer-backdrop" role="presentation" onclick={close}></div>
  <div class="tx-drawer box" aria-label="Transaction Workbench" role="dialog" aria-modal="true">
    <header class="tx-drawer-header">
      <div>
        <p class="title is-6 mb-1">Edit Transaction</p>
        <p class="is-size-7 has-text-grey">
          {transaction.fileName}:{transaction.beginLine}-{transaction.endLine}
          {#if transaction.postings[0]}
            ·
            <a href={postingUrl(transaction.postings[0])} class="is-underlined"> Open in Editor </a>
          {/if}
        </p>
      </div>
      <button type="button" class="delete" aria-label="Close drawer" onclick={close}></button>
    </header>

    <div class="tabs is-small is-boxed mb-3">
      <ul>
        <li class:is-active={activeTab === "form"}>
          <a
            href="#form"
            onclick={(e) => {
              e.preventDefault();
              switchTab("form");
            }}
          >
            Edit / Split
          </a>
        </li>
        <li class:is-active={activeTab === "raw"}>
          <a
            href="#raw"
            onclick={(e) => {
              e.preventDefault();
              switchTab("raw");
            }}
          >
            Raw Journal
          </a>
        </li>
      </ul>
    </div>

    <div class="tx-drawer-body">
      {#if loadError}
        <article class="message is-warning is-small">
          <div class="message-body">{loadError}</div>
        </article>
      {/if}

      {#if validationErrors.length > 0}
        <article class="message is-danger is-small">
          <div class="message-body">
            {#each validationErrors as err}
              <div>Line {err.line_from}: {err.message}</div>
            {/each}
          </div>
        </article>
      {/if}

      {#if isLoadingRaw}
        <p class="has-text-grey is-size-7">Loading transaction…</p>
      {:else if activeTab === "raw"}
        <div class="field">
          <label class="label is-small" for="drawer-raw-text">Raw Ledger Entry</label>
          <textarea
            id="drawer-raw-text"
            class="textarea is-small is-family-monospace"
            rows="10"
            bind:value={rawText}
          ></textarea>
        </div>
      {:else if editable}
        <div class="columns is-mobile mb-0">
          <div class="column is-6">
            <div class="field">
              <label class="label is-small" for="drawer-date">Date</label>
              <input
                id="drawer-date"
                class="input is-small"
                type="date"
                bind:value={editable.date}
              />
            </div>
          </div>
          <div class="column is-6">
            <div class="field">
              <label class="label is-small" for="drawer-status">Status</label>
              <div class="select is-small is-fullwidth">
                <select id="drawer-status" bind:value={editable.status}>
                  <option value="">Unmarked</option>
                  <option value="*">Cleared (*)</option>
                  <option value="!">Pending (!)</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div class="columns is-mobile mb-0">
          <div class="column is-6">
            <div class="field">
              <label class="label is-small" for="drawer-payee">Payee</label>
              <input
                id="drawer-payee"
                class="input is-small"
                type="text"
                bind:value={editable.payee}
              />
            </div>
          </div>
          <div class="column is-6">
            <div class="field">
              <label class="label is-small" for="drawer-narration">Narration</label>
              <input
                id="drawer-narration"
                class="input is-small"
                type="text"
                bind:value={editable.narration}
                placeholder="Optional note"
              />
            </div>
          </div>
        </div>

        <div class="is-flex is-justify-content-space-between is-align-items-center mt-2 mb-2">
          <span class="label is-small mb-0">Postings &amp; Splits</span>
          <div class="is-flex is-align-items-center" style="gap: 0.5rem;">
            <span
              class="tag is-small {Math.abs(runningTotal) < 0.005
                ? 'is-success is-light'
                : 'is-warning is-light'}"
            >
              Net: {runningTotal.toFixed(2)}
            </span>
            <button type="button" class="button is-small is-link is-light" onclick={addSplit}>
              + Add Split
            </button>
          </div>
        </div>

        {#each editable.postings as posting, idx}
          {#if !posting.isCommentOnly}
            <div class="box p-2 mb-2 posting-edit-card">
              <div class="field mb-2">
                <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
                  <span class="is-size-7 has-text-weight-semibold">Account #{idx + 1}</span>
                  {#if editable.postings.filter((p) => !p.isCommentOnly).length > 2}
                    <button
                      type="button"
                      class="button is-ghost is-small has-text-danger p-0"
                      style="height: auto;"
                      onclick={() => removePosting(idx)}
                    >
                      Remove
                    </button>
                  {/if}
                </div>
                <Select
                  value={posting.account
                    ? { id: posting.account, name: posting.account }
                    : undefined}
                  items={selectItems}
                  label="name"
                  itemId="id"
                  placeholder="Select or type account..."
                  searchable={true}
                  clearable={false}
                  showChevron={true}
                  on:change={(e) => updatePostingAccount(idx, e.detail?.name || "")}
                  on:input={(e) => {
                    if (typeof e.detail === "string" && e.detail.trim()) {
                      updatePostingAccount(idx, e.detail.trim());
                    }
                  }}
                />
              </div>
              <div class="columns is-mobile mb-0">
                <div class="column is-5 py-1">
                  <input
                    class="input is-small"
                    type="text"
                    bind:value={posting.amount}
                    placeholder="Amount (e.g. -50.00)"
                  />
                </div>
                <div class="column is-3 py-1">
                  <input
                    class="input is-small"
                    type="text"
                    bind:value={posting.commodity}
                    placeholder="Currency"
                  />
                </div>
                <div class="column is-4 py-1">
                  <input
                    class="input is-small"
                    type="text"
                    bind:value={posting.suffix}
                    placeholder="FX / ; note"
                  />
                </div>
              </div>
            </div>
          {/if}
        {/each}
      {/if}
    </div>

    <footer class="tx-drawer-footer">
      <div>
        {#if !confirmDelete}
          <button
            type="button"
            class="button is-small is-danger is-light"
            disabled={isSaving}
            onclick={() => (confirmDelete = true)}
          >
            Delete
          </button>
        {:else}
          <div class="buttons are-small">
            <button
              type="button"
              class="button is-danger {isSaving ? 'is-loading' : ''}"
              disabled={isSaving}
              onclick={deleteTransaction}
            >
              Confirm Delete
            </button>
            <button
              type="button"
              class="button is-light"
              disabled={isSaving}
              onclick={() => (confirmDelete = false)}
            >
              Cancel
            </button>
          </div>
        {/if}
      </div>
      <div class="buttons are-small mb-0">
        <button type="button" class="button" onclick={close} disabled={isSaving}>Cancel</button>
        <button
          type="button"
          class="button is-primary {isSaving ? 'is-loading' : ''}"
          disabled={isSaving || isLoadingRaw}
          onclick={savePatch}
        >
          Save Changes
        </button>
      </div>
    </footer>
  </div>
{/if}

<style lang="scss">
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(10, 10, 10, 0.45);
    z-index: 60;
  }

  .tx-drawer {
    position: fixed;
    top: 0;
    right: 0;
    bottom: 0;
    width: min(560px, 100vw);
    z-index: 61;
    display: flex;
    flex-direction: column;
    border-radius: 0;
    margin: 0 !important;
    padding: 1rem 1.25rem;
    overflow: hidden;
  }

  .tx-drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding-bottom: 0.75rem;
    border-bottom: 1px solid var(--fa-border-color, #dbdbdb);
    margin-bottom: 0.75rem;
  }

  .tx-drawer-body {
    flex: 1;
    overflow-y: auto;
    padding-right: 0.25rem;
  }

  .tx-drawer-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding-top: 0.75rem;
    margin-top: 0.75rem;
    border-top: 1px solid var(--fa-border-color, #dbdbdb);
  }
</style>
