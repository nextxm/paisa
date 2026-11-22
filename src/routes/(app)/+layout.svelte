<script lang="ts">
  import { afterNavigate, beforeNavigate } from "$app/navigation";
  import { followCursor, delegate, hideAll } from "tippy.js";
  import { navigating } from "$app/stores";
  import _ from "lodash";
  import type { Snippet } from "svelte";
  import Spinner from "$lib/components/Spinner.svelte";
  import Navbar from "$lib/components/Navbar.svelte";
  import ReloadPrompt from "$lib/components/ReloadPrompt.svelte";
  import CommandPalette from "$lib/components/CommandPalette.svelte";
  import QuickAddModal from "$lib/components/QuickAddModal.svelte";
  import { quickAddOpen, willClearTippy, willRefresh } from "../../store";
  import { isMobile } from "$lib/utils";
  import { onMount } from "svelte";

  let { children }: { children: Snippet; data?: any } = $props();
  let isBurger: boolean | null = $state(null);
  const mobile = isMobile();

  function handleGlobalKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.shiftKey && e.key.toLowerCase() === "a") {
      e.preventDefault();
      quickAddOpen.set(true);
    }
  }

  function clearTippy() {
    hideAll();
  }

  function setupTippy() {
    return delegate("body", {
      target: "[data-tippy-content]",
      theme: "light",
      onShow: (instance) => {
        const content = instance.reference.getAttribute("data-tippy-content");
        if (content && !_.isEmpty(content)) {
          instance.setContent(content);
        } else {
          return false;
        }
      },
      maxWidth: "none",
      delay: 0,
      allowHTML: true,
      followCursor: true,
      popperOptions: {
        modifiers: [
          {
            name: "flip",
            options: {
              fallbackPlacements: ["auto"]
            }
          }
        ]
      },
      plugins: [followCursor]
    });
  }

  onMount(() => {
    const tippyDelegate = setupTippy();
    const unsubClear = willClearTippy.subscribe(clearTippy);
    const unsubRefresh = willRefresh.subscribe(clearTippy);

    return () => {
      unsubClear();
      unsubRefresh();
      const delegates = Array.isArray(tippyDelegate) ? tippyDelegate : [tippyDelegate];
      delegates.forEach((instance) => instance.destroy());
    };
  });

  beforeNavigate(clearTippy);

  afterNavigate(() => {
    isBurger = null;
  });
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<Navbar bind:isBurger />

<Spinner>
  {#if $navigating}
    <div class="route-loading-overlay">Loading…</div>
  {/if}
  {#key $willRefresh}
    {@render children()}
  {/key}
</Spinner>

<CommandPalette />
<QuickAddModal bind:open={$quickAddOpen} />
<ReloadPrompt />

{#if mobile}
  <button
    type="button"
    class="button is-primary is-rounded mobile-quick-add-fab"
    aria-label="Quick Add Transaction (Ctrl+Shift+A)"
    onclick={() => quickAddOpen.set(true)}
  >
    <span class="icon">
      <i class="fa-solid fa-plus"></i>
    </span>
  </button>
{/if}

<style lang="scss">
  .mobile-quick-add-fab {
    position: fixed;
    right: 1.25rem;
    bottom: 1.25rem;
    z-index: 45;
    width: 3.25rem;
    height: 3.25rem;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.25);
  }

  .route-loading-overlay {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 100;
    background: rgba(0, 0, 0, 0.7);
    color: #fff;
    border-radius: 999px;
    padding: 0.25rem 0.75rem;
    font-size: 0.75rem;
    font-weight: 600;
    letter-spacing: 0.02em;
  }
</style>
