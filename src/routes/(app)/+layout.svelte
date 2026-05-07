<script lang="ts">
  import { afterNavigate, beforeNavigate } from "$app/navigation";
  import { followCursor, delegate, hideAll } from "tippy.js";
  import _ from "lodash";
  import type { Snippet } from "svelte";
  import Spinner from "$lib/components/Spinner.svelte";
  import Navbar from "$lib/components/Navbar.svelte";
  import ReloadPrompt from "$lib/components/ReloadPrompt.svelte";
  import ReconciliationModal from "$lib/components/ReconciliationModal.svelte";
  import { willClearTippy, willRefresh, reconciliationModalState } from "../../store";
  import { onMount } from "svelte";

  let { children, data }: { children: Snippet; data?: any } = $props();
  let isBurger: boolean = $state(null);

  onMount(() => {
    (window as any).openReconciliationModal = (account: string) => {
      reconciliationModalState.set({ account, open: true });
    };

    if (USER_CONFIG.enable_reconciliation) {
      const searchParams = new URLSearchParams(window.location.search);
      const reconcile = searchParams.get("reconcile");
      const account = searchParams.get("account") || (data as any)?.account || (data as any)?.name;
      if (reconcile === "1" && account) {
        (window as any).openReconciliationModal(account);
      }
    }
  });

  function clearTippy() {
    hideAll();
  }

  function setupTippy() {
    delegate("body", {
      target: "[data-tippy-content]",
      theme: "light",
      onShow: (instance) => {
        const content = instance.reference.getAttribute("data-tippy-content");
        if (!_.isEmpty(content)) {
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

<ReloadPrompt />
{#if USER_CONFIG.enable_reconciliation}
  <ReconciliationModal />
{/if}
