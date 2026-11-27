import * as d3 from "d3";
import dayjs from "dayjs";
import COLORS from "$lib/colors";
import {
  formatCurrency,
  type ProjectionLifeGoal,
  type SimulationMonthlyPoint,
  type SimulationResponse
} from "$lib/utils";

export function renderFanChart(data: SimulationResponse, container: HTMLDivElement) {
  const sim = data.simulation;
  const bands = sim.bands;
  if (!bands || !bands["p50"] || bands["p50"].length === 0) return;

  d3.select(container).selectAll("*").remove();

  const margin = { top: 30, right: 30, bottom: 50, left: 90 };
  const width = Math.max(240, container.clientWidth - margin.left - margin.right);
  const height = 420 - margin.top - margin.bottom;

  const svg = d3
    .select(container)
    .append("svg")
    .attr("width", width + margin.left + margin.right)
    .attr("height", height + margin.top + margin.bottom)
    .append("g")
    .attr("transform", `translate(${margin.left},${margin.top})`);

  const parseDate = (d: SimulationMonthlyPoint) => dayjs(d.date).toDate();
  const getValue = (d: SimulationMonthlyPoint) => d.balance_amount;
  const bisect = d3.bisector((d: SimulationMonthlyPoint) => dayjs(d.date).toDate()).left;
  const goalDate = (goal: ProjectionLifeGoal) => {
    const rawDate = goal.target_date || goal.end_date || goal.start_date;
    if (!rawDate) return null;
    const parsed = dayjs(`${rawDate}-01`);
    return parsed.isValid() ? parsed.toDate() : null;
  };

  const allPoints = [
    ...bands["p10"],
    ...bands["p25"],
    ...bands["p50"],
    ...bands["p75"],
    ...bands["p90"]
  ];

  const xExtent = d3.extent(bands["p50"], parseDate) as [Date, Date];
  const yMax = d3.max(allPoints, getValue) || 0;
  const yMin = Math.min(0, d3.min(allPoints, getValue) || 0);

  const x = d3.scaleTime().domain(xExtent).range([0, width]);
  const y = d3
    .scaleLinear()
    .domain([yMin, yMax * 1.05])
    .range([height, 0]);

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
        .ticks(8)
        .tickFormat((d) => {
          const val = d as number;
          if (Math.abs(val) >= 10000000) return `${(val / 10000000).toFixed(1)}Cr`;
          if (Math.abs(val) >= 100000) return `${(val / 100000).toFixed(1)}L`;
          if (Math.abs(val) >= 1000) return `${(val / 1000).toFixed(0)}K`;
          return `${val}`;
        })
    )
    .selectAll("text")
    .style("font-size", "11px");

  const area1090 = d3
    .area<number>()
    .x((_d, i) => x(parseDate(bands["p10"][i])))
    .y0((_d, i) => y(getValue(bands["p10"][i])))
    .y1((_d, i) => y(getValue(bands["p90"][i])))
    .curve(d3.curveMonotoneX);

  svg
    .append("path")
    .datum(d3.range(bands["p10"].length))
    .attr("d", area1090)
    .attr("fill", COLORS.primary)
    .attr("opacity", 0.08);

  const area2575 = d3
    .area<number>()
    .x((_d, i) => x(parseDate(bands["p25"][i])))
    .y0((_d, i) => y(getValue(bands["p25"][i])))
    .y1((_d, i) => y(getValue(bands["p75"][i])))
    .curve(d3.curveMonotoneX);

  svg
    .append("path")
    .datum(d3.range(bands["p25"].length))
    .attr("d", area2575)
    .attr("fill", COLORS.primary)
    .attr("opacity", 0.18);

  const line50 = d3
    .line<SimulationMonthlyPoint>()
    .x((d) => x(parseDate(d)))
    .y((d) => y(getValue(d)))
    .curve(d3.curveMonotoneX);

  svg
    .append("path")
    .datum(bands["p50"])
    .attr("d", line50)
    .attr("fill", "none")
    .attr("stroke", COLORS.primary)
    .attr("stroke-width", 2.5);

  const lineBoundary = d3
    .line<SimulationMonthlyPoint>()
    .x((d) => x(parseDate(d)))
    .y((d) => y(getValue(d)))
    .curve(d3.curveMonotoneX);

  svg
    .append("path")
    .datum(bands["p10"])
    .attr("d", lineBoundary)
    .attr("fill", "none")
    .attr("stroke", COLORS.lossText)
    .attr("stroke-width", 1)
    .attr("stroke-dasharray", "4,4")
    .attr("opacity", 0.5);

  svg
    .append("path")
    .datum(bands["p90"])
    .attr("d", lineBoundary)
    .attr("fill", "none")
    .attr("stroke", COLORS.gainText)
    .attr("stroke-width", 1)
    .attr("stroke-dasharray", "4,4")
    .attr("opacity", 0.5);

  const targetCorpus = sim.target_corpus;
  if (targetCorpus > 0 && targetCorpus < yMax * 1.05) {
    svg
      .append("line")
      .attr("x1", 0)
      .attr("x2", width)
      .attr("y1", y(targetCorpus))
      .attr("y2", y(targetCorpus))
      .attr("stroke", COLORS.tertiary)
      .attr("stroke-width", 1.5)
      .attr("stroke-dasharray", "8,4");

    svg
      .append("text")
      .attr("x", width - 5)
      .attr("y", y(targetCorpus) - 8)
      .attr("text-anchor", "end")
      .attr("fill", COLORS.tertiary)
      .style("font-size", "11px")
      .style("font-weight", "600")
      .text(`FIRE Target: ${formatCurrency(targetCorpus)}`);
  }

  svg
    .append("circle")
    .attr("cx", x(xExtent[0]))
    .attr("cy", y(data.profile.current_networth))
    .attr("r", 5)
    .attr("fill", COLORS.primary)
    .attr("stroke", "white")
    .attr("stroke-width", 2);

  const goalMarkers = (data.goals || [])
    .map((goal) => ({ goal, date: goalDate(goal) }))
    .filter((entry): entry is { goal: ProjectionLifeGoal; date: Date } => entry.date !== null)
    .filter((entry) => entry.date >= xExtent[0] && entry.date <= xExtent[1]);

  goalMarkers.forEach(({ goal, date }) => {
    const idx = Math.min(Math.max(bisect(bands["p50"], date), 0), bands["p50"].length - 1);
    const point = bands["p50"][idx];
    const gx = x(date);
    const gy = y(point.balance_amount);
    const probabilityPct = Math.round(goal.probability * 100);

    svg
      .append("line")
      .attr("x1", gx)
      .attr("x2", gx)
      .attr("y1", gy)
      .attr("y2", height)
      .attr("stroke", COLORS.secondary)
      .attr("stroke-width", 1)
      .attr("stroke-dasharray", "3,3")
      .attr("opacity", 0.5);

    svg
      .append("circle")
      .attr("cx", gx)
      .attr("cy", gy)
      .attr("r", 4)
      .attr("fill", COLORS.secondary)
      .attr("stroke", "white")
      .attr("stroke-width", 1.5);

    svg
      .append("text")
      .attr("x", Math.min(gx + 6, width - 6))
      .attr("y", Math.max(gy - 8, 12))
      .style("font-size", "10px")
      .style("font-weight", "600")
      .style("fill", COLORS.secondary)
      .text(`${goal.name} ${probabilityPct}%`);
  });

  const legendData = [
    { label: "P50 (Median)", color: COLORS.primary, dash: false },
    { label: "P25–P75", color: COLORS.primary, opacity: 0.18, isArea: true },
    { label: "P10–P90", color: COLORS.primary, opacity: 0.08, isArea: true },
    { label: "FIRE Target", color: COLORS.tertiary, dash: true }
  ];

  const legend = svg.append("g").attr("transform", `translate(${Math.max(0, width - 160)}, 0)`);

  legendData.forEach((d, i) => {
    const g = legend.append("g").attr("transform", `translate(0, ${i * 18})`);
    if (d.isArea) {
      g.append("rect")
        .attr("width", 16)
        .attr("height", 10)
        .attr("fill", d.color)
        .attr("opacity", d.opacity || 0.2);
    } else {
      g.append("line")
        .attr("x1", 0)
        .attr("x2", 16)
        .attr("y1", 5)
        .attr("y2", 5)
        .attr("stroke", d.color)
        .attr("stroke-width", 2)
        .attr("stroke-dasharray", d.dash ? "4,4" : "none");
    }
    g.append("text")
      .attr("x", 22)
      .attr("y", 9)
      .style("font-size", "10px")
      .style("fill", "currentColor")
      .text(d.label);
  });
}
