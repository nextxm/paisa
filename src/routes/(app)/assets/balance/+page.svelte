<script lang="ts">
  import AssetsBalance from "$lib/components/AssetsBalance.svelte";
  import { downloadAssetBalanceCSV, downloadAssetBalanceExcel } from "$lib/export";
  import { ajax, type AccountReconciliationStatus, type AssetBreakdown } from "$lib/utils";
  import { onMount } from "svelte";

  let breakdowns: Record<string, AssetBreakdown> = $state({});
  let reportCurrency = $state("");
  let availableCurrencies: string[] = $state([]);
  let flatAccounts = $state(false);
  let reconciliationStatuses = $state<Record<string, AccountReconciliationStatus>>({});

  async function fetchBreakdowns() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
    if (flatAccounts) params.set("flat", "true");
    const query = params.toString();
    ({ asset_breakdowns: breakdowns } = await ajax(
      query ? `/api/assets/balance?${query}` : "/api/assets/balance"
    ));
  }

  onMount(async () => {
    const [, currencyResult, reconciliationResult] = await Promise.all([
      fetchBreakdowns(),
      ajax("/api/price/currencies"),
      USER_CONFIG.enable_reconciliation
        ? ajax("/api/accounts/reconciliation")
        : Promise.resolve({ reconciliations: [] })
    ]);
    availableCurrencies = currencyResult.currencies || [];
    reconciliationStatuses = Object.fromEntries(
      (reconciliationResult.reconciliations || []).map((status) => [status.account, status])
    );
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
              <div class="field mb-0">
                <input
                  id="flat-assets-balance"
                  type="checkbox"
                  class="switch is-rounded is-small"
                  bind:checked={flatAccounts}
                  onchange={() => fetchBreakdowns()}
                />
                <label for="flat-assets-balance">Flat Accounts</label>
              </div>
            </div>
            <p class="control">
              <button
                type="button"
                class="button is-small is-text"
                onclick={() => downloadAssetBalanceCSV(breakdowns, flatAccounts)}
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
                onclick={() => downloadAssetBalanceExcel(breakdowns, flatAccounts)}
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
        <AssetsBalance {breakdowns} {reconciliationStatuses} indent={!flatAccounts} />
      </div>
    </div>
  </div>
</section>
