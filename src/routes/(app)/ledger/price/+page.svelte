<script lang="ts">
  import Toggleable from "$lib/components/Toggleable.svelte";
  import ValueChange from "$lib/components/ValueChange.svelte";
  import { ajax, formatCurrency, type Price } from "$lib/utils";
  import { toast } from "bulma-toast";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";

  // Legacy (backward-compatible) data: keyed by commodity name.
  let legacyPrices: Record<string, Price[]> = {};
  // Flat list returned by the filtered API.
  let filteredPrices: Price[] = [];

  // Filter state – all empty means "no filter" (backward-compatible mode).
  let filterBase = "";
  let filterQuote = "";
  let filterSource = "";
  let reportCurrency = "";

  // Options for filter dropdowns, derived from the legacy data on first load.
  let availableBases: string[] = [];
  let availableQuotes: string[] = [];
  let availableSources: string[] = [];

  const ITEM_SIZE = 18;

  $: isFiltered =
    filterBase !== "" || filterQuote !== "" || filterSource !== "" || reportCurrency !== "";

  // Reactively re-derive dropdown options whenever legacyPrices changes.
  $: {
    availableBases = _.sortBy(Object.keys(legacyPrices));
    const allPrices = _.flatten(Object.values(legacyPrices));
    availableQuotes = _.sortBy(_.uniq(allPrices.map((p) => p.quote_commodity).filter(Boolean)));
    availableSources = _.sortBy(_.uniq(allPrices.map((p) => p.source).filter(Boolean)));
  }

  function change(prices: Price[], days: number, tolerance: number) {
    const first = prices[0];
    if (!first) return null;

    const date = first.date.subtract(days, "day");
    const last = _.find(prices, (p) => p.date.isSameOrBefore(date, "day"));
    if (!last) return null;

    const diffDays = first.date.diff(last.date, "day");
    if (Math.abs(diffDays - days) <= tolerance) {
      return (first.value - last.value) / last.value;
    }
    return null;
  }

  async function clearPriceCache() {
    const { success, message } = await ajax("/api/price/delete", { method: "POST" });
    if (!success) {
      toast({
        message: `Failed to clear price cache. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
    } else {
      toast({
        message: "Price cache cleared.",
        type: "is-success"
      });
    }
    await fetchPrice();
  }

  async function fetchPrice() {
    // Compute filter state directly so this function always sees latest values,
    // even when called immediately after clearFilters() in the same event handler.
    const filtered =
      filterBase !== "" || filterQuote !== "" || filterSource !== "" || reportCurrency !== "";
    if (filtered) {
      const params = new URLSearchParams();
      if (filterBase) params.set("base", filterBase);
      if (filterQuote) params.set("quote", filterQuote);
      if (filterSource) params.set("source", filterSource);
      if (reportCurrency) params.set("report_currency", reportCurrency);
      const result: { prices: Price[] } = await ajax(`/api/price?${params.toString()}`);
      filteredPrices = result.prices || [];
    } else {
      const result: { prices: Record<string, Price[]> } = await ajax("/api/price");
      legacyPrices = _.omitBy(result.prices, (v) => v.length === 0);
    }
  }

  function clearFilters() {
    filterBase = "";
    filterQuote = "";
    filterSource = "";
    reportCurrency = "";
  }

  onMount(async () => {
    await fetchPrice();
  });
</script>

<section class="section tab-price">
  <div class="container is-fluid">
    <div class="columns is-multiline">
      <div class="column is-12">
        <div class="box p-3">
          <div class="field is-grouped is-grouped-multiline mb-0">
            <p class="control">
              <button
                class="button is-small is-link invertable is-light is-danger"
                on:click={(_e) => clearPriceCache()}
              >
                <span class="icon is-small">
                  <i class="fas fa-trash-can" />
                </span>
                <span>Clear Price Cache</span>
              </button>
            </p>
            <p class="control">
              <div class="select is-small">
                <select bind:value={filterBase} on:change={() => fetchPrice()}>
                  <option value="">All Base</option>
                  {#each availableBases as base}
                    <option value={base}>{base}</option>
                  {/each}
                </select>
              </div>
            </p>
            <p class="control">
              <div class="select is-small">
                <select bind:value={filterQuote} on:change={() => fetchPrice()}>
                  <option value="">All Quote</option>
                  {#each availableQuotes as quote}
                    <option value={quote}>{quote}</option>
                  {/each}
                </select>
              </div>
            </p>
            <p class="control">
              <div class="select is-small">
                <select bind:value={filterSource} on:change={() => fetchPrice()}>
                  <option value="">All Sources</option>
                  {#each availableSources as src}
                    <option value={src}>{src}</option>
                  {/each}
                </select>
              </div>
            </p>
            <p class="control">
              <div class="select is-small">
                <select bind:value={reportCurrency} on:change={() => fetchPrice()}>
                  <option value="">No Conversion</option>
                  {#each availableQuotes as quote}
                    <option value={quote}>{quote}</option>
                  {/each}
                </select>
              </div>
            </p>
            {#if isFiltered}
              <p class="control">
                <button
                  class="button is-small is-light"
                  on:click={() => {
                    clearFilters();
                    fetchPrice();
                  }}
                >
                  <span class="icon is-small">
                    <i class="fas fa-times" />
                  </span>
                  <span>Clear Filters</span>
                </button>
              </p>
            {/if}
          </div>
        </div>
      </div>

      {#if isFiltered}
        <!-- Filtered flat-list view -->
        <div class="column is-12">
          <div class="box overflow-x-auto">
            <table class="table is-narrow is-fullwidth is-light-border is-hoverable">
              <thead>
                <tr>
                  <th>Date</th>
                  <th>Base</th>
                  <th>Quote</th>
                  <th class="has-text-right">Value</th>
                  <th>Source</th>
                </tr>
              </thead>
              <tbody class="has-text-grey-dark">
                {#each filteredPrices as p}
                  <tr>
                    <td class="whitespace-nowrap">{p.date.format("DD MMM YYYY")}</td>
                    <td>{p.commodity_name}</td>
                    <td>{p.quote_commodity}</td>
                    <td class="has-text-right">{formatCurrency(p.value, 4)}</td>
                    <td>{p.source}</td>
                  </tr>
                {/each}
                {#if filteredPrices.length === 0}
                  <tr>
                    <td colspan="5" class="has-text-centered has-text-grey">No prices found.</td>
                  </tr>
                {/if}
              </tbody>
            </table>
          </div>
        </div>
      {:else}
        <!-- Default backward-compatible grouped view -->
        <div class="column is-12">
          <div class="box overflow-x-auto">
            <table class="table is-narrow is-fullwidth is-light-border is-hoverable">
              <thead>
                <tr>
                  <th />
                  <th>Commodity Name</th>
                  <th>Last Date</th>
                  <th class="has-text-right">Last Price</th>
                  <th class="has-text-right">1 Day</th>
                  <th class="has-text-right">1 Week</th>
                  <th class="has-text-right">4 Weeks</th>
                  <th class="has-text-right">1 Year</th>
                  <th class="has-text-right">3 Years</th>
                  <th class="has-text-right">5 Years</th>
                  <th>Commodity Type</th>
                  <th>Commodity ID</th>
                </tr>
              </thead>
              <tbody class="has-text-grey-dark">
                {#each Object.keys(legacyPrices) as commodity}
                  {@const p = legacyPrices[commodity][0]}
                  <Toggleable>
                    <tr
                      class={active ? "is-active" : ""}
                      style="cursor: pointer;"
                      slot="toggle"
                      let:active
                      let:onclick
                      on:click={(e) => onclick(e)}
                    >
                      <td>
                        <span class="icon has-text-link">
                          <i
                            class="fas {active ? 'fa-chevron-up' : 'fa-chevron-down'}"
                            aria-hidden="true"
                          />
                        </span>
                      </td>

                      <td>{p.commodity_name}</td>
                      <td class="whitespace-nowrap">{p.date.format("DD MMM YYYY")}</td>
                      <td class="has-text-right">{formatCurrency(p.value, 4)}</td>
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 1, 0)} /></td
                      >
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 7, 2)} /></td
                      >
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 28, 4)} />
                      </td>
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 365, 7)} />
                      </td>
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 365 * 3, 7)} /></td
                      >
                      <td class="has-text-right"
                        ><ValueChange value={change(legacyPrices[commodity], 365 * 5, 7)} /></td
                      >
                      <td>{p.commodity_type}</td>
                      <td>{p.commodity_id}</td>
                    </tr>
                    <tr slot="content">
                      <td colspan="10" />
                      <td colspan="2" class="p-0">
                        <div>
                          <VirtualList
                            width="100%"
                            height={_.min([
                              ITEM_SIZE * legacyPrices[commodity].length,
                              ITEM_SIZE * 20
                            ])}
                            itemCount={legacyPrices[commodity].length}
                            itemSize={ITEM_SIZE}
                          >
                            <div
                              slot="item"
                              let:index
                              let:style
                              {style}
                              class="small-box is-flex is-flex-wrap-wrap is-justify-content-space-between is-size-7"
                            >
                              {@const hp = legacyPrices[commodity][index]}
                              <div class="pl-1">{hp.date.format("DD MMM YYYY")}</div>
                              <div class="has-text-grey-light is-size-7 pl-1">
                                {hp.quote_commodity || ""}
                              </div>
                              <div class="has-text-grey-light is-size-7 pl-1">
                                {hp.source || ""}
                              </div>
                              <div class="pr-1 has-text-right">
                                {formatCurrency(hp.value, 4)}
                              </div>
                            </div>
                          </VirtualList>
                        </div>
                      </td>
                    </tr>
                  </Toggleable>
                {/each}
              </tbody>
            </table>
          </div>
        </div>
      {/if}
    </div>
  </div>
</section>
