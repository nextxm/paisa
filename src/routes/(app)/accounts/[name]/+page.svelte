<script lang="ts">
  import { ajax, type AccountNote, type AccountReconciliationStatus } from "$lib/utils";
  import { onMount } from "svelte";
  import type { PageData } from "./$types";
  import * as toast from "bulma-toast";
  import Modal from "$lib/components/Modal.svelte";
  import {
    reconciliationLabel,
    reconciliationTagClass,
    reconciliationIcon,
    reconciliationTextClass
  } from "$lib/reconciliation";

  let { data }: { data: PageData } = $props();

  let accountNote: AccountNote | null = $state(null);
  let noteText: string = $state("");
  let saving = $state(false);
  let loaded = $state(false);
  let reconciliationStatus: AccountReconciliationStatus | null = $state(null);
  let reconciliationModalOpen = $state(false);
  let reconciliationFrequencyDays = $state(30);

  onMount(async () => {
    const [noteResult, reconciliationResult] = await Promise.all([
      ajax("/api/account_notes/:account", null, { account: data.account }),
      ajax("/api/accounts/:account/reconciliation", null, { account: data.account })
    ]);
    accountNote = noteResult.account_note ?? null;
    noteText = accountNote?.note ?? "";
    reconciliationStatus = reconciliationResult;
    reconciliationFrequencyDays = reconciliationResult.frequency_days;
    const searchParams = new URLSearchParams(window.location.search);
    reconciliationModalOpen = searchParams.get("reconcile") === "1";
    loaded = true;
  });

  async function save() {
    if (saving) return;
    saving = true;
    try {
      const result = await ajax("/api/account_notes/upsert", {
        method: "POST",
        body: JSON.stringify({ account: data.account, note: noteText })
      });
      if (result.saved) {
        accountNote = result.account_note;
        toast.toast({ message: "Note saved.", type: "is-success", duration: 3000 });
      }
    } catch (err) {
      console.error("Failed to save account note:", err);
      toast.toast({ message: "Failed to save note.", type: "is-danger", duration: 3000 });
    } finally {
      saving = false;
    }
  }

  async function deleteNote() {
    if (!accountNote) return;
    if (!confirm("Delete note for this account?")) return;
    saving = true;
    try {
      await ajax("/api/account_notes/delete", {
        method: "POST",
        body: JSON.stringify({ account: data.account })
      });
      accountNote = null;
      noteText = "";
      toast.toast({ message: "Note deleted.", type: "is-success", duration: 3000 });
    } catch (err) {
      console.error("Failed to delete account note:", err);
      toast.toast({ message: "Failed to delete note.", type: "is-danger", duration: 3000 });
    } finally {
      saving = false;
    }
  }

  async function markReconciledNow() {
    if (saving) return;
    saving = true;
    try {
      reconciliationStatus = await ajax(
        "/api/accounts/:account/reconciliation",
        {
          method: "PATCH",
          body: JSON.stringify({
            mark_reconciled_now: true,
            frequency_days: reconciliationFrequencyDays
          })
        },
        { account: data.account }
      );
      toast.toast({ message: "Account marked as reconciled.", type: "is-success", duration: 3000 });
      reconciliationModalOpen = false;
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
</script>

{#if loaded}
  <section class="section tab-account-overview">
    <div class="container is-fluid">
      <div class="columns">
        <div class="column is-12">
          <nav class="level">
            <div class="level-left">
              <div class="level-item">
                <p class="title is-5">{data.account}</p>
              </div>
              {#if reconciliationStatus}
                <div class="level-item">
                  <button
                    type="button"
                    class="button is-ghost p-0 h-auto {reconciliationTextClass(
                      reconciliationStatus
                    )}"
                    onclick={() => (reconciliationModalOpen = true)}
                    title={reconciliationLabel(reconciliationStatus)}
                  >
                    <span class="custom-icon">{reconciliationIcon(reconciliationStatus)}</span>
                  </button>
                </div>
              {/if}
            </div>
            <div class="level-right">
              <div class="level-item">
                <a
                  href="/accounts/{encodeURIComponent(data.account)}/transactions"
                  class="button is-small is-light"
                >
                  <span class="icon is-small"><i class="fas fa-list"></i></span>
                  <span>Transactions</span>
                </a>
              </div>
            </div>
          </nav>
        </div>
      </div>

      <div class="columns">
        <div class="column is-12 is-6-widescreen">
          <div class="box">
            <p class="title is-6 mb-3">Account Note</p>
            <p class="subtitle is-7 has-text-grey mb-3">
              Add context about this account (e.g. "Emergency fund", "Company 401k"). Notes are
              stored locally and not written to your ledger file.
            </p>
            <div class="field">
              <div class="control">
                <textarea
                  class="textarea"
                  rows="5"
                  placeholder="Enter a note for this account…"
                  bind:value={noteText}
                  disabled={saving}
                ></textarea>
              </div>
            </div>
            <div class="field is-grouped">
              <div class="control">
                <button class="button is-primary is-small" onclick={save} disabled={saving}>
                  <span class="icon is-small"><i class="fas fa-save"></i></span>
                  <span>{saving ? "Saving…" : "Save Note"}</span>
                </button>
              </div>
              {#if accountNote}
                <div class="control">
                  <button
                    class="button is-danger is-light is-small"
                    onclick={deleteNote}
                    disabled={saving}
                  >
                    <span class="icon is-small"><i class="fas fa-trash"></i></span>
                    <span>Delete Note</span>
                  </button>
                </div>
              {/if}
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
{/if}

<Modal bind:active={reconciliationModalOpen}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">Reconciliation — {data.account}</p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
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
  {#snippet foot(close)}
    <button class="du-btn du-btn-success du-btn-sm" disabled={saving} onclick={markReconciledNow}>
      Mark Reconciled
    </button>
    <button class="du-btn du-btn-sm" onclick={() => close()}>Close</button>
  {/snippet}
</Modal>
