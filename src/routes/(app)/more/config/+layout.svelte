<script lang="ts">
  import { setContext } from "svelte";
  import { goto } from "$app/navigation";
  import { page } from "$app/stores";
  import { ajax, configUpdated } from "$lib/utils";
  import { onMount } from "svelte";
  import type { JSONSchema7 } from "json-schema";
  import _ from "lodash";
  import * as toast from "bulma-toast";
  import { refresh } from "../../../../store";
  import { sync } from "$lib/sync";
  import { CONFIG_GROUPS, ALL_SECTIONS, DEFAULT_SECTION_ID } from "$lib/config-sections";
  import type { Snippet } from "svelte";

  let { children }: { children: Snippet } = $props();

  let lastConfig: typeof globalThis.USER_CONFIG = $state(null);
  let config: typeof globalThis.USER_CONFIG = $state(null);
  let schema: JSONSchema7 = $state(null);
  let isLoading = $state(false);
  let isTogglingProviderDebug = $state(false);
  let error: string = $state(null);
  let accounts: string[] = $state([]);
  let sidebarOpen = $state(false);

  onMount(async () => {
    ({ config, schema, accounts } = await ajax("/api/config"));
    lastConfig = _.cloneDeep(config);
  });

  const hasChanges = $derived(!_.isEqual(config, lastConfig));

  const currentSectionId = $derived($page.params.section ?? DEFAULT_SECTION_ID);
  const currentSection = $derived(ALL_SECTIONS.find((s) => s.id === currentSectionId));

  // Share reactive config state with child section pages via context
  setContext("paisa-config", {
    get config() {
      return config;
    },
    set config(v: any) {
      config = v;
    },
    get schema() {
      return schema;
    },
    get accounts() {
      return accounts;
    },
    get error() {
      return error;
    },
    set error(v: string | null) {
      error = v;
    },
    get isTogglingProviderDebug() {
      return isTogglingProviderDebug;
    },
    applyProviderHTTPDebug
  });

  async function save() {
    isLoading = true;
    try {
      let success = false;
      ({ success, error } = await ajax("/api/config", {
        method: "POST",
        body: JSON.stringify(config),
        background: true
      }));
      if (success) {
        lastConfig = _.cloneDeep(config);
        globalThis.USER_CONFIG = _.cloneDeep(config);
        configUpdated();
        refresh();
        toast.toast({ message: "Saved config", type: "is-success" });
        await sync({ journal: true });
      }
    } finally {
      isLoading = false;
    }
  }

  async function discard() {
    config = _.cloneDeep(lastConfig);
  }

  async function resetToDefault() {
    if (
      confirm(
        "Are you sure you want to reset the config to defaults? This action is not reversible."
      )
    ) {
      const minimal = { journal_path: lastConfig.journal_path, db_path: lastConfig.db_path } as any;
      isLoading = true;
      try {
        let success = false;
        ({ success, error } = await ajax("/api/config", {
          method: "POST",
          body: JSON.stringify(minimal),
          background: true
        }));
        if (success) {
          lastConfig = _.cloneDeep(minimal);
          config = _.cloneDeep(minimal);
          globalThis.USER_CONFIG = _.cloneDeep(minimal);
          configUpdated();
          refresh();
          toast.toast({ message: "Config reset to defaults", type: "is-warning" });
        }
      } finally {
        isLoading = false;
      }
    }
  }

  async function applyProviderHTTPDebug(enabled: boolean) {
    isTogglingProviderDebug = true;
    try {
      const response: {
        success?: boolean;
        enabled?: boolean;
        error?: { code: string; message: string };
      } = await ajax("/api/config/provider-debug-http", {
        method: "POST",
        body: JSON.stringify({ enabled }),
        background: true
      });
      if (response.success) {
        config.provider_debug_http = enabled;
        lastConfig.provider_debug_http = enabled;
        globalThis.USER_CONFIG = _.cloneDeep(config);
        configUpdated();
        toast.toast({
          message: `Provider HTTP debug logging ${enabled ? "enabled" : "disabled"}`,
          type: "is-success"
        });
        error = null;
      } else if (response.error?.message) {
        error = response.error.message;
      }
    } finally {
      isTogglingProviderDebug = false;
    }
  }

  function navigateSection(sectionId: string) {
    sidebarOpen = false;
    goto(`/more/config/${sectionId}`);
  }
</script>

<div class="config-layout-wrapper">
  <!-- Mobile overlay backdrop -->
  {#if sidebarOpen}
    <button
      class="config-sidebar-backdrop"
      aria-label="Close sidebar"
      onclick={() => (sidebarOpen = false)}
    ></button>
  {/if}

  <!-- Left sidebar -->
  <aside class="config-sidebar" class:is-open={sidebarOpen} aria-label="Configuration sections">
    <div class="config-sidebar-inner">
      {#each CONFIG_GROUPS as group}
        <div class="config-nav-group-label">{group.label}</div>
        {#each group.sections as section}
          <button
            class="config-nav-item"
            class:is-active={currentSectionId === section.id}
            onclick={() => navigateSection(section.id)}
            aria-current={currentSectionId === section.id ? "page" : undefined}
          >
            <span class="icon is-small">
              <i class="fas {section.icon}"></i>
            </span>
            <span>{section.label}</span>
          </button>
        {/each}
      {/each}
    </div>
  </aside>

  <!-- Main content column -->
  <div class="config-main">
    <!-- Sticky action bar -->
    <div class="config-sticky-bar">
      <div class="config-sticky-bar-left">
        <button
          class="button is-ghost is-small config-menu-toggle is-hidden-desktop"
          onclick={() => (sidebarOpen = !sidebarOpen)}
          aria-label="Toggle section menu"
          aria-expanded={sidebarOpen}
        >
          <span class="icon is-small"><i class="fas fa-bars"></i></span>
          <span class="ml-1 is-size-7 has-text-grey">{currentSection?.label ?? "Config"}</span>
        </button>
      </div>
      <div class="config-sticky-bar-right">
        {#if error}
          <span class="tag is-danger is-light mr-2">Error — see below</span>
        {/if}
        {#if hasChanges}
          <span class="config-unsaved-dot" title="Unsaved changes">●</span>
        {/if}
        <button
          onclick={save}
          class="button is-success is-small {isLoading && 'is-loading'}"
          disabled={!hasChanges}
        >
          Save
        </button>
        <button onclick={discard} class="button is-light is-small" disabled={!hasChanges}>
          Discard
        </button>
        <div class="dropdown is-right is-hoverable">
          <div class="dropdown-trigger">
            <button class="button is-light is-small" aria-haspopup="true" aria-label="More options">
              <span class="icon is-small"><i class="fas fa-ellipsis"></i></span>
            </button>
          </div>
          <div class="dropdown-menu" role="menu">
            <div class="dropdown-content">
              <button class="dropdown-item has-text-danger" onclick={resetToDefault}>
                <span class="icon is-small"><i class="fas fa-rotate-left"></i></span>
                Reset to Defaults
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Info banner -->
    {#if schema}
      <div class="config-content">
        <article class="message is-info is-small config-info-banner">
          <div class="message-body py-2 px-3">
            Prices are <b>not</b> automatically updated after a config change. Use the menu at the
            top right to update prices. If the journal failed to sync, fix the issues and sync
            again.
          </div>
        </article>

        {#if error}
          <article class="message is-danger is-small">
            <div class="message-body" style="overflow: auto; white-space: pre;">{error}</div>
          </article>
        {/if}

        {@render children()}
      </div>
    {/if}
  </div>
</div>

<style lang="scss">
  .config-layout-wrapper {
    display: flex;
    min-height: calc(100vh - 120px);
    position: relative;
  }

  /* ── Sidebar ──────────────────────────────────────────── */
  .config-sidebar {
    width: 220px;
    flex-shrink: 0;
    border-right: 1px solid var(--bulma-border, #dbdbdb);
    background: var(--bulma-scheme-main, #fff);

    @media screen and (max-width: 1023px) {
      position: fixed;
      top: 0;
      left: 0;
      bottom: 0;
      z-index: 40;
      transform: translateX(calc(-100% - 1px));
      transition: transform 220ms ease;
      box-shadow: none;

      &.is-open {
        transform: translateX(0);
        box-shadow: 4px 0 20px rgba(0, 0, 0, 0.12);
      }
    }
  }

  .config-sidebar-inner {
    position: sticky;
    top: 0;
    padding: 1rem 0 2rem;
    max-height: 100vh;
    overflow-y: auto;
  }

  .config-sidebar-backdrop {
    display: none;
    @media screen and (max-width: 1023px) {
      display: block;
      position: fixed;
      inset: 0;
      background: rgba(10, 10, 10, 0.45);
      z-index: 39;
      border: none;
      cursor: pointer;
    }
  }

  /* ── Main content ─────────────────────────────────────── */
  .config-main {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
  }

  .config-content {
    padding: 1rem 1.5rem 2rem;
  }

  .config-info-banner {
    margin-bottom: 1rem;
  }

  /* ── Sticky action bar ────────────────────────────────── */
  .config-sticky-bar {
    position: sticky;
    top: 0;
    z-index: 10;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
    padding: 0.45rem 1rem;
    background: rgba(255, 255, 255, 0.92);
    backdrop-filter: blur(6px);
    border-bottom: 1px solid var(--bulma-border, #dbdbdb);
  }

  .config-sticky-bar-left {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  .config-sticky-bar-right {
    display: flex;
    align-items: center;
    gap: 0.4rem;
  }

  .config-unsaved-dot {
    color: #48c78e;
    font-size: 0.85rem;
    line-height: 1;
    margin-right: 0.1rem;
    animation: pulse 1.5s ease-in-out infinite;
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.4;
    }
  }

  .config-menu-toggle {
    display: flex;
    align-items: center;
  }
</style>
