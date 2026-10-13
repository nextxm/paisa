<script lang="ts">
  import type { StagedTransaction } from "$lib/utils";
  import _ from "lodash";

  let {
    transactions = $bindable([]),
    included = $bindable([]),
    accounts = [],
    defaultCurrency = "INR",
    oneditrule = (_tx: StagedTransaction) => {}
  }: {
    transactions: StagedTransaction[];
    included: boolean[];
    accounts?: string[];
    defaultCurrency?: string;
    oneditrule?: (tx: StagedTransaction) => void;
  } = $props();

  let allIncluded = $derived(
    transactions.length > 0 && transactions.every((_, i) => !!included[i])
  );

  let filterStatus: "all" | "new" | "duplicate" | "unassigned" = $state("all");

  let filteredIndices = $derived.by(() => {
    return transactions
      .map((tx, idx) => ({ tx, idx }))
      .filter(({ tx }) => {
        if (filterStatus === "duplicate") return tx.is_duplicate;
        if (filterStatus === "unassigned") return !tx.selected_account;
        if (filterStatus === "new") return !tx.is_duplicate && !!tx.selected_account;
        return true;
      })
      .map(({ idx }) => idx);
  });

  function selectAllFiltered(val: boolean) {
    filteredIndices.forEach((idx) => {
      included[idx] = val;
    });
  }
</script>

<div class="staging-container">
  <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
    <div class="tags has-addons mb-0">
      <button
        class="button is-small {filterStatus === 'all' ? 'is-info' : 'is-light'}"
        onclick={() => (filterStatus = "all")}
      >
        All ({transactions.length})
      </button>
      <button
        class="button is-small {filterStatus === 'new' ? 'is-success' : 'is-light'}"
        onclick={() => (filterStatus = "new")}
      >
        Ready ({transactions.filter((t) => !t.is_duplicate && !!t.selected_account).length})
      </button>
      <button
        class="button is-small {filterStatus === 'duplicate' ? 'is-warning' : 'is-light'}"
        onclick={() => (filterStatus = "duplicate")}
      >
        Duplicates ({transactions.filter((t) => t.is_duplicate).length})
      </button>
      <button
        class="button is-small {filterStatus === 'unassigned' ? 'is-danger' : 'is-light'}"
        onclick={() => (filterStatus = "unassigned")}
      >
        Uncategorized ({transactions.filter((t) => !t.selected_account).length})
      </button>
    </div>

    <div class="buttons are-small mb-0">
      {#if transactions.some((t) => !t.is_duplicate && !t.selected_account && !!t.suggested_account)}
        <button
          class="button is-small is-link is-light"
          onclick={() => {
            transactions.forEach((tx, idx) => {
              if (!tx.is_duplicate && !tx.selected_account && tx.suggested_account) {
                tx.selected_account = tx.suggested_account;
                tx.confidence = "accepted";
                included[idx] = true;
              }
            });
          }}
        >
          <span class="icon is-small"><i class="fas fa-check-double"></i></span>
          <span>Accept All Suggestions</span>
        </button>
      {/if}
      <button class="button is-white" onclick={() => selectAllFiltered(true)}
        >Select Filtered</button
      >
      <button class="button is-white" onclick={() => selectAllFiltered(false)}
        >Deselect Filtered</button
      >
    </div>
  </div>

  <div class="table-wrapper">
    <table class="table is-bordered is-striped is-narrow is-hoverable is-fullwidth is-size-7">
      <thead>
        <tr>
          <th style="width: 40px; text-align: center;">
            <input
              type="checkbox"
              checked={allIncluded}
              onchange={(e) => {
                const checked = (e.currentTarget as HTMLInputElement).checked;
                included = transactions.map(() => checked);
              }}
            />
          </th>
          <th style="width: 90px;">Date</th>
          <th>Payee & Narration</th>
          <th style="width: 115px; text-align: right;">Amount</th>
          <th style="width: 220px;">Target Offset Account</th>
          <th style="width: 120px;">Status / Rule</th>
          <th style="width: 45px;">Action</th>
        </tr>
      </thead>
      <tbody>
        {#each filteredIndices as idx}
          {@const tx = transactions[idx]}
          <tr
            class:has-background-warning-light={tx.is_duplicate}
            class:has-background-danger-light={!tx.selected_account && !tx.is_duplicate}
          >
            <td style="text-align: center;">
              <input type="checkbox" bind:checked={included[idx]} />
            </td>
            <td class="font-mono">{tx.date}</td>
            <td>
              <div class="has-text-weight-semibold">{tx.payee}</div>
              {#if tx.memo && tx.memo !== tx.payee}
                <div class="is-size-7 has-text-grey">{tx.memo}</div>
              {/if}
              {#if tx.tags && tx.tags.length > 0}
                <div class="tags mt-1 mb-0">
                  {#each tx.tags as tag}
                    <span class="tag is-info is-light is-rounded is-small">{tag}</span>
                  {/each}
                </div>
              {/if}
            </td>
            <td style="text-align: right;" class="font-mono">
              <span class={tx.is_debit ? "has-text-danger" : "has-text-success"}>
                {tx.is_debit ? "-" : "+"}{tx.amount.toFixed(2)}
                {tx.currency || defaultCurrency}
              </span>
            </td>
            <td>
              <div class="control">
                <input
                  class="input is-small {!tx.selected_account ? 'is-danger' : ''}"
                  type="text"
                  list="account-options"
                  value={tx.selected_account}
                  placeholder={tx.suggested_account
                    ? `Suggested: ${tx.suggested_account}`
                    : "Select account..."}
                  oninput={(e) => {
                    const newVal = (e.currentTarget as HTMLInputElement).value;
                    tx.selected_account = newVal;
                    if (newVal) {
                      if (tx.suggested_account && newVal === tx.suggested_account) {
                        tx.confidence = "accepted";
                      } else {
                        tx.confidence = "manual";
                      }
                      included[idx] = true;
                    } else {
                      tx.confidence = tx.suggested_account ? "ai_suggested" : undefined;
                      included[idx] = false;
                    }
                  }}
                />
              </div>
              {#if tx.suggested_account && !tx.selected_account}
                <div class="is-flex is-align-items-center mt-1">
                  <span class="is-size-7 has-text-grey mr-2 truncate" title={tx.suggested_account}>
                    <i class="fas fa-lightbulb has-text-warning mr-1"></i>{tx.suggested_account}
                  </span>
                  <button
                    class="button is-small is-link is-light py-0 px-2 ml-auto"
                    style="height: 20px; font-size: 0.7rem;"
                    onclick={() => {
                      tx.selected_account = tx.suggested_account || "";
                      tx.confidence = "accepted";
                      included[idx] = true;
                    }}
                  >
                    Accept
                  </button>
                </div>
              {/if}
            </td>
            <td>
              {#if tx.is_duplicate}
                <span class="tag is-warning is-rounded is-small" title={tx.duplicate_reason}>
                  <i class="fas fa-copy mr-1"></i> Dup ({tx.duplicate_score}%)
                </span>
              {:else if tx.matched_rule_name}
                <span
                  class="tag is-success is-light is-rounded is-small"
                  title={tx.matched_rule_name}
                >
                  <i class="fas fa-wand-magic-sparkles mr-1"></i>
                  {tx.matched_rule_name}
                </span>
              {:else if tx.confidence === "ai_suggested"}
                <span
                  class="tag is-warning is-light is-rounded is-small"
                  title="AI Suggested (Click Accept or edit to confirm)"
                >
                  <i class="fas fa-robot mr-1"></i> Suggested
                </span>
              {:else if tx.confidence === "accepted"}
                <span
                  class="tag is-success is-light is-rounded is-small"
                  title="Accepted AI recommendation"
                >
                  <i class="fas fa-check mr-1"></i> Accepted
                </span>
              {:else if tx.confidence === "manual"}
                <span
                  class="tag is-info is-light is-rounded is-small"
                  title="Manually entered account"
                >
                  <i class="fas fa-user-pen mr-1"></i> Manual
                </span>
              {:else if tx.selected_account}
                <span class="tag is-info is-light is-rounded is-small">Assigned</span>
              {:else}
                <span class="tag is-danger is-light is-rounded is-small">Needs Account</span>
              {/if}
            </td>
            <td style="text-align: center;">
              <button
                class="button is-small is-ghost p-1"
                title="Create rule from this transaction"
                onclick={() => oneditrule(tx)}
              >
                <span class="icon is-small has-text-primary">
                  <i class="fas fa-wand-magic-sparkles"></i>
                </span>
              </button>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</div>

<datalist id="account-options">
  {#each accounts as acc}
    <option value={acc}>{acc}</option>
  {/each}
</datalist>

<style lang="scss">
  .staging-container {
    border: 1px solid rgba(127, 127, 127, 0.2);
    border-radius: 6px;
    padding: 0.75rem;
    background: var(--bulma-card-background-color, #fff);
  }
  .table-wrapper {
    max-height: calc(100vh - 350px);
    overflow-y: auto;
  }
</style>
