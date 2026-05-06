<script lang="ts">
  import { formatCurrency, type ExpenseTrend } from "$lib/utils";
  import { iconify } from "$lib/icon";
  import COLORS from "$lib/colors";
  import SparklineChart from "./SparklineChart.svelte";

  let {
    trend,
    color,
    sparkData = {}
  }: { trend: ExpenseTrend; color: string; sparkData?: Record<string, number> } = $props();

  const isIncrease = $derived(trend.variance > 0);
  const isDecrease = $derived(trend.variance < 0);
  const hasPrevious = $derived(trend.variance_pct !== null);

  const trendColor = $derived(
    isIncrease ? COLORS.lossText : isDecrease ? COLORS.gainText : "inherit"
  );

  const arrowIcon = $derived(
    isIncrease ? "fa-arrow-up" : isDecrease ? "fa-arrow-down" : "fa-minus"
  );

  const hasSparkline = $derived(Object.keys(sparkData).length > 1);
</script>

<div
  class="box p-2 my-2 has-background-white"
  style="border-left: 2px solid {color}"
  title="Previous 30 days: {formatCurrency(
    trend.previous_month
  )} · Current 30 days: {formatCurrency(trend.current_month)}"
>
  <div class="is-flex is-justify-content-space-between is-align-items-center">
    <div class="has-text-grey has-text-weight-semibold custom-icon truncate">
      {iconify(trend.category, { group: "Expenses", suffix: true })}
    </div>
    <div class="has-text-weight-bold is-size-6 ml-2" style="white-space: nowrap">
      {formatCurrency(trend.current_month)}
    </div>
  </div>
  <div class="is-flex is-justify-content-space-between is-align-items-center mt-1">
    <div class="has-text-grey-light is-size-7">
      prev: {formatCurrency(trend.previous_month)}
    </div>
    {#if hasPrevious}
      <div
        class="is-size-7 has-text-weight-semibold"
        style="color: {trendColor}; white-space: nowrap"
      >
        <span class="icon is-small">
          <i class="fas {arrowIcon}"></i>
        </span>
        {Math.abs(trend.variance_pct).toFixed(2)}%
      </div>
    {/if}
  </div>
  {#if hasSparkline}
    <div class="mt-2" style="overflow: hidden">
      <SparklineChart data={sparkData} {color} width={160} height={32} />
    </div>
  {/if}
</div>
