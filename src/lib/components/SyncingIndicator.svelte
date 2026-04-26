<script lang="ts">
  import { fade } from "svelte/transition";
  import { isJobRunning } from "$lib/stores/jobs";
</script>

{#if $isJobRunning}
  <span
    class="syncing-indicator navbar-item"
    aria-live="polite"
    aria-label="Syncing records"
    title="Syncing records…"
    transition:fade={{ duration: 200 }}
  >
    <span class="icon is-small">
      <i class="fas fa-rotate fa-spin" aria-hidden="true" />
    </span>
    <span class="syncing-label is-hidden-mobile">Syncing…</span>
  </span>
{/if}

<style lang="scss">
  .syncing-indicator {
    display: inline-flex;
    align-items: center;
    gap: 0.3rem;
    color: var(--bulma-primary, #485fc7);
    font-size: 0.85rem;
    padding: 0 0.25rem;
    white-space: nowrap;
    pointer-events: none;
    user-select: none;
  }
</style>
