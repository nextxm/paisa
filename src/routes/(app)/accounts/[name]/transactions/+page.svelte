<script lang="ts">
  import {
    ajax,
    isMobile,
    type AccountNote,
    type Transaction as T,
    type AccountReconciliationStatus
  } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";
  import Transaction from "$lib/components/Transaction.svelte";
  import TransactionHeader from "$lib/components/TransactionHeader.svelte";
  import type { PageData } from "./$types";
  import {
    reconciliationLabel,
    reconciliationIcon,
    reconciliationTextClass
  } from "$lib/reconciliation";
  import {
    reconciliationModalState,
    reconciliationStatuses,
    updateReconciliationStatus
  } from "../../../../../store";

  let { data }: { data: PageData } = $props();

  let transactions: T[] | null = $state(null);
  let accountNote: AccountNote | null = $state(null);

  const mobile = isMobile();

  const itemSize = (i: number) => {
    const t = transactions[i];
    const count = t.postings.length;
    return 8 + count * 22 + (mobile ? 25 : 0);
  };

  onMount(async () => {
    const encoded = encodeURIComponent(data.account);
    transactions = (await ajax(`/api/transaction?account=${encoded}`)).transactions;
    const [noteResult, reconciliationResult] = await Promise.all([
      ajax("/api/account_notes/:account", null, { account: data.account }),
      USER_CONFIG.enable_reconciliation
        ? ajax("/api/accounts/:account/reconciliation", null, { account: data.account })
        : Promise.resolve(null)
    ]);
    accountNote = noteResult.account_note;
    updateReconciliationStatus(data.account, reconciliationResult);
    const searchParams = new URLSearchParams(window.location.search);
    if (USER_CONFIG.enable_reconciliation && searchParams.get("reconcile") === "1") {
      reconciliationModalState.set({ account: data.account, open: true });
    }
  });
</script>

{#if transactions}
  <section class="section tab-journal">
    <div class="container is-fluid">
      <div class="columns">
        <div class="column is-12">
          <nav class="level">
            <div class="level-left">
              <div class="level-item">
                <p class="title is-5 has-text-info">{data.account}</p>
              </div>
              {#if accountNote?.note}
                <div class="level-item">
                  <span class="tag is-info is-light">
                    <span class="icon is-small mr-1"><i class="fas fa-sticky-note"></i></span>
                    {accountNote.note}
                  </span>
                </div>
              {/if}
              {#if USER_CONFIG.enable_reconciliation && $reconciliationStatuses[data.account]}
                <div class="level-item">
                  <button
                    type="button"
                    class="button is-ghost p-0 h-auto is-small {reconciliationTextClass(
                      $reconciliationStatuses[data.account]
                    )}"
                    onclick={() =>
                      reconciliationModalState.set({ account: data.account, open: true })}
                    title={reconciliationLabel($reconciliationStatuses[data.account])}
                    style="vertical-align: baseline; height: 1.2em; width: 1.2em; line-height: 1;"
                  >
                    <span class="custom-icon" style="font-size: 0.9em;"
                      >{reconciliationIcon($reconciliationStatuses[data.account])}</span
                    >
                  </button>
                </div>
              {/if}
            </div>
            <div class="level-right">
              <div class="level-item">
                <p class="is-6"><b>{transactions.length}</b> transaction(s)</p>
              </div>
              <div class="level-item">
                <a
                  href="/accounts/{encodeURIComponent(data.account)}"
                  class="button is-small is-light"
                >
                  <span class="icon is-small"><i class="fas fa-sticky-note"></i></span>
                  <span>Notes</span>
                </a>
              </div>
              <div class="level-item">
                <a href="/ledger/transaction" class="button is-small is-light">
                  <span class="icon is-small"><i class="fas fa-list"></i></span>
                  <span>All Transactions</span>
                </a>
              </div>
            </div>
          </nav>
        </div>
      </div>

      <div class="columns">
        <div class="column is-12">
          <div class="box">
            {#if transactions.length === 0}
              <p class="has-text-grey has-text-centered py-4">
                No transactions found for <strong>{data.account}</strong>.
              </p>
            {:else}
              <TransactionHeader showExtraColumns={true} />
              <VirtualList
                width="100%"
                height={window.innerHeight - 150}
                itemCount={transactions.length}
                {itemSize}
              >
                <div slot="item" let:index let:style {style}>
                  {@const t = transactions[index]}
                  <Transaction {t} highlightAccount={data.account} />
                </div>
              </VirtualList>
            {/if}
          </div>
        </div>
      </div>
    </div>
  </section>
{/if}
