<script lang="ts">
  import { untrack } from "svelte";
  import { reconciliationModalState, updateReconciliationStatus } from "../../store";
  import { ajax, type AccountReconciliationStatus } from "$lib/utils";
  import { reconciliationLabel } from "$lib/reconciliation";
  import Modal from "$lib/components/Modal.svelte";
  import * as toast from "bulma-toast";

  let account = $derived($reconciliationModalState.account);
  let open = $state(false);
  let reconciliationStatus: AccountReconciliationStatus | null = $state(null);
  let reconciliationFrequencyDays = $state(30);
  let saving = $state(false);

  $effect(() => {
    open = $reconciliationModalState.open;
  });

  $effect(() => {
    if (open && account) {
      load();
    }
  });

  async function load() {
    try {
      const res = await ajax(
        "/api/accounts/:account/reconciliation",
        { background: true },
        { account }
      );
      reconciliationStatus = res;
      reconciliationFrequencyDays = res.frequency_days;
    } catch (err) {
      console.error("Failed to load reconciliation status:", err);
    }
  }

  async function markReconciledNow() {
    if (saving || !account) return;
    saving = true;
    try {
      const res = await ajax(
        "/api/accounts/:account/reconciliation",
        {
          method: "PATCH",
          body: JSON.stringify({
            mark_reconciled_now: true,
            frequency_days: reconciliationFrequencyDays
          }),
          background: true
        },
        { account }
      );
      toast.toast({ message: "Account marked as reconciled.", type: "is-success", duration: 3000 });
      close();
      updateReconciliationStatus(account, res);
    } catch (err) {
      console.error("Failed to update reconciliation:", err);
      toast.toast({
        message: "Failed to update reconciliation.",
        type: "is-danger",
        duration: 3000
      });
    } finally {
      saving = false;
    }
  }

  function close() {
    reconciliationModalState.set({ account: null, open: false });
    reconciliationStatus = null;
  }

  $effect(() => {
    if (!open && untrack(() => $reconciliationModalState.open)) {
      close();
    }
  });
</script>

<Modal bind:active={open}>
  {#snippet head()}
    <p class="text-base font-semibold flex-1">Reconciliation — {account}</p>
    <button class="du-btn du-btn-sm du-btn-circle du-btn-ghost" aria-label="close" onclick={close}>
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}
  {#snippet body()}
    {#if reconciliationStatus}
      <p class="mb-2">{reconciliationLabel(reconciliationStatus)}</p>
      <p class="mb-4">Frequency: every {reconciliationStatus.frequency_days} days</p>
    {/if}
    <div class="field">
      <label class="label" for="reconciliation-frequency">Frequency (days)</label>
      <div class="control">
        <input
          id="reconciliation-frequency"
          class="input"
          type="number"
          min="1"
          bind:value={reconciliationFrequencyDays}
          disabled={saving}
        />
      </div>
    </div>
  {/snippet}
  {#snippet foot()}
    <button class="du-btn du-btn-success du-btn-sm" disabled={saving} onclick={markReconciledNow}>
      Mark Reconciled
    </button>
    <button class="du-btn du-btn-sm" onclick={close}>Close</button>
  {/snippet}
</Modal>
