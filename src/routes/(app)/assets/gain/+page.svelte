<script lang="ts">
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import { buildLegends, renderOverview } from "$lib/gain";
  import { ajax, formatCurrency, formatPercentage, type Gain, type Legend } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";

  let legends: Legend[] = $state([]);
  let gains: Gain[] = $state([]);

  onMount(async () => {
    const { gain_breakdown } = await ajax("/api/gain");
    gains = gain_breakdown;

    legends = buildLegends();
    renderOverview(gains);
  });
</script>

<section class="section tab-gain">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto">
          <LegendCard {legends} clazz="ml-4" />
          <svg id="d3-gain-overview" />
        </div>
      </div>
    </div>
    <BoxLabel text="Gain Overview" />
  </div>
</section>
<section class="section tab-gain">
  <div class="container is-fluid d3-gain-timeline-breakdown">
    <div class="columns">
      <div id="d3-gain-timeline-breakdown" class="column is-12"></div>
    </div>
  </div>
</section>
<section class="section tab-gain">
  <div class="container is-fluid">
    <div class="box overflow-x-auto">
      <table class="table is-fullwidth is-striped is-hoverable is-narrow">
        <thead>
          <tr>
            <th>Account</th>
            <th class="has-text-right">Price Appreciation</th>
            <th class="has-text-right">Income Received</th>
            <th class="has-text-right">Total Return</th>
            <th class="has-text-right">TTM Yield</th>
          </tr>
        </thead>
        <tbody>
          {#each _.sortBy(gains, (g) => g.account) as gain}
            <tr>
              <td>
                <a href="/assets/gain/{encodeURIComponent(gain.account)}">{gain.account}</a>
              </td>
              <td class="has-text-right">{formatCurrency(gain.price_appreciation)}</td>
              <td class="has-text-right">{formatCurrency(gain.income_received)}</td>
              <td class="has-text-right">{formatCurrency(gain.total_return)}</td>
              <td class="has-text-right">{formatPercentage(gain.ttm_yield, 2)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
</section>
