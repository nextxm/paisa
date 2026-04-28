<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    toggle,
    content
  }: {
    toggle: Snippet<[{ active: boolean; onclick: (e: any) => void }]>;
    content: Snippet;
  } = $props();

  let active = $state(false);
  const onclick = function (_e: any) {
    active = !active;
  };
</script>

{@render toggle({ active, onclick })}
{#if active}
  {@render content()}
{/if}
