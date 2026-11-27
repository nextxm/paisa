<script lang="ts">
  import { page } from "$app/stores";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);

  const tabs = [
    {
      label: "Balances",
      href: "/liabilities/balance",
      icon: "fa-scale-balanced",
      match: (path: string) => path.startsWith("/liabilities/balance")
    },
    {
      label: "Credit Cards",
      href: "/liabilities/credit_cards",
      icon: "fa-credit-card",
      match: (path: string) => path.startsWith("/liabilities/credit_cards")
    },
    {
      label: "Repayment",
      href: "/liabilities/repayment",
      icon: "fa-calendar-check",
      match: (path: string) => path.startsWith("/liabilities/repayment")
    },
    {
      label: "Interest",
      href: "/liabilities/interest",
      icon: "fa-percent",
      match: (path: string) => path.startsWith("/liabilities/interest")
    }
  ];
</script>

<div class="liabilities-tabs-bar">
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

{@render children()}

<style>
  .liabilities-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .liabilities-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
