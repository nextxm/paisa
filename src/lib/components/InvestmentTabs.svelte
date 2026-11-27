<script lang="ts">
  import { page } from "$app/stores";

  const pathname = $derived($page.url.pathname);

  const tabs = [
    {
      label: "Holdings & Valuation",
      href: "/assets/investment",
      icon: "fa-briefcase",
      match: (path: string) => path.startsWith("/assets/investment")
    },
    {
      label: "Gain / XIRR",
      href: "/assets/gain",
      icon: "fa-chart-line",
      match: (path: string) => path.startsWith("/assets/gain")
    },
    {
      label: "Allocation",
      href: "/assets/allocation",
      icon: "fa-chart-pie",
      match: (path: string) => path.startsWith("/assets/allocation")
    },
    {
      label: "Underlying Portfolio",
      href: "/assets/portfolio",
      icon: "fa-layer-group",
      match: (path: string) => path.startsWith("/assets/portfolio")
    },
    {
      label: "Investment Income",
      href: "/income/investment",
      icon: "fa-coins",
      match: (path: string) => path.startsWith("/income/investment")
    }
  ];
</script>

<div class="investment-tabs-bar">
  <div class="container is-fluid">
    <div class="tabs is-boxed is-small mb-0">
      <ul>
        {#each tabs as tab}
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

<style>
  .investment-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .investment-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
