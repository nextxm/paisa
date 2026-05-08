<script lang="ts">
  import { accountColorStyle } from "$lib/colors";
  import { iconText } from "$lib/icon";
  import { restName, type AssetBreakdown, formatCurrency, firstName } from "$lib/utils";

  let { assetBreakdown }: { assetBreakdown: AssetBreakdown } = $props();
</script>

<div class="box p-3 has-background-white">
  <div class="my-1 is-flex is-justify-content-space-between">
    <div class="has-text-grey truncate custom-icon" title={assetBreakdown.group}>
      <span style={accountColorStyle(firstName(assetBreakdown.group))}
        >{iconText(assetBreakdown.group)}</span
      >
      <a class="secondary-link" href="/assets/gain/{assetBreakdown.group}">
        {restName(restName(assetBreakdown.group))}</a
      >
    </div>
    <div class="has-text-weight-bold is-size-6">
      <a
        class="secondary-link has-text-weight-bold"
        href="/accounts/{encodeURIComponent(assetBreakdown.group)}/transactions"
        title="View transactions for {assetBreakdown.group}"
      >
        {#if assetBreakdown.originalBalances && assetBreakdown.originalBalances.length > 0}
          {#each assetBreakdown.originalBalances as ob}
            {formatCurrency(ob.amount)}
            {ob.currency}
          {/each}
        {:else}
          {formatCurrency(assetBreakdown.marketAmount)}
          {USER_CONFIG.default_currency}
        {/if}
      </a>
    </div>
  </div>
</div>
