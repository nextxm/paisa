<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import { generateColorScheme } from "$lib/colors";
  import { formatCurrencyCrude } from "$lib/utils";
  import { rem } from "$lib/utils";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import type { Legend } from "$lib/utils";

  interface EntityTrend {
    key: string;
    series: Record<string, number>;
  }

  interface CompositionPoint {
    month: string;
    [key: string]: string | number;
  }

  let {
    id,
    data = [],
    months = [],
    title = "Composition Over Time"
  }: {
    id: string;
    data: EntityTrend[];
    months: string[];
    title?: string;
  } = $props();

  let legends: Legend[] = $state([]);
  let svgEl: SVGSVGElement = $state();

  function buildCompositionData(): CompositionPoint[] {
    const result: CompositionPoint[] = months.map((month) => ({ month }));

    for (const entity of data) {
      for (const point of result) {
        const monthValue = entity.series[point.month] || 0;
        point[entity.key] = monthValue;
      }
    }

    return result;
  }

  function render() {
    if (typeof document === "undefined" || !svgEl || data.length === 0 || months.length < 2) return;

    const svg = d3.select(svgEl);
    svg.selectAll("*").remove();
    d3.select(svgEl.parentElement).selectAll(".mom-composition-tooltip").remove();

    const compositionData = buildCompositionData();
    const keys = data.map((d) => d.key);

    const margin = {
      top: rem(20),
      right: rem(40),
      bottom: rem(40),
      left: rem(60)
    };

    const width =
      Math.max(svgEl.parentElement?.clientWidth || 600, 400) - margin.left - margin.right;
    const height = 280 - margin.top - margin.bottom;

    const x = d3.scaleBand<string>().domain(months).range([0, width]).padding(0.1);

    // Calculate max total for Y scale
    const maxTotal = Math.max(
      ...compositionData.map((point) => _.sumBy(keys, (key) => Number(point[key] || 0)))
    );

    const y = d3
      .scaleLinear()
      .domain([0, maxTotal * 1.1])
      .range([height, 0]);

    const color = generateColorScheme(keys);

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

    // Stack the data
    const stack = d3
      .stack<CompositionPoint, string>()
      .keys(keys)
      .value((point, key) => Number(point[key] || 0));

    const stackedData = stack(compositionData);

    // Create stacked area generator
    const area = d3
      .area<d3.SeriesPoint<CompositionPoint>>()
      .x((d) => (x(d.data.month) || 0) + x.bandwidth() / 2)
      .y0((d) => y(d[0]))
      .y1((d) => y(d[1]))
      .curve(d3.curveMonotoneX);

    // Draw stacked areas
    root
      .selectAll(".stack-area")
      .data(stackedData)
      .enter()
      .append("path")
      .attr("class", "stack-area")
      .attr("fill", (d) => color(d.key))
      .attr("opacity", 0.75)
      .attr("d", area)
      .on("mouseover", function (event, d) {
        d3.select(this).attr("opacity", 0.95);
        tooltip
          .style("opacity", 1)
          .html(`<strong>${d.key}</strong><br/>` + `Click legend to toggle visibility`)
          .style("left", event.pageX + 10 + "px")
          .style("top", event.pageY - 30 + "px");
      })
      .on("mousemove", (event) => {
        tooltip.style("left", event.pageX + 10 + "px").style("top", event.pageY - 30 + "px");
      })
      .on("mouseout", function () {
        d3.select(this).attr("opacity", 0.75);
        tooltip.style("opacity", 0);
      });

    // Axes
    root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${height})`)
      .call(
        d3.axisBottom(x).tickValues(months.filter((_, i) => i % Math.ceil(months.length / 6) === 0))
      );

    root
      .append("g")
      .attr("class", "axis y")
      .call(d3.axisLeft(y).tickFormat(formatCurrencyCrude as any));

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

    // Tooltip
    const tooltip = d3
      .select(svgEl.parentElement)
      .append("div")
      .attr("class", "mom-composition-tooltip")
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
    legends = keys.map((key) => ({
      label: key,
      color: color(key),
      shape: "square",
      onClick: (legend: Legend) => {
        const path = root
          .selectAll<SVGPathElement, d3.Series<CompositionPoint, string>>(".stack-area")
          .filter((d) => d.key === legend.label);

        const currentOpacity = +path.attr("opacity");
        path.attr("opacity", currentOpacity > 0.1 ? 0.1 : 0.75);
      }
    }));
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

  :global(.grid line) {
    stroke: currentColor;
  }
</style>
