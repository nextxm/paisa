<script lang="ts">
  import { formatPercentage } from "$lib/utils";

  let { value }: { value: number } = $props();
  const icon = $derived(value > 0 ? "fa-arrow-up rotate-45" : "fa-arrow-down -rotate-45");
  const color = $derived(value > 0 ? "has-text-success" : "has-text-danger");
</script>

{#if value === null}
  <span></span>
{:else if value == 0}
  <span class="has-text-grey-light">{formatPercentage(value, 2)}</span>
{:else}
  <span class="{color} whitespace-nowrap">
    <span class="icon is-small">
      <i class="fas {icon}"></i>
    </span>
    {formatPercentage(value, 2)}
  </span>
{/if}
