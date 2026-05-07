<script lang="ts">
  import {
    reconciliationLabel,
    reconciliationTagClass,
    reconciliationIcon
  } from "$lib/reconciliation";
  import type { AccountReconciliationStatus } from "$lib/utils";

  let { reconciliations = [] }: { reconciliations: AccountReconciliationStatus[] } = $props();

  const upToDate = $derived(reconciliations.filter((status) => !status.is_overdue));
  const overdue = $derived(reconciliations.filter((status) => status.is_overdue));
</script>

<p class="subtitle">
  <span class="secondary-link has-text-grey">Account Reconciliation</span>
</p>
<div class="content box">
  <p>
    <strong>{upToDate.length}</strong> accounts up-to-date, <strong>{overdue.length}</strong> overdue
  </p>
  {#if overdue.length === 0}
    <p class="has-text-grey">No overdue accounts.</p>
  {:else}
    <div class="is-flex is-flex-direction-column" style="gap: 8px;">
      {#each overdue as status}
        <div class="is-flex is-align-items-center is-justify-content-space-between">
          <a href="/accounts/{encodeURIComponent(status.account)}">{status.account}</a>
          <a
            href="/accounts/{encodeURIComponent(status.account)}?reconcile=1"
            class="tag is-light {reconciliationTagClass(status)} is-rounded"
            title={reconciliationLabel(status)}
            style="padding: 0 0.5em;"
          >
            <span class="custom-icon">{reconciliationIcon(status)}</span>
          </a>
        </div>
      {/each}
    </div>
  {/if}
</div>
