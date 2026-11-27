<script lang="ts">
  import { page } from "$app/stores";
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);
  const isOperationalRoute = $derived(pathname.startsWith("/expense/budget"));

  const tabs: SectionTab[] = [
    {
      label: "Monthly",
      href: "/expense/monthly",
      icon: "fa-calendar-days",
      match: (path) => path.startsWith("/expense/monthly")
    },
    {
      label: "Yearly",
      href: "/expense/yearly",
      icon: "fa-calendar",
      match: (path) => path.startsWith("/expense/yearly")
    },
    {
      label: "Breakdown (Sankey)",
      href: "/expense/sankey",
      icon: "fa-diagram-project",
      match: (path) => path.startsWith("/expense/sankey")
    },
    {
      label: "Heatmap",
      href: "/expense/heatmap",
      icon: "fa-border-all",
      match: (path) => path.startsWith("/expense/heatmap")
    },
    {
      label: "MoM",
      href: "/expense/mom",
      icon: "fa-chart-column",
      match: (path) => path.startsWith("/expense/mom")
    },
    {
      label: "YoY",
      href: "/expense/yoy",
      icon: "fa-chart-bar",
      match: (path) => path.startsWith("/expense/yoy")
    }
  ];
</script>

{#if !isOperationalRoute}
  <SectionTabs {tabs} />
{/if}

{@render children()}
