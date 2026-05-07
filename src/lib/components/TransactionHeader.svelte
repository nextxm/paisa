<script lang="ts">
  import { isMobile } from "$lib/utils";

  let { showExtraColumns = false }: { showExtraColumns?: boolean } = $props();
  const mobile = isMobile();
</script>

{#if !mobile}
  <div
    class="columns is-flex-wrap-wrap transaction-header has-text-grey is-size-7 has-text-weight-bold mx-0 mb-2"
  >
    <div class="column py-0" style="flex: 0 0 30%; max-width: 30%;">Date & Narration</div>
    <div class="column py-0" style="flex: 0 0 70%; max-width: 70%;">
      <div class="is-flex">
        <div style="flex: 0 0 40%; max-width: 40%;">Account</div>
        <div class="has-text-right" style="flex: 0 0 20%; max-width: 20%;">Conversion</div>
        <div class="has-text-right" style="flex: 0 0 15%; max-width: 15%;">Amount</div>
        {#if showExtraColumns}
          <div class="has-text-right" style="flex: 0 0 10%; max-width: 10%;">Quantity</div>
          <div class="has-text-right" style="flex: 0 0 15%; max-width: 15%;">Balance</div>
        {:else}
          <div style="flex: 0 0 25%; max-width: 25%;"></div>
        {/if}
      </div>
    </div>
  </div>
{/if}

<style lang="scss">
  .transaction-header {
    border-bottom: 1px solid var(--fa-border-color, #dbdbdb);
    padding-bottom: 4px;
    margin-top: -12px;
    opacity: 0.8;
  }
</style>
