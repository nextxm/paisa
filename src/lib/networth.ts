import * as d3 from "d3";
import { Delaunay } from "d3";
import _ from "lodash";
import tippy from "tippy.js";
import COLORS from "./colors";
import {
  formatCurrency,
  formatCurrencyCrude,
  isMobile,
  now,
  svgUrl,
  tooltip,
  type Legend,
  type Networth,
  type NetworthProjectionMilestone,
  type NetworthProjectionPoint
} from "./utils";

function networth(d: Networth) {
  return d.investmentAmount + d.gainAmount - d.withdrawalAmount;
}

function investment(d: Networth) {
  return d.investmentAmount - d.withdrawalAmount;
}

function investmentReturn(d: Networth) {
  return d.investment_return ?? d.gainAmount - (d.fx_impact ?? 0);
}

function fxImpact(d: Networth) {
  return d.fx_impact ?? 0;
}

export interface NetworthProjectionSeries {
  label: string;
  color: string;
  points: NetworthProjectionPoint[];
}

interface ProjectionBandPoint {
  date: NetworthProjectionPoint["date"];
  min: number;
  max: number;
}

export function renderNetworth(
  points: Networth[],
  element: Element,
  options: {
    showFXImpact?: boolean;
    projections?: NetworthProjectionSeries[];
    milestones?: NetworthProjectionMilestone[];
  } = {}
): { destroy: () => void; legends: Legend[] } {
  const { showFXImpact = false, projections = [], milestones = [] } = options;
  const projectionEnd = _.max(
    _.flatMap(projections, (p) => _.map(p.points, (point) => point.date))
  );
  const start = _.min(_.map(points, (p) => p.date)),
    end = _.max([now(), projectionEnd || now()]);

  const svg = d3.select(element);

  svg.selectAll("*").remove();

  const right = isMobile() ? 10 : 80,
    margin = { top: 15, right: right, bottom: 20, left: 40 },
    width = Math.max(element.parentElement.clientWidth, 800) - margin.left - margin.right,
    height = +svg.attr("height") - margin.top - margin.bottom,
    g = svg.append("g").attr("transform", "translate(" + margin.left + "," + margin.top + ")");

  svg.attr("width", width + margin.left + margin.right);

  const areaKeys = ["gain", "loss"];
  const colors = [COLORS.gain, COLORS.loss];
  const areaScale = d3.scaleOrdinal<string>().domain(areaKeys).range(colors);

  const lineKeys = ["networth", "investment"];
  const lineScale = d3
    .scaleOrdinal<string>()
    .domain(lineKeys)
    .range([COLORS.primary, COLORS.secondary]);

  const positions = _.flatMap(points, (p) => [
    p.gainAmount + p.investmentAmount - p.withdrawalAmount,
    p.investmentAmount - p.withdrawalAmount
  ]);
  positions.push(
    ..._.flatMap(projections, (series) => _.map(series.points, (point) => point.balanceAmount))
  );
  positions.push(..._.map(milestones, (milestone) => milestone.amount));
  positions.push(0);

  const x = d3.scaleTime().range([0, width]).domain([start, end]),
    y = d3.scaleLinear().range([height, 0]).domain(d3.extent(positions)),
    z = d3.scaleOrdinal<string>(colors).domain(areaKeys);

  const area = (y0: number | ((d: Networth) => number), y1: (d: Networth) => number) => {
    const builder = d3
      .area<Networth>()
      .curve(d3.curveMonotoneX)
      .x((d) => x(d.date))
      .y1(y1);
    if (typeof y0 === "function") {
      return builder.y0(y0);
    }
    return builder.y0(y0);
  };

  g.append("g")
    .attr("class", "axis x")
    .attr("transform", "translate(0," + height + ")")
    .call(d3.axisBottom(x));

  if (!isMobile()) {
    g.append("g")
      .attr("class", "axis y")
      .attr("transform", `translate(${width},0)`)
      .call(d3.axisRight(y).tickPadding(5).tickFormat(formatCurrencyCrude));
  }

  g.append("g")
    .attr("class", "axis y")
    .call(d3.axisLeft(y).tickSize(-width).tickFormat(formatCurrencyCrude));

  const layer = g.selectAll(".layer").data([points]).enter().append("g").attr("class", "layer");

  const clipAboveID = _.uniqueId("clip-above");
  layer
    .append("clipPath")
    .attr("id", clipAboveID)
    .append("path")
    .attr(
      "d",
      area(height, (d) => {
        return y(d.gainAmount + d.investmentAmount - d.withdrawalAmount);
      })
    );

  const clipBelowID = _.uniqueId("clip-below");
  layer
    .append("clipPath")
    .attr("id", clipBelowID)
    .append("path")
    .attr(
      "d",
      area(0, (d) => {
        return y(d.gainAmount + d.investmentAmount - d.withdrawalAmount);
      })
    );

  layer
    .append("path")
    .style("fill", z("gain"))
    .style("opacity", "0.2")
    .attr("clip-path", svgUrl(clipAboveID))
    .attr(
      "d",
      area(0, (d) => {
        return y(d.investmentAmount - d.withdrawalAmount);
      })
    );

  layer
    .append("path")
    .attr("clip-path", svgUrl(clipBelowID))
    .style("fill", z("loss"))
    .style("opacity", "0.2")
    .attr(
      "d",
      area(height, (d) => {
        return y(d.investmentAmount - d.withdrawalAmount);
      })
    );

  if (showFXImpact) {
    layer
      .append("path")
      .style("fill", COLORS.primary)
      .style("opacity", "0.12")
      .attr(
        "d",
        area(
          (d) => y(investment(d)),
          (d) => y(investment(d) + investmentReturn(d))
        )
      );

    layer
      .append("path")
      .style("fill", COLORS.tertiary)
      .style("opacity", "0.15")
      .attr(
        "d",
        area(
          (d) => y(investment(d) + investmentReturn(d)),
          (d) => y(networth(d))
        )
      );
  }

  layer
    .append("path")
    .style("stroke", lineScale("investment"))
    .style("stroke-width", "1.5")
    .style("fill", "none")
    .attr(
      "d",
      d3
        .line<Networth>()
        .curve(d3.curveMonotoneX)
        .x((d) => x(d.date))
        .y((d) => y(investment(d)))
    );

  layer
    .append("path")
    .style("stroke", lineScale("networth"))
    .style("stroke-width", "1.5")
    .style("fill", "none")
    .attr(
      "d",
      d3
        .line<Networth>()
        .curve(d3.curveMonotoneX)
        .x((d) => x(d.date))
        .y((d) => y(networth(d)))
    );

  if (projections.length > 1) {
    const bandPoints: ProjectionBandPoint[] = _.map(projections[0].points, (point, index) => {
      const balances = _.chain(projections)
        .map((scenario) => scenario.points[index]?.balanceAmount)
        .filter((value) => value !== undefined)
        .value() as number[];
      return {
        date: point.date,
        min: _.min(balances) || point.balanceAmount,
        max: _.max(balances) || point.balanceAmount
      };
    });

    layer
      .append("path")
      .style("fill", COLORS.primary)
      .style("opacity", "0.08")
      .attr(
        "d",
        d3
          .area<ProjectionBandPoint>()
          .curve(d3.curveMonotoneX)
          .x((d) => x(d.date))
          .y0((d) => y(d.min))
          .y1((d) => y(d.max))(bandPoints)
      );
  }

  for (const projection of projections) {
    layer
      .append("path")
      .style("stroke", projection.color)
      .style("stroke-width", "1.5")
      .style("stroke-dasharray", "4,3")
      .style("fill", "none")
      .attr(
        "d",
        d3
          .line<NetworthProjectionPoint>()
          .curve(d3.curveMonotoneX)
          .x((d) => x(d.date))
          .y((d) => y(d.balanceAmount))(projection.points)
      );
  }

  milestones.forEach((milestone, idx) => {
    const milestoneX = x(milestone.date);
    layer
      .append("line")
      .attr("x1", milestoneX)
      .attr("x2", milestoneX)
      .attr("y1", 0)
      .attr("y2", height)
      .style("stroke", COLORS.neutral)
      .style("stroke-width", "1")
      .style("stroke-dasharray", "3,3")
      .style("opacity", "0.3");

    layer
      .append("text")
      .attr("x", milestoneX + 6)
      .attr("y", 12 + idx * 14)
      .style("font-size", "10px")
      .style("fill", COLORS.neutral)
      .style("opacity", "0.75")
      .text(milestone.label);
  });

  const hoverCircle = layer.append("circle").attr("r", "3").attr("fill", "none");
  const t = tippy(hoverCircle.node(), { theme: "light", delay: 0, allowHTML: true });

  const networthVoronoiPoints: Delaunay.Point[] = _.map(points, (d) => [x(d.date), y(networth(d))]);
  const investmentVoronoiPoints: Delaunay.Point[] = _.map(points, (d) => [
    x(d.date),
    y(investment(d))
  ]);

  const projectionVoronoiPoints: Delaunay.Point[] = [];
  const expectedSeries = projections[1]; // expected is at index 1
  if (expectedSeries) {
    expectedSeries.points.forEach((p) => {
      projectionVoronoiPoints.push([x(p.date), y(p.balanceAmount)]);
    });
  }

  const allVoronoiPoints = networthVoronoiPoints
    .concat(investmentVoronoiPoints)
    .concat(projectionVoronoiPoints);

  const voronoi = Delaunay.from(allVoronoiPoints).voronoi([0, 0, width, height]);

  const dataList: any[] = points
    .map((p) => ["networth", p])
    .concat(points.map((p) => ["investment", p]));

  if (expectedSeries) {
    expectedSeries.points.forEach((p, idx) => {
      dataList.push(["projection", p, idx]);
    });
  }

  layer
    .append("g")
    .selectAll("path")
    .data(dataList)
    .enter()
    .append("path")
    .style("pointer-events", "all")
    .style("fill", "none")
    .attr("d", (_, i) => {
      return voronoi.renderCell(i);
    })
    .on("mouseover", (event, item) => {
      const itemType = item[0];
      if (itemType === "projection") {
        const p = item[1];
        const idx = item[2];
        const consVal = projections[0]?.points[idx]?.balanceAmount ?? 0;
        const expVal = projections[1]?.points[idx]?.balanceAmount ?? 0;
        const optVal = projections[2]?.points[idx]?.balanceAmount ?? 0;

        hoverCircle
          .attr("cx", x(p.date))
          .attr("cy", y(p.balanceAmount))
          .attr("fill", COLORS.primary);

        t.setProps({
          placement: "top",
          content: tooltip([
            ["Date", p.date.format("MMM YYYY")],
            ["Conservative", [formatCurrency(consVal), "has-text-weight-bold has-text-right"]],
            ["Expected", [formatCurrency(expVal), "has-text-weight-bold has-text-right"]],
            ["Optimistic", [formatCurrency(optVal), "has-text-weight-bold has-text-right"]]
          ])
        });
        t.show();
      } else {
        const d = item[1];
        hoverCircle
          .attr("cx", x(d.date))
          .attr("cy", y(itemType == "networth" ? networth(d) : investment(d)))
          .attr("fill", lineScale(itemType));

        t.setProps({
          placement: itemType == "networth" ? "top" : "bottom",
          content: tooltip([
            ["Date", d.date.format("DD MMM YYYY")],
            ["Net Worth", [formatCurrency(networth(d)), "has-text-weight-bold has-text-right"]],
            [
              "Net Investment",
              [formatCurrency(investment(d)), "has-text-weight-bold has-text-right"]
            ],
            ["Gain / Loss", [formatCurrency(d.gainAmount), "has-text-weight-bold has-text-right"]],
            [
              "Contribution",
              [formatCurrency(d.contribution), "has-text-weight-bold has-text-right"]
            ],
            [
              "Investment Return",
              [formatCurrency(investmentReturn(d)), "has-text-weight-bold has-text-right"]
            ],
            ["FX Impact", [formatCurrency(fxImpact(d)), "has-text-weight-bold has-text-right"]]
          ])
        });
        t.show();
      }
    })
    .on("mouseout", () => {
      t.hide();
      hoverCircle.attr("fill", "none");
    });

  const legends: Legend[] = [
    {
      label: "Net Worth",
      color: lineScale("networth"),
      shape: "line"
    },
    {
      label: "Net Investment",
      color: lineScale("investment"),
      shape: "line"
    },
    {
      label: "Gain",
      color: areaScale("gain"),
      shape: "square"
    },
    {
      label: "Loss",
      color: areaScale("loss"),
      shape: "square"
    }
  ];
  if (showFXImpact) {
    legends.push(
      { label: "Investment Return", color: COLORS.primary, shape: "square" },
      { label: "FX Impact", color: COLORS.tertiary, shape: "square" }
    );
  }
  if (projections.length > 1) {
    legends.push({
      label: "Projection Band",
      color: COLORS.primary,
      shape: "square"
    });
  }
  legends.push(
    ...projections.map((scenario) => ({
      label: scenario.label,
      color: scenario.color,
      shape: "line" as const
    }))
  );

  const destroy = () => {
    t.destroy();
  };

  return { destroy, legends };
}
