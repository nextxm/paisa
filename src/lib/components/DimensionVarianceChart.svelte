<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import COLORS from "$lib/colors";
  import { formatCurrencyCrude } from "$lib/utils";
  import { rem } from "$lib/utils";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import type { Legend } from "$lib/utils";

  interface VarianceEntry {
    key: string;
    previous: number;
    current: number;
    change: number;
    changePct: number | null;
  }

  let {
    id,
    data = [],
    title = "Top Movers"
  }: {
    id: string;
    data: VarianceEntry[];
    title?: string;
  } = $props();

  let legends: Legend[] = $state([]);
  let svgEl: SVGSVGElement = $state();

  function render() {
    if (typeof document === "undefined" || !svgEl || data.length === 0) return;

    const svg = d3.select(svgEl);
    svg.selectAll("*").remove();

    if (data.length < 1) return;

    const margin = {
      top: rem(20),
      right: rem(30),
      bottom: rem(40),
      left: rem(100)
    };

    const width =
      Math.max(svgEl.parentElement?.clientWidth || 400, 300) - margin.left - margin.right;
    const height = Math.min(data.length * 35 + 40, 400) - margin.top - margin.bottom;

    // Prepare data: sort by absolute change
    const sortedData = _.orderBy(
      data,
      [(d) => Math.abs(d.change), (d) => d.current],
      ["desc", "desc"]
    );

    const yDomain = sortedData.map((d) => d.key);
    const xMin = Math.min(0, ...sortedData.map((d) => Math.min(d.previous, d.current)));
    const xMax = Math.max(...sortedData.map((d) => Math.max(d.previous, d.current))) * 1.1;

    const y = d3.scaleBand<string>().domain(yDomain).range([0, height]).padding(0.3);

    const x = d3.scaleLinear().domain([xMin, xMax]).range([0, width]);

    const root = svg
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    // Zero line
    root
      .append("line")
      .attr("x1", x(0))
      .attr("x2", x(0))
      .attr("y1", 0)
      .attr("y2", height)
      .attr("stroke", "#ccc")
      .attr("stroke-width", 1)
      .attr("stroke-dasharray", "2,2");

    // Previous bars (background, lighter)
    root
      .selectAll(".bar-previous")
      .data(sortedData)
      .enter()
      .append("rect")
      .attr("class", "bar-previous")
      .attr("x", (d) => Math.min(x(0), x(d.previous)))
      .attr("y", (d) => y(d.key) || 0)
      .attr("width", (d) => Math.abs(x(d.previous) - x(0)))
      .attr("height", y.bandwidth())
      .attr("fill", COLORS.primary)
      .attr("opacity", 0.3)
      .on("mouseover", (event, d) => {
        tooltip
          .style("opacity", 1)
          .html(`<strong>${d.key}</strong><br/>Previous: ${formatCurrencyCrude(d.previous)}`)
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 30 + "px");
      })
      .on("mousemove", (event) => {
        tooltip.style("left", event.pageX + 10 + "px").style("top", event.pageY - 30 + "px");
      })
      .on("mouseout", () => {
        tooltip.style("opacity", 0);
      });

    // Current bars (foreground, vibrant)
    root
      .selectAll(".bar-current")
      .data(sortedData)
      .enter()
      .append("rect")
      .attr("class", "bar-current")
      .attr("x", (d) => Math.min(x(0), x(d.current)))
      .attr("y", (d) => y(d.key) || 0)
      .attr("width", (d) => Math.abs(x(d.current) - x(0)))
      .attr("height", y.bandwidth())
      .attr("fill", (d) =>
        d.change > 0 ? COLORS.danger : d.change < 0 ? COLORS.success : COLORS.primary
      )
      .attr("opacity", 0.8)
      .on("mouseover", (event, d) => {
        const changeTxt = d.changePct !== null ? ` (${(d.changePct * 100).toFixed(1)}%)` : "";
        tooltip
          .style("opacity", 1)
          .html(
            `<strong>${d.key}</strong><br/>` +
              `Current: ${formatCurrencyCrude(d.current)}<br/>` +
              `Change: ${d.change > 0 ? "+" : ""}${formatCurrencyCrude(d.change)}${changeTxt}`
          )
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 30 + "px");
      })
      .on("mousemove", (event) => {
        tooltip.style("left", event.pageX + 10 + "px").style("top", event.pageY - 30 + "px");
      })
      .on("mouseout", () => {
        tooltip.style("opacity", 0);
      });

    // Value labels on bars
    root
      .selectAll(".label-current")
      .data(sortedData)
      .enter()
      .append("text")
      .attr("class", "label-current")
      .attr("x", (d) => x(d.current) + (d.current > 0 ? 5 : -5))
      .attr("y", (d) => (y(d.key) || 0) + y.bandwidth() / 2 + 4)
      .attr("text-anchor", (d) => (d.current > 0 ? "start" : "end"))
      .attr("font-size", "0.75rem")
      .attr("fill", "var(--bulma-text)")
      .attr("font-weight", "500")
      .text((d) => formatCurrencyCrude(d.current));

    // Axes
    root
      .append("g")
      .attr("class", "axis y")
      .call(d3.axisLeft(y).tickSize(0))
      .selectAll("text")
      .style("font-size", "0.875rem");

    root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${height})`)
      .call(d3.axisBottom(x).tickFormat(formatCurrencyCrude as any))
      .selectAll("text")
      .style("font-size", "0.75rem");

    // Tooltip
    const tooltip = d3
      .select(svgEl.parentElement)
      .append("div")
      .style("position", "absolute")
      .style("background", "rgba(0,0,0,0.8)")
      .style("color", "white")
      .style("padding", "6px 10px")
      .style("border-radius", "4px")
      .style("font-size", "0.875rem")
      .style("pointer-events", "none")
      .style("opacity", 0)
      .style("z-index", 999);

    // Update legends
    legends = [
      { label: "Previous Month", color: COLORS.primary, shape: "square" },
      { label: "Current Month", color: COLORS.danger, shape: "square" }
    ];
  }

  onMount(() => {
    render();
  });

  $effect(() => {
    render();
  });
</script>

<div style="position: relative;">
  <h3 class="heading is-size-7 mb-3">{title}</h3>
  <LegendCard {legends} clazz="mb-3" />
  <svg bind:this={svgEl} {id} style="overflow: visible; display: block; width: 100%;" />
</div>

<style lang="scss">
  :global(.axis) {
    font-size: 0.875rem;
  }

  :global(.axis text) {
    font-size: 0.75rem !important;
  }
</style>
