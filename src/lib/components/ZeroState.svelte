<script lang="ts">
  import _ from "lodash";

  let { item }: { item: any } = $props();
</script>

{#if item == null || item == false || (item != true && _.isEmpty(item))}
  <div class="has-text-centered p-6">
    <slot />
  </div>
{/if}
