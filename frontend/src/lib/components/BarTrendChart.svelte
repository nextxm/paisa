<script lang="ts">
  import dayjs from "dayjs";
  import { formatMoney } from "$lib/api";

  export interface MonthlyBarPoint {
    date: string;
    income: number;
    expenses: number;
    investment: number;
    balance: number;
  }

  let { data }: { data: MonthlyBarPoint[] } = $props();

  let container = $state<HTMLDivElement | null>(null);
  let width = $state(640);
  let hoveredIndex = $state<number | null>(null);

  $effect(() => {
    if (!container) return;
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) {
        width = Math.max(280, entry.contentRect.width);
      }
    });
    observer.observe(container);
    return () => observer.disconnect();
  });

  const height = 240;
  const margin = { top: 16, right: 16, bottom: 28, left: 64 };
  const plotWidth = $derived(Math.max(100, width - margin.left - margin.right));
  const plotHeight = height - margin.top - margin.bottom;

  const maxVal = $derived.by(() => {
    let m = 0;
    for (const d of data) {
      m = Math.max(m, Math.abs(d.income || 0), Math.abs(d.expenses || 0));
    }
    return m > 0 ? m * 1.1 : 1000;
  });

  const ticks = $derived([0, 0.25, 0.5, 0.75, 1].map((t) => maxVal * t));

  function yPos(val: number): number {
    return plotHeight - (Math.max(0, val) / maxVal) * plotHeight;
  }

  const slotWidth = $derived(data.length > 0 ? plotWidth / data.length : plotWidth);
  const barWidth = $derived(Math.max(4, Math.min(20, slotWidth * 0.34)));

  function formatAxis(val: number): string {
    if (val >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
    if (val >= 100000) return `${(val / 100000).toFixed(1)}L`;
    if (val >= 1000) return `${(val / 1000).toFixed(0)}K`;
    return `${Math.round(val)}`;
  }
</script>

<div bind:this={container} class="relative w-full select-none">
  {#if data.length === 0}
    <div class="h-56 flex items-center justify-center text-sm text-[var(--muted)]">
      No cash flow data in this range.
    </div>
  {:else}
    <svg {width} {height} class="overflow-visible">
      <g transform="translate({margin.left},{margin.top})">
        {#each ticks as tick}
          {@const y = yPos(tick)}
          <line
            x1="0"
            x2={plotWidth}
            y1={y}
            y2={y}
            stroke="var(--border)"
            stroke-dasharray={tick === 0 ? undefined : "3 3"}
          />
          <text
            x="-8"
            y={y + 4}
            text-anchor="end"
            font-size="11"
            fill="var(--muted)"
            class="tabular-nums"
          >
            {formatAxis(tick)}
          </text>
        {/each}

        {#each data as d, i}
          {@const cx = i * slotWidth + slotWidth / 2}
          {@const inc = Math.max(0, d.income || 0)}
          {@const exp = Math.max(0, d.expenses || 0)}
          <g
            role="presentation"
            onmouseenter={() => (hoveredIndex = i)}
            onmouseleave={() => (hoveredIndex = null)}
          >
            <rect
              x={i * slotWidth}
              y="0"
              width={slotWidth}
              height={plotHeight}
              fill={hoveredIndex === i ? "var(--surface-elevated)" : "transparent"}
              opacity="0.5"
              rx="4"
            />
            <rect
              x={cx - barWidth - 1.5}
              y={yPos(inc)}
              width={barWidth}
              height={Math.max(0, plotHeight - yPos(inc))}
              fill="var(--gain)"
              rx="3"
              opacity="0.88"
            />
            <rect
              x={cx + 1.5}
              y={yPos(exp)}
              width={barWidth}
              height={Math.max(0, plotHeight - yPos(exp))}
              fill="var(--loss)"
              rx="3"
              opacity="0.88"
            />
            {#if data.length <= 14 || i % Math.ceil(data.length / 10) === 0}
              <text
                x={cx}
                y={plotHeight + 18}
                text-anchor="middle"
                font-size="11"
                fill="var(--muted)"
              >
                {dayjs(d.date).format("MMM YY")}
              </text>
            {/if}
          </g>
        {/each}
      </g>
    </svg>

    {#if hoveredIndex !== null && data[hoveredIndex]}
      {@const item = data[hoveredIndex]}
      <div
        class="mt-2 px-3 py-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] flex flex-wrap items-center justify-between gap-4 text-xs tabular-nums"
      >
        <span class="font-semibold">{dayjs(item.date).format("MMMM YYYY")}</span>
        <div class="flex flex-wrap items-center gap-4">
          <span class="text-[var(--gain)]">Income: {formatMoney(item.income)}</span>
          <span class="text-[var(--loss)]">Expenses: {formatMoney(item.expenses)}</span>
          <span>Investments: {formatMoney(item.investment)}</span>
        </div>
      </div>
    {/if}
  {/if}
</div>
