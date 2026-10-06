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
    expenses: number;
    returnAmount: number;
    endingCorpus: number;
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
    const annualReturnRate = cagrPercent / 100;

    for (let year = 1; year <= years; year++) {
      const inflatedExpenses =
        annualExpenses * Math.pow(1 + inflationPercent / 100, Math.max(year - 1, 0));
      const investmentReturn = currentCorpus * annualReturnRate;
      const endingCorpus = currentCorpus + investmentReturn + annualContribution - inflatedExpenses;

      rows.push({
        year,
        startingCorpus: currentCorpus,
        expenses: inflatedExpenses,
        returnAmount: investmentReturn,
        endingCorpus
      });

      currentCorpus = endingCorpus;
    }

    return rows;
  }

  const updateCalculations = debounce(() => {
    calcYears = years;
    calcConservativeCagr = conservativeCagr;
    calcExpectedCagr = expectedCagr;
    calcOptimisticCagr = optimisticCagr;
    calcMonthlyContribution = monthlyContribution;
    calcSwr = swr;
    calcInflationRate = inflationRate;
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
            <label class="label is-size-7" for="projection-years">Projection Years: {years}</label>
            <input
              id="projection-years"
              type="range"
              min="1"
              max="40"
              step="1"
              bind:value={years}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-conservative-cagr"
              >Conservative CAGR: {formatFloat(conservativeCagr)}%</label
            >

            <input
              id="projection-conservative-cagr"
              type="range"
              min="-5"
              max="30"
              step="0.5"
              bind:value={conservativeCagr}
            />
          </div>

          <div class="field">
            <label class="label is-size-7" for="projection-expected-cagr"
              >Expected CAGR: {formatFloat(expectedCagr)}%</label
            >
            <input
              id="projection-expected-cagr"
              type="range"
              min="-5"
              max="35"
              step="0.5"
              bind:value={expectedCagr}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-optimistic-cagr"
              >Optimistic CAGR: {formatFloat(optimisticCagr)}%</label
            >
            <input
              id="projection-optimistic-cagr"
              type="range"
              min="-5"
              max="40"
              step="0.5"
              bind:value={optimisticCagr}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-monthly-contribution"
              >Monthly Contribution: {formatCurrency(monthlyContribution)}</label
            >
            <input
              id="projection-monthly-contribution"
              type="range"
              min="-100000"
              max="1000000"
              step="1000"
              bind:value={monthlyContribution}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-swr"
              >Safe Withdrawal Rate (SWR): {formatFloat(swr)}%</label
            >
            <input id="projection-swr" type="range" min="2" max="8" step="0.1" bind:value={swr} />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-inflation"
              >Inflation Rate: {formatFloat(inflationRate)}%</label
            >
            <input
              id="projection-inflation"
              type="range"
              min="0"
              max="15"
              step="0.1"
              bind:value={inflationRate}
            />
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
        <div class="content mb-4">
          <h3 class="title is-5 mb-2">Your Financial DNA</h3>
          <p class="is-size-7 has-text-grey">
            Parameters inferred from your ledger history. These feed into the projection engine and
            can be used to calibrate the sliders above.
          </p>
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
            Expenses are inflated year over year using the selected inflation rate. Ending corpus is
            calculated as starting corpus + return + annual contribution - annual expenses.
          </p>
        </div>

        {#each projectionScenarioTables as scenario}
          <div class="mb-5">
            <div class="is-flex is-justify-content-space-between is-align-items-baseline mb-2">
              <h4 class="title is-6 mb-0">{scenario.label}</h4>
              <span class="tag is-light">{formatFloat(scenario.cagr)}% CAGR</span>
            </div>

            <div class="table-container">
              <table class="table is-striped is-hoverable is-fullwidth is-size-7">
                <thead>
                  <tr>
                    <th>Year</th>
                    <th class="has-text-right">Starting Corpus</th>
                    <th class="has-text-right">Expenses</th>
                    <th class="has-text-right">Return</th>
                    <th class="has-text-right">Ending Corpus</th>
                  </tr>
                </thead>
                <tbody>
                  {#each scenario.rows as row}
                    <tr>
                      <td>{row.year}</td>
                      <td class="has-text-right">{formatCurrency(row.startingCorpus)}</td>
                      <td class="has-text-right">{formatCurrency(row.expenses)}</td>
                      <td
                        class={`has-text-right ${row.returnAmount >= 0 ? "has-text-success" : "has-text-danger"}`}
                      >
                        {formatCurrency(row.returnAmount)}
                      </td>
                      <td class="has-text-right">{formatCurrency(row.endingCorpus)}</td>
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
