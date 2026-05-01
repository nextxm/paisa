<script lang="ts">
  import { goto } from "$app/navigation";
  import { onMount } from "svelte";
  import { DEFAULT_SECTION_ID } from "$lib/config-sections";

  onMount(() => {
    goto(`/more/config/${DEFAULT_SECTION_ID}`, { replaceState: true });
  });
</script>
