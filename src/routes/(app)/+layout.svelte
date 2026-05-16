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
  import { ajax } from "$lib/utils";
  import { onMount } from "svelte";

  let { children }: { children: Snippet; data?: any } = $props();
  let isBurger: boolean = $state(null);

  /** How often (ms) to poll the journal dirty state in the background. */
  const JOURNAL_POLL_INTERVAL_MS = 30_000;

  /**
   * Fetches GET /api/journal/status and updates USER_CONFIG.is_journal_dirty
   * when the value has changed.  On change the willRefresh counter is bumped
   * so derived UI state (e.g. the sync icon colour) re-evaluates immediately
   * without triggering a full page reload.
   */
  async function checkJournalDirty() {
    try {
      const { is_dirty } = await ajax("/api/journal/status", { background: true });
      if (is_dirty !== USER_CONFIG?.is_journal_dirty) {
        USER_CONFIG.is_journal_dirty = is_dirty;
        willRefresh.update((n) => n + 1);
      }
    } catch {
      // Network errors are silently ignored; the next poll will retry.
    }
  }

  onMount(() => {
    // Poll periodically so the journal-dirty indicator stays in sync even
    // when the user edits the ledger file in an external editor.
    const intervalId = setInterval(checkJournalDirty, JOURNAL_POLL_INTERVAL_MS);

    // Also check immediately when the user returns to this tab, since they
    // may have been editing the file while the app was in the background.
    function handleVisibilityChange() {
      if (!document.hidden) {
        checkJournalDirty();
      }
    }
    document.addEventListener("visibilitychange", handleVisibilityChange);

    return () => {
      clearInterval(intervalId);
      document.removeEventListener("visibilitychange", handleVisibilityChange);
    };
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

<CommandPalette />
<ReloadPrompt />
