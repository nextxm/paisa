<script lang="ts">
  import dayjs from "dayjs";
  import { formatMoney } from "$lib/api";

  export interface ProjectionPoint {
    date: string;
    balance: number;
  }

  export interface FanBand {
    p10: ProjectionPoint[];
    p25: ProjectionPoint[];
    p50: ProjectionPoint[];
    p75: ProjectionPoint[];
    p90: ProjectionPoint[];
  }

  let {
    mode = "simple",
    conservative = [],
    expected = [],
    optimistic = [],
    fan = null,
    targetCorpus = 0
  }: {
    mode?: "simple" | "montecarlo";
    conservative?: ProjectionPoint[];
    expected?: ProjectionPoint[];
    optimistic?: ProjectionPoint[];
    fan?: FanBand | null;
    targetCorpus?: number;
  } = $props();

  let container = $state<HTMLDivElement | null>(null);
  let width = $state(640);

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
  const margin = { top: 20, right: 20, bottom: 28, left: 68 };
  const plotWidth = $derived(Math.max(100, width - margin.left - margin.right));
  const plotHeight = height - margin.top - margin.bottom;

  const series = $derived.by(() => {
    if (mode === "montecarlo" && fan) {
      return [...fan.p10, ...fan.p50, ...fan.p90];
    }
    return [...conservative, ...expected, ...optimistic];
  });

  const bounds = $derived.by(() => {
    let min = 0;
    let max = targetCorpus > 0 ? targetCorpus : 1000;
    for (const pt of series) {
      min = Math.min(min, pt.balance);
      max = Math.max(max, pt.balance);
    }
    return { min, max: max * 1.06 };
  });

  const primaryLine = $derived(mode === "montecarlo" && fan ? fan.p50 : expected);

  function xPos(index: number, len: number): number {
    if (len <= 1) return 0;
    return (index / (len - 1)) * plotWidth;
  }

  function yPos(val: number): number {
    const span = bounds.max - bounds.min || 1;
    return plotHeight - ((val - bounds.min) / span) * plotHeight;
  }

  function linePath(pts: ProjectionPoint[]): string {
    return pts
      .map(
        (p, i) => `${i === 0 ? "M" : "L"}${xPos(i, pts.length).toFixed(1)},${yPos(p.balance).toFixed(1)}`
      )
      .join(" ");
  }

  function bandPath(low: ProjectionPoint[], high: ProjectionPoint[]): string {
    if (low.length === 0 || high.length === 0) return "";
    const top = high
      .map(
        (p, i) =>
          `${i === 0 ? "M" : "L"}${xPos(i, high.length).toFixed(1)},${yPos(p.balance).toFixed(1)}`
      )
      .join(" ");
    const bottom = [...low]
      .reverse()
      .map(
        (p, i) =>
          `L${xPos(low.length - 1 - i, low.length).toFixed(1)},${yPos(p.balance).toFixed(1)}`
      )
      .join(" ");
    return `${top} ${bottom} Z`;
  }

  const ticks = $derived(
    [0, 0.25, 0.5, 0.75, 1].map((t) => bounds.min + (bounds.max - bounds.min) * t)
  );

  function formatAxis(val: number): string {
    if (Math.abs(val) >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
    if (Math.abs(val) >= 100000) return `${(val / 100000).toFixed(1)}L`;
    if (Math.abs(val) >= 1000) return `${(val / 1000).toFixed(0)}K`;
    return `${Math.round(val)}`;
  }
</script>

<div bind:this={container} class="relative w-full select-none">
  {#if primaryLine.length === 0}
    <div class="h-64 flex items-center justify-center text-sm text-[var(--muted)]">
      No projection points to display.
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

        {#if targetCorpus > 0 && targetCorpus <= bounds.max}
          {@const ty = yPos(targetCorpus)}
          <line
            x1="0"
            x2={plotWidth}
            y1={ty}
            y2={ty}
            stroke="var(--warning)"
            stroke-width="1.5"
            stroke-dasharray="6 4"
          />
          <text x={plotWidth} y={ty - 6} text-anchor="end" font-size="11" fill="var(--warning)">
            FIRE Target: {formatMoney(targetCorpus)}
          </text>
        {/if}

        {#if mode === "montecarlo" && fan}
          <path d={bandPath(fan.p10, fan.p90)} fill="var(--accent)" opacity="0.08" />
          <path d={bandPath(fan.p25, fan.p75)} fill="var(--accent)" opacity="0.18" />
          <path d={linePath(fan.p50)} fill="none" stroke="var(--accent)" stroke-width="2.5" />
        {:else}
          <path
            d={linePath(conservative)}
            fill="none"
            stroke="var(--loss)"
            stroke-width="1.75"
            opacity="0.75"
          />
          <path
            d={linePath(optimistic)}
            fill="none"
            stroke="var(--gain)"
            stroke-width="1.75"
            opacity="0.75"
          />
          <path d={linePath(expected)} fill="none" stroke="var(--accent)" stroke-width="2.5" />
        {/if}

        {#each primaryLine as pt, i}
          {#if i === 0 || i === primaryLine.length - 1 || (primaryLine.length > 6 && i % Math.ceil(primaryLine.length / 6) === 0)}
            <text
              x={xPos(i, primaryLine.length)}
              y={plotHeight + 18}
              text-anchor="middle"
              font-size="11"
              fill="var(--muted)"
            >
              {dayjs(pt.date).format("YYYY")}
            </text>
          {/if}
        {/each}
      </g>
    </svg>
  {/if}
</div>
