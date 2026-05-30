<script lang="ts">
  import type { SavedTransactionSearch, TransactionFilters } from "$lib/transaction_filters";

  let {
    filters,
    accounts = [],
    commodities = [],
    savedSearches = [],
    showSavedSearches = false,
    autoFocusSearch = false,
    onFiltersChange = (_filters: TransactionFilters) => {},
    onSaveSearch = (_name: string) => {},
    onLoadSavedSearch = (_name: string) => {}
  }: {
    filters: TransactionFilters;
    accounts?: string[];
    commodities?: string[];
    savedSearches?: SavedTransactionSearch[];
    showSavedSearches?: boolean;
    autoFocusSearch?: boolean;
    onFiltersChange?: (filters: TransactionFilters) => void;
    onSaveSearch?: (name: string) => void;
    onLoadSavedSearch?: (name: string) => void;
  } = $props();

  let expanded = $state(false);
  let selectedSavedSearch = $state("");
  let searchInput: HTMLInputElement = $state();

  $effect(() => {
    if (autoFocusSearch && searchInput) {
      searchInput.focus();
      searchInput.select();
    }
  });

  function update(patch: Partial<TransactionFilters>) {
    filters = { ...filters, ...patch };
    onFiltersChange(filters);
  }

  function clearAll() {
    update({
      q: "",
      amountMin: "",
      amountMax: "",
      account: "",
      commodity: "",
      dateFrom: "",
      dateTo: ""
    });
  }

  function saveCurrentSearch() {
    if (typeof window === "undefined") return;
    const name = window.prompt("Save search as");
    if (!name) return;
    onSaveSearch(name.trim());
  }

  function loadSavedSearch(name: string) {
    if (!name) return;
    onLoadSavedSearch(name);
    selectedSavedSearch = "";
  }
</script>

<div class="box p-3 mb-3">
  <div class="field is-grouped is-grouped-multiline mb-0">
    <p class="control is-expanded">
      <input
        bind:this={searchInput}
        class="input is-small"
        type="search"
        placeholder="Search payee or narration..."
        value={filters.q}
        oninput={(e) => update({ q: (e.currentTarget as HTMLInputElement).value })}
      />
    </p>
    <p class="control">
      <button class="button is-small is-light" onclick={() => (expanded = !expanded)}>
        <span class="icon is-small"
          ><i class="fas {expanded ? 'fa-angle-up' : 'fa-sliders'}"></i></span
        >
        <span>{expanded ? "Hide Filters" : "Filters"}</span>
      </button>
    </p>
    {#if showSavedSearches}
      <p class="control">
        <button
          type="button"
          class="button is-small is-light"
          title="Save current filters"
          onclick={saveCurrentSearch}
        >
          <span class="icon is-small"><i class="fas fa-bookmark"></i></span>
        </button>
      </p>
      <p class="control">
        <span class="select is-small">
          <select
            bind:value={selectedSavedSearch}
            onchange={(e) => loadSavedSearch((e.currentTarget as HTMLSelectElement).value)}
          >
            <option value="">Saved searches</option>
            {#each savedSearches as search}
              <option value={search.name}>{search.name}</option>
            {/each}
          </select>
        </span>
      </p>
    {/if}
    <p class="control">
      <button class="button is-small is-light" onclick={clearAll}>
        <span class="icon is-small"><i class="fas fa-xmark"></i></span>
        <span>Clear</span>
      </button>
    </p>
  </div>

  {#if expanded}
    <div class="field is-grouped is-grouped-multiline mt-3 mb-0">
      <p class="control">
        <input
          class="input is-small"
          type="number"
          placeholder="Min amount"
          value={filters.amountMin}
          oninput={(e) => update({ amountMin: (e.currentTarget as HTMLInputElement).value })}
        />
      </p>
      <p class="control">
        <input
          class="input is-small"
          type="number"
          placeholder="Max amount"
          value={filters.amountMax}
          oninput={(e) => update({ amountMax: (e.currentTarget as HTMLInputElement).value })}
        />
      </p>
      <p class="control">
        <input
          class="input is-small"
          type="date"
          value={filters.dateFrom}
          oninput={(e) => update({ dateFrom: (e.currentTarget as HTMLInputElement).value })}
        />
      </p>
      <p class="control">
        <input
          class="input is-small"
          type="date"
          value={filters.dateTo}
          oninput={(e) => update({ dateTo: (e.currentTarget as HTMLInputElement).value })}
        />
      </p>
      <p class="control">
        <span class="select is-small">
          <select
            value={filters.account}
            onchange={(e) => update({ account: (e.currentTarget as HTMLSelectElement).value })}
          >
            <option value="">All Accounts</option>
            {#each accounts as account}
              <option value={account}>{account}</option>
            {/each}
          </select>
        </span>
      </p>
      <p class="control">
        <span class="select is-small">
          <select
            value={filters.commodity}
            onchange={(e) => update({ commodity: (e.currentTarget as HTMLSelectElement).value })}
          >
            <option value="">All Commodities</option>
            {#each commodities as commodity}
              <option value={commodity}>{commodity}</option>
            {/each}
          </select>
        </span>
      </p>
    </div>
  {/if}
</div>
