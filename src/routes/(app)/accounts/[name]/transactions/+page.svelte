<script lang="ts">
  import { ajax, isMobile, type AccountNote, type Transaction as T } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";
  import Transaction from "$lib/components/Transaction.svelte";
  import TransactionHeader from "$lib/components/TransactionHeader.svelte";
  import DateRange from "$lib/components/DateRange.svelte";
  import type { PageData } from "./$types";
  import dayjs from "dayjs";

  let { data }: { data: PageData } = $props();

  let transactions: T[] | null = $state(null);
  let accountNote: AccountNote | null = $state(null);
  let localDateMin = $state(dayjs("1980", "YYYY"));
  let localDateMax = $state(dayjs());
  let localRangeOption = $state(-1);

  const mobile = isMobile();

  const localDateRange = $derived(
    localRangeOption === -1
      ? { from: localDateMin, to: localDateMax }
      : { from: localDateMax.subtract(localRangeOption, "year"), to: localDateMax }
  );

  const filteredTransactions = $derived(
    transactions === null
      ? []
      : transactions.filter(
          (t) =>
            t.date.isSameOrAfter(localDateRange.from, "day") &&
            t.date.isSameOrBefore(localDateRange.to, "day")
        )
  );

  const itemSize = (i: number) => {
    const t = filteredTransactions[i];
    const count = t.postings.length;
    return 8 + count * 22 + (mobile ? 25 : 0);
  };

  onMount(async () => {
    const encoded = encodeURIComponent(data.account);
    transactions = (await ajax(`/api/transaction?account=${encoded}`)).transactions;
    const noteResult = await ajax("/api/account_notes/:account", null, { account: data.account });
    accountNote = noteResult.account_note;

    if (transactions && transactions.length > 0) {
      const dates = transactions.map((t) => t.date);
      localDateMin = _.minBy(dates, (d) => d.valueOf()) ?? dayjs("1980", "YYYY");
      localDateMax = _.maxBy(dates, (d) => d.valueOf()) ?? dayjs();
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
            </div>
            <div class="level-right">
              <div class="level-item">
                <DateRange
                  bind:value={localRangeOption}
                  dateMin={localDateMin}
                  dateMax={localDateMax}
                />
              </div>
              <div class="level-item">
                <p class="is-6"><b>{filteredTransactions.length}</b> transaction(s)</p>
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
            {#if filteredTransactions.length === 0}
              <p class="has-text-grey has-text-centered py-4">
                No transactions found for <strong>{data.account}</strong>.
              </p>
            {:else}
              <TransactionHeader showExtraColumns={true} />
              <VirtualList
                width="100%"
                height={window.innerHeight - 150}
                itemCount={filteredTransactions.length}
                {itemSize}
              >
                <div slot="item" let:index let:style {style}>
                  {@const t = filteredTransactions[index]}
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
