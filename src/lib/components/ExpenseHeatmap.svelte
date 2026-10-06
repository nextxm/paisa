<script lang="ts">
  import * as d3 from "d3";
  import { buildHeatmapWeeks, buildHeatmapWeeksOfMonth } from "$lib/expense_heatmap";
  import { formatCurrency, type DailyExpenseDay } from "$lib/utils";
  import type dayjs from "dayjs";

  let {
    days = [],
    from,
    to,
    category = "",
    viewMode = "day"
  }: {
    days: DailyExpenseDay[];
    from: dayjs.Dayjs;
    to: dayjs.Dayjs;
    category?: string;
    viewMode?: "day" | "week";
  } = $props();

  const weekdayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
  const weekLabels = ["Week 1", "Week 2", "Week 3", "Week 4", "Week 5"];

  let currentLabels = $derived(viewMode === "day" ? weekdayLabels : weekLabels);

  let weeks = $derived(
    viewMode === "day"
      ? buildHeatmapWeeks(days, from, to, category)
      : buildHeatmapWeeksOfMonth(days, from, to, category)
  );
  let colWidth = $derived(viewMode === "day" ? "minmax(0, 16px)" : "minmax(0, 22px)");
  let maxSpend = $derived(Math.max(0, ...weeks.flat().map((cell) => cell.total)));
  let colorScale = $derived(
    d3
      .scaleQuantize<string>()
      .domain([0, Math.max(maxSpend, 1)])
      .range(["#fee2e2", "#fecaca", "#fca5a5", "#f87171", "#ef4444", "#b91c1c"])
  );
  let monthHeaders = $derived(
    viewMode === "day"
      ? weeks.map((week, index) => {
          const firstInRange = week.find((cell) => cell.inRange);
          const monthLabel = firstInRange ? firstInRange.date.format("MMM") : "";
          const previous = index > 0 ? weeks[index - 1].find((cell) => cell.inRange) : undefined;
          return monthLabel !== (previous ? previous.date.format("MMM") : "") ? monthLabel : "";
        })
      : weeks.map((week) => {
          return week[0]?.date.format("MMM") || "";
        })
  );

  function colorFor(total: number, inRange: boolean) {
    if (!inRange) return "transparent";
    if (total <= 0) return "var(--p-surface-2, #f3f4f6)";
    return colorScale(total);
  }

  function cellTitle(date: dayjs.Dayjs, total: number, inRange: boolean, tooltipRange?: string) {
    if (!inRange) return "";
    const suffix = category ? ` • ${category}` : "";
    const dateStr = tooltipRange || date.format("MMM D, YYYY");
    return `${dateStr} • ${formatCurrency(total)}${suffix}`;
  }
</script>

<div class="expense-heatmap" data-testid="expense-heatmap">
  <div class="months">
    <div class="months-spacer"></div>
    <div class="months-grid" style={`grid-template-columns: repeat(${weeks.length}, ${colWidth});`}>
      {#each monthHeaders as month}
        <div class="month-label">{month}</div>
      {/each}
    </div>
  </div>

  <div class="heatmap-body">
    <div class="weekday-labels" class:week-mode={viewMode === "week"}>
      {#each currentLabels as label}
        <div>{label}</div>
      {/each}
    </div>

    <div class="weeks-grid" style={`grid-template-columns: repeat(${weeks.length}, ${colWidth});`}>
      {#each weeks as week}
        <div class="week-column" class:week-mode={viewMode === "week"}>
          {#each week as cell}
            <div
              class:placeholder={!cell.inRange}
              class="day-cell"
              style={`background:${colorFor(cell.total, cell.inRange)}`}
              title={cellTitle(cell.date, cell.total, cell.inRange, cell.tooltipRange)}
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
    grid-template-columns: 3.5rem minmax(0, 1fr);
    gap: 0.5rem;
    align-items: end;
  }

  .months-grid {
    display: grid;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: var(--p-text-2, #6b7280);
    justify-content: start;
  }

  .month-label {
    min-width: 0;
  }

  .heatmap-body {
    display: grid;
    grid-template-columns: 3.5rem minmax(0, 1fr);
    gap: 0.5rem;
    align-items: stretch;
  }

  .weekday-labels {
    display: grid;
    grid-template-rows: repeat(7, minmax(0, 1fr));
    gap: 0.25rem;
    font-size: 0.7rem;
    color: var(--p-text-2, #6b7280);
    line-height: 1;
  }

  .weekday-labels.week-mode {
    grid-template-rows: repeat(5, minmax(0, 1fr));
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
    justify-content: start;
  }

  .week-column {
    display: grid;
    grid-template-rows: repeat(7, minmax(0, 1fr));
    gap: 0.25rem;
  }

  .week-column.week-mode {
    grid-template-rows: repeat(5, minmax(0, 1fr));
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
