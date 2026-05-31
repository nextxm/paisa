<script lang="ts">
  import { onMount } from "svelte";
  import * as d3 from "d3";
  import dayjs from "dayjs";
  import COLORS from "$lib/colors";
  import {
    formatCurrency,
    formatFloat,
    type SimulationMonthlyPoint,
    type WhatIfResponse
  } from "$lib/utils";

  type ScenarioSeries = {
    name: string;
    points: SimulationMonthlyPoint[];
    color: string;
    dashed?: boolean;
  };

  const scenarioPalette = [
    COLORS.secondary,
    COLORS.tertiary,
    COLORS.gainText,
    COLORS.lossText,
    "#8f6ed5"
  ];

  let { comparison }: { comparison: WhatIfResponse | null } = $props();

  let container: HTMLDivElement | undefined = $state();
  let selectedScenarioNames: string[] = $state([]);

  $effect(() => {
    if (comparison) {
      selectedScenarioNames = comparison.scenarios.slice(0, 3).map((scenario) => scenario.name);
    }
  });

  $effect(() => {
    if (comparison && container) {
      renderChart(comparison, container, selectedScenarioNames);
    }
  });

  onMount(() => {
    if (comparison) {
      selectedScenarioNames = comparison.scenarios.slice(0, 3).map((scenario) => scenario.name);
    }
  });

  function toggleScenario(name: string, enabled: boolean) {
    if (enabled) {
      selectedScenarioNames = [...selectedScenarioNames, name];
      return;
    }
    selectedScenarioNames = selectedScenarioNames.filter((scenarioName) => scenarioName !== name);
  }

  function renderChart(data: WhatIfResponse, host: HTMLDivElement, enabledScenarios: string[]) {
    d3.select(host).selectAll("*").remove();

    const baseline = data.baseline.simulation;
    const bands = baseline.bands;
    if (!bands?.p50?.length) return;

    const activeScenarios = data.scenarios.filter((scenario) =>
      enabledScenarios.includes(scenario.name)
    );
    const series: ScenarioSeries[] = [
      {
        name: "Baseline",
        points: bands.p50,
        color: COLORS.primary
      },
      ...activeScenarios.map((scenario, index) => ({
        name: scenario.name,
        points: scenario.simulation.bands.p50,
        color: scenarioPalette[index % scenarioPalette.length],
        dashed: true
      }))
    ];

    const margin = { top: 24, right: 24, bottom: 48, left: 82 };
    const width = host.clientWidth - margin.left - margin.right;
    const height = 360 - margin.top - margin.bottom;
    const parseDate = (point: SimulationMonthlyPoint) => dayjs(point.date).toDate();
    const value = (point: SimulationMonthlyPoint) => point.balance_amount;

    const svg = d3
      .select(host)
      .append("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    const allPoints = [
      ...bands.p10,
      ...bands.p25,
      ...bands.p50,
      ...bands.p75,
      ...bands.p90,
      ...activeScenarios.flatMap((scenario) => scenario.simulation.bands.p50 || [])
    ];

    const xExtent = d3.extent(bands.p50, parseDate) as [Date, Date];
    const yMax = d3.max(allPoints, value) || 0;
    const yMin = Math.min(0, d3.min(allPoints, value) || 0);

    const x = d3.scaleTime().domain(xExtent).range([0, width]);
    const y = d3
      .scaleLinear()
      .domain([yMin, yMax * 1.05])
      .range([height, 0]);

    const band = d3
      .area<number>()
      .x((_d, index) => x(parseDate(bands.p25[index])))
      .y0((_d, index) => y(value(bands.p25[index])))
      .y1((_d, index) => y(value(bands.p75[index])))
      .curve(d3.curveMonotoneX);

    svg
      .append("path")
      .datum(d3.range(bands.p25.length))
      .attr("d", band)
      .attr("fill", COLORS.primary)
      .attr("opacity", 0.12);

    svg
      .append("g")
      .attr("transform", `translate(0,${height})`)
      .call(d3.axisBottom(x).ticks(8))
      .selectAll("text")
      .style("font-size", "11px");

    svg
      .append("g")
      .call(
        d3
          .axisLeft(y)
          .ticks(7)
          .tickFormat((tick) => {
            const current = tick as number;
            if (Math.abs(current) >= 10000000) return `${(current / 10000000).toFixed(1)}Cr`;
            if (Math.abs(current) >= 100000) return `${(current / 100000).toFixed(1)}L`;
            if (Math.abs(current) >= 1000) return `${(current / 1000).toFixed(0)}K`;
            return `${current}`;
          })
      )
      .selectAll("text")
      .style("font-size", "11px");

    const line = d3
      .line<SimulationMonthlyPoint>()
      .x((point) => x(parseDate(point)))
      .y((point) => y(value(point)))
      .curve(d3.curveMonotoneX);

    series.forEach((scenario) => {
      svg
        .append("path")
        .datum(scenario.points)
        .attr("d", line)
        .attr("fill", "none")
        .attr("stroke", scenario.color)
        .attr("stroke-width", scenario.name === "Baseline" ? 2.5 : 2)
        .attr("stroke-dasharray", scenario.dashed ? "7,5" : "none");
    });

    const legend = svg.append("g").attr("transform", `translate(${Math.max(width - 220, 0)}, 0)`);
    series.forEach((scenario, index) => {
      const row = legend.append("g").attr("transform", `translate(0, ${index * 18})`);
      row
        .append("line")
        .attr("x1", 0)
        .attr("x2", 18)
        .attr("y1", 8)
        .attr("y2", 8)
        .attr("stroke", scenario.color)
        .attr("stroke-width", 2)
        .attr("stroke-dasharray", scenario.dashed ? "7,5" : "none");
      row.append("text").attr("x", 24).attr("y", 11).style("font-size", "10px").text(scenario.name);
    });
  }
</script>

{#if comparison}
  <div class="scenario-comparison">
    <div class="scenario-toggle-list">
      {#each comparison.scenarios as scenario}
        <label class="checkbox scenario-toggle">
          <input
            type="checkbox"
            checked={selectedScenarioNames.includes(scenario.name)}
            onchange={(event) =>
              toggleScenario(scenario.name, (event.currentTarget as HTMLInputElement).checked)}
          />
          <span>{scenario.name}</span>
          <span class="scenario-toggle-meta">
            {formatFloat(scenario.simulation.fire_probability * 100)}% FIRE
          </span>
        </label>
      {/each}
    </div>
    <div bind:this={container} class="chart-host"></div>
    <p class="comparison-caption">
      Baseline shows the P25-P75 fan band. Enabled scenarios are overlaid as dashed median
      trajectories.
    </p>
  </div>
{/if}

<style>
  .chart-host {
    min-height: 360px;
  }

  .scenario-toggle-list {
    display: grid;
    gap: 0.6rem;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    margin-bottom: 1rem;
  }

  .scenario-toggle {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    padding: 0.6rem 0.75rem;
    border-radius: 8px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  .scenario-toggle-meta {
    margin-left: auto;
    font-size: 0.75rem;
    opacity: 0.7;
  }

  .comparison-caption {
    margin-top: 0.75rem;
    font-size: 0.75rem;
    opacity: 0.7;
  }

  @media (max-width: 768px) {
    .scenario-toggle {
      align-items: flex-start;
      flex-direction: column;
    }

    .scenario-toggle-meta {
      margin-left: 0;
    }
  }
</style>
