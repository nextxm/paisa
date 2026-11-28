<script lang="ts">
  import dayjs from "dayjs";
  import { formatMoney } from "$lib/api";
  import {
    buildBandPath,
    buildLinePath,
    computeNetworthBounds,
    getInvestmentReturn,
    type NetworthPoint
  } from "$lib/networth_chart";

  export type { NetworthPoint };

  let { data, showFxImpact = true }: { data: NetworthPoint[]; showFxImpact?: boolean } = $props();

  let container = $state<HTMLDivElement | null>(null);
  let width = $state(640);
  let hoveredIndex = $state<number | null>(null);
  const uid = Math.random().toString(36).slice(2, 9);
  const clipAboveId = `nw-clip-above-${uid}`;
  const clipBelowId = `nw-clip-below-${uid}`;

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

  const height = 280;
  const margin = { top: 16, right: 20, bottom: 28, left: 68 };
  const plotWidth = $derived(Math.max(100, width - margin.left - margin.right));
  const plotHeight = height - margin.top - margin.bottom;

  const bounds = $derived(computeNetworthBounds(data, showFxImpact));

  const timestamps = $derived(data.map((d) => dayjs(d.date).valueOf()));
  const minTime = $derived(timestamps.length > 0 ? Math.min(...timestamps) : 0);
  const maxTime = $derived(timestamps.length > 0 ? Math.max(...timestamps) : 1);

  function xPos(index: number): number {
    if (data.length <= 1) return 0;
    const span = maxTime - minTime;
    if (span <= 0) {
      return (index / (data.length - 1)) * plotWidth;
    }
    return ((timestamps[index] - minTime) / span) * plotWidth;
  }

  function yPos(val: number): number {
    const span = bounds.max - bounds.min || 1;
    return plotHeight - ((val - bounds.min) / span) * plotHeight;
  }

  const ticks = $derived(
    [0, 0.25, 0.5, 0.75, 1].map((t) => bounds.min + (bounds.max - bounds.min) * t)
  );

  const xs = $derived(data.map((_, i) => xPos(i)));
  const balanceYs = $derived(data.map((d) => yPos(d.balanceAmount || 0)));
  const investedYs = $derived(data.map((d) => yPos(d.netInvestmentAmount || 0)));
  const exFxYs = $derived(
    data.map((d) => yPos((d.netInvestmentAmount || 0) + getInvestmentReturn(d)))
  );
  const zeroYs = $derived(data.map(() => 0));
  const bottomYs = $derived(data.map(() => plotHeight));

  const balancePath = $derived(buildLinePath(xs, balanceYs));
  const investedPath = $derived(buildLinePath(xs, investedYs));

  // Clip paths matching the old UI: gain fills between Net Investment and Net Worth where Net Worth >= Net Investment;
  // loss fills where Net Worth < Net Investment.
  const clipAbovePath = $derived(buildBandPath(xs, bottomYs, balanceYs));
  const clipBelowPath = $derived(buildBandPath(xs, zeroYs, balanceYs));
  const gainAreaPath = $derived(buildBandPath(xs, zeroYs, investedYs));
  const lossAreaPath = $derived(buildBandPath(xs, bottomYs, investedYs));

  // FX Impact stacked bands matching old UI:
  // 1) Net Investment -> Net Investment + Investment Return
  // 2) Net Investment + Investment Return -> Net Worth
  const investmentReturnBandPath = $derived(
    showFxImpact ? buildBandPath(xs, investedYs, exFxYs) : ""
  );
  const fxImpactBandPath = $derived(showFxImpact ? buildBandPath(xs, exFxYs, balanceYs) : "");

  function formatAxis(val: number): string {
    if (Math.abs(val) >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
    if (Math.abs(val) >= 100000) return `${(val / 100000).toFixed(1)}L`;
    if (Math.abs(val) >= 1000) return `${(val / 1000).toFixed(0)}K`;
    return `${Math.round(val)}`;
  }

  function handlePointerMove(e: MouseEvent) {
    if (data.length === 0 || !container) return;
    const rect = container.getBoundingClientRect();
    const relX = Math.min(plotWidth, Math.max(0, e.clientX - rect.left - margin.left));
    let bestIdx = 0;
    let bestDist = Infinity;
    for (let i = 0; i < xs.length; i++) {
      const dist = Math.abs(xs[i] - relX);
      if (dist < bestDist) {
        bestDist = dist;
        bestIdx = i;
      }
    }
    hoveredIndex = bestIdx;
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
      <defs>
        <clipPath id={clipAboveId}>
          <path d={clipAbovePath} />
        </clipPath>
        <clipPath id={clipBelowId}>
          <path d={clipBelowPath} />
        </clipPath>
      </defs>

      <g transform="translate({margin.left},{margin.top})">
        {#each ticks as tick}
          {@const y = yPos(tick)}
          <line x1="0" x2={plotWidth} y1={y} y2={y} stroke="var(--border)" stroke-dasharray="3 3" />
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

        <path d={gainAreaPath} clip-path="url(#{clipAboveId})" fill="var(--gain)" opacity="0.2" />
        <path d={lossAreaPath} clip-path="url(#{clipBelowId})" fill="var(--loss)" opacity="0.2" />

        {#if showFxImpact}
          <path d={investmentReturnBandPath} fill="var(--accent)" opacity="0.12" />
          <path d={fxImpactBandPath} fill="var(--warning)" opacity="0.15" />
        {/if}

        <path d={investedPath} fill="none" stroke="var(--muted)" stroke-width="1.5" />
        <path d={balancePath} fill="none" stroke="var(--accent)" stroke-width="1.75" />

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
          {@const hx = xs[hoveredIndex]}
          <line x1={hx} x2={hx} y1="0" y2={plotHeight} stroke="var(--border)" />
          <circle cx={hx} cy={investedYs[hoveredIndex]} r="3.5" fill="var(--muted)" />
          <circle cx={hx} cy={balanceYs[hoveredIndex]} r="4" fill="var(--accent)" />
        {/if}
      </g>
    </svg>

    {#if hoveredIndex !== null && data[hoveredIndex]}
      {@const pt = data[hoveredIndex]}
      {@const invRet = getInvestmentReturn(pt)}
      <div
        class="mt-2 px-3 py-2 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] flex flex-wrap items-center justify-between gap-4 text-xs tabular-nums"
      >
        <span class="font-semibold">{dayjs(pt.date).format("DD MMM YYYY")}</span>
        <div class="flex flex-wrap items-center gap-4">
          <span>Net Worth: <strong>{formatMoney(pt.balanceAmount)}</strong></span>
          <span class="text-[var(--muted)]">
            Net Investment: {formatMoney(pt.netInvestmentAmount)}
          </span>
          <span class={pt.gainAmount >= 0 ? "text-[var(--gain)]" : "text-[var(--loss)]"}>
            Gain / Loss: {formatMoney(pt.gainAmount)}
          </span>
          <span class={invRet >= 0 ? "text-[var(--gain)]" : "text-[var(--loss)]"}>
            Investment Return: {formatMoney(invRet)}
          </span>
          {#if showFxImpact}
            <span class="text-[var(--warning)]">
              FX Impact: {formatMoney(pt.fx_impact || 0)}
            </span>
          {/if}
        </div>
      </div>
    {/if}
  {/if}
</div>
