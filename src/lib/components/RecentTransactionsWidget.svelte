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

<div class="content">
  <p class="subtitle">
    <a class="secondary-link has-text-grey" href="/ledger/transaction">Recent Transactions</a>
  </p>
  <div>
    <div class="masonry-grid masonry-grid-500">
      {#each visible as t (t.id)}
        <div class="mr-3 is-flex-grow-1">
          <TransactionCard {t} />
        </div>
      {/each}
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
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  }
</style>
