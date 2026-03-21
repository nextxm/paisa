<script lang="ts">
  import ValueChange from "$lib/components/ValueChange.svelte";
  import { ajax, formatCurrency, type Price, type PriceFilters } from "$lib/utils";
  import { toast } from "bulma-toast";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";

  type HistoryMode = "latest" | "all";

  let groupedPrices: Record<string, Price[]> = {};
  let expandedRows: Record<string, boolean> = {};
  let loadingHistoryRows: Record<string, boolean> = {};
  let loadedHistoryRows: Record<string, boolean> = {};

  let filterBase = "";
  let filterQuote = "";
  let filterSource = "";
  let reportCurrency = "";
  let historyMode: HistoryMode = "latest";

  let availableBases: string[] = [];
  let availableQuotes: string[] = [];
  let availableSources: string[] = [];

  const ITEM_SIZE = 18;

  $: hasActiveFilters =
    filterBase !== "" || filterQuote !== "" || filterSource !== "" || reportCurrency !== "";
  $: priceEntries = _.sortBy(Object.entries(groupedPrices), ([commodity]) => commodity);

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

  function buildPriceRoute(base = filterBase, mode: HistoryMode = historyMode) {
    const params = new URLSearchParams();
    if (base) params.set("base", base);
    if (filterQuote) params.set("quote", filterQuote);
    if (filterSource) params.set("source", filterSource);
    if (reportCurrency) params.set("report_currency", reportCurrency);
    if (mode === "all") params.set("history", mode);

    const query = params.toString();
    return query ? `/api/price?${query}` : "/api/price";
  }

  async function loadFilterOptions() {
    const result: PriceFilters = await ajax("/api/price/filters");
    availableBases = result.bases || [];
    availableQuotes = result.quotes || [];
    availableSources = result.sources || [];
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
    await Promise.all([fetchPrice(), loadFilterOptions()]);
  }

  async function fetchPrice() {
    const result: { prices: Record<string, Price[]> } = await ajax(buildPriceRoute());
    groupedPrices = _.omitBy(result.prices || {}, (prices) => prices.length === 0);
    expandedRows = {};
    loadingHistoryRows = {};
    loadedHistoryRows =
      historyMode === "all"
        ? Object.fromEntries(Object.keys(groupedPrices).map((commodity) => [commodity, true]))
        : {};
  }

  async function loadHistoryForCommodity(commodity: string) {
    if (historyMode === "all" || loadedHistoryRows[commodity] || loadingHistoryRows[commodity]) {
      return;
    }

    loadingHistoryRows = { ...loadingHistoryRows, [commodity]: true };
    try {
      const result: { prices: Record<string, Price[]> } = await ajax(buildPriceRoute(commodity, "all"));
      groupedPrices = {
        ...groupedPrices,
        [commodity]: result.prices?.[commodity] || groupedPrices[commodity] || []
      };
      loadedHistoryRows = { ...loadedHistoryRows, [commodity]: true };
    } finally {
      loadingHistoryRows = { ...loadingHistoryRows, [commodity]: false };
    }
  }

  async function toggleCommodity(commodity: string) {
    const isExpanded = !expandedRows[commodity];
    expandedRows = { ...expandedRows, [commodity]: isExpanded };
    if (isExpanded) {
      await loadHistoryForCommodity(commodity);
    }
  }

  function clearFilters() {
    filterBase = "";
    filterQuote = "";
    filterSource = "";
    reportCurrency = "";
  }

  onMount(async () => {
    await Promise.all([loadFilterOptions(), fetchPrice()]);
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
              <span class="select is-small">
                <select bind:value={filterBase} on:change={() => fetchPrice()}>
                  <option value="">All Base</option>
                  {#each availableBases as base}
                    <option value={base}>{base}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={filterQuote} on:change={() => fetchPrice()}>
                  <option value="">All Quote</option>
                  {#each availableQuotes as quote}
                    <option value={quote}>{quote}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={filterSource} on:change={() => fetchPrice()}>
                  <option value="">All Sources</option>
                  {#each availableSources as src}
                    <option value={src}>{src}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={reportCurrency} on:change={() => fetchPrice()}>
                  <option value="">No Conversion</option>
                  {#each availableQuotes as quote}
                    <option value={quote}>{quote}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={historyMode} on:change={() => fetchPrice()}>
                  <option value="latest">Latest Only</option>
                  <option value="all">Load History</option>
                </select>
              </span>
            </p>
            {#if hasActiveFilters}
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

      <div class="column is-12">
        <div class="box overflow-x-auto">
          <table class="table is-narrow is-fullwidth is-light-border is-hoverable">
            <thead>
              <tr>
                <th />
                <th>Commodity Name</th>
                <th>Last Date</th>
                <th>Quote</th>
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
              {#each priceEntries as [commodity, prices]}
                {@const latest = prices[0]}
                <tr
                  class={expandedRows[commodity] ? "is-active" : ""}
                  style="cursor: pointer;"
                  on:click={() => toggleCommodity(commodity)}
                >
                  <td>
                    <span class="icon has-text-link">
                      {#if loadingHistoryRows[commodity]}
                        <i class="fas fa-spinner fa-spin" aria-hidden="true" />
                      {:else}
                        <i
                          class="fas {expandedRows[commodity] ? 'fa-chevron-up' : 'fa-chevron-down'}"
                          aria-hidden="true"
                        />
                      {/if}
                    </span>
                  </td>
                  <td>{latest.commodity_name}</td>
                  <td class="whitespace-nowrap">{latest.date.format("DD MMM YYYY")}</td>
                  <td>{latest.quote_commodity}</td>
                  <td class="has-text-right">{formatCurrency(latest.value, 4)}</td>
                  <td class="has-text-right"><ValueChange value={change(prices, 1, 0)} /></td>
                  <td class="has-text-right"><ValueChange value={change(prices, 7, 2)} /></td>
                  <td class="has-text-right"><ValueChange value={change(prices, 28, 4)} /></td>
                  <td class="has-text-right"><ValueChange value={change(prices, 365, 7)} /></td>
                  <td class="has-text-right"><ValueChange value={change(prices, 365 * 3, 7)} /></td>
                  <td class="has-text-right"><ValueChange value={change(prices, 365 * 5, 7)} /></td>
                  <td>{latest.commodity_type}</td>
                  <td>{latest.commodity_id}</td>
                </tr>
                {#if expandedRows[commodity]}
                  <tr>
                    <td colspan="13" class="p-0">
                      <VirtualList
                        width="100%"
                        height={_.min([ITEM_SIZE * prices.length, ITEM_SIZE * 20])}
                        itemCount={prices.length}
                        itemSize={ITEM_SIZE}
                      >
                        <div
                          slot="item"
                          let:index
                          let:style
                          {style}
                          class="small-box is-flex is-flex-wrap-wrap is-justify-content-space-between is-size-7"
                        >
                          {@const historyPrice = prices[index]}
                          <div class="pl-1">{historyPrice.date.format("DD MMM YYYY")}</div>
                          <div class="has-text-grey-light is-size-7 pl-1">
                            {historyPrice.quote_commodity || ""}
                          </div>
                          <div class="has-text-grey-light is-size-7 pl-1">
                            {historyPrice.source || ""}
                          </div>
                          <div class="pr-1 has-text-right">
                            {formatCurrency(historyPrice.value, 4)}
                          </div>
                        </div>
                      </VirtualList>
                    </td>
                  </tr>
                {/if}
              {/each}
              {#if priceEntries.length === 0}
                <tr>
                  <td colspan="13" class="has-text-centered has-text-grey">No prices found.</td>
                </tr>
              {/if}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</section>
