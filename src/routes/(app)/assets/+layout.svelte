<script lang="ts">
  import { page } from "$app/stores";
  import InvestmentTabs from "$lib/components/InvestmentTabs.svelte";
  import SectionTabs, { type SectionTab } from "$lib/components/SectionTabs.svelte";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);
  const isOverviewRoute = $derived(
    pathname.startsWith("/assets/networth") || pathname.startsWith("/assets/balance")
  );
  const isInvestmentRoute = $derived(
    pathname.startsWith("/assets/investment") ||
      pathname.startsWith("/assets/gain") ||
      pathname.startsWith("/assets/allocation") ||
      pathname.startsWith("/assets/portfolio")
  );

  const overviewTabs: SectionTab[] = [
    {
      label: "Net Worth History",
      href: "/assets/networth",
      icon: "fa-chart-area",
      match: (path) => path.startsWith("/assets/networth")
    },
    {
      label: "Asset Balances",
      href: "/assets/balance",
      icon: "fa-scale-balanced",
      match: (path) => path.startsWith("/assets/balance")
    }
  ];
</script>

{#if isOverviewRoute}
  <SectionTabs tabs={overviewTabs} />
{:else if isInvestmentRoute}
  <InvestmentTabs />
{/if}

{@render children()}
