<script lang="ts">
  import { ajax, formatCurrency } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import dayjs from "dayjs";

  let base = "USD";
  let quote = "INR";
  let rates: any[] = [];
  let loading = false;
  let availableCurrencies: string[] = [];

  async function loadCurrencies() {
    const result = await ajax("/api/price/currencies");
    availableCurrencies = result.currencies || [];
  }

  async function loadRates() {
    if (!base || !quote) return;
    loading = true;
    try {
      const result = await ajax(`/api/fx/rates?base=${base}&quote=${quote}`);
      rates = result.rates || [];
    } finally {
      loading = false;
    }
  }

  function swap() {
    const tmp = base;
    base = quote;
    quote = tmp;
    loadRates();
  }

  onMount(async () => {
    await loadCurrencies();
    if (availableCurrencies.length > 0) {
      if (!availableCurrencies.includes(base)) base = availableCurrencies[0];
      if (!availableCurrencies.includes(quote))
        quote = availableCurrencies[1] || availableCurrencies[0];
    }
    await loadRates();
  });
</script>

<section class="section tab-fx">
  <div class="container is-fluid">
    <div class="columns is-multiline">
      <div class="column is-12">
        <div class="box p-3">
          <div class="field is-grouped is-grouped-multiline mb-0 is-align-items-center">
            <p class="control">
              <span class="select is-small">
                <select bind:value={base} on:change={() => loadRates()}>
                  {#each availableCurrencies as cur}
                    <option value={cur}>{cur}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <button class="button is-small is-ghost px-1" on:click={swap}>
                <span class="icon is-small">
                  <i class="fas fa-arrows-left-right" />
                </span>
              </button>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={quote} on:change={() => loadRates()}>
                  {#each availableCurrencies as cur}
                    <option value={cur}>{cur}</option>
                  {/each}
                </select>
              </span>
            </p>
            {#if loading}
              <p class="control ml-2">
                <span class="icon is-small has-text-grey-light">
                  <i class="fas fa-spinner fa-spin" />
                </span>
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
                <th>Date</th>
                <th class="has-text-right">Rate</th>
                <th>Type</th>
                <th>Method</th>
              </tr>
            </thead>
            <tbody class="has-text-grey-dark">
              {#each rates as entry}
                <tr>
                  <td>{dayjs(entry.date).format("DD MMM YYYY")}</td>
                  <td class="has-text-right family-monospace">
                    {formatCurrency(entry.rate, 6)}
                  </td>
                  <td>
                    {#if entry.resolution_type === "direct"}
                      <span class="tag is-success is-light is-rounded">Direct</span>
                    {:else if entry.resolution_type === "inverse"}
                      <span class="tag is-info is-light is-rounded">Inverse</span>
                    {:else if entry.resolution_type === "cross"}
                      <span class="tag is-warning is-light is-rounded">Derived</span>
                    {/if}
                  </td>
                  <td>
                    {#if entry.resolution_type === "direct"}
                      {base} &rarr; {quote}
                    {:else if entry.resolution_type === "inverse"}
                      1 / ({quote} &rarr; {base})
                    {:else if entry.resolution_type === "cross"}
                      ({base} &rarr; {entry.anchor}) &times; ({entry.anchor} &rarr; {quote})
                    {/if}
                  </td>
                </tr>
              {/each}
              {#if rates.length === 0 && !loading}
                <tr>
                  <td colspan="4" class="has-text-centered has-text-grey py-5">
                    No exchange rates found for this pair.
                  </td>
                </tr>
              {/if}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</section>

<style>
  .family-monospace {
    font-family: monospace;
  }
</style>
