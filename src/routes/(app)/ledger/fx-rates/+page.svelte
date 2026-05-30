<script lang="ts">
  import CurrencyExposureWidget from "$lib/components/CurrencyExposureWidget.svelte";
  import {
    ajax,
    formatCurrency,
    now,
    type CurrencyExposure,
    type FXRate,
    type PriceFilters
  } from "$lib/utils";
  import dayjs from "dayjs";
  import { onMount } from "svelte";

  let rates: FXRate[] = $state([]);
  let currencyExposure: CurrencyExposure[] = $state([]);
  let base = $state("USD");
  let quote = $state("");
  let year = $state(now().year());
  let month = $state(now().month() + 1);

  let availableBases: string[] = $state([]);
  let availableQuotes: string[] = $state([]);

  const years = Array.from({ length: 10 }, (_, i) => now().year() - i);
  const months = Array.from({ length: 12 }, (_, i) => ({
    value: i + 1,
    label: dayjs().month(i).format("MMMM")
  }));

  async function loadFilterOptions() {
    const result: PriceFilters = await ajax("/api/price/filters");
    availableBases = result.bases || [];
    availableQuotes = result.quotes || [];
    if (!quote && availableQuotes.length > 0) {
      // The backend default is config.DefaultCurrency(), so we don't strictly need to set it here,
      // but it's good for the UI.
    }
  }

  async function fetchRates() {
    const params = new URLSearchParams();
    params.set("base", base);
    if (quote) params.set("quote", quote);
    params.set("year", year.toString());
    params.set("month", month.toString().padStart(2, "0"));

    const result = await ajax(`/api/fx-rates?${params.toString()}`);
    rates = result.rates || [];
    if (result.base) base = result.base;
    if (result.quote) quote = result.quote;
  }

  onMount(async () => {
    const exposureResult = await ajax("/api/currency-exposure");
    currencyExposure = exposureResult.currency_exposure || [];
    await loadFilterOptions();
    await fetchRates();
  });
</script>

<section class="section tab-fx-rates">
  <div class="container is-fluid">
    <div class="columns is-multiline">
      <div class="column is-12">
        <div class="box p-3">
          <CurrencyExposureWidget exposures={currencyExposure} />
        </div>
      </div>

      <div class="column is-12">
        <div class="box p-3">
          <div class="field is-grouped is-grouped-multiline mb-0">
            <p class="control">
              <span class="select is-small">
                <select bind:value={base} onchange={() => fetchRates()}>
                  {#each availableBases as b}
                    <option value={b}>{b}</option>
                  {/each}
                  {#if !availableBases.includes("USD")}
                    <option value="USD">USD</option>
                  {/if}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={quote} onchange={() => fetchRates()}>
                  {#each availableQuotes as q}
                    <option value={q}>{q}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={year} onchange={() => fetchRates()}>
                  {#each years as y}
                    <option value={y}>{y}</option>
                  {/each}
                </select>
              </span>
            </p>
            <p class="control">
              <span class="select is-small">
                <select bind:value={month} onchange={() => fetchRates()}>
                  {#each months as m}
                    <option value={m.value}>{m.label}</option>
                  {/each}
                </select>
              </span>
            </p>
          </div>
        </div>
      </div>

      <div class="column is-12">
        <div class="box overflow-x-auto">
          <table class="table is-narrow is-fullwidth is-light-border is-hoverable">
            <thead>
              <tr>
                <th>Date</th>
                <th class="has-text-right">Rate ({base} → {quote})</th>
                <th class="has-text-centered">Type</th>
              </tr>
            </thead>
            <tbody class="has-text-grey-dark">
              {#each rates as rate}
                <tr>
                  <td>{rate.date.format("DD MMM YYYY")}</td>
                  <td class="has-text-right">{formatCurrency(rate.rate, 4)}</td>
                  <td class="has-text-centered">
                    {#if rate.derived}
                      <span class="tag is-warning is-light">Derived</span>
                    {:else}
                      <span class="tag is-success is-light">Direct</span>
                    {/if}
                  </td>
                </tr>
              {/each}
              {#if rates.length === 0}
                <tr>
                  <td colspan="3" class="has-text-centered has-text-grey"
                    >No exchange rates found for this period.</td
                  >
                </tr>
              {/if}
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</section>
