<script lang="ts">
  import {
    reconciliationLabel,
    reconciliationTagClass,
    reconciliationIcon,
    reconciliationTextClass
  } from "$lib/reconciliation";
  import type { AccountReconciliationStatus } from "$lib/utils";
  import { reconciliationModalState } from "../../store";

  let props: { reconciliations: AccountReconciliationStatus[] } = $props();

  const upToDate = $derived((props.reconciliations || []).filter((status) => !status.is_overdue));
  const overdue = $derived((props.reconciliations || []).filter((status) => status.is_overdue));
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
          <button
            type="button"
            class="button is-ghost p-0 h-auto {reconciliationTextClass(status)}"
            onclick={() => reconciliationModalState.set({ account: status.account, open: true })}
            title={reconciliationLabel(status)}
          >
            <span class="custom-icon">{reconciliationIcon(status)}</span>
          </button>
        </div>
      {/each}
    </div>
  {/if}
</div>
