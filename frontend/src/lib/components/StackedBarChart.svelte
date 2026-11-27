<script lang="ts">
  import { formatMoney } from "$lib/api";

  export interface StackedBarColumn {
    label: string;
    total: number;
    segments: Array<{ name: string; amount: number; color: string }>;
  }

  let { columns }: { columns: StackedBarColumn[] } = $props();

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

  const height = 250;
  const margin = { top: 16, right: 16, bottom: 28, left: 64 };
  const plotWidth = $derived(Math.max(100, width - margin.left - margin.right));
  const plotHeight = height - margin.top - margin.bottom;

  const maxVal = $derived.by(() => {
    let m = 0;
    for (const c of columns) {
      m = Math.max(m, c.total || 0);
    }
    return m > 0 ? m * 1.1 : 1000;
  });

  const ticks = $derived([0, 0.25, 0.5, 0.75, 1].map((t) => maxVal * t));

  function yPos(val: number): number {
    return plotHeight - (Math.max(0, val) / maxVal) * plotHeight;
  }

  const slotWidth = $derived(columns.length > 0 ? plotWidth / columns.length : plotWidth);
  const barWidth = $derived(Math.max(6, Math.min(34, slotWidth * 0.62)));

  function formatAxis(val: number): string {
    if (val >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
    if (val >= 100000) return `${(val / 100000).toFixed(1)}L`;
    if (val >= 1000) return `${(val / 1000).toFixed(0)}K`;
    return `${Math.round(val)}`;
  }
</script>

<div bind:this={container} class="relative w-full select-none">
  {#if columns.length === 0}
    <div class="h-56 flex items-center justify-center text-sm text-[var(--muted)]">
      No categorized expense data available.
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
          <text x="-8" y={y + 4} text-anchor="end" font-size="11" fill="var(--muted)" class="tabular-nums">
            {formatAxis(tick)}
          </text>
        {/each}

        {#each columns as col, i}
          {@const cx = i * slotWidth + slotWidth / 2}
          {@const bx = cx - barWidth / 2}
          {@const stackedSegments = col.segments.reduce<Array<{ name: string; amount: number; color: string; yTop: number; h: number; _cum: number }>>(
            (acc, seg) => {
              const prevCum = acc.length > 0 ? acc[acc.length - 1]._cum : 0;
              const nextCum = prevCum + Math.max(0, seg.amount);
              const yTop = yPos(nextCum);
              const yBot = yPos(prevCum);
              return [...acc, { ...seg, yTop, h: Math.max(0, yBot - yTop), _cum: nextCum }];
            },
            []
          )}
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
            {#each stackedSegments as seg}
              <rect
                x={bx}
                y={seg.yTop}
                width={barWidth}
                height={seg.h}
                fill={seg.color}
                opacity={hoveredIndex === null || hoveredIndex === i ? 0.9 : 0.55}
              />
            {/each}
            {#if columns.length <= 14 || i % Math.ceil(columns.length / 10) === 0}
              <text x={cx} y={plotHeight + 18} text-anchor="middle" font-size="11" fill="var(--muted)">
                {col.label}
              </text>
            {/if}
          </g>
        {/each}
      </g>
    </svg>

    {#if hoveredIndex !== null && columns[hoveredIndex]}
      {@const item = columns[hoveredIndex]}
      <div class="mt-2 p-3 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] text-xs space-y-2">
        <div class="flex items-center justify-between font-semibold">
          <span>{item.label}</span>
          <span class="tabular-nums text-[var(--loss)]">Total: {formatMoney(item.total)}</span>
        </div>
        <div class="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-x-4 gap-y-1 tabular-nums">
          {#each [...item.segments].sort((a, b) => b.amount - a.amount) as seg}
            <div class="flex items-center justify-between gap-2">
              <span class="inline-flex items-center gap-1.5 truncate">
                <span class="w-2 h-2 rounded-xs shrink-0" style="background: {seg.color}"></span>
                <span class="truncate">{seg.name}</span>
              </span>
              <span class="font-medium shrink-0">{formatMoney(seg.amount)}</span>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  {/if}
</div>
