<script lang="ts">
  import AssetsBalance from "$lib/components/AssetsBalance.svelte";
  import { downloadAssetBalanceCSV, downloadAssetBalanceExcel } from "$lib/export";
  import { ajax, now, type AssetBreakdown } from "$lib/utils";
  import dayjs from "dayjs";
  import { onMount } from "svelte";

  let breakdowns: Record<string, AssetBreakdown> = $state({});
  let reportCurrency = $state("");
  let availableCurrencies: string[] = $state([]);
  let flatAccounts = $state(false);
  let filterInactive = $state(true);
  let filterZero = $state(true);
  let asOfDate = $state(now().format("YYYY-MM-DD"));

  async function fetchBreakdowns() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
    params.set("as_of_date", asOfDate);
    const query = params.toString();
    ({ asset_breakdowns: breakdowns } = await ajax(
      query ? `/api/assets/balance?${query}` : "/api/assets/balance"
    ));
  }

  onMount(async () => {
    const [, currencyResult] = await Promise.all([
      fetchBreakdowns(),
      ajax("/api/price/currencies")
    ]);
    availableCurrencies = currencyResult.currencies || [];
  });
</script>

<section class="section pb-0">
  <div class="container is-fluid">
    <div class="columns is-flex-wrap-wrap">
      <div class="column is-12 pb-0">
        <div class="box p-3">
          <div class="field is-grouped is-grouped-multiline mb-0">
            {#if availableCurrencies.length > 1}
              <p class="control">
                <span class="select is-small">
                  <select
                    bind:value={reportCurrency}
                    onchange={() => fetchBreakdowns()}
                    title="Report Currency"
                  >
                    <option value="">Default Currency</option>
                    {#each availableCurrencies as currency}
                      <option value={currency}>{currency}</option>
                    {/each}
                  </select>
                </span>
              </p>
              {#if reportCurrency}
                <p class="control">
                  <button
                    class="button is-small is-light"
                    onclick={() => {
                      reportCurrency = "";
                      fetchBreakdowns();
                    }}
                  >
                    <span class="icon is-small"><i class="fas fa-times"></i></span>
                    <span>Reset Currency</span>
                  </button>
                </p>
              {/if}
            {/if}
            <div class="control">
              <div class="field mb-0 is-flex is-align-items-center">
                <label
                  class="label is-size-7 mb-0 mr-2"
                  for="assets-balance-as-of"
                  style="min-width: fit-content;">View as of</label
                >
                <input
                  id="assets-balance-as-of"
                  class="input is-small"
                  type="date"
                  bind:value={asOfDate}
                  onchange={() => fetchBreakdowns()}
                />
              </div>
            </div>
            <div class="control">
              <div class="field mb-0">
                <input
                  id="flat-assets-balance"
                  type="checkbox"
                  class="switch is-rounded is-small"
                  bind:checked={flatAccounts}
                />
                <label for="flat-assets-balance">Flat Accounts</label>
              </div>
            </div>
            <div class="control">
              <div class="field mb-0">
                <input
                  id="filter-inactive-balance"
                  type="checkbox"
                  class="switch is-rounded is-small"
                  bind:checked={filterInactive}
                />
                <label for="filter-inactive-balance">Filter Inactive</label>
              </div>
            </div>
            <div class="control">
              <div class="field mb-0">
                <input
                  id="filter-zero-balance"
                  type="checkbox"
                  class="switch is-rounded is-small"
                  bind:checked={filterZero}
                />
                <label for="filter-zero-balance">Filter 0</label>
              </div>
            </div>
            <p class="control">
              <button
                type="button"
                class="button is-small is-text"
                onclick={() =>
                  downloadAssetBalanceCSV(breakdowns, flatAccounts, filterInactive, filterZero)}
              >
                <span class="icon is-small">
                  <i class="fa-solid fa-file-csv"></i>
                </span>
                <span>CSV</span>
              </button>
            </p>
            <p class="control">
              <button
                type="button"
                class="button is-small is-text"
                onclick={() =>
                  downloadAssetBalanceExcel(breakdowns, flatAccounts, filterInactive, filterZero)}
              >
                <span class="icon is-small">
                  <i class="fa-solid fa-file-excel"></i>
                </span>
                <span>Excel</span>
              </button>
            </p>
          </div>
        </div>
      </div>
      <div class="column is-12 pb-0">
        <p class="is-size-7 has-text-grey">
          Balances as of {dayjs(asOfDate).format("MMM D, YYYY")}
        </p>
      </div>
      <div class="column is-12 pb-0">
        <AssetsBalance {breakdowns} {filterInactive} {filterZero} indent={!flatAccounts} />
      </div>
    </div>
  </div>
</section>
