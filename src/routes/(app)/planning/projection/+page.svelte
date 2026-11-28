<script lang="ts">
  import { page } from "$app/stores";
  import COLORS from "$lib/colors";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import { renderNetworth } from "$lib/networth";
  import {
    ajax,
    appPath,
    formatCurrency,
    formatFloat,
    type FinancialProfile,
    type Legend,
    type Networth,
    type NetworthProjectionResponse,
    type SimulationResponse
  } from "$lib/utils";
  import { onDestroy, onMount } from "svelte";
  import dayjs from "dayjs";
  import { debounce } from "lodash";
  import { renderFanChart } from "./fan_chart";
  import {
    buildProjectionRows,
    calibrateFromDna,
    firstFireCrossingDate,
    generateScenarioCSV,
    getScenarioCSVFilename,
    monthDiff,
    parseVal,
    projectNetworth,
    toRealCagr,
    type ProjectionScenarioKey,
    type ProjectionScenarioTable
  } from "./projection";

  type ProjectionMode = "simple" | "montecarlo";

  let mode: ProjectionMode = $state(
    $page.url.searchParams.get("mode") === "montecarlo" ? "montecarlo" : "simple"
  );

  let svg: Element | undefined = $state();
  let destroy: () => void;
  let legends: Legend[] = $state([]);
  let points: Networth[] = $state([]);
  let baseData: NetworthProjectionResponse | null = $state(null);
  let dnaProfile: FinancialProfile | null = $state(null);

  // Monte Carlo state
  let fanChartContainer: HTMLDivElement | undefined = $state();
  let simData: SimulationResponse | null = $state(null);
  let simLoading = $state(false);
  let simError: string | null = $state(null);

  // Core 3 levers (shared across Simple & Monte Carlo)
  let years = $state(15);
  let monthlyContribution = $state(0);
  let expectedCagr = $state(9);

  // Collapsible assumptions
  let swr = $state(4);
  let inflationRate = $state(6);
  let cagrSpread = $state(3);
  let adjustForInflation = $state(true);

  // Monte Carlo specific assumptions
  let volatility = $state(18);
  let contribGrowth = $state(5);
  let iterations = $state(1000);

  let showAssumptions = $state(false);
  let showDnaDrawer = $state(false);
  let activeScenarioKey: ProjectionScenarioKey = $state("expected");
  let controlsInitialized = $state(false);

  // Debounced calculation values (Simple mode)
  let calcYears = $state(15);
  let calcMonthlyContribution = $state(0);
  let calcExpectedCagr = $state(9);
  let calcSwr = $state(4);
  let calcInflationRate = $state(6);
  let calcCagrSpread = $state(3);
  let calcAdjustForInflation = $state(true);

  function calibrateSliders() {
    if (!dnaProfile) return;

    const calibrated = calibrateFromDna(dnaProfile, {
      monthlyContribution,
      expectedCagr,
      cagrSpread,
      inflationRate
    });

    monthlyContribution = calibrated.monthlyContribution;
    expectedCagr = calibrated.expectedCagr;
    cagrSpread = calibrated.cagrSpread;
    inflationRate = calibrated.inflationRate;

    if (dnaProfile.return_volatility > 0) {
      volatility = Math.min(40, Math.max(0, Math.round(dnaProfile.return_volatility * 10) / 10));
    }
    if (dnaProfile.income_growth_rate > 0) {
      contribGrowth = Math.min(
        20,
        Math.max(0, Math.round(dnaProfile.income_growth_rate * 10) / 10)
      );
    }

    calcMonthlyContribution = monthlyContribution;
    calcExpectedCagr = expectedCagr;
    calcCagrSpread = cagrSpread;
    calcInflationRate = inflationRate;
    updateCalculations.flush();

    if (mode === "montecarlo") {
      debouncedSimulate.flush();
    }
  }

  function exportScenarioCSV(scenario: ProjectionScenarioTable) {
    const csvContent = generateScenarioCSV(scenario);
    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = getScenarioCSVFilename(scenario.label);
    link.click();
    URL.revokeObjectURL(url);
  }

  const updateCalculations = debounce(() => {
    calcYears = Math.max(1, parseVal(years, 15));
    calcMonthlyContribution = parseVal(monthlyContribution, 0);
    calcExpectedCagr = parseVal(expectedCagr, 9);
    calcSwr = parseVal(swr, 4);
    calcInflationRate = parseVal(inflationRate, 6);
    calcCagrSpread = Math.max(0.5, parseVal(cagrSpread, 3));
    calcAdjustForInflation = adjustForInflation;
  }, 100);

  async function runSimulation() {
    simLoading = true;
    simError = null;
    try {
      simData = await ajax("/api/projection/simulate", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          iterations: Math.max(100, parseVal(iterations, 1000)),
          months_to_project: Math.max(1, parseVal(years, 15)) * 12,
          monthly_contribution: parseVal(monthlyContribution, 0),
          contribution_growth_rate: parseVal(contribGrowth, 5),
          expected_return: parseVal(expectedCagr, 9),
          return_volatility: parseVal(volatility, 18),
          inflation_rate: parseVal(inflationRate, 6),
          swr: parseVal(swr, 4)
        })
      });
      if (simData?.profile) {
        dnaProfile = simData.profile;
      }
    } catch (e) {
      simError = e instanceof Error ? e.message : "Simulation failed";
    } finally {
      simLoading = false;
    }
  }

  const debouncedSimulate = debounce(runSimulation, 500);

  $effect(() => {
    const _y = years;
    const _mc = monthlyContribution;
    const _ec = expectedCagr;
    const _s = swr;
    const _i = inflationRate;
    const _sp = cagrSpread;
    const _adj = adjustForInflation;
    const _v = volatility;
    const _cg = contribGrowth;
    const _it = iterations;

    if (!controlsInitialized) return;

    if (mode === "simple") {
      updateCalculations();
    } else {
      debouncedSimulate();
    }
  });

  $effect(() => {
    if (mode === "montecarlo" && controlsInitialized && !simData && !simLoading && !simError) {
      void runSimulation();
    }
  });

  $effect(() => {
    if (mode === "montecarlo" && simData && fanChartContainer) {
      renderFanChart(simData, fanChartContainer);
    }
  });

  const projection = $derived.by(() => {
    if (!baseData) return null;

    const currentNetworth = baseData.current_networth;
    const annualExpenses = baseData.annual_expenses;
    const now = dayjs();
    const months = calcYears * 12;

    const nominalConservative = calcExpectedCagr - calcCagrSpread;
    const nominalExpected = calcExpectedCagr;
    const nominalOptimistic = calcExpectedCagr + calcCagrSpread;

    const effectiveConservative = calcAdjustForInflation
      ? toRealCagr(nominalConservative, calcInflationRate)
      : nominalConservative;
    const effectiveExpected = calcAdjustForInflation
      ? toRealCagr(nominalExpected, calcInflationRate)
      : nominalExpected;
    const effectiveOptimistic = calcAdjustForInflation
      ? toRealCagr(nominalOptimistic, calcInflationRate)
      : nominalOptimistic;

    const conservative = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      effectiveConservative,
      months
    );
    const expected = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      effectiveExpected,
      months
    );
    const optimistic = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      effectiveOptimistic,
      months
    );

    const baseTargetCorpus = calcSwr > 0 ? annualExpenses / (calcSwr / 100) : 0;
    const targetInflation = calcAdjustForInflation ? 0 : calcInflationRate;

    let displayTargetCorpus = baseTargetCorpus;
    let fireProgress = 0;
    let yearsToFIRE: number | null = null;
    let fireDate: dayjs.Dayjs | null = null;

    const milestones = [];

    if (baseTargetCorpus > 0) {
      fireProgress = Math.min(100, (currentNetworth / baseTargetCorpus) * 100);
      const crossing = firstFireCrossingDate(expected, baseTargetCorpus, targetInflation);
      if (crossing) {
        const monthsToFire = monthDiff(now, crossing.date);
        yearsToFIRE = Math.round((monthsToFire / 12) * 100) / 100;
        fireDate = crossing.date;
        displayTargetCorpus = crossing.targetAtCrossing;
        milestones.push({
          label: "FIRE target reached",
          date: crossing.date,
          amount: crossing.targetAtCrossing
        });
      }
    }

    return {
      current_networth: currentNetworth,
      savings_rate: baseData.savings_rate,
      monthly_contribution: calcMonthlyContribution,
      annual_expenses: annualExpenses,
      swr: calcSwr,
      base_target_corpus: baseTargetCorpus,
      target_corpus: displayTargetCorpus,
      years_to_fire: yearsToFIRE,
      fire_date: fireDate,
      fire_progress_percent: fireProgress,
      projection: {
        conservative,
        expected,
        optimistic
      },
      milestones,
      conservative_cagr: effectiveConservative,
      expected_cagr: effectiveExpected,
      optimistic_cagr: effectiveOptimistic,
      table_inflation: targetInflation
    };
  });

  const projectionScenarioTables = $derived.by((): ProjectionScenarioTable[] | null => {
    if (!projection) return null;

    return [
      {
        key: "expected",
        label: "Expected Scenario",
        cagr: projection.expected_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          projection.table_inflation,
          projection.expected_cagr,
          projection.monthly_contribution,
          calcYears,
          projection.swr
        )
      },
      {
        key: "conservative",
        label: "Conservative Scenario",
        cagr: projection.conservative_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          projection.table_inflation,
          projection.conservative_cagr,
          projection.monthly_contribution,
          calcYears,
          projection.swr
        )
      },
      {
        key: "optimistic",
        label: "Optimistic Scenario",
        cagr: projection.optimistic_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          projection.table_inflation,
          projection.optimistic_cagr,
          projection.monthly_contribution,
          calcYears,
          projection.swr
        )
      }
    ];
  });

  const activeScenario = $derived(
    projectionScenarioTables?.find((s) => s.key === activeScenarioKey) ??
      projectionScenarioTables?.[0] ??
      null
  );

  const firstFireYearInTable = $derived(
    activeScenario?.rows.find((r) => r.isFireReached)?.year ?? null
  );

  $effect(() => {
    if (mode === "simple" && svg && points.length > 0) {
      if (destroy) {
        destroy();
      }
      ({ destroy, legends } = renderNetworth(points, svg, {
        showFXImpact: true,
        projections: projection
          ? [
              {
                label: "Conservative Projection",
                color: COLORS.lossText,
                points: projection.projection.conservative
              },
              {
                label: "Expected Projection",
                color: COLORS.primary,
                points: projection.projection.expected
              },
              {
                label: "Optimistic Projection",
                color: COLORS.gainText,
                points: projection.projection.optimistic
              }
            ]
          : [],
        milestones: projection?.milestones || []
      }));
    }
  });

  onMount(async () => {
    const [projData, dnaData] = await Promise.all([
      ajax("/api/networth/projection") as Promise<NetworthProjectionResponse>,
      ajax("/api/projection/dna") as Promise<{ profile: FinancialProfile }>
    ]);

    baseData = projData;
    if (dnaData) {
      dnaProfile = dnaData.profile;
      if (dnaProfile.return_volatility > 0) {
        volatility = Math.min(40, Math.max(0, Math.round(dnaProfile.return_volatility * 10) / 10));
      }
      if (dnaProfile.income_growth_rate > 0) {
        contribGrowth = Math.min(
          20,
          Math.max(0, Math.round(dnaProfile.income_growth_rate * 10) / 10)
        );
      }
    }

    if (baseData) {
      points = [
        {
          date: dayjs(),
          investmentAmount: baseData.current_networth,
          withdrawalAmount: 0,
          gainAmount: 0,
          contribution: baseData.current_networth,
          investment_return: 0,
          fx_impact: 0,
          balanceAmount: baseData.current_networth,
          balanceUnits: 0,
          netInvestmentAmount: baseData.current_networth
        }
      ];

      years = Math.round(baseData.projection.expected.length / 12) || 15;
      expectedCagr = baseData.expected_cagr;
      const derivedSpread = Math.max(
        0.5,
        Math.round(((baseData.optimistic_cagr - baseData.conservative_cagr) / 2 || 3) * 10) / 10
      );
      cagrSpread = derivedSpread;
      monthlyContribution = baseData.monthly_contribution;
      swr = baseData.swr;

      calcYears = years;
      calcExpectedCagr = expectedCagr;
      calcCagrSpread = cagrSpread;
      calcMonthlyContribution = monthlyContribution;
      calcSwr = swr;
      calcInflationRate = inflationRate;
      calcAdjustForInflation = adjustForInflation;

      controlsInitialized = true;
    }
  });

  onDestroy(() => {
    updateCalculations.cancel();
    debouncedSimulate.cancel();
    if (destroy) {
      destroy();
    }
  });
</script>

<section class="section tab-projection py-5">
  <div class="container is-fluid">
    <!-- Compact Header with Mode Toggle + DNA Actions -->
    <div
      class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-3 mb-4"
    >
      <div>
        <h1 class="title is-4 mb-1">Net Worth Projection</h1>
        {#if projection}
          <p class="is-size-7 has-text-grey">
            Baseline: <strong>{formatCurrency(projection.current_networth)}</strong> net worth ·
            <strong>{formatCurrency(projection.annual_expenses)}</strong>/yr expenses ·
            {#if mode === "montecarlo"}
              Probabilistic Monte Carlo ({iterations.toLocaleString()} runs)
            {:else}
              {calcAdjustForInflation
                ? "Shown in today's money (real)"
                : "Shown in future money (nominal)"}
            {/if}
          </p>
        {/if}
      </div>

      <div class="is-flex is-align-items-center is-flex-wrap-wrap gap-2">
        <div class="tabs is-toggle is-small mb-0">
          <ul>
            <li class={mode === "simple" ? "is-active" : ""}>
              <a
                href="#simple"
                onclick={(e) => {
                  e.preventDefault();
                  mode = "simple";
                }}
              >
                <span>Simple</span>
              </a>
            </li>
            <li class={mode === "montecarlo" ? "is-active" : ""}>
              <a
                href="#montecarlo"
                onclick={(e) => {
                  e.preventDefault();
                  mode = "montecarlo";
                }}
              >
                <span>Monte Carlo</span>
              </a>
            </li>
          </ul>
        </div>

        {#if dnaProfile}
          <button
            class="button is-small is-primary is-light"
            type="button"
            onclick={calibrateSliders}
          >
            <span class="icon is-small"><i class="fa-solid fa-wand-magic-sparkles"></i></span>
            <span>Calibrate from Ledger</span>
          </button>
          <button
            class="button is-small is-light"
            type="button"
            aria-expanded={showDnaDrawer}
            onclick={() => (showDnaDrawer = !showDnaDrawer)}
          >
            <span class="icon is-small"><i class="fa-solid fa-dna"></i></span>
            <span>Financial DNA</span>
            <span class="icon is-small">
              <i class="fa-solid {showDnaDrawer ? 'fa-chevron-up' : 'fa-chevron-down'}"></i>
            </span>
          </button>
        {/if}
      </div>
    </div>

    <!-- Collapsible Financial DNA Drawer -->
    {#if dnaProfile && showDnaDrawer}
      <div class="box mb-4 dna-drawer">
        <div
          class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-2 mb-3"
        >
          <p class="is-size-7 has-text-grey mb-0">
            Inferred from your ledger history. Click <strong>Apply to Sliders</strong> to calibrate your
            projection parameters.
          </p>
          <button
            class="button is-small is-primary"
            type="button"
            onclick={() => {
              calibrateSliders();
              showDnaDrawer = false;
            }}
          >
            Apply to Sliders
          </button>
        </div>
        <div class="columns is-multiline">
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Annual Income</div>
              <div class="dna-value">{formatCurrency(dnaProfile.annual_income)}</div>
              <div class="dna-meta">
                Growth: <span
                  class={dnaProfile.income_growth_rate >= 0
                    ? "has-text-success"
                    : "has-text-danger"}>{formatFloat(dnaProfile.income_growth_rate)}%</span
                > p.a.
              </div>
              <div class="dna-quality">{dnaProfile.income_years_covered} year(s) analyzed</div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Annual Expenses</div>
              <div class="dna-value">{formatCurrency(dnaProfile.annual_expenses)}</div>
              <div class="dna-meta">
                Growth: <span
                  class={dnaProfile.expense_growth_rate >= 0
                    ? "has-text-danger"
                    : "has-text-success"}>{formatFloat(dnaProfile.expense_growth_rate)}%</span
                > p.a.
              </div>
              <div class="dna-quality">{dnaProfile.expense_years_covered} year(s) analyzed</div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Savings Rate</div>
              <div class="dna-value">{formatFloat(dnaProfile.savings_rate)}%</div>
              <div class="dna-meta">
                {formatCurrency(dnaProfile.monthly_contribution)}/month
              </div>
            </div>
          </div>
          <div class="column is-3">
            <div class="dna-card">
              <div class="dna-label">Historical Return</div>
              <div class="dna-value has-text-success">
                {formatFloat(dnaProfile.historical_return)}%
              </div>
              <div class="dna-meta">
                Volatility: {formatFloat(dnaProfile.return_volatility)}%
              </div>
              <div class="dna-quality">{dnaProfile.price_months_covered} month(s) analyzed</div>
            </div>
          </div>
        </div>
      </div>
    {/if}

    <!-- Outcome KPIs Strip -->
    {#if mode === "simple"}
      <div class="columns is-multiline mb-4">
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">Years to FIRE</div>
            <div class="outcome-value">
              {#if projection?.years_to_fire !== null && projection?.years_to_fire !== undefined}
                {formatFloat(projection.years_to_fire)} years
              {:else}
                Not in {calcYears}y window
              {/if}
            </div>
            <div class="outcome-sub">
              {#if projection?.fire_date}
                Estimated by {projection.fire_date.format("MMM YYYY")}
              {:else}
                Increase horizon or monthly savings
              {/if}
            </div>
          </div>
        </div>
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">FIRE Progress</div>
            <div class="outcome-value">{formatFloat(projection?.fire_progress_percent || 0)}%</div>
            <progress
              class="progress is-small is-primary mt-2 mb-1"
              value={projection?.fire_progress_percent || 0}
              max="100"
            ></progress>
            <div class="outcome-sub">
              {formatCurrency(projection?.current_networth || 0)} of
              {formatCurrency(projection?.base_target_corpus || 0)} today's target
            </div>
          </div>
        </div>
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">
              Target Corpus
              <span class="tag is-light is-small ml-1">
                {calcAdjustForInflation ? "Real" : "Nominal"}
              </span>
            </div>
            <div class="outcome-value">{formatCurrency(projection?.target_corpus || 0)}</div>
            <div class="outcome-sub">
              {formatFloat(calcSwr)}% SWR on {formatCurrency(projection?.annual_expenses || 0)}/yr
            </div>
          </div>
        </div>
      </div>
    {:else}
      <div class="columns is-multiline mb-4">
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">Years to FIRE (Median P50)</div>
            <div class="outcome-value">
              {#if simData?.simulation.fire_year_p50 && simData.simulation.fire_year_p50 > 0}
                {simData.simulation.fire_year_p50} years
              {:else}
                Not in {years}y window
              {/if}
            </div>
            <div class="outcome-sub">
              {#if simData}
                P25: {simData.simulation.fire_year_p25 > 0
                  ? `${simData.simulation.fire_year_p25}y`
                  : "—"} · P75: {simData.simulation.fire_year_p75 > 0
                  ? `${simData.simulation.fire_year_p75}y`
                  : "—"}
              {:else}
                Running simulation…
              {/if}
            </div>
          </div>
        </div>
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">FIRE Probability</div>
            <div class="outcome-value">
              {simData ? `${formatFloat(simData.simulation.fire_probability * 100)}%` : "—"}
            </div>
            {#if simData}
              <progress
                class="progress is-small mt-2 mb-1 {simData.simulation.fire_probability >= 0.7
                  ? 'is-success'
                  : simData.simulation.fire_probability >= 0.4
                    ? 'is-warning'
                    : 'is-danger'}"
                value={simData.simulation.fire_probability * 100}
                max="100"
              ></progress>
            {/if}
            <div class="outcome-sub">Likelihood of reaching FIRE within {years} years</div>
          </div>
        </div>
        <div class="column is-4">
          <div class="box outcome-card">
            <div class="outcome-label">Target Corpus</div>
            <div class="outcome-value">
              {simData ? formatCurrency(simData.simulation.target_corpus) : "—"}
            </div>
            <div class="outcome-sub">
              {formatFloat(swr)}% SWR · {formatFloat(volatility)}% volatility
            </div>
          </div>
        </div>
      </div>
    {/if}

    <!-- Main Workspace: 3 Core Levers + Chart -->
    <div class="columns">
      <div class="column is-4">
        <div class="box">
          <h2 class="title is-6 mb-3">Core Levers</h2>

          <div class="field">
            <label class="label is-size-7" for="projection-years"
              >Projection Horizon: {years ?? ""} years</label
            >
            <div class="control-row">
              <input
                id="projection-years"
                class="slider-input"
                type="range"
                min="1"
                max={Math.max(40, Number(years) || 40)}
                step="1"
                bind:value={years}
              />
              <input
                id="projection-years-input"
                class="input is-small number-input"
                type="number"
                min="1"
                step="1"
                bind:value={years}
                aria-label="Projection Years"
              />
            </div>
          </div>

          <div class="field">
            <label class="label is-size-7" for="projection-monthly-contribution"
              >Monthly Contribution: {formatCurrency(monthlyContribution ?? 0)}</label
            >
            <div class="control-row">
              <input
                id="projection-monthly-contribution"
                class="slider-input"
                type="range"
                min={Math.min(-100000, Number(monthlyContribution) || 0)}
                max={Math.max(1000000, Number(monthlyContribution) || 0)}
                step="any"
                bind:value={monthlyContribution}
              />
              <input
                id="projection-monthly-contribution-input"
                class="input is-small number-input"
                type="number"
                step="any"
                bind:value={monthlyContribution}
                aria-label="Monthly Contribution"
              />
            </div>
          </div>

          <div class="field">
            <label class="label is-size-7" for="projection-expected-cagr"
              >Expected Return (CAGR): {formatFloat(expectedCagr ?? 0)}%</label
            >
            <div class="control-row">
              <input
                id="projection-expected-cagr"
                class="slider-input"
                type="range"
                min={Math.min(-5, Number(expectedCagr) || -5)}
                max={Math.max(35, Number(expectedCagr) || 35)}
                step="0.5"
                bind:value={expectedCagr}
              />
              <input
                id="projection-expected-cagr-input"
                class="input is-small number-input"
                type="number"
                step="0.1"
                bind:value={expectedCagr}
                aria-label="Expected CAGR"
              />
            </div>
          </div>

          <!-- Collapsible Assumptions -->
          <div class="assumptions-section mt-4 pt-3">
            <button
              type="button"
              class="assumptions-toggle"
              aria-expanded={showAssumptions}
              onclick={() => (showAssumptions = !showAssumptions)}
            >
              <span class="is-size-7 has-text-weight-semibold">
                Assumptions
                <span class="has-text-grey has-text-weight-normal ml-1">
                  {#if mode === "simple"}
                    ({formatFloat(swr)}% SWR · {formatFloat(inflationRate)}% inf · ±{formatFloat(
                      cagrSpread
                    )}%)
                  {:else}
                    ({formatFloat(swr)}% SWR · {formatFloat(volatility)}% vol · {iterations.toLocaleString()}
                    runs)
                  {/if}
                </span>
              </span>
              <span class="icon is-small">
                <i class="fa-solid {showAssumptions ? 'fa-chevron-up' : 'fa-chevron-down'}"></i>
              </span>
            </button>

            {#if showAssumptions}
              <div class="mt-3">
                {#if mode === "simple"}
                  <div class="field">
                    <label class="checkbox is-size-7">
                      <input type="checkbox" bind:checked={adjustForInflation} />
                      <span class="ml-1">Adjust for inflation (show in today's money)</span>
                    </label>
                  </div>
                {/if}

                <div class="field">
                  <label class="label is-size-7" for="projection-swr"
                    >Safe Withdrawal Rate (SWR): {formatFloat(swr ?? 0)}%</label
                  >
                  <div class="control-row">
                    <input
                      id="projection-swr"
                      class="slider-input"
                      type="range"
                      min={Math.min(2, Number(swr) || 2)}
                      max={Math.max(8, Number(swr) || 8)}
                      step="0.1"
                      bind:value={swr}
                    />
                    <input
                      id="projection-swr-input"
                      class="input is-small number-input"
                      type="number"
                      min="0"
                      step="0.1"
                      bind:value={swr}
                      aria-label="Safe Withdrawal Rate"
                    />
                  </div>
                </div>

                <div class="field">
                  <label class="label is-size-7" for="projection-inflation"
                    >Inflation Rate: {formatFloat(inflationRate ?? 0)}%</label
                  >
                  <div class="control-row">
                    <input
                      id="projection-inflation"
                      class="slider-input"
                      type="range"
                      min={Math.min(0, Number(inflationRate) || 0)}
                      max={Math.max(15, Number(inflationRate) || 15)}
                      step="0.1"
                      bind:value={inflationRate}
                    />
                    <input
                      id="projection-inflation-input"
                      class="input is-small number-input"
                      type="number"
                      min="0"
                      step="0.1"
                      bind:value={inflationRate}
                      aria-label="Inflation Rate"
                    />
                  </div>
                </div>

                {#if mode === "simple"}
                  <div class="field mb-0">
                    <label class="label is-size-7" for="projection-spread"
                      >Scenario Spread: ±{formatFloat(cagrSpread ?? 0)}%</label
                    >
                    <div class="control-row">
                      <input
                        id="projection-spread"
                        class="slider-input"
                        type="range"
                        min="0.5"
                        max="15"
                        step="0.5"
                        bind:value={cagrSpread}
                      />
                      <input
                        id="projection-spread-input"
                        class="input is-small number-input"
                        type="number"
                        min="0.5"
                        max="15"
                        step="0.5"
                        bind:value={cagrSpread}
                        aria-label="Scenario Spread"
                      />
                    </div>
                  </div>
                {:else}
                  <div class="field">
                    <label class="label is-size-7" for="mc-volatility"
                      >Volatility (Risk): {formatFloat(volatility)}%</label
                    >
                    <div class="control-row">
                      <input
                        id="mc-volatility"
                        class="slider-input"
                        type="range"
                        min="0"
                        max="40"
                        step="0.5"
                        bind:value={volatility}
                      />
                      <input
                        id="mc-volatility-input"
                        class="input is-small number-input"
                        type="number"
                        min="0"
                        max="40"
                        step="0.5"
                        bind:value={volatility}
                        aria-label="Volatility"
                      />
                    </div>
                  </div>

                  <div class="field">
                    <label class="label is-size-7" for="mc-contrib-growth"
                      >Contribution Growth: {formatFloat(contribGrowth)}%</label
                    >
                    <div class="control-row">
                      <input
                        id="mc-contrib-growth"
                        class="slider-input"
                        type="range"
                        min="0"
                        max="20"
                        step="0.5"
                        bind:value={contribGrowth}
                      />
                      <input
                        id="mc-contrib-growth-input"
                        class="input is-small number-input"
                        type="number"
                        min="0"
                        max="20"
                        step="0.5"
                        bind:value={contribGrowth}
                        aria-label="Contribution Growth"
                      />
                    </div>
                  </div>

                  <div class="field mb-0">
                    <label class="label is-size-7" for="mc-iterations"
                      >Simulations: {iterations.toLocaleString()}</label
                    >
                    <div class="control-row">
                      <input
                        id="mc-iterations"
                        class="slider-input"
                        type="range"
                        min="100"
                        max="10000"
                        step="100"
                        bind:value={iterations}
                      />
                      <input
                        id="mc-iterations-input"
                        class="input is-small number-input"
                        type="number"
                        min="100"
                        max="10000"
                        step="100"
                        bind:value={iterations}
                        aria-label="Simulations"
                      />
                    </div>
                  </div>
                {/if}
              </div>
            {/if}
          </div>
        </div>
      </div>

      <div class="column is-8">
        <div class="box overflow-x-auto">
          {#if mode === "simple"}
            <LegendCard {legends} clazz="mb-2 overflow-x-auto" />
            <svg bind:this={svg} height={440} width="100%" />
          {:else if simLoading}
            <div class="has-text-centered py-6">
              <span class="icon is-large">
                <i class="fas fa-spinner fa-pulse fa-2x"></i>
              </span>
              <p class="mt-3 has-text-grey">Running {iterations.toLocaleString()} simulations…</p>
            </div>
          {:else if simError}
            <div class="notification is-danger is-light">
              <p>{simError}</p>
            </div>
          {:else}
            <div
              bind:this={fanChartContainer}
              class="fan-chart-container"
              style="position: relative;"
            ></div>
          {/if}
        </div>
      </div>
    </div>

    <!-- Bottom Section: Year-by-Year Table (Simple) or Goal Funding Probabilities (Monte Carlo) -->
    {#if mode === "simple" && projectionScenarioTables && activeScenario}
      <div class="box mt-4">
        <div
          class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-3 mb-3"
        >
          <div>
            <h3 class="title is-5 mb-1">Year-by-Year Projection</h3>
            <p class="is-size-7 has-text-grey mb-0">
              {calcAdjustForInflation
                ? "Values adjusted for inflation (real terms)."
                : "Nominal values with expenses and FIRE target growing by inflation."}
            </p>
          </div>

          <div class="is-flex is-align-items-center is-flex-wrap-wrap gap-2">
            <div class="tabs is-toggle is-small mb-0">
              <ul>
                <li class={activeScenarioKey === "expected" ? "is-active" : ""}>
                  <a
                    href="#expected"
                    onclick={(e) => {
                      e.preventDefault();
                      activeScenarioKey = "expected";
                    }}>Expected</a
                  >
                </li>
                <li class={activeScenarioKey === "conservative" ? "is-active" : ""}>
                  <a
                    href="#conservative"
                    onclick={(e) => {
                      e.preventDefault();
                      activeScenarioKey = "conservative";
                    }}>Conservative</a
                  >
                </li>
                <li class={activeScenarioKey === "optimistic" ? "is-active" : ""}>
                  <a
                    href="#optimistic"
                    onclick={(e) => {
                      e.preventDefault();
                      activeScenarioKey = "optimistic";
                    }}>Optimistic</a
                  >
                </li>
              </ul>
            </div>

            <span class="tag is-light">
              {formatFloat(activeScenario.cagr)}% {calcAdjustForInflation ? "Real " : ""}CAGR
            </span>
            <button
              type="button"
              class="button is-small is-light"
              onclick={() => exportScenarioCSV(activeScenario)}
            >
              <span class="icon is-small"><i class="fa-solid fa-file-csv"></i></span>
              <span>Export CSV</span>
            </button>
          </div>
        </div>

        <div class="table-container">
          <table class="table is-striped is-hoverable is-fullwidth is-size-7">
            <thead>
              <tr>
                <th>Year</th>
                <th class="has-text-right">Starting Corpus</th>
                <th class="has-text-right">Annual Contribution</th>
                <th class="has-text-right">Return</th>
                <th class="has-text-right">Ending Corpus</th>
                <th class="has-text-right">Annual Expenses</th>
              </tr>
            </thead>
            <tbody>
              {#each activeScenario.rows as row}
                <tr class={row.year === firstFireYearInTable ? "is-fire-row" : ""}>
                  <td>
                    {row.year}
                    {#if row.year === firstFireYearInTable}
                      <span class="tag is-primary is-light is-small ml-1">FIRE</span>
                    {/if}
                  </td>
                  <td class="has-text-right">{formatCurrency(row.startingCorpus)}</td>
                  <td class="has-text-right">{formatCurrency(row.annualContribution)}</td>
                  <td
                    class={`has-text-right ${row.returnAmount >= 0 ? "has-text-success" : "has-text-danger"}`}
                  >
                    {formatCurrency(row.returnAmount)}
                  </td>
                  <td class="has-text-right has-text-weight-semibold">
                    {formatCurrency(row.endingCorpus)}
                  </td>
                  <td class="has-text-right has-text-grey">{formatCurrency(row.expenses)}</td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      </div>
    {:else if mode === "montecarlo" && simData?.goals && simData.goals.length > 0 && !simLoading}
      <div class="box mt-4">
        <div
          class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-2 mb-3"
        >
          <h3 class="title is-5 mb-0">Life Goal Funding Probabilities</h3>
          <a class="button is-small is-light" href={appPath("/planning/goals?tab=life")}
            >Edit Life Goals</a
          >
        </div>
        <div class="columns is-multiline">
          {#each simData.goals as goal}
            <div class="column is-4">
              <div class="dna-card">
                <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
                  <strong>{goal.name}</strong>
                  <span
                    class="tag is-small {goal.probability >= 0.75
                      ? 'is-success'
                      : goal.probability >= 0.45
                        ? 'is-warning'
                        : 'is-danger'} is-light"
                  >
                    {Math.round(goal.probability * 100)}%
                  </span>
                </div>
                <div class="dna-meta">
                  Target: {formatCurrency(goal.target_amount)}
                  {#if goal.target_date || goal.end_date}
                    · {goal.target_date || goal.end_date}
                  {/if}
                </div>
              </div>
            </div>
          {/each}
        </div>
      </div>
    {/if}
  </div>
</section>

<style>
  .outcome-card {
    height: 100%;
    padding: 1rem 1.25rem;
  }
  .outcome-label {
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    opacity: 0.65;
    margin-bottom: 0.25rem;
  }
  .outcome-value {
    font-size: 1.5rem;
    font-weight: 700;
    line-height: 1.2;
  }
  .outcome-sub {
    font-size: 0.75rem;
    opacity: 0.7;
    margin-top: 0.25rem;
  }
  .assumptions-section {
    border-top: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
  .assumptions-toggle {
    display: flex;
    justify-content: space-between;
    align-items: center;
    width: 100%;
    background: transparent;
    border: 0;
    padding: 0;
    cursor: pointer;
    color: inherit;
    text-align: left;
  }
  .control-row {
    display: flex !important;
    align-items: center !important;
    gap: 0.5rem;
    width: 100%;
    min-width: 0;
    box-sizing: border-box;
  }
  .slider-input {
    flex: 1 1 auto !important;
    min-width: 0 !important;
    width: auto !important;
    cursor: pointer;
    box-sizing: border-box;
  }
  .number-input {
    flex: 0 0 6.5rem !important;
    width: 6.5rem !important;
    max-width: 6.5rem !important;
    min-width: 6.5rem !important;
    height: 2rem !important;
    min-height: 2rem !important;
    padding: 0.25rem 0.5rem !important;
    font-size: 0.85rem !important;
    text-align: right;
    box-sizing: border-box;
  }
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
  .is-fire-row {
    background-color: rgba(72, 95, 199, 0.08) !important;
  }
</style>
