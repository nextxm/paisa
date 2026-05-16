<script lang="ts">
  import COLORS from "$lib/colors";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import SparklineChart from "$lib/components/SparklineChart.svelte";
  import {
    ajax,
    formatCurrency,
    formatPercentage,
    type InvestmentIncomeHolding,
    type InvestmentIncomeTimelinePoint
  } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";

  let holdings: InvestmentIncomeHolding[] = $state([]);
  let timeline: InvestmentIncomeTimelinePoint[] = $state([]);
  let ttmTotal = $state(0);
  let dividendTTM = $state(0);
  let interestTTM = $state(0);
  let distributionTTM = $state(0);

  function yearlyMapForSparkline(holding: InvestmentIncomeHolding): Record<string, number> {
    return _.chain(holding.yearly_income)
      .toPairs()
      .sortBy(([year]) => year)
      .fromPairs()
      .value();
  }

  onMount(async () => {
    const response = await ajax("/api/income/investment");
    ({ holdings, timeline, ttm_total: ttmTotal } = response);
    dividendTTM = _.sumBy(response.income_by_type.Dividend ?? [], (h) => h.ttm_income);
    interestTTM = _.sumBy(response.income_by_type.Interest ?? [], (h) => h.ttm_income);
    distributionTTM = _.sumBy(response.income_by_type.Distribution ?? [], (h) => h.ttm_income);
  });
</script>

<section class="section tab-income">
  <div class="container">
    <nav class="level grid-4">
      <LevelItem
        title="TTM Investment Income"
        value={formatCurrency(ttmTotal)}
        color={COLORS.gainText}
      />
      <LevelItem
        title="Dividend (TTM)"
        value={formatCurrency(dividendTTM)}
        color={COLORS.primary}
      />
      <LevelItem
        title="Interest (TTM)"
        value={formatCurrency(interestTTM)}
        color={COLORS.secondary}
      />
      <LevelItem
        title="Distribution (TTM)"
        value={formatCurrency(distributionTTM)}
        color={COLORS.tertiary}
      />
    </nav>
  </div>
</section>

<section class="section">
  <div class="container is-fluid">
    <div class="box overflow-x-auto">
      <p class="is-size-6 mb-2">Investment Income by Holding</p>
      <table class="table is-fullwidth is-hoverable is-striped is-narrow">
        <thead>
          <tr>
            <th>Holding</th>
            <th>Type</th>
            <th class="has-text-right">Total Income</th>
            <th class="has-text-right">TTM Income</th>
            <th class="has-text-right">Current Balance</th>
            <th class="has-text-right">TTM Yield</th>
            <th>Dividend Growth</th>
          </tr>
        </thead>
        <tbody>
          {#each holdings as holding}
            <tr>
              <td>{holding.holding}</td>
              <td>{holding.type}</td>
              <td class="has-text-right">{formatCurrency(holding.total_income)}</td>
              <td class="has-text-right">{formatCurrency(holding.ttm_income)}</td>
              <td class="has-text-right">{formatCurrency(holding.current_balance)}</td>
              <td class="has-text-right">{formatPercentage(holding.ttm_yield, 2)}</td>
              <td>
                <SparklineChart
                  data={yearlyMapForSparkline(holding)}
                  color={COLORS.gain}
                  height={28}
                  width={120}
                />
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
</section>

<section class="section">
  <div class="container is-fluid">
    <div class="box overflow-x-auto">
      <p class="is-size-6 mb-2">Investment Income Timeline</p>
      <table class="table is-fullwidth is-hoverable is-striped is-narrow">
        <thead>
          <tr>
            <th>Month</th>
            <th class="has-text-right">Dividend</th>
            <th class="has-text-right">Interest</th>
            <th class="has-text-right">Distribution</th>
            <th class="has-text-right">Total</th>
          </tr>
        </thead>
        <tbody>
          {#each timeline as point}
            <tr>
              <td>{point.date.format("MMM YYYY")}</td>
              <td class="has-text-right">{formatCurrency(point.dividend)}</td>
              <td class="has-text-right">{formatCurrency(point.interest)}</td>
              <td class="has-text-right">{formatCurrency(point.distribution)}</td>
              <td class="has-text-right">{formatCurrency(point.total)}</td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  </div>
</section>
