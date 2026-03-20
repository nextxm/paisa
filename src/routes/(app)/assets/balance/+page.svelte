<script lang="ts">
  import AssetsBalance from "$lib/components/AssetsBalance.svelte";
  import { ajax, type AssetBreakdown } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";

  let breakdowns: Record<string, AssetBreakdown> = {};
  let reportCurrency = "";
  let availableCurrencies: string[] = [];

  async function fetchBreakdowns() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
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
      {#if availableCurrencies.length > 1}
        <div class="column is-12 pb-0">
          <div class="box p-3">
            <div class="field is-grouped is-grouped-multiline mb-0">
              <p class="control">
                <span class="select is-small">
                  <select
                    bind:value={reportCurrency}
                    on:change={() => fetchBreakdowns()}
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
                    on:click={() => {
                      reportCurrency = "";
                      fetchBreakdowns();
                    }}
                  >
                    <span class="icon is-small"><i class="fas fa-times" /></span>
                    <span>Reset Currency</span>
                  </button>
                </p>
              {/if}
            </div>
          </div>
        </div>
      {/if}
      <div class="column is-12 pb-0">
        <AssetsBalance {breakdowns} />
      </div>
    </div>
  </div>
</section>
