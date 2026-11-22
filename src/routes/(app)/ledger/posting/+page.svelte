<script lang="ts">
  import { accountColorStyle } from "$lib/colors";
  import TransactionFilterBar from "$lib/components/TransactionFilterBar.svelte";
  import PostingNote from "$lib/components/PostingNote.svelte";
  import PostingStatus from "$lib/components/PostingStatus.svelte";
  import TransactionDrawer from "$lib/components/TransactionDrawer.svelte";
  import { iconText } from "$lib/icon";
  import { change } from "$lib/posting";
  import {
    buildTransactionFiltersQuery,
    emptyTransactionFilters,
    type TransactionFilters
  } from "$lib/transaction_filters";
  import {
    ajax,
    postingUrl,
    type Posting,
    type Transaction,
    formatCurrency,
    formatFloat,
    firstName,
    isMobile
  } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";

  let accounts: string[] = $state([]);
  let commodities: string[] = $state([]);

  let filteredPostings: Posting[] = $state([]);
  let transactionByPostingKey = $state(new Map<string, Transaction>());
  let selectedTransaction = $state<Transaction | null>(null);
  let filters: TransactionFilters = $state(emptyTransactionFilters());

  function postingKey(p: Posting) {
    return `${p.file_name}:${p.transaction_begin_line}:${p.transaction_end_line}`;
  }

  async function loadPostings(currentFilters = filters) {
    const query = buildTransactionFiltersQuery(currentFilters);
    const route = query ? `/api/transaction?${query}` : "/api/transaction";
    const { transactions } = await ajax(route);
    const map = new Map<string, Transaction>();
    for (const t of transactions as Transaction[]) {
      map.set(`${t.fileName}:${t.beginLine}:${t.endLine}`, t);
    }
    transactionByPostingKey = map;
    filteredPostings = _.flatMap(transactions, (transaction: Transaction) => transaction.postings);
  }

  function openPostingDrawer(e: MouseEvent, p: Posting) {
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return;
    const tx = transactionByPostingKey.get(postingKey(p));
    if (tx) {
      e.preventDefault();
      selectedTransaction = tx;
    }
  }

  const debouncedLoadPostings = _.debounce((currentFilters: TransactionFilters) => {
    loadPostings(currentFilters);
  }, 200);

  function handleFiltersChange(nextFilters: TransactionFilters) {
    filters = { ...nextFilters };
    debouncedLoadPostings(filters);
  }

  onMount(async () => {
    ({ accounts, commodities } = await ajax("/api/editor/files"));
    await loadPostings();
  });

  function unlessDefault(p: Posting, text: string) {
    if (p.commodity !== USER_CONFIG.default_currency) {
      return text;
    }
    return "";
  }

  function unlessZero(value: number, text: string) {
    if (value > 0) {
      return text;
    }
    return "";
  }
</script>

<section class="section tab-journal">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <nav class="level">
          <div class="level-left">
            <div class="level-item">
              <TransactionFilterBar
                {filters}
                {accounts}
                {commodities}
                onFiltersChange={handleFiltersChange}
              />
            </div>
          </div>
        </nav>
      </div>
    </div>
    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto" style="max-width: 98rem;">
          <div style="width: 98rem;">
            <div
              class="px-3 pt-1 grid grid-cols-7x gap-1 posting-row items-baseline has-text-weight-bold"
            >
              <div>Date</div>
              <div>Description</div>
              <div>Account</div>
              <div class="has-text-right">Amount</div>
              <div class="has-text-right">Balance</div>
              <div class="has-text-right">Units</div>
              <div class="has-text-right">Unit Price</div>
              <div class="has-text-right">Market Value</div>
              <div class="has-text-right">Change</div>
              <div class="has-text-right">CAGR</div>
            </div>
            <VirtualList
              height={window.innerHeight - 245}
              itemCount={filteredPostings.length}
              itemSize={isMobile() ? 32 : 27}
            >
              <div
                slot="item"
                class="px-3 pt-1 grid grid-cols-7x gap-1 posting-row items-baseline is-hoverable"
                let:index
                let:style
                style="{style} overflow: hidden;"
              >
                {@const p = filteredPostings[index]}
                {@const c = change(p)}
                <div>{p.date.format("DD MMM YYYY")}</div>
                <div class="is-size-7 truncate" title={p.payee}>
                  <PostingStatus posting={p} />
                  <PostingNote posting={p} />
                  <a
                    class="secondary-link"
                    href={postingUrl(p)}
                    onclick={(e) => openPostingDrawer(e, p)}>{p.payee}</a
                  >
                </div>
                <div class="custom-icon truncate" title={p.account}>
                  <div class="flex">
                    <span class="mr-1" style={accountColorStyle(firstName(p.account))}
                      >{iconText(p.account)}</span
                    >
                    <a
                      class="secondary-link"
                      href="/accounts/{encodeURIComponent(p.account)}/transactions">{p.account}</a
                    >
                  </div>
                </div>
                <div class="has-text-right">{formatCurrency(p.amount, 2)}</div>
                <div class="has-text-right">{formatCurrency(p.balance, 2)}</div>
                <div class="has-text-right">{unlessDefault(p, formatFloat(p.quantity, 4))}</div>
                <div class="has-text-right">
                  {unlessDefault(p, formatCurrency(Math.abs(p.amount / p.quantity), 4))}
                </div>
                <div class="has-text-right">
                  {unlessDefault(p, unlessZero(c.days, formatCurrency(p.market_amount)))}
                </div>
                <div class="has-text-right {c.class}">
                  {unlessZero(c.value, formatCurrency(c.value))}
                </div>
                <div class="has-text-right {c.class}">
                  {unlessZero(c.percentage, formatFloat(c.percentage))}
                </div>
              </div>
            </VirtualList>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>

<TransactionDrawer
  bind:transaction={selectedTransaction}
  {accounts}
  onsaved={() => loadPostings()}
/>
