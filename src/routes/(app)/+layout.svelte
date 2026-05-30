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
  import { willClearTippy, willRefresh } from "../../store";
  import { onMount } from "svelte";

  let { children }: { children: Snippet; data?: any } = $props();
  let isBurger: boolean | null = $state(null);

  onMount(() => {});

  function clearTippy() {
    hideAll();
  }

  function setupTippy() {
    delegate("body", {
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

  willClearTippy.subscribe(clearTippy);
  beforeNavigate(clearTippy);
  willRefresh.subscribe(() => {
    clearTippy();
    setupTippy();
  });

  afterNavigate(() => {
    isBurger = null;
    setupTippy();
  });
</script>

{#key $willRefresh}
  <Navbar bind:isBurger />

  <Spinner>
    {#if $navigating}
      <div class="route-loading-overlay">Loading…</div>
    {/if}
    {@render children()}
  </Spinner>
{/key}

<CommandPalette />
<ReloadPrompt />

<style lang="scss">
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
