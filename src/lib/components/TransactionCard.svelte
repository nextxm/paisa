<script lang="ts">
  import { accountColorStyle } from "$lib/colors";
  import { iconText } from "$lib/icon";
  import {
    formatCurrency,
    formatFloat,
    postingUrl,
    restName,
    type Posting,
    type Transaction,
    firstName
  } from "$lib/utils";
  import PostingStatus from "./PostingStatus.svelte";
  import TransactionNote from "./TransactionNote.svelte";

  let { t }: { t: Transaction } = $props();
  const posting = $derived(t.postings[0]);
</script>

<div class="box p-2 has-background-white">
  <div class="is-flex is-justify-content-space-between is-align-items-baseline">
    <div class="has-text-grey is-size-7 truncate">
      <PostingStatus {posting} />
      <TransactionNote transaction={t} />
      <a class="secondary-link" href={postingUrl(posting)}>{posting.payee}</a>
    </div>
    <div class="has-text-grey min-w-[110px] has-text-right">
      <span class="icon is-small has-text-grey-light">
        <i class="fas fa-calendar"></i>
      </span>
      {posting.date.format("DD MMM YYYY")}
    </div>
  </div>
  <hr class="my-1" />
  {#each t.postings as posting}
    {@const account = posting.account ?? ""}
    <div class="my-1 is-flex is-justify-content-space-between">
      <div class="has-text-grey truncate custom-icon" title={account}>
        <span style={accountColorStyle(firstName(account))}>{iconText(account)}</span>
        {restName(account)}
      </div>
      <div class="has-text-weight-bold is-size-6 has-text-right whitespace-nowrap">
        {#if posting.commodity !== USER_CONFIG.default_currency}
          <span>{formatFloat(posting.quantity)} {posting.commodity}</span>
          <span class="is-size-7 has-text-grey has-text-weight-normal ml-1">
            ({formatCurrency(posting.amount)})
          </span>
        {:else}
          {formatCurrency(posting.amount)}
        {/if}
      </div>
    </div>
  {/each}
</div>
