<script lang="ts">
  import * as d3 from "d3";
  import { sankeyCircular, sankeyJustify } from "d3-sankey-circular";
  import _ from "lodash";
  import { onMount, onDestroy } from "svelte";
  import { willClearTippy } from "../../store";
  import COLORS from "$lib/colors";
  import { formatCurrency, rem, type SankeyLink, type SankeyNode } from "$lib/utils";
  import { tooltip } from "$lib/utils";
  import { iconify } from "$lib/icon";

  // Props
  export let nodes: SankeyNode[] = [];
  export let links: SankeyLink[] = [];
  export let height: number = 500;
  export let loading: boolean = false;
  export let emptyMessage: string = "No flow data available for this period.";
  export let onLinkClick: ((link: SankeyLink) => void) | undefined = undefined;

  let svgEl: SVGSVGElement;
  let containerEl: HTMLDivElement;
  let resizeObserver: ResizeObserver;
  let containerWidth = 0;

  // Color mapping by account kind
  const kindColors: Record<string, string> = {
    income: COLORS.income,
    asset: COLORS.assets,
    liability: COLORS.liabilities,
    expense: COLORS.expenses,
    equity: COLORS.equity,
    other: COLORS.neutral
  };

  function nodeColor(kind: string): string {
    return kindColors[kind] ?? COLORS.neutral;
  }

  function draw() {
    if (!svgEl || !containerEl) return;

    const svgSel = d3.select(svgEl);
    svgSel.selectAll("*").remove();
    willClearTippy.update((n) => n + 1);

    if (_.isEmpty(nodes) || _.isEmpty(links)) return;

    const margin = { top: rem(10), right: rem(120), bottom: rem(10), left: rem(120) };
    const w = containerWidth - margin.left - margin.right;
    const h = height - margin.top - margin.bottom;

    if (w <= 0 || h <= 0) return;

    svgSel.attr("width", containerWidth).attr("height", height);

    const g = svgSel.append("g").attr("transform", `translate(${margin.left},${margin.top})`);

    // Build graph for d3-sankey-circular
    const graph = {
      nodes: nodes.map((n) => ({ ...n })),
      links: links.map((l) => ({ ...l }))
    };

    const sankey = sankeyCircular()
      .nodeWidth(rem(15))
      .nodePaddingRatio(0.6)
      .size([w, h])
      .nodeId((d: any) => d.id)
      .nodeAlign(sankeyJustify)
      .iterations(32)
      .circularLinkGap(2);

    let sankeyData: any;
    try {
      sankeyData = sankey(graph);
    } catch (_err) {
      return;
    }

    const sankeyNodes: any[] = sankeyData.nodes;
    const sankeyLinks: any[] = sankeyData.links;

    // ── Links ────────────────────────────────────────────────────────────────
    const linkG = g
      .append("g")
      .attr("class", "sankey-links")
      .attr("fill", "none")
      .attr("stroke-opacity", 0.35);

    const linkSel = linkG
      .selectAll("path")
      .data(sankeyLinks)
      .enter()
      .append("path")
      .attr("class", "sankey-link")
      .attr("d", (d: any) => d.path)
      .style("stroke-width", (d: any) => Math.max(1, d.width))
      .style("stroke", (d: any) => nodeColor(d.target.kind))
      .style("opacity", 0.5)
      .attr("tabindex", "0")
      .attr("role", "img")
      .attr("aria-label", (d: any) => `Flow from ${d.source.name} to ${d.target.name}: ${d.value}`)
      .attr("data-tippy-content", (d: any) =>
        tooltip([
          ["Source", iconify(d.source.name)],
          ["Target", iconify(d.target.name)],
          ["Amount", [formatCurrency(d.value), "has-text-right has-text-weight-bold"]],
          ["Transactions", [String(d.txnCount), "has-text-right"]]
        ])
      );

    if (onLinkClick) {
      linkSel
        .style("cursor", "pointer")
        .on("click", (_event: MouseEvent, d: any) => {
          onLinkClick({
            source: d.source.id,
            target: d.target.id,
            value: d.value,
            txnCount: d.txnCount
          });
        })
        .on("keydown", (event: KeyboardEvent, d: any) => {
          if (event.key === "Enter" || event.key === " ") {
            event.preventDefault();
            onLinkClick({
              source: d.source.id,
              target: d.target.id,
              value: d.value,
              txnCount: d.txnCount
            });
          }
        });
    }

    // ── Hover: highlight neighbours ──────────────────────────────────────────
    function highlightNeighbors(nodeId: string | null) {
      if (nodeId == null) {
        linkSel.style("opacity", 0.5);
        nodeSel.style("opacity", 1);
        labelSel.style("opacity", 1);
        return;
      }
      linkSel.style("opacity", (d: any) =>
        d.source.id === nodeId || d.target.id === nodeId ? 0.85 : 0.1
      );
      nodeSel.style("opacity", (d: any) => {
        if (d.id === nodeId) return 1;
        const linked = sankeyLinks.some(
          (l: any) =>
            (l.source.id === nodeId && l.target.id === d.id) ||
            (l.target.id === nodeId && l.source.id === d.id)
        );
        return linked ? 1 : 0.25;
      });
      labelSel.style("opacity", (d: any) => {
        if (d.id === nodeId) return 1;
        const linked = sankeyLinks.some(
          (l: any) =>
            (l.source.id === nodeId && l.target.id === d.id) ||
            (l.target.id === nodeId && l.source.id === d.id)
        );
        return linked ? 1 : 0.25;
      });
    }

    // ── Nodes ────────────────────────────────────────────────────────────────
    const nodeGroup = g
      .append("g")
      .attr("class", "sankey-nodes")
      .selectAll("g")
      .data(sankeyNodes)
      .enter()
      .append("g");

    const nodeSel = nodeGroup
      .append("rect")
      .attr("x", (d: any) => d.x0)
      .attr("y", (d: any) => d.y0)
      .attr("height", (d: any) => Math.max(1, d.y1 - d.y0))
      .attr("width", (d: any) => d.x1 - d.x0)
      .attr("rx", 2)
      .attr("ry", 2)
      .style("fill", (d: any) => nodeColor(d.kind))
      .style("cursor", "default")
      .attr("tabindex", "0")
      .attr("role", "img")
      .attr("aria-label", (d: any) => `${d.name}: ${formatCurrency(d.value)}`)
      .attr("data-tippy-content", (d: any) =>
        tooltip([
          ["Account", iconify(d.name)],
          ["Total", [formatCurrency(d.value), "has-text-right has-text-weight-bold"]]
        ])
      )
      .on("mouseenter", (_event: MouseEvent, d: any) => highlightNeighbors(d.id))
      .on("mouseleave", () => highlightNeighbors(null))
      .on("focus", (_event: FocusEvent, d: any) => highlightNeighbors(d.id))
      .on("blur", () => highlightNeighbors(null));

    // ── Labels ───────────────────────────────────────────────────────────────
    const LABEL_MAX_CHARS = 20;

    function truncateLabel(name: string) {
      const parts = name.split(":");
      const short = parts[parts.length - 1];
      return short.length > LABEL_MAX_CHARS ? short.slice(0, LABEL_MAX_CHARS) + "…" : short;
    }

    const labelSel = nodeGroup
      .append("text")
      .attr("x", (d: any) => {
        if (_.isEmpty(d.sourceLinks)) return d.x0 - 6;
        if (_.isEmpty(d.targetLinks)) return d.x1 + 6;
        return (d.x0 + d.x1) / 2;
      })
      .attr("y", (d: any) => (d.y0 + d.y1) / 2)
      .attr("dy", "0.35em")
      .attr("text-anchor", (d: any) => {
        if (_.isEmpty(d.sourceLinks)) return "end";
        if (_.isEmpty(d.targetLinks)) return "start";
        return "middle";
      })
      .classed("svg-text-grey-dark", true)
      .style("font-size", "0.75rem")
      .style("pointer-events", "none")
      .attr("data-tippy-content", (d: any) =>
        d.name.length > LABEL_MAX_CHARS ? tooltip([["Account", iconify(d.name)]]) : null
      )
      .text((d: any) => truncateLabel(d.name));
  }

  $: if (svgEl && containerWidth > 0) {
    draw();
  }

  // Re-draw whenever props change
  $: nodes, links, height, draw();

  onMount(() => {
    resizeObserver = new ResizeObserver((entries) => {
      containerWidth = entries[0].contentRect.width;
    });
    if (containerEl) {
      resizeObserver.observe(containerEl);
      containerWidth = containerEl.clientWidth;
    }
  });

  onDestroy(() => {
    resizeObserver?.disconnect();
  });
</script>

<div bind:this={containerEl} style="width: 100%; position: relative;">
  {#if loading}
    <div class="sankey-skeleton" aria-busy="true" aria-label="Loading Sankey diagram">
      {#each Array(5) as _}
        <div class="skeleton-bar" />
      {/each}
    </div>
  {:else if _.isEmpty(nodes) || _.isEmpty(links)}
    <div class="has-text-centered p-6 has-text-grey">
      <span class="icon is-large"><i class="fas fa-chart-bar fa-2x" /></span>
      <p class="mt-2">{emptyMessage}</p>
    </div>
  {:else}
    <svg
      bind:this={svgEl}
      role="img"
      aria-label="Sankey flow diagram"
      style="width: 100%; display: block;"
    />
  {/if}
</div>

<style lang="scss">
  .sankey-skeleton {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 24px;
  }

  .skeleton-bar {
    height: 20px;
    border-radius: 4px;
    background: linear-gradient(90deg, #e0e0e0 25%, #f0f0f0 50%, #e0e0e0 75%);
    background-size: 200% 100%;
    animation: shimmer 1.4s infinite;

    &:nth-child(1) {
      width: 80%;
    }
    &:nth-child(2) {
      width: 60%;
    }
    &:nth-child(3) {
      width: 90%;
    }
    &:nth-child(4) {
      width: 50%;
    }
    &:nth-child(5) {
      width: 70%;
    }
  }

  @keyframes shimmer {
    0% {
      background-position: 200% 0;
    }
    100% {
      background-position: -200% 0;
    }
  }
</style>
