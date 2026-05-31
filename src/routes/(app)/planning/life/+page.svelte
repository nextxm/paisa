<script lang="ts">
  import { onMount } from "svelte";
  import {
    ajax,
    formatCurrency,
    formatFloat,
    type FinancialProfile,
    type ProjectionLifeGoal,
    type SimulationResponse,
    type SimulationMonthlyPoint
  } from "$lib/utils";
  import COLORS from "$lib/colors";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import * as d3 from "d3";
  import dayjs from "dayjs";
  import { debounce } from "lodash";

  let simData: SimulationResponse | null = $state(null);
  let profile: FinancialProfile | null = $state(null);
  let loading = $state(true);
  let error: string | null = $state(null);

  // Slider controls — initialized from DNA, user-adjustable
  let expectedReturn = $state(12);
  let volatility = $state(18);
  let monthlyContrib = $state(0);
  let contribGrowth = $state(5);
  let inflationRate = $state(6);
  let swr = $state(4);
  let projectionYears = $state(30);
  let iterations = $state(1000);

  let controlsInitialized = $state(false);
  let chartContainer: HTMLDivElement | undefined = $state();

  async function runSimulation() {
    loading = true;
    error = null;
    try {
      simData = await ajax("/api/projection/simulate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          iterations: iterations,
          months_to_project: projectionYears * 12,
          monthly_contribution: monthlyContrib,
          contribution_growth_rate: contribGrowth,
          expected_return: expectedReturn,
          return_volatility: volatility,
          inflation_rate: inflationRate,
          swr: swr
        })
      });
      if (simData) {
        profile = simData.profile;
      }
    } catch (e) {
      error = e instanceof Error ? e.message : "Simulation failed";
    } finally {
      loading = false;
    }
  }

  const debouncedSimulate = debounce(runSimulation, 600);

  $effect(() => {
    // Track all slider values to trigger re-simulation
    const _er = expectedReturn;
    const _v = volatility;
    const _mc = monthlyContrib;
    const _cg = contribGrowth;
    const _ir = inflationRate;
    const _s = swr;
    const _py = projectionYears;
    const _it = iterations;

    if (controlsInitialized) {
      debouncedSimulate();
    }
  });

  $effect(() => {
    if (simData && chartContainer) {
      renderFanChart(simData, chartContainer);
    }
  });

  function renderFanChart(data: SimulationResponse, container: HTMLDivElement) {
    const sim = data.simulation;
    const bands = sim.bands;
    if (!bands || !bands["p50"] || bands["p50"].length === 0) return;

    // Clear previous chart
    d3.select(container).selectAll("*").remove();

    const margin = { top: 30, right: 30, bottom: 50, left: 90 };
    const width = container.clientWidth - margin.left - margin.right;
    const height = 420 - margin.top - margin.bottom;

    const svg = d3
      .select(container)
      .append("svg")
      .attr("width", width + margin.left + margin.right)
      .attr("height", height + margin.top + margin.bottom)
      .append("g")
      .attr("transform", `translate(${margin.left},${margin.top})`);

    // Parse dates
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

    // Axes
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

    // P10-P90 band (outermost)
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

    // P25-P75 band (inner)
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

    // P50 median line
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

    // P10 and P90 boundary lines (dashed)
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

    // FIRE target line
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

    // Current networth marker
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

    // Legend
    const legendData = [
      { label: "P50 (Median)", color: COLORS.primary, dash: false },
      { label: "P25–P75", color: COLORS.primary, opacity: 0.18, isArea: true },
      { label: "P10–P90", color: COLORS.primary, opacity: 0.08, isArea: true },
      { label: "FIRE Target", color: COLORS.tertiary, dash: true }
    ];

    const legend = svg.append("g").attr("transform", `translate(${width - 200}, 0)`);

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

    // Tooltip
    const tooltip = d3
      .select(container)
      .append("div")
      .style("position", "absolute")
      .style("background", "rgba(0,0,0,0.85)")
      .style("color", "white")
      .style("padding", "8px 12px")
      .style("border-radius", "6px")
      .style("font-size", "12px")
      .style("pointer-events", "none")
      .style("opacity", 0)
      .style("z-index", "10");
    svg
      .append("rect")
      .attr("width", width)
      .attr("height", height)
      .attr("fill", "transparent")
      .on("mousemove", (event: MouseEvent) => {
        const [mx] = d3.pointer(event);
        const dateAtMouse = x.invert(mx);
        const idx = Math.min(bisect(bands["p50"], dateAtMouse), bands["p50"].length - 1);
        if (idx < 0) return;

        const dateStr = dayjs(bands["p50"][idx].date).format("MMM YYYY");
        tooltip
          .style("opacity", 1)
          .style("left", `${mx + margin.left + 15}px`)
          .style("top", `${margin.top + 10}px`)
          .html(
            `<strong>${dateStr}</strong><br/>` +
              `P90: ${formatCurrency(bands["p90"][idx].balance_amount)}<br/>` +
              `P75: ${formatCurrency(bands["p75"][idx].balance_amount)}<br/>` +
              `<strong>P50: ${formatCurrency(bands["p50"][idx].balance_amount)}</strong><br/>` +
              `P25: ${formatCurrency(bands["p25"][idx].balance_amount)}<br/>` +
              `P10: ${formatCurrency(bands["p10"][idx].balance_amount)}`
          );
      })
      .on("mouseleave", () => {
        tooltip.style("opacity", 0);
      });
  }

  onMount(async () => {
    await runSimulation();
    if (simData && simData.profile) {
      profile = simData.profile;
      // Initialize sliders from DNA-derived values
      if (profile.historical_return > 0)
        expectedReturn = Math.round(profile.historical_return * 10) / 10;
      if (profile.return_volatility > 0)
        volatility = Math.round(profile.return_volatility * 10) / 10;
      if (profile.monthly_contribution > 0)
        monthlyContrib = Math.round(profile.monthly_contribution);
      if (profile.income_growth_rate > 0)
        contribGrowth = Math.round(profile.income_growth_rate * 10) / 10;
      controlsInitialized = true;
      // Re-run with DNA-calibrated values
      await runSimulation();
    }
  });
</script>

<section class="section tab-life-projection">
  <div class="container is-fluid">
    <div class="mb-5">
      <div class="is-flex is-justify-content-space-between is-align-items-flex-start life-header">
        <div>
          <h1 class="title is-4 is-spaced mb-1">Life Projection</h1>
          <p class="subtitle is-6 has-text-grey">
            Monte Carlo simulation using {iterations.toLocaleString()} scenarios to project your financial
            future probabilistically
          </p>
        </div>
        <div class="buttons">
          <a class="button is-light" href="/planning/life/whatif">What-If Scenarios</a>
          <a class="button is-light" href="/planning/life/drawdown">Tax-Aware Drawdown</a>
          <a class="button is-light" href="/planning/life/goals">Manage Goals</a>
        </div>
      </div>
    </div>

    <div class="columns">
      <!-- Left: Controls + Summary -->
      <div class="column is-4">
        <!-- Key metrics -->
        {#if simData}
          <div class="box">
            <nav class="level grid-2">
              <LevelItem
                title="Current Net Worth"
                color={COLORS.primary}
                value={formatCurrency(profile?.current_networth || 0)}
              />
              <LevelItem
                title="FIRE Probability"
                color={simData.simulation.fire_probability >= 0.7
                  ? COLORS.gainText
                  : COLORS.lossText}
                value={`${formatFloat(simData.simulation.fire_probability * 100)}%`}
              />
              <LevelItem
                title="Target Corpus"
                color={COLORS.tertiary}
                value={formatCurrency(simData.simulation.target_corpus)}
              />
              <LevelItem
                title="Years to FIRE (P50)"
                value={simData.simulation.fire_year_p50 > 0
                  ? `${simData.simulation.fire_year_p50} years`
                  : "Not in window"}
              />
              <LevelItem
                title="Monthly Contribution"
                color={COLORS.secondary}
                value={formatCurrency(monthlyContrib)}
              />
              <LevelItem
                title="Savings Rate"
                value={`${formatFloat(profile?.savings_rate || 0)}%`}
              />
            </nav>
          </div>
        {/if}

        <!-- Simulation controls -->
        <div class="box">
          <h3 class="title is-6 mb-3">Simulation Parameters</h3>
          <div class="field">
            <label class="label is-size-7" for="life-years"
              >Projection Years: {projectionYears}</label
            >
            <input
              id="life-years"
              type="range"
              min="5"
              max="50"
              step="1"
              bind:value={projectionYears}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-return"
              >Expected Return: {formatFloat(expectedReturn)}%</label
            >
            <input
              id="life-return"
              type="range"
              min="0"
              max="30"
              step="0.5"
              bind:value={expectedReturn}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-volatility"
              >Volatility (Risk): {formatFloat(volatility)}%</label
            >
            <input
              id="life-volatility"
              type="range"
              min="0"
              max="40"
              step="0.5"
              bind:value={volatility}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-contrib"
              >Monthly Contribution: {formatCurrency(monthlyContrib)}</label
            >
            <input
              id="life-contrib"
              type="range"
              min="0"
              max="1000000"
              step="1000"
              bind:value={monthlyContrib}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-contrib-growth"
              >Contribution Growth: {formatFloat(contribGrowth)}%</label
            >
            <input
              id="life-contrib-growth"
              type="range"
              min="0"
              max="20"
              step="0.5"
              bind:value={contribGrowth}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-inflation"
              >Inflation Rate: {formatFloat(inflationRate)}%</label
            >
            <input
              id="life-inflation"
              type="range"
              min="0"
              max="15"
              step="0.5"
              bind:value={inflationRate}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-swr"
              >Safe Withdrawal Rate: {formatFloat(swr)}%</label
            >
            <input id="life-swr" type="range" min="2" max="8" step="0.1" bind:value={swr} />
          </div>
          <div class="field">
            <label class="label is-size-7" for="life-iterations"
              >Iterations: {iterations.toLocaleString()}</label
            >
            <input
              id="life-iterations"
              type="range"
              min="100"
              max="10000"
              step="100"
              bind:value={iterations}
            />
          </div>
        </div>
      </div>

      <!-- Right: Fan Chart -->
      <div class="column is-8">
        <div class="box">
          {#if loading}
            <div class="has-text-centered py-6">
              <span class="icon is-large">
                <i class="fas fa-spinner fa-pulse fa-2x"></i>
              </span>
              <p class="mt-3 has-text-grey">Running {iterations.toLocaleString()} simulations…</p>
            </div>
          {:else if error}
            <div class="notification is-danger is-light">
              <p>{error}</p>
            </div>
          {:else}
            <div
              bind:this={chartContainer}
              class="fan-chart-container"
              style="position: relative;"
            ></div>
          {/if}
        </div>

        <!-- FIRE timeline -->
        {#if simData && !loading}
          <div class="box">
            <h3 class="title is-6 mb-3">FIRE Timeline</h3>
            <div class="columns is-multiline">
              <div class="column is-4">
                <div class="fire-metric">
                  <div class="fire-label">Optimistic (P25)</div>
                  <div class="fire-value has-text-success">
                    {simData.simulation.fire_year_p25 > 0
                      ? `${simData.simulation.fire_year_p25} years`
                      : "—"}
                  </div>
                </div>
              </div>
              <div class="column is-4">
                <div class="fire-metric">
                  <div class="fire-label">Expected (P50)</div>
                  <div class="fire-value">
                    {simData.simulation.fire_year_p50 > 0
                      ? `${simData.simulation.fire_year_p50} years`
                      : "—"}
                  </div>
                </div>
              </div>
              <div class="column is-4">
                <div class="fire-metric">
                  <div class="fire-label">Conservative (P75)</div>
                  <div class="fire-value has-text-danger">
                    {simData.simulation.fire_year_p75 > 0
                      ? `${simData.simulation.fire_year_p75} years`
                      : "—"}
                  </div>
                </div>
              </div>
            </div>

            <!-- FIRE probability bar -->
            <div class="mt-3">
              <div class="is-flex is-justify-content-space-between mb-1">
                <span class="is-size-7 has-text-weight-semibold">FIRE Probability</span>
                <span class="is-size-7 has-text-weight-bold"
                  >{formatFloat(simData.simulation.fire_probability * 100)}%</span
                >
              </div>
              <progress
                class="progress {simData.simulation.fire_probability >= 0.7
                  ? 'is-success'
                  : simData.simulation.fire_probability >= 0.4
                    ? 'is-warning'
                    : 'is-danger'}"
                value={simData.simulation.fire_probability * 100}
                max="100"
              >
                {formatFloat(simData.simulation.fire_probability * 100)}%
              </progress>
            </div>
          </div>
        {/if}

        {#if simData?.goals?.length}
          <div class="box">
            <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
              <h3 class="title is-6 mb-0">Goal Probabilities</h3>
              <span class="is-size-7 has-text-grey">Priority-funded in simulation order</span>
            </div>
            <div class="goal-list">
              {#each simData.goals as goal (goal.goal_id)}
                <div class="goal-card">
                  <div>
                    <div class="goal-name">{goal.name}</div>
                    <div class="goal-meta">
                      {#if goal.type === "milestone"}
                        {formatCurrency(goal.target_amount)} by {goal.target_date}
                      {:else}
                        {formatCurrency(goal.monthly_allocation)}
                        {goal.frequency} from {goal.start_date} to {goal.end_date}
                      {/if}
                    </div>
                  </div>
                  <div
                    class="goal-probability {goal.probability >= 0.7
                      ? 'has-text-success'
                      : goal.probability >= 0.4
                        ? 'has-text-warning-dark'
                        : 'has-text-danger'}"
                  >
                    {formatFloat(goal.probability * 100)}%
                  </div>
                </div>
              {/each}
            </div>
          </div>
        {/if}
      </div>
    </div>

    <!-- Financial DNA -->
    {#if profile}
      <div class="box mt-2">
        <div class="content mb-4">
          <h3 class="title is-5 mb-2">Your Financial DNA</h3>
          <p class="is-size-7 has-text-grey">
            Parameters inferred from your ledger history. The simulation is initialized with these
            values — adjust the sliders above to explore alternatives.
          </p>
        </div>
        <div class="columns is-multiline">
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Annual Income</div>
              <div class="dna-value">{formatCurrency(profile.annual_income)}</div>
              <div class="dna-meta">
                Growth: <span
                  class={profile.income_growth_rate >= 0 ? "has-text-success" : "has-text-danger"}
                  >{formatFloat(profile.income_growth_rate)}%</span
                > p.a.
              </div>
              <div class="dna-quality">{profile.income_years_covered} year(s) analyzed</div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Annual Expenses</div>
              <div class="dna-value">{formatCurrency(profile.annual_expenses)}</div>
              <div class="dna-meta">
                Growth: <span
                  class={profile.expense_growth_rate >= 0 ? "has-text-danger" : "has-text-success"}
                  >{formatFloat(profile.expense_growth_rate)}%</span
                > p.a.
              </div>
              <div class="dna-quality">{profile.expense_years_covered} year(s) analyzed</div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Savings Rate</div>
              <div class="dna-value">{formatFloat(profile.savings_rate)}%</div>
              <div class="dna-meta">{formatCurrency(profile.monthly_contribution)}/month</div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Historical Return</div>
              <div class="dna-value has-text-success">
                {formatFloat(profile.historical_return)}%
              </div>
              <div class="dna-meta">Volatility: {formatFloat(profile.return_volatility)}%</div>
              <div class="dna-quality">{profile.price_months_covered} month(s) analyzed</div>
            </div>
          </div>
        </div>
      </div>
    {/if}
  </div>
</section>

<style>
  .dna-card {
    padding: 0.75rem;
    border-radius: 6px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .dna-label {
    font-size: 0.7rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    opacity: 0.6;
    margin-bottom: 0.25rem;
  }
  .dna-value {
    font-size: 1.25rem;
    font-weight: 700;
    margin-bottom: 0.25rem;
  }
  .dna-meta {
    font-size: 0.75rem;
    opacity: 0.7;
  }
  .dna-quality {
    font-size: 0.65rem;
    opacity: 0.5;
    margin-top: 0.25rem;
    font-style: italic;
  }
  .fire-metric {
    text-align: center;
    padding: 0.5rem;
  }
  .fire-label {
    font-size: 0.75rem;
    font-weight: 600;
    opacity: 0.6;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }
  .fire-value {
    font-size: 1.5rem;
    font-weight: 700;
    margin-top: 0.25rem;
  }
  .fan-chart-container {
    min-height: 420px;
  }

  .life-header {
    gap: 1rem;
  }

  .goal-list {
    display: grid;
    gap: 0.75rem;
  }

  .goal-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 1rem;
    padding: 0.75rem;
    border-radius: 8px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  .goal-name {
    font-weight: 600;
  }

  .goal-meta {
    font-size: 0.75rem;
    opacity: 0.7;
    margin-top: 0.1rem;
  }

  .goal-probability {
    font-size: 1.1rem;
    font-weight: 700;
    white-space: nowrap;
  }

  .progress {
    height: 0.5rem;
    border-radius: 999px;
  }

  @media (max-width: 768px) {
    .life-header {
      flex-direction: column;
      align-items: stretch;
    }

    .goal-card {
      align-items: flex-start;
      flex-direction: column;
    }
  }
</style>
