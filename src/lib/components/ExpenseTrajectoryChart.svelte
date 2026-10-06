<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import COLORS from "$lib/colors";
  import { formatCurrencyCrude } from "$lib/utils";
  import { rem } from "$lib/utils";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import type { Legend } from "$lib/utils";

  interface MonthlyPoint {
    month: string;
    label?: string;
    total: number;
    movingAverage3: number;
    changePct: number | null;
  }

  let {
    id,
    data = []
  }: {
    id: string;
    data: MonthlyPoint[];
  } = $props();

  let legends: Legend[] = $state([]);
  let svgEl: SVGSVGElement = $state();

  function render() {
    if (typeof document === "undefined" || !svgEl || data.length === 0) return;

    const svg = d3.select(svgEl);
    svg.selectAll("*").remove();
    d3.select(svgEl.parentElement).selectAll(".mom-trajectory-tooltip").remove();

    if (data.length < 2) return;

    const margin = {
      top: rem(30),
      right: rem(40),
      bottom: rem(40),
      left: rem(60)
    };

    const width =
      Math.max(svgEl.parentElement?.clientWidth || 800, 600) - margin.left - margin.right;
    const height = 350 - margin.top - margin.bottom;

    const xDomain = data.map((d) => d.label ?? d.month);
    const minSeriesValue = Math.min(...data.map((d) => Math.min(d.total, d.movingAverage3)), 0);
    const maxSeriesValue = Math.max(...data.map((d) => Math.max(d.total, d.movingAverage3)), 0);

    // Symmetric-ish padding around real min/max while always keeping zero in-domain.
    const valueSpan = Math.max(maxSeriesValue - minSeriesValue, 1);
    const yDomain = [minSeriesValue - valueSpan * 0.1, maxSeriesValue + valueSpan * 0.1];

    const x = d3.scaleBand<string>().domain(xDomain).range([0, width]).padding(0.2);

    const y = d3.scaleLinear().domain(yDomain).nice().range([height, 0]).clamp(true);

    const root = svg
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    // Add grid lines
    root
      .append("g")
      .attr("class", "grid")
      .attr("opacity", 0.1)
      .call(
        d3
          .axisLeft(y)
          .tickSize(-width)
          .tickFormat(() => "")
      );

    // Area for actual expense (subtle background)
    const areaGenerator = d3
      .area<MonthlyPoint>()
      .x((d) => (x(d.label ?? d.month) || 0) + x.bandwidth() / 2)
      .y0(y(0))
      .y1((d) => y(d.total));

    root
      .append("path")
      .datum(data)
      .attr("fill", COLORS.expenses)
      .attr("opacity", 0.1)
      .attr("d", areaGenerator);

    // Zero baseline for mixed positive/negative windows.
    root
      .append("line")
      .attr("x1", 0)
      .attr("x2", width)
      .attr("y1", y(0))
      .attr("y2", y(0))
      .attr("stroke", "currentColor")
      .attr("stroke-opacity", 0.2)
      .attr("stroke-width", 1);

    // Line for actual expenses
    const lineGenerator = d3
      .line<MonthlyPoint>()
      .x((d) => (x(d.label ?? d.month) || 0) + x.bandwidth() / 2)
      .y((d) => y(d.total))
      .curve(d3.curveMonotoneX);

    root
      .append("path")
      .datum(data)
      .attr("fill", "none")
      .attr("stroke", COLORS.expenses)
      .attr("stroke-width", 2.5)
      .attr("d", lineGenerator);

    // Line for 3-month moving average
    const maLineGenerator = d3
      .line<MonthlyPoint>()
      .x((d) => (x(d.label ?? d.month) || 0) + x.bandwidth() / 2)
      .y((d) => y(d.movingAverage3))
      .curve(d3.curveMonotoneX);

    root
      .append("path")
      .datum(data)
      .attr("fill", "none")
      .attr("stroke", COLORS.primary)
      .attr("stroke-width", 2)
      .attr("stroke-dasharray", "5,5")
      .attr("d", maLineGenerator);

    // Axes
    root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${height})`)
      .call(
        d3
          .axisBottom(x)
          .tickValues(x.domain().filter((_, i) => i % Math.ceil(data.length / 6) === 0))
      );

    root.append("g").attr("class", "axis y").call(d3.axisLeft(y).tickFormat(formatCurrencyCrude));

    // Y-axis label
    svg
      .append("text")
      .attr("transform", "rotate(-90)")
      .attr("y", 0 - margin.left + rem(15))
      .attr("x", 0 - height / 2 - margin.top)
      .attr("dy", "1em")
      .style("text-anchor", "middle")
      .style("font-size", "0.875rem")
      .style("fill", "var(--bulma-text)")
      .text("Amount");

    // Interactive dots and tooltips
    const tooltip = d3
      .select(svgEl.parentElement)
      .append("div")
      .attr("class", "mom-trajectory-tooltip")
      .style("position", "absolute")
      .style("background", "rgba(0,0,0,0.8)")
      .style("color", "white")
      .style("padding", "6px 10px")
      .style("border-radius", "4px")
      .style("font-size", "0.875rem")
      .style("pointer-events", "none")
      .style("opacity", 0)
      .style("z-index", 999);

    root
      .selectAll(".dot-actual")
      .data(data)
      .enter()
      .append("circle")
      .attr("class", "dot-actual")
      .attr("cx", (d) => (x(d.label ?? d.month) || 0) + x.bandwidth() / 2)
      .attr("cy", (d) => y(d.total))
      .attr("r", 3)
      .attr("fill", COLORS.expenses)
      .attr("opacity", 0.7)
      .on("mouseover", (event, d) => {
        const changeTxt = d.changePct !== null ? ` (${(d.changePct * 100).toFixed(1)}%)` : "";
        tooltip
          .style("opacity", 1)
          .html(
            `<strong>${d.label ?? d.month}</strong><br/>` +
              `Expense: ${formatCurrencyCrude(d.total)}<br/>` +
              `3M Avg: ${formatCurrencyCrude(d.movingAverage3)}${changeTxt}`
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

    // Update legends
    legends = [
      { label: "Actual Expense", color: COLORS.expenses, shape: "line" },
      { label: "3-Month Moving Avg", color: COLORS.primary, shape: "line" }
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
  <LegendCard {legends} clazz="mb-3" />
  <svg bind:this={svgEl} {id} style="overflow: visible; display: block; width: 100%;" />
</div>

<style lang="scss">
  :global(.axis) {
    font-size: 0.875rem;
  }

  :global(.grid line) {
    stroke: currentColor;
  }
</style>
