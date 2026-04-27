<script lang="ts">
  import _ from "lodash";
  import type { FilterType, FilterOperator, Filter, LogicOperator } from "./query_builder_utils";
  import {
    combinePredicate,
    filterLabel,
    OPERATORS_BY_TYPE,
    defaultOperator
  } from "./query_builder_utils";
  import type { TransactionPredicate } from "./query_builder_utils";

  /** List of known account names for autocomplete. */
  export let allAccounts: string[] = [];

  /** The currently active filter predicate – bind this in the parent to react to changes. */
  export let predicate: TransactionPredicate = () => true;

  let filters: Filter[] = [];
  let logic: LogicOperator = "AND";

  // Form state
  let showForm = false;
  let selectedType: FilterType = "account";
  let selectedOperator: FilterOperator = defaultOperator("account");
  let inputValue = "";

  $: predicate = combinePredicate(filters, logic);
  $: operators = OPERATORS_BY_TYPE[selectedType];

  function handleTypeChange() {
    selectedOperator = defaultOperator(selectedType);
    inputValue = "";
  }

  function addFilter() {
    const needsValue = selectedOperator !== "is_set" && selectedOperator !== "is_not_set";
    if (needsValue && inputValue.trim() === "") return;

    filters = [
      ...filters,
      {
        id: crypto.randomUUID(),
        type: selectedType,
        operator: selectedOperator,
        value: inputValue.trim()
      }
    ];

    showForm = false;
    inputValue = "";
  }

  function removeFilter(id: string) {
    filters = filters.filter((f) => f.id !== id);
  }

  function clearAll() {
    filters = [];
  }

  function toggleLogic() {
    logic = logic === "AND" ? "OR" : "AND";
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Enter") addFilter();
    if (e.key === "Escape") {
      showForm = false;
      inputValue = "";
    }
  }
</script>

<div class="query-builder is-flex is-flex-wrap-wrap is-align-items-center" style="gap: 0.4rem;">
  <!-- Active filter chips -->
  {#each filters as filter, i}
    {#if i > 0}
      <button
        class="button is-small is-rounded logic-toggle"
        on:click={toggleLogic}
        title="Click to toggle AND / OR"
      >
        {logic}
      </button>
    {/if}
    <span class="tags has-addons mb-0">
      <span class="tag is-info is-light">{filterLabel(filter)}</span>
      <a
        class="tag is-delete"
        role="button"
        title="Remove filter"
        on:click={() => removeFilter(filter.id)}
        on:keydown={(e) => e.key === "Enter" && removeFilter(filter.id)}
        tabindex="0"
      />
    </span>
  {/each}

  <!-- Inline add-filter form -->
  {#if showForm}
    <div class="is-flex is-align-items-center is-flex-wrap-wrap" style="gap: 0.35rem;">
      <!-- Filter type selector -->
      <div class="select is-small">
        <select bind:value={selectedType} on:change={handleTypeChange}>
          <option value="account">account</option>
          <option value="amount">amount</option>
          <option value="date">date</option>
          <option value="tag">tag</option>
        </select>
      </div>

      <!-- Operator selector -->
      <div class="select is-small">
        <select bind:value={selectedOperator}>
          {#each operators as op}
            <option value={op.value}>{op.label}</option>
          {/each}
        </select>
      </div>

      <!-- Value input (hidden for is_set / is_not_set) -->
      {#if selectedOperator !== "is_set" && selectedOperator !== "is_not_set"}
        {#if selectedType === "account"}
          <input
            class="input is-small"
            list="qb-account-suggestions"
            type="text"
            bind:value={inputValue}
            placeholder="account name"
            style="width: 170px"
            on:keydown={handleKeydown}
          />
          <datalist id="qb-account-suggestions">
            {#each allAccounts as account}
              <option value={account} />
            {/each}
          </datalist>
        {:else if selectedType === "amount"}
          <input
            class="input is-small"
            type="number"
            bind:value={inputValue}
            placeholder="0"
            style="width: 100px"
            on:keydown={handleKeydown}
          />
        {:else if selectedType === "date"}
          <input
            class="input is-small"
            type="date"
            bind:value={inputValue}
            on:keydown={handleKeydown}
          />
        {:else if selectedType === "tag"}
          <input
            class="input is-small"
            type="text"
            bind:value={inputValue}
            placeholder="tag value"
            style="width: 130px"
            on:keydown={handleKeydown}
          />
        {/if}
      {/if}

      <button class="button is-small is-primary is-light" on:click={addFilter}>Add</button>
      <button
        class="button is-small is-light"
        on:click={() => {
          showForm = false;
          inputValue = "";
        }}
      >
        Cancel
      </button>
    </div>
  {:else}
    <button class="button is-small is-light" on:click={() => (showForm = true)} title="Add filter">
      <span class="icon is-small">
        <i class="fas fa-filter" />
      </span>
      <span>Add Filter</span>
    </button>

    {#if filters.length > 0}
      <button class="button is-small is-light" on:click={clearAll} title="Clear all filters">
        <span class="icon is-small">
          <i class="fas fa-times" />
        </span>
        <span>Clear</span>
      </button>
    {/if}
  {/if}
</div>

<style>
  .logic-toggle {
    min-width: 2.5rem;
  }
</style>
