<script lang="ts">
  import { formatTextAsHtml, type Transaction } from "$lib/utils";
  let { transaction }: { transaction: Transaction } = $props();
</script>

{#if transaction.note != ""}
  <span class="icon is-small" data-tippy-content={formatTextAsHtml(transaction.note)}>
    <i class="fa-regular fa-comment-dots"></i>
  </span>
{/if}
