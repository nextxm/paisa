<script lang="ts">
  import { accountColorStyle } from "$lib/colors";
  import { iconText } from "$lib/icon";
  import { firstName, formatCurrency, formatFloatUptoPrecision, type Posting } from "$lib/utils";

  const unlessDefaultCurrency = (p: Posting) => {
    if (p.commodity == USER_CONFIG.default_currency) {
      return "";
    } else {
      return `${formatFloatUptoPrecision(p.quantity, 3)} ${
        p.commodity
      } @ ${formatFloatUptoPrecision(p.amount / p.quantity, 4)}`;
    }
  };

  let { postings, highlightAccount = "" }: { postings: Posting[]; highlightAccount?: string } =
    $props();
</script>

<div style="margin: 4px 0;">
  {#each postings as p}
    <div class="is-flex is-hoverable" style="margin: 1px 0;">
      <div
        class="truncate custom-icon mr-2"
        style="flex: 0 0 40%; max-width: 40%;"
        title={p.account}
      >
        <span style={accountColorStyle(firstName(p.account))}>{iconText(p.account)}</span>
        <a class="secondary-link" href="/accounts/{encodeURIComponent(p.account)}/transactions"
          >{p.account}</a
        >
      </div>
      <div
        class="has-text-right has-text-grey is-size-7 truncate"
        style="flex: 0 0 20%; max-width: 20%;"
      >
        {unlessDefaultCurrency(p)}
      </div>
      <div class="has-text-right" style="flex: 0 0 15%; max-width: 15%;">
        {formatCurrency(p.amount, 2)}
      </div>
      {#if highlightAccount}
        <div class="has-text-right" style="flex: 0 0 10%; max-width: 10%;">
          {#if p.account === highlightAccount || p.account.startsWith(highlightAccount + ":")}
            {formatFloatUptoPrecision(p.quantity, 2)}
            {p.commodity}
          {/if}
        </div>
        <div class="has-text-right has-text-grey" style="flex: 0 0 15%; max-width: 15%;">
          {#if p.account === highlightAccount || p.account.startsWith(highlightAccount + ":")}
            {formatCurrency(p.balance)}
            {p.commodity}
          {/if}
        </div>
      {:else}
        <div style="flex: 0 0 25%; max-width: 25%;"></div>
      {/if}
    </div>
  {/each}
</div>
