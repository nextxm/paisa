<script lang="ts">
  import { page } from "$app/stores";
  import InvestmentTabs from "$lib/components/InvestmentTabs.svelte";
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

  const overviewTabs = [
    {
      label: "Net Worth History",
      href: "/assets/networth",
      icon: "fa-chart-area",
      match: (path: string) => path.startsWith("/assets/networth")
    },
    {
      label: "Asset Balances",
      href: "/assets/balance",
      icon: "fa-scale-balanced",
      match: (path: string) => path.startsWith("/assets/balance")
    }
  ];
</script>

{#if isOverviewRoute}
  <div class="assets-tabs-bar">
    <div class="container is-fluid">
      <div class="tabs is-boxed is-small mb-0">
        <ul>
          {#each overviewTabs as tab}
            <li class={tab.match(pathname) ? "is-active" : ""}>
              <a href={tab.href}>
                <span class="icon is-small"><i class="fa-solid {tab.icon}"></i></span>
                <span>{tab.label}</span>
              </a>
            </li>
          {/each}
        </ul>
      </div>
    </div>
  </div>
{:else if isInvestmentRoute}
  <InvestmentTabs />
{/if}

{@render children()}

<style>
  .assets-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .assets-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
