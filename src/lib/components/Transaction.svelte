<script lang="ts">
  import { postingUrl, type Transaction } from "$lib/utils";
  import Postings from "$lib/components/Postings.svelte";
  import _ from "lodash";
  import PostingStatus from "$lib/components/PostingStatus.svelte";
  import TransactionNote from "./TransactionNote.svelte";

  let {
    compact = false,
    t,
    highlightAccount = ""
  }: { compact?: boolean; t: Transaction; highlightAccount?: string } = $props();
</script>

<div class="column is-12">
  {#if compact}
    <div class="columns is-flex-wrap-wrap transaction">
      <div class="column is-12 py-0 truncate">
        <div class="description is-size-7">
          <b>{t.date.format("DD MMM YYYY")}</b>
          <span title={t.payee}>
            <PostingStatus posting={t.postings[0]} />
            <TransactionNote transaction={t} />
            <a class="secondary-link" href={postingUrl(t.postings[0])}>{t.payee}</a></span
          >
        </div>
      </div>
      <div class="column is-12 py-0">
        <Postings postings={t.postings} {highlightAccount} />
      </div>
    </div>
  {:else}
    <div class="columns is-flex-wrap-wrap transaction bordered">
      <div class="column py-0 truncate" style="flex: 0 0 30%; max-width: 30%;">
        <div class="description mt-2 is-size-7">
          <b>{t.date.format("DD MMM YYYY")}</b>
          <span title={t.payee}
            ><PostingStatus posting={t.postings[0]} />
            <TransactionNote transaction={t} />
            <a class="secondary-link" href={postingUrl(t.postings[0])}>{t.payee}</a></span
          >
        </div>
      </div>
      <div class="column py-0" style="flex: 0 0 70%; max-width: 70%;">
        <Postings postings={t.postings} {highlightAccount} />
      </div>
    </div>
  {/if}
</div>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  .description {
    display: inline-block;
    white-space: nowrap;
    overflow: hidden;
  }
</style>
