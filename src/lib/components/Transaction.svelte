<script lang="ts">
  import { isMobile, postingUrl, type Transaction } from "$lib/utils";
  import Postings from "$lib/components/Postings.svelte";
  import _ from "lodash";
  import PostingStatus from "$lib/components/PostingStatus.svelte";
  import TransactionNote from "./TransactionNote.svelte";

  let {
    compact = false,
    t,
    highlightAccount = "",
    onselect
  }: {
    compact?: boolean;
    t: Transaction;
    highlightAccount?: string;
    onselect?: (t: Transaction) => void;
  } = $props();

  const useCompact = $derived(compact || isMobile());

  function handleSelect(e: MouseEvent) {
    if (!onselect) return;
    if (e.metaKey || e.ctrlKey || e.shiftKey || e.altKey || e.button !== 0) return;
    e.preventDefault();
    onselect(t);
  }
</script>

<div class="column is-12" class:py-1={useCompact}>
  {#if useCompact}
    <div class="columns is-flex-wrap-wrap transaction" class:bordered={!compact}>
      <div class="column is-12 py-0 truncate" style="line-height: 22px; height: 22px;">
        <div class="description is-size-7" style="max-width: 100%;">
          <b>{t.date.format("DD MMM YYYY")}</b>
          <span title={t.payee}>
            <PostingStatus posting={t.postings[0]} />
            <TransactionNote transaction={t} />
            <a class="secondary-link" href={postingUrl(t.postings[0])} onclick={handleSelect}
              >{t.payee}</a
            ></span
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
            <a class="secondary-link" href={postingUrl(t.postings[0])} onclick={handleSelect}
              >{t.payee}</a
            ></span
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
