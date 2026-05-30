<script lang="ts">
  import { generateColorScheme } from "$lib/colors";
  import { formatCurrency, type CurrencyExposure } from "$lib/utils";
  import _ from "lodash";

  let { exposures = [] as CurrencyExposure[] } = $props();

  const total = $derived(_.sumBy(exposures, (e) => e.amount));
  const currencies = $derived(exposures.map((e) => e.currency));
  const colors = $derived(generateColorScheme(currencies));
  const gradient = $derived(
    (() => {
      let offset = 0;
      return exposures
        .map((exposure) => {
          const span = total > 0 ? (exposure.amount / total) * 100 : 0;
          const start = offset;
          const end = offset + span;
          offset = end;
          return `${colors(exposure.currency)} ${start}% ${end}%`;
        })
        .join(", ");
    })()
  );
</script>

<div class="columns is-mobile is-vcentered">
  <div class="column is-narrow">
    <div class="currency-exposure-donut" style="background: conic-gradient({gradient})">
      <div class="currency-exposure-donut-hole"></div>
    </div>
  </div>
  <div class="column">
    {#if exposures.length === 0}
      <p class="is-size-7 has-text-grey">No currency exposure available.</p>
    {:else}
      {#each exposures as exposure}
        <div class="is-flex is-justify-content-space-between is-size-7 mb-1">
          <div class="is-flex is-align-items-center">
            <span
              class="mr-2"
              style="display:inline-block;width:10px;height:10px;border-radius:50%;background:{colors(
                exposure.currency
              )}"
            ></span>
            <span>{exposure.currency}</span>
          </div>
          <span>{formatCurrency(exposure.amount)} ({exposure.percentage.toFixed(1)}%)</span>
        </div>
      {/each}
    {/if}
  </div>
</div>

<style>
  .currency-exposure-donut {
    width: 140px;
    height: 140px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .currency-exposure-donut-hole {
    width: 70px;
    height: 70px;
    border-radius: 50%;
    background: var(--bulma-scheme-main, #fff);
  }
</style>
