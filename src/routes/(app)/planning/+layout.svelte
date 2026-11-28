<script lang="ts">
  import { page } from "$app/stores";
  import { stripBase } from "$lib/utils";
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived(stripBase($page.url.pathname));
  const isTaxRoute = $derived(pathname.startsWith("/planning/tax"));

  const tabs: SectionTab[] = [
    {
      label: "Projection",
      href: "/planning/projection",
      icon: "fa-chart-line",
      match: (path) => path.startsWith("/planning/projection") || path === "/planning/life"
    },
    {
      label: "Goals",
      href: "/planning/goals",
      icon: "fa-bullseye",
      match: (path) => path.startsWith("/planning/goals") || path.startsWith("/planning/life/goals")
    },
    {
      label: "What-If Scenarios",
      href: "/planning/life/whatif",
      icon: "fa-code-branch",
      match: (path) => path.startsWith("/planning/life/whatif")
    },
    {
      label: "Drawdown Strategy",
      href: "/planning/life/drawdown",
      icon: "fa-layer-group",
      match: (path) => path.startsWith("/planning/life/drawdown")
    }
  ];
</script>

{#if !isTaxRoute}
  <SectionTabs {tabs} />
{/if}

{@render children()}
