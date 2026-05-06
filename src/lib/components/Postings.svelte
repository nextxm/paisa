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
    <div class="is-flex is-justify-content-space-between is-hoverable" style="margin: 1px 0;">
      <div class="truncate custom-icon" style="min-width: 100px;" title={p.account}>
        <span style={accountColorStyle(firstName(p.account))}>{iconText(p.account)}</span>
        {p.account}
      </div>
      <div class="is-flex is-align-items-baseline is-justify-content-right">
        <div class="has-text-right has-text-grey is-size-7 mr-2 truncate">
          {unlessDefaultCurrency(p)}
        </div>
        <div class="has-text-right" style="min-width: 50px;">
          {formatCurrency(p.amount, 2)}
        </div>
        {#if highlightAccount && (p.account === highlightAccount || p.account.startsWith(highlightAccount + ":"))}
          <div class="has-text-right ml-4" style="min-width: 80px;">
            {formatFloatUptoPrecision(p.quantity, 2)}
            {p.commodity}
          </div>
          <div class="has-text-right ml-4 has-text-grey" style="min-width: 100px;">
            {formatCurrency(p.balance)}
            {p.commodity}
          </div>
        {/if}
      </div>
    </div>
  {/each}
</div>
