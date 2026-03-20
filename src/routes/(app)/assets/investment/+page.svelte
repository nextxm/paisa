<script lang="ts">
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import {
    renderMonthlyInvestmentTimeline,
    renderYearlyCards,
    renderYearlyInvestmentTimeline
  } from "$lib/investment";
  import { ajax, type Legend } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";

  let monthlyInvestmentTimelineLegends: Legend[] = [];
  let yearlyInvestmentTimelineLegends: Legend[] = [];
  let reportCurrency = "";
  let availableCurrencies: string[] = [];

  async function fetchInvestment() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
    const query = params.toString();
    const { assets: assets, yearly_cards: yearlyCards } = await ajax(
      query ? `/api/investment?${query}` : "/api/investment"
    );
    monthlyInvestmentTimelineLegends = renderMonthlyInvestmentTimeline(assets);
    yearlyInvestmentTimelineLegends = renderYearlyInvestmentTimeline(yearlyCards);
    renderYearlyCards(yearlyCards);
  }

  onMount(async () => {
    const [, currencyResult] = await Promise.all([
      fetchInvestment(),
      ajax("/api/price/currencies")
    ]);
    availableCurrencies = currencyResult.currencies || [];
  });
</script>

<section class="section tab-investment">
  <div class="container is-fluid">
    {#if availableCurrencies.length > 1}
      <div class="box p-3 mb-4">
        <div class="field is-grouped is-grouped-multiline mb-0">
          <p class="control">
            <span class="select is-small">
              <select
                bind:value={reportCurrency}
                on:change={() => fetchInvestment()}
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
                  fetchInvestment();
                }}
              >
                <span class="icon is-small"><i class="fas fa-times" /></span>
                <span>Reset Currency</span>
              </button>
            </p>
          {/if}
        </div>
      </div>
    {/if}
    <div class="columns">
      <div class="column is-12">
        <div class="box">
          <LegendCard legends={monthlyInvestmentTimelineLegends} clazz="ml-4" />
          <svg id="d3-investment-timeline" width="100%" height="500" />
        </div>
      </div>
    </div>
    <BoxLabel text="Monthly Investment Timeline" />
  </div>
</section>
<section class="section tab-investment">
  <div class="container is-fluid">
    <div class="columns is-flex-wrap-wrap">
      <div class="column is-full-tablet is-half-fullhd">
        <div class="box px-2">
          <LegendCard legends={yearlyInvestmentTimelineLegends} clazz="ml-4" />
          <svg id="d3-yearly-investment-timeline" width="100%" />
        </div>
        <BoxLabel text="Financial Year Investment Timeline" />
      </div>
      <div class="column is-full-tablet is-half-fullhd">
        <div class="columns is-flex-wrap-wrap" id="d3-yearly-investment-cards" />
      </div>
    </div>
  </div>
</section>
