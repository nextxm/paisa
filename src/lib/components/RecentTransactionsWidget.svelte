<script lang="ts">
  import { appPath, type Transaction } from "$lib/utils";
  import TransactionCard from "./TransactionCard.svelte";
  import _ from "lodash";

  let {
    transactions,
    limit = 15
  }: {
    transactions: Transaction[];
    limit?: number;
  } = $props();

  let visible = $derived(_.take(transactions, limit));
</script>

<div class="box dashboard-card" style="padding: 1.15rem 1.25rem;">
  <div class="content mb-0">
    <div
      class="is-flex is-justify-content-space-between is-align-items-center mb-3"
      style="font-size: 0.72rem; font-weight: 600; text-transform: uppercase; letter-spacing: 0.05em;"
    >
      <a class="secondary-link has-text-grey" href={appPath("/ledger/transaction")}
        >Recent Transactions</a
      >
      <a
        href={appPath("/ledger/transaction")}
        style="font-size: 0.72rem; font-weight: 500; text-transform: none; letter-spacing: normal; opacity: 0.75;"
        >All transactions →</a
      >
    </div>
    <div>
      <div class="masonry-grid masonry-grid-500">
        {#each visible as t (t.id)}
          <div class="mr-3 is-flex-grow-1 recent-tx-item">
            <TransactionCard {t} />
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

<style lang="scss">
  .masonry-grid {
    display: grid;
    gap: 10px;
    align-items: stretch;
  }

  .masonry-grid-500 {
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 320px), 1fr));
  }

  @media screen and (max-width: 768px) {
    .recent-tx-item {
      margin-right: 0 !important;
      min-width: 0;
    }
  }
</style>
