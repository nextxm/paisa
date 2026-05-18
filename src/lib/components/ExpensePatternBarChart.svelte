<script lang="ts">
  import * as d3 from "d3";
  import { onMount } from "svelte";
  import type { ExpensePatternPoint } from "$lib/expense_heatmap";
  import { formatCurrency, formatCurrencyCrude } from "$lib/utils";

  let {
    id,
    points,
    title,
    subtitle = "",
    color = "#ef4444"
  }: {
    id: string;
    points: ExpensePatternPoint[];
    title: string;
    subtitle?: string;
    color?: string;
  } = $props();

  function render() {
    if (typeof document === "undefined") return;
    const svg = d3.select(`#${id}`);
    if (!svg.node()) return;

    svg.selectAll("*").remove();

    const margin = { top: 20, right: 16, bottom: 40, left: 52 };
    const width = document.getElementById(id)?.parentElement?.clientWidth || 480;
    const height = 250 - margin.top - margin.bottom;
    const innerWidth = width - margin.left - margin.right;

    const root = svg
      .attr("height", 250)
      .attr("width", "100%")
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    const x = d3
      .scaleBand<string>()
      .domain(points.map((point) => point.label))
      .range([0, innerWidth])
      .padding(0.2);

    const y = d3
      .scaleLinear()
      .domain([0, d3.max(points, (point) => point.value) || 1])
      .nice()
      .range([height, 0]);

    root
      .append("g")
      .attr("class", "axis x")
      .attr("transform", `translate(0,${height})`)
      .call(d3.axisBottom(x));

    root
      .append("g")
      .attr("class", "axis y")
      .call(d3.axisLeft(y).ticks(5).tickFormat((value: number) => formatCurrencyCrude(value)));

    root
      .append("g")
      .selectAll("rect")
      .data(points)
      .join("rect")
      .attr("x", (point) => x(point.label) || 0)
      .attr("y", (point) => y(point.value))
      .attr("width", x.bandwidth())
      .attr("height", (point) => height - y(point.value))
      .attr("rx", 4)
      .attr("fill", color)
      .append("title")
      .text(
        (point) => `${point.label}: ${formatCurrency(point.value)} average (${formatCurrency(point.total)} total)`
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

<div class="pattern-chart">
  <div class="mb-3">
    <p class="has-text-weight-semibold">{title}</p>
    {#if subtitle}
      <p class="is-size-7 has-text-grey">{subtitle}</p>
    {/if}
  </div>

  <svg {id}></svg>
</div>
