<script lang="ts">
  import { afterNavigate, beforeNavigate } from "$app/navigation";
  import { followCursor, delegate, hideAll } from "tippy.js";
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
    {@render children()}
  </Spinner>
{/key}

<CommandPalette />
<ReloadPrompt />
