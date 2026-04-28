<script lang="ts">
  import { dueDateIcon } from "$lib/utils";
  import dayjs from "dayjs";

  let { dueDate, paidDate }: { dueDate: dayjs.Dayjs; paidDate: dayjs.Dayjs } = $props();
  const icon = $derived(dueDateIcon(dueDate, paidDate));
</script>

<span title="due on {dueDate.format('DD MMM YYYY')}">
  <span class="icon is-small {icon.color}">
    <i class="fas {icon.icon}"></i>
  </span>
  {#if paidDate}
    <span>paid on {paidDate.format("DD MMM YYYY")}</span>
  {:else}
    <span>due {dueDate.fromNow()}</span>
  {/if}
</span>
