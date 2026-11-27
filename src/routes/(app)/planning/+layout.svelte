<script lang="ts">
  import { page } from "$app/stores";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  const pathname = $derived($page.url.pathname);
  const isTaxRoute = $derived(pathname.startsWith("/planning/tax"));

  const tabs = [
    {
      label: "Projection",
      href: "/planning/projection",
      icon: "fa-chart-line",
      match: (path: string) => path.startsWith("/planning/projection") || path === "/planning/life"
    },
    {
      label: "Goals",
      href: "/planning/goals",
      icon: "fa-bullseye",
      match: (path: string) =>
        path.startsWith("/planning/goals") || path.startsWith("/planning/life/goals")
    },
    {
      label: "What-If Scenarios",
      href: "/planning/life/whatif",
      icon: "fa-code-branch",
      match: (path: string) => path.startsWith("/planning/life/whatif")
    },
    {
      label: "Drawdown Strategy",
      href: "/planning/life/drawdown",
      icon: "fa-layer-group",
      match: (path: string) => path.startsWith("/planning/life/drawdown")
    }
  ];
</script>

{#if !isTaxRoute}
  <div class="planning-tabs-bar">
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
  .planning-tabs-bar {
    padding-top: 0.75rem;
    border-bottom: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .planning-tabs-bar :global(.tabs ul) {
    border-bottom: 0;
  }
</style>
