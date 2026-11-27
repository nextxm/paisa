<script lang="ts">
  import { page } from "$app/stores";
  import CashFlowTabs from "$lib/components/CashFlowTabs.svelte";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);
  const isOperationalRoute = $derived(pathname.startsWith("/cash_flow/recurring"));
</script>

{#if !isOperationalRoute}
  <CashFlowTabs />
{/if}

{@render children()}
