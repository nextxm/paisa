<script lang="ts">
  import { type Transaction } from "$lib/utils";
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

<div class="box px-3">
  <div class="content">
    <p class="subtitle">
      <a class="secondary-link has-text-grey" href="/ledger/transaction">Recent Transactions</a>
    </p>
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

  p.subtitle {
    margin-bottom: 0.5rem !important;
  }

  p.subtitle a.secondary-link {
    text-transform: uppercase;
    font-size: 1rem;
  }
</style>
