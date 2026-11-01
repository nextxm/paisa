<script lang="ts">
  import COLORS from "$lib/colors";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import { renderNetworth } from "$lib/networth";
  import {
    ajax,
    formatCurrency,
    formatFloat,
    type FinancialProfile,
    type Legend,
    type Networth,
    type NetworthProjectionResponse
  } from "$lib/utils";
  import { onDestroy, onMount } from "svelte";
  import dayjs from "dayjs";
  import { debounce } from "lodash";

  let svg: Element | undefined = $state();
  let destroy: () => void;
  let legends: Legend[] = $state([]);
  let points: Networth[] = $state([]);
  let baseData: NetworthProjectionResponse | null = $state(null);
  let dnaProfile: FinancialProfile | null = $state(null);

  let years = $state(15);
  let conservativeCagr = $state(6);
  let expectedCagr = $state(9);
  let optimisticCagr = $state(12);
  let monthlyContribution = $state(0);
  let swr = $state(4);
  let inflationRate = $state(6);
  let controlsInitialized = $state(false);

  let calcYears = $state(15);
  let calcConservativeCagr = $state(6);
  let calcExpectedCagr = $state(9);
  let calcOptimisticCagr = $state(12);
  let calcMonthlyContribution = $state(0);
  let calcSwr = $state(4);
  let calcInflationRate = $state(6);

  type ProjectionTableRow = {
    year: number;
    startingCorpus: number;
    annualContribution: number;
    returnAmount: number;
    endingCorpus: number;
    expenses: number;
  };

  type ProjectionScenarioTable = {
    label: string;
    cagr: number;
    rows: ProjectionTableRow[];
  };

  function buildProjectionRows(
    startingCorpus: number,
    annualExpenses: number,
    inflationPercent: number,
    cagrPercent: number,
    monthlyContribution: number,
    years: number
  ): ProjectionTableRow[] {
    const rows: ProjectionTableRow[] = [];
    let currentCorpus = startingCorpus;
    const annualContribution = monthlyContribution * 12;
    const cagr = cagrPercent / 100;
    const monthlyRate = Math.pow(1 + cagr, 1 / 12) - 1;

    for (let year = 1; year <= years; year++) {
      const inflatedExpenses =
        annualExpenses * Math.pow(1 + inflationPercent / 100, Math.max(year - 1, 0));
      const yearStartCorpus = currentCorpus;
      let yearlyReturn = 0;

      for (let m = 0; m < 12; m++) {
        const monthReturn = currentCorpus * monthlyRate;
        yearlyReturn += monthReturn;
        currentCorpus = currentCorpus + monthReturn + monthlyContribution;
      }

      const endingCorpus = currentCorpus;

      rows.push({
        year,
        startingCorpus: yearStartCorpus,
        annualContribution,
        returnAmount: yearlyReturn,
        endingCorpus,
        expenses: inflatedExpenses
      });
    }

    return rows;
  }

  function calibrateSliders() {
    if (!dnaProfile) return;

    if (dnaProfile.monthly_contribution !== undefined && dnaProfile.monthly_contribution !== null) {
      monthlyContribution = Math.min(
        1000000,
        Math.max(-100000, Math.round(dnaProfile.monthly_contribution))
      );
    }

    if (dnaProfile.price_months_covered >= 3 && dnaProfile.historical_return > 0) {
      const returnVal = Math.round(dnaProfile.historical_return * 10) / 10;
      expectedCagr = Math.min(35, Math.max(-5, returnVal));
      const spread =
        dnaProfile.return_volatility > 0 ? Math.round(dnaProfile.return_volatility * 10) / 10 : 3;
      conservativeCagr = Math.min(30, Math.max(-5, Math.round((expectedCagr - spread) * 10) / 10));
      optimisticCagr = Math.min(40, Math.max(-5, Math.round((expectedCagr + spread) * 10) / 10));
    }

    if (dnaProfile.expense_years_covered >= 2 && dnaProfile.expense_growth_rate > 0) {
      const expenseRate = Math.round(dnaProfile.expense_growth_rate * 10) / 10;
      inflationRate = Math.min(15, Math.max(0, expenseRate));
    }

    calcMonthlyContribution = monthlyContribution;
    calcExpectedCagr = expectedCagr;
    calcConservativeCagr = conservativeCagr;
    calcOptimisticCagr = optimisticCagr;
    calcInflationRate = inflationRate;
    updateCalculations.flush();
  }

  function parseVal(val: number | null | undefined, fallback: number): number {
    if (val === null || val === undefined || isNaN(Number(val))) {
      return fallback;
    }
    return Number(val);
  }

  function exportScenarioCSV(scenario: ProjectionScenarioTable) {
    const headers = [
      "Year",
      "Starting Corpus",
      "Annual Contribution",
      "Return",
      "Ending Corpus",
      "Inflated Expenses"
    ];
    const round = (val: number) => Math.round((val || 0) * 100) / 100;
    const csvContent = [
      headers.join(","),
      ...scenario.rows.map((row) =>
        [
          row.year,
          round(row.startingCorpus),
          round(row.annualContribution),
          round(row.returnAmount),
          round(row.endingCorpus),
          round(row.expenses)
        ].join(",")
      )
    ].join("\n");

    const blob = new Blob([csvContent], { type: "text/csv;charset=utf-8;" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    const slug = scenario.label
      .toLowerCase()
      .replace(/[^a-z0-9]+/g, "-")
      .replace(/(^-|-$)/g, "");
    link.href = url;
    link.download = `paisa-projection-${slug || "scenario"}.csv`;
    link.click();
    URL.revokeObjectURL(url);
  }

  const updateCalculations = debounce(() => {
    calcYears = Math.max(1, parseVal(years, 15));
    calcConservativeCagr = parseVal(conservativeCagr, 6);
    calcExpectedCagr = parseVal(expectedCagr, 9);
    calcOptimisticCagr = parseVal(optimisticCagr, 12);
    calcMonthlyContribution = parseVal(monthlyContribution, 0);
    calcSwr = parseVal(swr, 4);
    calcInflationRate = parseVal(inflationRate, 6);
  }, 100);

  $effect(() => {
    const _y = years;
    const _cc = conservativeCagr;
    const _ec = expectedCagr;
    const _oc = optimisticCagr;
    const _mc = monthlyContribution;
    const _s = swr;
    const _i = inflationRate;

    if (controlsInitialized) {
      updateCalculations();
    }
  });

  function projectNetworth(
    startDate: dayjs.Dayjs,
    currentNetworth: number,
    monthlyContribution: number,
    cagrPercent: number,
    months: number
  ) {
    if (months <= 0) return [];
    const cagr = cagrPercent / 100;
    const monthlyRate = Math.pow(1 + cagr, 1 / 12) - 1;
    const points = [];
    let current = currentNetworth;
    for (let i = 1; i <= months; i++) {
      current = current * (1 + monthlyRate) + monthlyContribution;
      points.push({
        date: startDate.add(i, "month"),
        balanceAmount: Math.round(current * 100) / 100
      });
    }
    return points;
  }

  function firstCrossingDate(
    points: { date: dayjs.Dayjs; balanceAmount: number }[],
    threshold: number
  ): dayjs.Dayjs | null {
    for (const p of points) {
      if (p.balanceAmount >= threshold) {
        return p.date;
      }
    }
    return null;
  }

  function monthDiff(from: dayjs.Dayjs, to: dayjs.Dayjs): number {
    const months = (to.year() - from.year()) * 12 + (to.month() - from.month());
    return months < 0 ? 0 : months;
  }

  function projectionMilestones(
    expected: { date: dayjs.Dayjs; balanceAmount: number }[],
    fireTarget: number
  ) {
    const milestones = [];
    const oneCrore = 10000000;
    const cDate1 = firstCrossingDate(expected, oneCrore);
    if (cDate1) {
      milestones.push({
        label: "You will hit 1Cr",
        date: cDate1,
        amount: oneCrore
      });
    }
    if (fireTarget > 0) {
      const cDate2 = firstCrossingDate(expected, fireTarget);
      if (cDate2) {
        milestones.push({
          label: "FIRE target reached",
          date: cDate2,
          amount: Math.round(fireTarget * 100) / 100
        });
      }
    }
    return milestones;
  }

  const projection = $derived.by(() => {
    if (!baseData) return null;

    const currentNetworth = baseData.current_networth;
    const annualExpenses = baseData.annual_expenses;
    const now = dayjs();
    const months = calcYears * 12;

    const conservative = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      calcConservativeCagr,
      months
    );
    const expected = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      calcExpectedCagr,
      months
    );
    const optimistic = projectNetworth(
      now,
      currentNetworth,
      calcMonthlyContribution,
      calcOptimisticCagr,
      months
    );

    let targetCorpus = 0;
    let fireProgress = 0;
    let yearsToFIRE: number | null = null;
    if (calcSwr > 0) {
      targetCorpus = annualExpenses / (calcSwr / 100);
      if (targetCorpus > 0) {
        fireProgress = (currentNetworth / targetCorpus) * 100;
        if (fireProgress > 100) {
          fireProgress = 100;
        }
        const crossedDate = firstCrossingDate(expected, targetCorpus);
        if (crossedDate) {
          const monthsToFire = monthDiff(now, crossedDate);
          yearsToFIRE = Math.round((monthsToFire / 12) * 100) / 100;
        }
      }
    }

    const milestones = projectionMilestones(expected, targetCorpus);

    return {
      current_networth: currentNetworth,
      savings_rate: baseData.savings_rate,
      monthly_contribution: calcMonthlyContribution,
      derived_contribution: baseData.derived_contribution,
      annual_expenses: annualExpenses,
      swr: calcSwr,
      target_corpus: targetCorpus,
      years_to_fire: yearsToFIRE,
      fire_progress_percent: fireProgress,
      projection: {
        conservative,
        expected,
        optimistic
      },
      milestones,
      conservative_cagr: calcConservativeCagr,
      expected_cagr: calcExpectedCagr,
      optimistic_cagr: calcOptimisticCagr
    };
  });

  const projectionScenarioTables = $derived.by((): ProjectionScenarioTable[] | null => {
    if (!projection) return null;

    return [
      {
        label: "Conservative Scenario",
        cagr: projection.conservative_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          calcInflationRate,
          projection.conservative_cagr,
          projection.monthly_contribution,
          calcYears
        )
      },
      {
        label: "Expected Scenario",
        cagr: projection.expected_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          calcInflationRate,
          projection.expected_cagr,
          projection.monthly_contribution,
          calcYears
        )
      },
      {
        label: "Optimistic Scenario",
        cagr: projection.optimistic_cagr,
        rows: buildProjectionRows(
          projection.current_networth,
          projection.annual_expenses,
          calcInflationRate,
          projection.optimistic_cagr,
          projection.monthly_contribution,
          calcYears
        )
      }
    ];
  });

  $effect(() => {
    if (svg && points.length > 0) {
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
      conservativeCagr = baseData.conservative_cagr;
      expectedCagr = baseData.expected_cagr;
      optimisticCagr = baseData.optimistic_cagr;
      monthlyContribution = baseData.monthly_contribution;
      swr = baseData.swr;

      calcYears = years;
      calcConservativeCagr = conservativeCagr;
      calcExpectedCagr = expectedCagr;
      calcOptimisticCagr = optimisticCagr;
      calcMonthlyContribution = monthlyContribution;
      calcSwr = swr;
      calcInflationRate = inflationRate;

      controlsInitialized = true;
    }
  });

  onDestroy(() => {
    updateCalculations.cancel();
    if (destroy) {
      destroy();
    }
  });
</script>

<section class="section tab-projection">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-4">
        <div class="box">
          <nav class="level grid-2">
            <LevelItem
              title="Current Net Worth"
              color={COLORS.primary}
              value={formatCurrency(projection?.current_networth || 0)}
            />
            <LevelItem
              title="Monthly Contribution"
              color={COLORS.secondary}
              value={formatCurrency(projection?.monthly_contribution || 0)}
            />
            <LevelItem
              title="Annual Expenses"
              color={COLORS.lossText}
              value={formatCurrency(projection?.annual_expenses || 0)}
            />
            <LevelItem
              title="Target Corpus"
              color={COLORS.tertiary}
              value={formatCurrency(projection?.target_corpus || 0)}
            />
            <LevelItem
              title="Years to FIRE"
              value={projection?.years_to_fire !== null && projection?.years_to_fire !== undefined
                ? `${formatFloat(projection?.years_to_fire || 0)} years`
                : "Not in projection window"}
            />
            <LevelItem
              title="FIRE Progress"
              value={`${formatFloat(projection?.fire_progress_percent || 0)}%`}
            />
          </nav>
        </div>
        <div class="box">
          <div class="field">
            <label class="label is-size-7" for="projection-years"
              >Projection Years: {years ?? ""}</label
            >
            <div class="is-flex is-align-items-center gap-2">
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
            <label class="label is-size-7" for="projection-conservative-cagr"
              >Conservative CAGR: {formatFloat(conservativeCagr ?? 0)}%</label
            >
            <div class="is-flex is-align-items-center gap-2">
              <input
                id="projection-conservative-cagr"
                class="slider-input"
                type="range"
                min={Math.min(-5, Number(conservativeCagr) || -5)}
                max={Math.max(30, Number(conservativeCagr) || 30)}
                step="0.5"
                bind:value={conservativeCagr}
              />
              <input
                id="projection-conservative-cagr-input"
                class="input is-small number-input"
                type="number"
                step="0.1"
                bind:value={conservativeCagr}
                aria-label="Conservative CAGR"
              />
            </div>
          </div>

          <div class="field">
            <label class="label is-size-7" for="projection-expected-cagr"
              >Expected CAGR: {formatFloat(expectedCagr ?? 0)}%</label
            >
            <div class="is-flex is-align-items-center gap-2">
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
          <div class="field">
            <label class="label is-size-7" for="projection-optimistic-cagr"
              >Optimistic CAGR: {formatFloat(optimisticCagr ?? 0)}%</label
            >
            <div class="is-flex is-align-items-center gap-2">
              <input
                id="projection-optimistic-cagr"
                class="slider-input"
                type="range"
                min={Math.min(-5, Number(optimisticCagr) || -5)}
                max={Math.max(40, Number(optimisticCagr) || 40)}
                step="0.5"
                bind:value={optimisticCagr}
              />
              <input
                id="projection-optimistic-cagr-input"
                class="input is-small number-input"
                type="number"
                step="0.1"
                bind:value={optimisticCagr}
                aria-label="Optimistic CAGR"
              />
            </div>
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-monthly-contribution"
              >Monthly Contribution: {formatCurrency(monthlyContribution ?? 0)}</label
            >
            <div class="is-flex is-align-items-center gap-2">
              <input
                id="projection-monthly-contribution"
                class="slider-input"
                type="range"
                min={Math.min(-100000, Number(monthlyContribution) || 0)}
                max={Math.max(1000000, Number(monthlyContribution) || 0)}
                step="1000"
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
            <label class="label is-size-7" for="projection-swr"
              >Safe Withdrawal Rate (SWR): {formatFloat(swr ?? 0)}%</label
            >
            <div class="is-flex is-align-items-center gap-2">
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
            <div class="is-flex is-align-items-center gap-2">
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
        </div>
      </div>
      <div class="column is-8">
        <div class="box overflow-x-auto">
          <LegendCard {legends} clazz="mb-2 overflow-x-auto" />
          <svg bind:this={svg} height={500} width="100%" />
        </div>
      </div>
    </div>

    {#if dnaProfile}
      <div class="box mt-5">
        <div
          class="is-flex is-justify-content-space-between is-align-items-start is-flex-wrap-wrap gap-2 mb-4"
        >
          <div>
            <h3 class="title is-5 mb-2">Your Financial DNA</h3>
            <p class="is-size-7 has-text-grey">
              Parameters inferred from your ledger history. These feed into the projection engine
              and can be used to calibrate the sliders above.
            </p>
          </div>
          <button
            class="button is-small is-primary is-light"
            type="button"
            onclick={calibrateSliders}
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

    {#if projectionScenarioTables}
      <div class="box mt-5">
        <div class="content mb-4">
          <h3 class="title is-5 mb-2">Year-by-Year Projection Tables</h3>
          <p class="is-size-7 has-text-grey">
            Annual contributions represent net savings after expenses. Ending corpus is calculated
            as starting corpus + annual contribution + return. Inflated expenses show the annual
            cost of living and FIRE benchmark adjusted for inflation.
          </p>
        </div>

        {#each projectionScenarioTables as scenario}
          <div class="mb-5">
            <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
              <h4 class="title is-6 mb-0">{scenario.label}</h4>
              <div class="is-flex is-align-items-center gap-2">
                <span class="tag is-light">{formatFloat(scenario.cagr)}% CAGR</span>
                <button
                  type="button"
                  class="button is-small is-light"
                  onclick={() => exportScenarioCSV(scenario)}
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
                    <th class="has-text-right">Inflated Expenses</th>
                  </tr>
                </thead>
                <tbody>
                  {#each scenario.rows as row}
                    <tr>
                      <td>{row.year}</td>
                      <td class="has-text-right">{formatCurrency(row.startingCorpus)}</td>
                      <td class="has-text-right">{formatCurrency(row.annualContribution)}</td>
                      <td
                        class={`has-text-right ${row.returnAmount >= 0 ? "has-text-success" : "has-text-danger"}`}
                      >
                        {formatCurrency(row.returnAmount)}
                      </td>
                      <td class="has-text-right">{formatCurrency(row.endingCorpus)}</td>
                      <td class="has-text-right has-text-grey">{formatCurrency(row.expenses)}</td>
                    </tr>
                  {/each}
                </tbody>
              </table>
            </div>
          </div>
        {/each}
      </div>
    {/if}
  </div>
</section>

<style>
  .slider-input {
    flex: 1;
    min-width: 0;
  }
  .number-input {
    width: 6.5rem;
    flex-shrink: 0;
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
</style>
