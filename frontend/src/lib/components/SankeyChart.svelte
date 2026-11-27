<script lang="ts">
  import { formatMoney } from "$lib/api";

  export interface SankeyLinkItem {
    source: string;
    target: string;
    value: number;
  }

  let { links }: { links: SankeyLinkItem[] } = $props();

  let container = $state<HTMLDivElement | null>(null);
  let width = $state(680);
  let hoveredIdx = $state<number | null>(null);

  $effect(() => {
    if (!container) return;
    const observer = new ResizeObserver((entries) => {
      const entry = entries[0];
      if (entry) {
        width = Math.max(320, entry.contentRect.width);
      }
    });
    observer.observe(container);
    return () => observer.disconnect();
  });

  const topLinks = $derived(links.filter((l) => l.value > 0).slice(0, 18));

  const layout = $derived.by(() => {
    const sources = new Map<string, number>();
    const targets = new Map<string, number>();

    for (const l of topLinks) {
      sources.set(l.source, (sources.get(l.source) || 0) + l.value);
      targets.set(l.target, (targets.get(l.target) || 0) + l.value);
    }

    const totalSrc = Math.max(1, Array.from(sources.values()).reduce((a, b) => a + b, 0));
    const totalTgt = Math.max(1, Array.from(targets.values()).reduce((a, b) => a + b, 0));
    const height = Math.max(260, Math.max(sources.size, targets.size) * 34);
    const padY = 12;
    const usableH = height - padY * 2;

    const srcNodes = new Map<string, { y: number; h: number; offset: number }>();
    let sy = padY;
    for (const [name, val] of sources.entries()) {
      const h = Math.max(10, (val / totalSrc) * (usableH - (sources.size - 1) * 6));
      srcNodes.set(name, { y: sy, h, offset: 0 });
      sy += h + 6;
    }

    const tgtNodes = new Map<string, { y: number; h: number; offset: number }>();
    let ty = padY;
    for (const [name, val] of targets.entries()) {
      const h = Math.max(10, (val / totalTgt) * (usableH - (targets.size - 1) * 6));
      tgtNodes.set(name, { y: ty, h, offset: 0 });
      ty += h + 6;
    }

    const xLeft = 150;
    const xRight = Math.max(260, width - 170);

    const ribbons = topLinks.map((l) => {
      const s = srcNodes.get(l.source)!;
      const t = tgtNodes.get(l.target)!;
      const sh = Math.max(2, (l.value / (sources.get(l.source) || 1)) * s.h);
      const th = Math.max(2, (l.value / (targets.get(l.target) || 1)) * t.h);
      const y1 = s.y + s.offset + sh / 2;
      const y2 = t.y + t.offset + th / 2;
      s.offset += sh;
      t.offset += th;
      const mx = (xLeft + xRight) / 2;
      return {
        ...l,
        d: `M ${xLeft} ${y1} C ${mx} ${y1}, ${mx} ${y2}, ${xRight} ${y2}`,
        strokeWidth: Math.max(2, Math.min(sh, th))
      };
    });

    return { height, xLeft, xRight, sources, targets, srcNodes, tgtNodes, ribbons };
  });
</script>

<div bind:this={container} class="w-full select-none">
  {#if topLinks.length === 0}
    <div class="h-48 flex items-center justify-center text-sm text-[var(--muted)]">
      No flow links to visualize.
    </div>
  {:else}
    <svg {width} height={layout.height} class="overflow-visible">
      {#each layout.ribbons as r, i}
        <path
          d={r.d}
          fill="none"
          stroke="var(--accent)"
          stroke-width={r.strokeWidth}
          opacity={hoveredIdx === null ? 0.28 : hoveredIdx === i ? 0.72 : 0.1}
          role="presentation"
          onmouseenter={() => (hoveredIdx = i)}
          onmouseleave={() => (hoveredIdx = null)}
        />
      {/each}

      {#each Array.from(layout.srcNodes.entries()) as [name, node]}
        <rect x={layout.xLeft - 8} y={node.y} width="8" height={node.h} rx="2" fill="var(--gain)" />
        <text
          x={layout.xLeft - 14}
          y={node.y + node.h / 2 + 4}
          text-anchor="end"
          font-size="11"
          fill="var(--text)"
        >
          {name.length > 20 ? `${name.slice(0, 19)}…` : name}
        </text>
      {/each}

      {#each Array.from(layout.tgtNodes.entries()) as [name, node]}
        <rect x={layout.xRight} y={node.y} width="8" height={node.h} rx="2" fill="var(--accent)" />
        <text
          x={layout.xRight + 14}
          y={node.y + node.h / 2 + 4}
          text-anchor="start"
          font-size="11"
          fill="var(--text)"
        >
          {name.length > 22 ? `${name.slice(0, 21)}…` : name}
        </text>
      {/each}
    </svg>

    {#if hoveredIdx !== null && layout.ribbons[hoveredIdx]}
      {@const item = layout.ribbons[hoveredIdx]}
      <div
        class="mt-2 px-3 py-1.5 rounded-lg bg-[var(--surface-elevated)] border border-[var(--border)] text-xs flex items-center justify-between"
      >
        <span>{item.source} → <strong>{item.target}</strong></span>
        <span class="tabular-nums font-semibold">{formatMoney(item.value)}</span>
      </div>
    {/if}
  {/if}
</div>
