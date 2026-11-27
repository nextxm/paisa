<script lang="ts">
  import dayjs from "dayjs";
  import { formatMoney } from "$lib/api";

  export interface NetworthPoint {
    date: string;
    balanceAmount: number;
    netInvestmentAmount: number;
    gainAmount: number;
  }

  let { data }: { data: NetworthPoint[] } = $props();

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

  const height = 260;
  const margin = { top: 16, right: 16, bottom: 28, left: 68 };
  const plotWidth = $derived(Math.max(100, width - margin.left - margin.right));
  const plotHeight = height - margin.top - margin.bottom;

  const bounds = $derived.by(() => {
    let min = 0;
    let max = 0;
    for (const d of data) {
      min = Math.min(min, d.balanceAmount || 0, d.netInvestmentAmount || 0);
      max = Math.max(max, d.balanceAmount || 0, d.netInvestmentAmount || 0);
    }
    if (max === min) max = min + 1000;
    return { min, max: max * 1.06 };
  });

  function xPos(index: number): number {
    if (data.length <= 1) return 0;
    return (index / (data.length - 1)) * plotWidth;
  }

  function yPos(val: number): number {
    const span = bounds.max - bounds.min || 1;
    return plotHeight - ((val - bounds.min) / span) * plotHeight;
  }

  const ticks = $derived(
    [0, 0.25, 0.5, 0.75, 1].map((t) => bounds.min + (bounds.max - bounds.min) * t)
  );

  const balancePath = $derived(
    data
      .map((d, i) => `${i === 0 ? "M" : "L"}${xPos(i).toFixed(1)},${yPos(d.balanceAmount).toFixed(1)}`)
      .join(" ")
  );

  const areaPath = $derived.by(() => {
    if (data.length === 0) return "";
    const baseY = yPos(bounds.min).toFixed(1);
    return `${balancePath} L${xPos(data.length - 1).toFixed(1)},${baseY} L${xPos(0).toFixed(1)},${baseY} Z`;
  });

  const investedPath = $derived(
    data
      .map(
        (d, i) =>
          `${i === 0 ? "M" : "L"}${xPos(i).toFixed(1)},${yPos(d.netInvestmentAmount).toFixed(1)}`
      )
      .join(" ")
  );

  function formatAxis(val: number): string {
    if (Math.abs(val) >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
    if (Math.abs(val) >= 100000) return `${(val / 100000).toFixed(1)}L`;
    if (Math.abs(val) >= 1000) return `${(val / 1000).toFixed(0)}K`;
    return `${Math.round(val)}`;
  }

  function handlePointerMove(e: MouseEvent) {
    if (data.length === 0 || !container) return;
    const rect = container.getBoundingClientRect();
    const relX = e.clientX - rect.left - margin.left;
    const ratio = Math.min(1, Math.max(0, relX / plotWidth));
    hoveredIndex = Math.round(ratio * (data.length - 1));
  }
</script>

<div bind:this={container} class="relative w-full select-none">
  {#if data.length === 0}
    <div class="h-60 flex items-center justify-center text-sm text-[var(--muted)]">
      No net worth history in this range.
    </div>
  {:else}
    <svg
      {width}
      {height}
      class="overflow-visible"
      role="img"
      aria-label="Net worth timeline"
      onmousemove={handlePointerMove}
      onmouseleave={() => (hoveredIndex = null)}
    >
      <g transform="translate({margin.left},{margin.top})">
        {#each ticks as tick}
          {@const y = yPos(tick)}
          <line
            x1="0"
            x2={plotWidth}
            y1={y}
            y2={y}
            stroke="var(--border)"
            stroke-dasharray="3 3"
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

        <path d={areaPath} fill="var(--accent)" opacity="0.1" />
        <path
          d={investedPath}
          fill="none"
          stroke="var(--muted)"
          stroke-width="1.5"
          stroke-dasharray="4 4"
        />
        <path d={balancePath} fill="none" stroke="var(--accent)" stroke-width="2.25" />

        {#each data as d, i}
          {#if i === 0 || i === data.length - 1 || (data.length > 6 && i % Math.ceil(data.length / 6) === 0)}
            <text
              x={xPos(i)}
              y={plotHeight + 18}
              text-anchor="middle"
              font-size="11"
              fill="var(--muted)"
            >
              {dayjs(d.date).format("MMM YY")}
            </text>
          {/if}
        {/each}

        {#if hoveredIndex !== null && data[hoveredIndex]}
          {@const pt = data[hoveredIndex]}
          {@const hx = xPos(hoveredIndex)}
          <line x1={hx} x2={hx} y1="0" y2={plotHeight} stroke="var(--border)" />
          <circle cx={hx} cy={yPos(pt.balanceAmount)} r="4" fill="var(--accent)" />
        {/if}
      </g>
    </svg>

    {#if hoveredIndex !== null && data[hoveredIndex]}
      {@const pt = data[hoveredIndex]}
      <div
        class="mt-2 px-3 py-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] flex flex-wrap items-center justify-between gap-4 text-xs tabular-nums"
      >
        <span class="font-semibold">{dayjs(pt.date).format("DD MMM YYYY")}</span>
        <div class="flex flex-wrap items-center gap-4">
          <span>Net Worth: <strong>{formatMoney(pt.balanceAmount)}</strong></span>
          <span class="text-[var(--muted)]">Invested: {formatMoney(pt.netInvestmentAmount)}</span>
          <span class={pt.gainAmount >= 0 ? "text-[var(--gain)]" : "text-[var(--loss)]"}>
            Gain: {formatMoney(pt.gainAmount)}
          </span>
        </div>
      </div>
    {/if}
  {/if}
</div>
