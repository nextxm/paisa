<script lang="ts">
  import COLORS from "$lib/colors";
  import LegendCard from "$lib/components/LegendCard.svelte";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import { renderNetworth } from "$lib/networth";
  import {
    ajax,
    formatCurrency,
    formatFloat,
    type Legend,
    type Networth,
    type NetworthProjectionResponse
  } from "$lib/utils";
  import { onDestroy, onMount } from "svelte";
  import dayjs from "dayjs";
  import { debounce } from "lodash";

  let svg: Element = $state();
  let destroy: () => void;
  let legends: Legend[] = $state([]);
  let points: Networth[] = $state([]);
  let baseData: NetworthProjectionResponse | null = $state(null);

  let years = $state(15);
  let conservativeCagr = $state(8);
  let expectedCagr = $state(12);
  let optimisticCagr = $state(16);
  let monthlyContribution = $state(0);
  let swr = $state(4);
  let controlsInitialized = $state(false);

  let calcYears = $state(15);
  let calcConservativeCagr = $state(8);
  let calcExpectedCagr = $state(12);
  let calcOptimisticCagr = $state(16);
  let calcMonthlyContribution = $state(0);
  let calcSwr = $state(4);

  const updateCalculations = debounce(() => {
    calcYears = years;
    calcConservativeCagr = conservativeCagr;
    calcExpectedCagr = expectedCagr;
    calcOptimisticCagr = optimisticCagr;
    calcMonthlyContribution = monthlyContribution;
    calcSwr = swr;
  }, 100);

  $effect(() => {
    const _y = years;
    const _cc = conservativeCagr;
    const _ec = expectedCagr;
    const _oc = optimisticCagr;
    const _mc = monthlyContribution;
    const _s = swr;

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
    const networthResult = await ajax("/api/networth");
    points = networthResult.networthTimeline;

    baseData = (await ajax("/api/networth/projection")) as NetworthProjectionResponse;
    if (baseData) {
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
        </div>
      </div>
      <div class="column is-8">
        <div class="box overflow-x-auto">
          <LegendCard {legends} clazz="mb-2 overflow-x-auto" />
          <svg bind:this={svg} height={500} width="100%" />
        </div>
      </div>
    </div>
  </div>
</section>
