<script lang="ts">
  import {
    ajax,
    formatCurrency,
    formatFloat,
    isMobile,
    type Legend,
    type Networth
  } from "$lib/utils";
  import COLORS from "$lib/colors";
  import { renderNetworth } from "$lib/networth";
  import _ from "lodash";
  import { onDestroy, onMount } from "svelte";
  import { dateRange, setAllowedDateRange } from "../../../../store";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let networth = 0;
  let investment = 0;
  let gain = 0;
  let xirr = 0;
  let svg: Element;
  let destroy: () => void;
  let points: Networth[] = [];
  let legends: Legend[] = [];
  let reportCurrency = "";
  let availableCurrencies: string[] = [];

  $: if (!_.isEmpty(points)) {
    if (destroy) {
      destroy();
    }

    ({ destroy, legends } = renderNetworth(
      _.filter(
        points,
        (p) => p.date.isSameOrBefore($dateRange.to) && p.date.isSameOrAfter($dateRange.from)
      ),
      svg
    ));
  }

  onDestroy(async () => {
    if (destroy) {
      destroy();
    }
  });

  async function fetchNetworth() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
    const query = params.toString();
    const result = await ajax(query ? `/api/networth?${query}` : "/api/networth");
    points = result.networthTimeline;
    setAllowedDateRange(_.map(points, (p) => p.date));

    const current = _.last(points);
    if (current) {
      networth = current.investmentAmount + current.gainAmount - current.withdrawalAmount;
      investment = current.investmentAmount - current.withdrawalAmount;
      gain = current.gainAmount;
    }
    xirr = result.xirr;
  }

  onMount(async () => {
    const [, currencyResult] = await Promise.all([
      fetchNetworth(),
      ajax("/api/price/currencies")
    ]);
    availableCurrencies = currencyResult.currencies || [];
  });
</script>

<section class="section tab-networth">
  <div class="container is-fluid">
    {#if availableCurrencies.length > 1}
      <div class="box p-3 mb-4">
        <div class="field is-grouped is-grouped-multiline mb-0">
          <p class="control">
            <span class="select is-small">
              <select
                bind:value={reportCurrency}
                on:change={() => fetchNetworth()}
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
                  fetchNetworth();
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
    <nav class="level {isMobile() && 'grid-2'}">
      <LevelItem title="Net worth" color={COLORS.primary} value={formatCurrency(networth)} />
      <LevelItem
        title="Net Investment"
        color={COLORS.secondary}
        value={formatCurrency(investment)}
      />
      <LevelItem
        title="Gain / Loss"
        color={gain >= 0 ? COLORS.gainText : COLORS.lossText}
        value={formatCurrency(gain)}
      />
      <LevelItem title="XIRR" value={formatFloat(xirr)} />
    </nav>
  </div>
</section>

<section class="section tab-networth">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto">
          <ZeroState item={points}>
            <strong>Oops!</strong> You have no transactions.
          </ZeroState>

          <LegendCard {legends} clazz="ml-4" />
          <svg id="d3-networth-timeline" height="500" bind:this={svg} />
        </div>
      </div>
    </div>
    <BoxLabel text="Networth Timeline" />
  </div>
</section>
