<script lang="ts">
  import type { Posting } from "$lib/utils";

  let { posting }: { posting: Posting } = $props();
</script>

{#if posting.status == "cleared"}
  <span class="icon is-small">
    <i class="fa-solid fa-check"></i>
  </span>
{:else if posting.status == "pending"}
  <span class="icon is-small">
    <i class="fa-solid fa-exclamation"></i>
  </span>
{/if}
