<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import _ from "lodash";
  import { generateColorScheme } from "$lib/colors";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import { buildMonthlyComparisonPoints, orderedYears, type YoYChartType } from "$lib/yoy_utils";
  import {
    formatCurrency,
    formatCurrencyCrude,
    tooltip,
    type Legend,
    type YoYSeries
  } from "$lib/utils";

  let {
    id,
    series,
    chartType = "line"
  }: {
    id: string;
    series: Record<string, YoYSeries>;
    chartType?: YoYChartType;
  } = $props();

  let legends: Legend[] = $state([]);
  let chartContainer: HTMLDivElement = $state();
  let compactMode = $state(false);

  function render() {
    if (typeof document === "undefined") return;
    const years = orderedYears(series);
    const points = buildMonthlyComparisonPoints(series);
    const svg = d3.select(`#${id}`);
    if (!svg.node()) return;

    svg.selectAll("*").remove();
    if (_.isEmpty(years)) return;

    const containerWidth =
      chartContainer?.clientWidth || document.getElementById(id)?.parentElement?.clientWidth || 800;
    compactMode = containerWidth < 480;

    const margin = compactMode
      ? { top: 16, right: 12, bottom: 36, left: 38 }
      : { top: 20, right: 30, bottom: 45, left: 50 };
    const chartHeight = compactMode ? 260 : 320;
    const width = Math.max(containerWidth - margin.left - margin.right, 120);
    const height = chartHeight - margin.top - margin.bottom;
    const root = svg
      .attr("height", chartHeight)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    const x = d3
      .scaleBand<string>()
      .domain(points.map((point) => point.month as string))
      .range([0, width])
      .padding(0.2);

    const y = d3
      .scaleLinear()
      .domain([
        0,
        d3.max(points, (point) => d3.max(years, (year) => Number(point[year] || 0)) || 0) || 0
      ])
      .nice()
      .range([height, 0]);

    const color = generateColorScheme(years);
    legends = years.map((year) => ({
      label: year,
      color: color(year),
      shape: chartType === "line" ? "line" : "square"
    }));

    const xAxis = root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${height})`)
      .call(d3.axisBottom(x));
    const yAxis = root
      .append("g")
      .attr("class", "axis y")
      .call(
        d3
          .axisLeft(y)
          .ticks(compactMode ? 4 : 6)
          .tickFormat(formatCurrencyCrude)
      );

    if (compactMode) {
      xAxis.selectAll("text").style("font-size", "10px");
      yAxis.selectAll("text").style("font-size", "10px");
    }

    if (chartType === "line") {
      for (const year of years) {
        const line = d3
          .line<Record<string, string | number>>()
          .x((point) => (x(point.month as string) || 0) + x.bandwidth() / 2)
          .y((point) => y(Number(point[year] || 0)));

        root
          .append("path")
          .datum(points)
          .attr("fill", "none")
          .attr("stroke", color(year))
          .attr("stroke-width", compactMode ? 1.5 : 2)
          .attr("d", line);

        root
          .selectAll(`.point-${year}`)
          .data(points)
          .join("circle")
          .attr("class", `point-${year}`)
          .attr("cx", (point) => (x(point.month as string) || 0) + x.bandwidth() / 2)
          .attr("cy", (point) => y(Number(point[year] || 0)))
          .attr("r", compactMode ? 2 : 3)
          .attr("fill", color(year));
      }
    } else {
      const xInner = d3.scaleBand().domain(years).range([0, x.bandwidth()]).padding(0.1);
      root
        .append("g")
        .selectAll("g")
        .data(points)
        .join("g")
        .attr("transform", (point) => `translate(${x(point.month as string) || 0},0)`)
        .selectAll("rect")
        .data((point) => years.map((year) => ({ year, value: Number(point[year] || 0) })))
        .join("rect")
        .attr("x", (d) => xInner(d.year) || 0)
        .attr("y", (d) => y(d.value))
        .attr("width", xInner.bandwidth())
        .attr("height", (d) => height - y(d.value))
        .attr("fill", (d) => color(d.year));
    }

    root
      .append("g")
      .selectAll("rect")
      .data(points)
      .join("rect")
      .attr("x", (point) => x(point.month as string) || 0)
      .attr("y", 0)
      .attr("width", x.bandwidth())
      .attr("height", height)
      .attr("fill", "transparent")
      .attr("data-tippy-content", (point) =>
        tooltip(
          years.map((year) => [
            year,
            [formatCurrency(Number(point[year] || 0)), "has-text-weight-bold has-text-right"]
          ]),
          { header: point.month as string }
        )
      );
  }

  onMount(() => {
    render();
    const onResize = () => render();
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  });

  $effect(() => {
    render();
  });
</script>

<div class="yoy-chart" bind:this={chartContainer}>
  <LegendCard {legends} clazz={compactMode ? "mb-2" : "ml-4 mb-3"} />
  <svg {id} width="100%" />
</div>

<style lang="scss">
  .yoy-chart {
    min-width: 0;
  }

  .yoy-chart :global(svg) {
    display: block;
  }
</style>
