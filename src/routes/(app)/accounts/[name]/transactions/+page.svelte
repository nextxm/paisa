<script lang="ts">
  import { ajax, isMobile, type AccountNote, type Transaction as T } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";
  import Transaction from "$lib/components/Transaction.svelte";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  let transactions: T[] | null = $state(null);
  let accountNote: AccountNote | null = $state(null);

  const mobile = isMobile();

  const debits = (t: T) => {
    return _.filter(t.postings, (p) => p.amount < 0);
  };

  const credits = (t: T) => {
    return _.filter(t.postings, (p) => p.amount >= 0);
  };

  const itemSize = (i: number) => {
    const t = transactions[i];
    const count = mobile ? t.postings.length : Math.max(credits(t).length, debits(t).length);
    return 8 + count * 22 + (mobile ? 25 : 0);
  };

  onMount(async () => {
    const encoded = encodeURIComponent(data.account);
    [{ transactions }, { account_note: accountNote }] = await Promise.all([
      ajax(`/api/transaction?account=${encoded}`),
      ajax("/api/account_notes/:account", null, { account: data.account })
    ]);
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
                <p class="title is-5">{data.account}</p>
              </div>
              {#if accountNote?.note}
                <div class="level-item">
                  <span class="tag is-info is-light">
                    <span class="icon is-small mr-1"><i class="fas fa-sticky-note"></i></span>
                    {accountNote.note}
                  </span>
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
              <VirtualList
                width="100%"
                height={window.innerHeight - 150}
                itemCount={transactions.length}
                {itemSize}
              >
                <div slot="item" let:index let:style {style}>
                  {@const t = transactions[index]}
                  <Transaction {t} />
                </div>
              </VirtualList>
            {/if}
          </div>
        </div>
      </div>
    </div>
  </section>
{/if}
