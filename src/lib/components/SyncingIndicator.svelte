<script lang="ts">
  import { fade } from "svelte/transition";
  import { isJobRunning, runningJob } from "$lib/stores/jobs";
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
      <i class="fas fa-rotate fa-spin" aria-hidden="true"></i>
    </span>
    {#if $runningJob?.total_items && $runningJob.total_items > 0}
      <span class="syncing-label is-hidden-mobile">
        {$runningJob.items_completed ?? 0}&nbsp;/&nbsp;{$runningJob.total_items}
      </span>
    {:else}
      <span class="syncing-label is-hidden-mobile">Syncing…</span>
    {/if}
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
