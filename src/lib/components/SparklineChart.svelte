<script lang="ts">
  import * as d3 from "d3";
  import _ from "lodash";
  import { formatCurrencyCrude } from "$lib/utils";
  import COLORS from "$lib/colors";

  /**
   * Monthly spending data for one expense category.
   * Key is "YYYY-MM", value is the total spend for that month.
   */
  let {
    data,
    color,
    height = 40,
    width = 120
  }: { data: Record<string, number>; color: string; height?: number; width?: number } = $props();

  interface Bar {
    month: string;
    value: number;
    x: number;
    barWidth: number;
    barHeight: number;
    y: number;
    isCurrent: boolean;
  }

  const bars = $derived.by<Bar[]>(() => {
    const allEntries: [string, number][] = _.chain(data)
      .entries()
      .sortBy(([m]: [string, number]) => m)
      .value() as [string, number][];

    const entries = allEntries.slice(-6);

    if (entries.length === 0) return [];

    const values = entries.map(([, v]: [string, number]) => v);
    const maxVal = Math.max(...values, 1);
    const padding = 2;
    const barWidth = (width - padding * (entries.length - 1)) / entries.length;
    const currentMonth = entries[entries.length - 1][0];

    return entries.map(([month, value]: [string, number], i: number) => {
      const barHeight = (value / maxVal) * height;
      return {
        month,
        value,
        x: i * (barWidth + padding),
        barWidth,
        barHeight,
        y: height - barHeight,
        isCurrent: month === currentMonth
      };
    });
  });

  const avgValue = $derived(bars.length > 0 ? _.sumBy(bars, (b) => b.value) / bars.length : 0);

  const avgY = $derived(
    bars.length > 0 && avgValue > 0
      ? (() => {
          const maxVal = Math.max(...bars.map((b) => b.value), 1);
          return height - (avgValue / maxVal) * height;
        })()
      : null
  );
</script>

<svg {width} {height} style="overflow: visible; display: block">
  {#each bars as bar}
    <rect
      x={bar.x}
      y={bar.y}
      width={bar.barWidth}
      height={bar.barHeight}
      fill={bar.isCurrent ? color : "#e0e0e0"}
      rx="1"
    >
      <title>{bar.month}: {formatCurrencyCrude(bar.value)}</title>
    </rect>
  {/each}

  {#if avgY !== null && bars.length > 1}
    <line
      x1={0}
      y1={avgY}
      x2={width}
      y2={avgY}
      stroke={COLORS.lossText}
      stroke-width="1"
      stroke-dasharray="3,2"
      opacity="0.6"
    >
      <title>6-month average: {formatCurrencyCrude(avgValue)}</title>
    </line>
  {/if}
</svg>
