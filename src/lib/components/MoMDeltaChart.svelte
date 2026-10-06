<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import COLORS from "$lib/colors";
  import { formatCurrencyCrude, formatPercentage } from "$lib/utils";
  import { rem } from "$lib/utils";

  interface DeltaPoint {
    month: string;
    label?: string;
    total: number;
    change: number | null;
    changePct: number | null;
  }

  let {
    id,
    data = []
  }: {
    id: string;
    data: DeltaPoint[];
  } = $props();

  let svgEl: SVGSVGElement = $state();

  // Only points that have a change value (skip the first month)
  let deltaPoints = $derived(
    data.filter((d) => d.change !== null) as (DeltaPoint & {
      change: number;
      changePct: number | null;
    })[]
  );

  function render() {
    if (typeof document === "undefined" || !svgEl || deltaPoints.length === 0) return;

    const svg = d3.select(svgEl);
    svg.selectAll("*").remove();
    d3.select(svgEl.parentElement).selectAll(".mom-delta-tooltip").remove();

    const margin = { top: rem(20), right: rem(20), bottom: rem(40), left: rem(60) };
    const width =
      Math.max(svgEl.parentElement?.clientWidth || 500, 300) - margin.left - margin.right;
    const height = 260 - margin.top - margin.bottom;

    const months = deltaPoints.map((d) => d.label ?? d.month);
    const maxAbs = Math.max(...deltaPoints.map((d) => Math.abs(d.change)), 1);

    const x = d3.scaleBand<string>().domain(months).range([0, width]).padding(0.25);
    const y = d3
      .scaleLinear()
      .domain([-maxAbs * 1.1, maxAbs * 1.1])
      .range([height, 0]);

    const root = svg
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    // Horizontal zero line
    root
      .append("line")
      .attr("x1", 0)
      .attr("x2", width)
      .attr("y1", y(0))
      .attr("y2", y(0))
      .attr("stroke", "currentColor")
      .attr("stroke-opacity", 0.3)
      .attr("stroke-width", 1.5);

    // Subtle grid lines at round values
    root
      .append("g")
      .attr("class", "grid")
      .attr("opacity", 0.08)
      .call(
        d3
          .axisLeft(y)
          .tickSize(-width)
          .tickFormat(() => "")
      );

    // Tooltip div
    const tooltip = d3
      .select(svgEl.parentElement)
      .append("div")
      .attr("class", "mom-delta-tooltip")
      .style("position", "absolute")
      .style("background", "rgba(0,0,0,0.82)")
      .style("color", "white")
      .style("padding", "6px 10px")
      .style("border-radius", "4px")
      .style("font-size", "0.8rem")
      .style("pointer-events", "none")
      .style("opacity", 0)
      .style("z-index", 999);

    // Bars
    root
      .selectAll(".delta-bar")
      .data(deltaPoints)
      .enter()
      .append("rect")
      .attr("class", "delta-bar")
      .attr("x", (d) => x(d.label ?? d.month) ?? 0)
      .attr("y", (d) => (d.change >= 0 ? y(d.change) : y(0)))
      .attr("width", x.bandwidth())
      .attr("height", (d) => Math.abs(y(d.change) - y(0)))
      .attr("fill", (d) => (d.change > 0 ? COLORS.danger : COLORS.success))
      .attr("opacity", 0.8)
      .attr("rx", 2)
      .on("mouseover", (event, d) => {
        d3.select(event.currentTarget).attr("opacity", 1);
        const pct =
          d.changePct !== null
            ? ` (${d.changePct >= 0 ? "+" : ""}${formatPercentage(d.changePct)})`
            : "";
        tooltip
          .style("opacity", 1)
          .html(
            `<strong>${d.label ?? d.month}</strong><br/>` +
              `Change: ${d.change >= 0 ? "+" : ""}${formatCurrencyCrude(d.change)}${pct}<br/>` +
              `Total: ${formatCurrencyCrude(d.total)}`
          )
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 38 + "px");
      })
      .on("mousemove", (event) => {
        tooltip.style("left", event.pageX + 10 + "px").style("top", event.pageY - 38 + "px");
      })
      .on("mouseout", (event) => {
        d3.select(event.currentTarget).attr("opacity", 0.8);
        tooltip.style("opacity", 0);
      });

    // Tick labels for significant bars: show % on bars taller than 1/5 of max
    const labelThreshold = (maxAbs * 1.1) / 5;
    root
      .selectAll(".delta-label")
      .data(deltaPoints.filter((d) => Math.abs(d.change) > labelThreshold))
      .enter()
      .append("text")
      .attr("class", "delta-label")
      .attr("x", (d) => (x(d.label ?? d.month) ?? 0) + x.bandwidth() / 2)
      .attr("y", (d) => (d.change >= 0 ? y(d.change) - 4 : y(d.change) - y(0) + y(0) + 13))
      .attr("text-anchor", "middle")
      .attr("font-size", "0.65rem")
      .attr("fill", (d) => (d.change > 0 ? COLORS.danger : COLORS.success))
      .attr("font-weight", "600")
      .text((d) =>
        d.changePct !== null
          ? `${d.changePct >= 0 ? "+" : ""}${Math.round(d.changePct * 100)}%`
          : ""
      );

    // X axis — subsample if many months
    const tickEvery = Math.ceil(months.length / 8);
    root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${y(0)})`)
      .call(d3.axisBottom(x).tickValues(months.filter((_, i) => i % tickEvery === 0)));

    // Y axis
    root
      .append("g")
      .attr("class", "axis y")
      .call(
        d3
          .axisLeft(y)
          .ticks(5)
          .tickFormat(formatCurrencyCrude as any)
      );
  }

  onMount(() => render());
  $effect(() => render());
</script>

<h3 class="heading is-size-7 mb-1">Month-over-Month Change</h3>
<p class="is-size-8 has-text-grey mb-3">
  How much more (🔴) or less (🟢) was spent vs the previous month
</p>

<div style="position: relative;">
  <svg bind:this={svgEl} {id} style="overflow: visible; display: block; width: 100%;" />
</div>

<style lang="scss">
  :global(.axis text) {
    font-size: 0.72rem;
  }
</style>
