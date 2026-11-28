<script lang="ts">
  import { page } from "$app/stores";
  import { appPath, stripBase } from "$lib/utils";

  export interface SectionTab {
    label: string;
    href: string;
    icon?: string;
    match: (path: string) => boolean;
  }

  let { tabs }: { tabs: SectionTab[] } = $props();

  const pathname = $derived(stripBase($page.url.pathname));
</script>

<div class="section-tabs-bar">
  <div class="container is-fluid">
    <nav class="section-tabs-nav" aria-label="Section navigation">
      {#each tabs as tab}
        {@const active = tab.match(pathname)}
        <a href={appPath(tab.href)} class="section-tab-link" class:is-active={active}>
          {#if tab.icon}
            <span class="icon is-small"><i class="fa-solid {tab.icon}"></i></span>
          {/if}
          <span>{tab.label}</span>
        </a>
      {/each}
    </nav>
  </div>
</div>

<style>
  .section-tabs-bar {
    border-bottom: 1px solid var(--color-border, rgba(128, 128, 128, 0.16));
    background: transparent;
  }
  .section-tabs-nav {
    display: flex;
    align-items: center;
    gap: 1.5rem;
    overflow-x: auto;
    scrollbar-width: none;
  }
  .section-tabs-nav::-webkit-scrollbar {
    display: none;
  }
  .section-tab-link {
    display: inline-flex;
    align-items: center;
    gap: 0.45rem;
    padding: 0.7rem 0.1rem;
    font-size: 0.85rem;
    font-weight: 500;
    color: inherit;
    opacity: 0.65;
    border-bottom: 2px solid transparent;
    white-space: nowrap;
    text-decoration: none;
    transition:
      opacity 0.15s ease,
      border-color 0.15s ease;
  }
  .section-tab-link:hover {
    opacity: 0.95;
  }
  .section-tab-link.is-active {
    opacity: 1;
    font-weight: 600;
    border-bottom-color: var(--primary, #485fc7);
  }
</style>
