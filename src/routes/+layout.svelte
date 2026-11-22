<script lang="ts">
  import { onMount, type Snippet } from "svelte";
  import { initializeTheme } from "$lib/theme";
  import { theme } from "../store";

  let { children }: { children: Snippet } = $props();

  onMount(() => {
    return initializeTheme((value) => {
      theme.set(value);
    });
  });
</script>

{@render children()}
