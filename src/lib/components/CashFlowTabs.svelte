<script lang="ts">
  import { page } from "$app/stores";

  const pathname = $derived($page.url.pathname);

  const tabs = [
    {
      label: "Monthly",
      href: "/cash_flow/monthly",
      icon: "fa-calendar-days",
      match: (path: string) => path.startsWith("/cash_flow/monthly")
    },
    {
      label: "Yearly",
      href: "/cash_flow/yearly",
      icon: "fa-calendar",
      match: (path: string) => path.startsWith("/cash_flow/yearly")
    },
    {
      label: "Income Statement",
      href: "/cash_flow/income_statement",
      icon: "fa-file-invoice-dollar",
      match: (path: string) => path.startsWith("/cash_flow/income_statement")
    },
    {
      label: "Money Flow (Sankey)",
      href: "/cash_flow/sankey",
      icon: "fa-diagram-project",
      match: (path: string) => path.startsWith("/cash_flow/sankey")
    },
    {
      label: "Income Timeline",
      href: "/income",
      icon: "fa-arrow-trend-up",
      match: (path: string) => path === "/income"
    }
  ];
</script>

<div class="cashflow-tabs-bar">
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
  .cashflow-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .cashflow-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
