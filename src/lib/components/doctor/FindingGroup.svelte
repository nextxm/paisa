<script lang="ts">
  import type { DoctorFinding } from "$lib/doctor_v2";
  import FindingCard from "./FindingCard.svelte";

  let {
    groupTitle,
    ruleId,
    findings = [],
    initiallyOpen = true,
    onDismiss,
    onUndismiss
  }: {
    groupTitle: string;
    ruleId: string;
    findings: DoctorFinding[];
    initiallyOpen?: boolean;
    onDismiss: (finding: DoctorFinding, note: string) => Promise<void>;
    onUndismiss: (finding: DoctorFinding) => Promise<void>;
  } = $props();

  let isOpen = $state(true);

  $effect(() => {
    isOpen = initiallyOpen;
  });
</script>

<div class="group-container mb-4">
  <button
    type="button"
    class="group-header-row mb-2 is-flex is-justify-content-space-between is-align-items-center w-100"
    onclick={() => (isOpen = !isOpen)}
  >
    <div class="is-flex is-align-items-center gap-2">
      <span class="icon is-small text-muted">
        <i class="fas {isOpen ? 'fa-chevron-down' : 'fa-chevron-right'}"></i>
      </span>
      <h3 class="title is-6 mb-0 group-title-text">{groupTitle}</h3>
      <span class="tag is-rounded is-small is-light">{findings.length}</span>
    </div>
  </button>

  {#if isOpen}
    <div class="group-cards-list">
      {#each findings as finding (finding.id)}
        <FindingCard {finding} {onDismiss} {onUndismiss} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .group-container {
    width: 100%;
  }

  .group-header-row {
    background: transparent;
    border: 0;
    cursor: pointer;
    text-align: left;
    color: inherit;
    padding: 0.35rem 0.25rem;
    width: 100%;
  }

  .group-header-row:hover .group-title-text {
    color: #3273dc;
  }

  .group-title-text {
    transition: color 0.15s ease;
  }

  .group-cards-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .text-muted {
    color: #9ca3af;
  }
</style>
