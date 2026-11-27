<script lang="ts">
  import { page } from "$app/stores";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);
  const isOperationalRoute = $derived(pathname.startsWith("/expense/budget"));

  const tabs = [
    {
      label: "Monthly",
      href: "/expense/monthly",
      icon: "fa-calendar-days",
      match: (path: string) => path.startsWith("/expense/monthly")
    },
    {
      label: "Yearly",
      href: "/expense/yearly",
      icon: "fa-calendar",
      match: (path: string) => path.startsWith("/expense/yearly")
    },
    {
      label: "Breakdown (Sankey)",
      href: "/expense/sankey",
      icon: "fa-diagram-project",
      match: (path: string) => path.startsWith("/expense/sankey")
    },
    {
      label: "Heatmap",
      href: "/expense/heatmap",
      icon: "fa-border-all",
      match: (path: string) => path.startsWith("/expense/heatmap")
    },
    {
      label: "MoM",
      href: "/expense/mom",
      icon: "fa-chart-column",
      match: (path: string) => path.startsWith("/expense/mom")
    },
    {
      label: "YoY",
      href: "/expense/yoy",
      icon: "fa-chart-bar",
      match: (path: string) => path.startsWith("/expense/yoy")
    }
  ];
</script>

{#if !isOperationalRoute}
  <div class="expense-tabs-bar">
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
{/if}

{@render children()}

<style>
  .expense-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .expense-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
