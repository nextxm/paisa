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

<div class="box group-box mb-4 p-0">
  <button
    type="button"
    class="group-header-btn p-3 is-flex is-justify-content-space-between is-align-items-center w-100"
    onclick={() => (isOpen = !isOpen)}
  >
    <div class="is-flex is-align-items-center gap-2">
      <i class="fas {isOpen ? 'fa-chevron-down' : 'fa-chevron-right'} has-text-grey"></i>
      <h3 class="title is-6 mb-0">{groupTitle}</h3>
      <span class="tag is-rounded is-light ml-1">{findings.length}</span>
    </div>

    <span class="is-size-7 has-text-grey">
      {isOpen ? "Collapse" : "Expand"}
    </span>
  </button>

  {#if isOpen}
    <div class="group-body p-3 border-top">
      {#each findings as finding (finding.id)}
        <FindingCard {finding} {onDismiss} {onUndismiss} />
      {/each}
    </div>
  {/if}
</div>

<style>
  .group-box {
    overflow: hidden;
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .group-header-btn {
    width: 100%;
    background: transparent;
    border: 0;
    cursor: pointer;
    text-align: left;
    color: inherit;
    transition: background 0.15s ease;
  }

  .group-header-btn:hover {
    background: rgba(255, 255, 255, 0.03);
  }

  .border-top {
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }
</style>
