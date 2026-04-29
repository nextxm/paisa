<script lang="ts">
  import type { Snippet } from "svelte";
  import _ from "lodash";

  let { item, children }: { item: any; children?: Snippet } = $props();
</script>

{#if item == null || item == false || (item != true && _.isEmpty(item))}
  <div class="has-text-centered p-6">
    {@render children?.()}
  </div>
{/if}
