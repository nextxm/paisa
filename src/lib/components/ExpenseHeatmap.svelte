<script lang="ts">
  import * as d3 from "d3";
  import { buildHeatmapWeeks } from "$lib/expense_heatmap";
  import { formatCurrency, type DailyExpenseDay } from "$lib/utils";
  import type dayjs from "dayjs";

  let {
    days = [],
    from,
    to,
    category = ""
  }: {
    days: DailyExpenseDay[];
    from: dayjs.Dayjs;
    to: dayjs.Dayjs;
    category?: string;
  } = $props();

  const weekdayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];

  let weeks = $derived(buildHeatmapWeeks(days, from, to, category));
  let maxSpend = $derived(Math.max(0, ...weeks.flat().map((cell) => cell.total)));
  let colorScale = $derived(
    d3
      .scaleQuantize<string>()
      .domain([0, Math.max(maxSpend, 1)])
      .range(["#fee2e2", "#fecaca", "#fca5a5", "#f87171", "#ef4444", "#b91c1c"])
  );
  let monthHeaders = $derived(
    weeks.map((week, index) => {
      const firstInRange = week.find((cell) => cell.inRange);
      const monthLabel = firstInRange ? firstInRange.date.format("MMM") : "";
      const previous = index > 0 ? weeks[index - 1].find((cell) => cell.inRange) : undefined;
      return monthLabel !== (previous ? previous.date.format("MMM") : "") ? monthLabel : "";
    })
  );

  function colorFor(total: number, inRange: boolean) {
    if (!inRange) return "transparent";
    if (total <= 0) return "var(--p-surface-2, #f3f4f6)";
    return colorScale(total);
  }

  function cellTitle(date: dayjs.Dayjs, total: number, inRange: boolean) {
    if (!inRange) return "";
    const suffix = category ? ` • ${category}` : "";
    return `${date.format("MMM D, YYYY")} • ${formatCurrency(total)}${suffix}`;
  }
</script>

<div class="expense-heatmap" data-testid="expense-heatmap">
  <div class="months">
    <div class="months-spacer"></div>
    <div
      class="months-grid"
      style={`grid-template-columns: repeat(${weeks.length}, minmax(0, 1fr));`}
    >
      {#each monthHeaders as month}
        <div class="month-label">{month}</div>
      {/each}
    </div>
  </div>

  <div class="heatmap-body">
    <div class="weekday-labels">
      {#each weekdayLabels as label}
        <div>{label}</div>
      {/each}
    </div>

    <div
      class="weeks-grid"
      style={`grid-template-columns: repeat(${weeks.length}, minmax(0, 1fr));`}
    >
      {#each weeks as week}
        <div class="week-column">
          {#each week as cell}
            <div
              class:placeholder={!cell.inRange}
              class="day-cell"
              style={`background:${colorFor(cell.total, cell.inRange)}`}
              title={cellTitle(cell.date, cell.total, cell.inRange)}
            ></div>
          {/each}
        </div>
      {/each}
    </div>
  </div>

  <div class="heatmap-legend">
    <span>Less</span>
    <div class="legend-scale">
      <span style="background: var(--p-surface-2, #f3f4f6)"></span>
      {#each ["#fee2e2", "#fecaca", "#fca5a5", "#f87171", "#ef4444", "#b91c1c"] as tone}
        <span style={`background:${tone}`}></span>
      {/each}
    </div>
    <span>More</span>
  </div>
</div>

<style>
  .expense-heatmap {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .months {
    display: grid;
    grid-template-columns: 3rem minmax(0, 1fr);
    gap: 0.5rem;
    align-items: end;
  }

  .months-grid {
    display: grid;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--p-text-2, #6b7280);
  }

  .month-label {
    min-width: 0;
  }

  .heatmap-body {
    display: grid;
    grid-template-columns: 3rem minmax(0, 1fr);
    gap: 0.5rem;
    align-items: start;
  }

  .weekday-labels {
    display: grid;
    grid-template-rows: repeat(7, minmax(0, 1fr));
    gap: 0.25rem;
    font-size: 0.7rem;
    color: var(--p-text-2, #6b7280);
    line-height: 1;
  }

  .weekday-labels div {
    min-height: 14px;
    display: flex;
    align-items: center;
  }

  .weeks-grid {
    display: grid;
    gap: 0.25rem;
    min-width: 0;
  }

  .week-column {
    display: grid;
    grid-template-rows: repeat(7, minmax(0, 1fr));
    gap: 0.25rem;
  }

  .day-cell {
    width: 100%;
    aspect-ratio: 1;
    min-height: 14px;
    border-radius: 0.2rem;
    border: 1px solid color-mix(in srgb, currentColor 10%, transparent);
  }

  .day-cell.placeholder {
    border-color: transparent;
  }

  .heatmap-legend {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    justify-content: flex-end;
    font-size: 0.75rem;
    color: var(--p-text-2, #6b7280);
  }

  .legend-scale {
    display: grid;
    grid-template-columns: repeat(7, 12px);
    gap: 0.2rem;
  }

  .legend-scale span {
    width: 12px;
    height: 12px;
    border-radius: 0.2rem;
  }
</style>
