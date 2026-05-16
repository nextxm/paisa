<script lang="ts">
  import {
    ajax,
    formatCurrency,
    formatFloat,
    isMobile,
    type Legend,
    type Networth,
    type NetworthProjectionResponse
  } from "$lib/utils";
  import COLORS from "$lib/colors";
  import { renderNetworth } from "$lib/networth";
  import _ from "lodash";
  import { onDestroy, onMount } from "svelte";
  import { dateRange, setAllowedDateRange } from "../../../../store";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let networth = $state(0);
  let investment = $state(0);
  let gain = $state(0);
  let contribution = $state(0);
  let investmentReturn = $state(0);
  let fxImpact = $state(0);
  let xirr = $state(0);
  let svg: Element = $state();
  let destroy: () => void;
  let points: Networth[] = $state([]);
  let legends: Legend[] = $state([]);
  let showFXImpact = $state(true);
  let showProjection = $state(false);
  let reportCurrency = $state("");
  let availableCurrencies: string[] = $state([]);
  let projection: NetworthProjectionResponse | null = $state(null);

  $effect(() => {
    if (!_.isEmpty(points)) {
      if (destroy) {
        destroy();
      }

      ({ destroy, legends } = renderNetworth(
        _.filter(
          points,
          (p) => p.date.isSameOrBefore($dateRange.to) && p.date.isSameOrAfter($dateRange.from)
        ),
        svg,
        {
          showFXImpact,
          projections: showProjection
            ? [
                {
                  label: "Conservative Projection",
                  color: COLORS.lossText,
                  points: projection?.projection.conservative || []
                },
                {
                  label: "Expected Projection",
                  color: COLORS.primary,
                  points: projection?.projection.expected || []
                },
                {
                  label: "Optimistic Projection",
                  color: COLORS.gainText,
                  points: projection?.projection.optimistic || []
                }
              ]
            : [],
          milestones: showProjection ? projection?.milestones || [] : []
        }
      ));
    }
  });

  onDestroy(async () => {
    if (destroy) {
      destroy();
    }
  });

  async function fetchNetworth() {
    const params = new URLSearchParams();
    if (reportCurrency) params.set("report_currency", reportCurrency);
    const query = params.toString();
    const result = await ajax(query ? `/api/networth?${query}` : "/api/networth");
    points = result.networthTimeline;
    setAllowedDateRange(_.map(points, (p) => p.date));

    const current = _.last(points);
    if (current) {
      networth = current.investmentAmount + current.gainAmount - current.withdrawalAmount;
      investment = current.investmentAmount - current.withdrawalAmount;
      gain = current.gainAmount;
      contribution = current.contribution;
      investmentReturn = current.investment_return;
      fxImpact = current.fx_impact;
    }
    xirr = result.xirr;
  }

  async function fetchProjection() {
    projection = (await ajax("/api/networth/projection")) as NetworthProjectionResponse;
  }

  onMount(async () => {
    const [, currencyResult] = await Promise.all([fetchNetworth(), ajax("/api/price/currencies")]);
    availableCurrencies = currencyResult.currencies || [];
  });

  $effect(() => {
    if (showProjection && !projection) {
      fetchProjection();
    }
  });
</script>

<section class="section tab-networth">
  <div class="container is-fluid">
    {#if availableCurrencies.length > 1}
      <div class="box p-3 mb-4">
        <div class="field is-grouped is-grouped-multiline mb-0">
          <p class="control">
            <span class="select is-small">
              <select
                bind:value={reportCurrency}
                onchange={() => fetchNetworth()}
                title="Report Currency"
              >
                <option value="">Default Currency</option>
                {#each availableCurrencies as currency}
                  <option value={currency}>{currency}</option>
                {/each}
              </select>
            </span>
          </p>
          {#if reportCurrency}
            <p class="control">
              <button
                class="button is-small is-light"
                onclick={() => {
                  reportCurrency = "";
                  fetchNetworth();
                }}
              >
                <span class="icon is-small"><i class="fas fa-times"></i></span>
                <span>Reset Currency</span>
              </button>
            </p>
          {/if}
        </div>
      </div>
    {/if}
    <div class="box p-3 mb-4">
      <div class="field mb-0">
        <input id="project-forward" type="checkbox" bind:checked={showProjection} class="switch" />
        <label for="project-forward">Project Forward</label>
      </div>
      {#if showProjection && projection?.milestones?.[0]}
        <p class="is-size-7 has-text-grey mt-2">
          {projection.milestones[0].label} by {projection.milestones[0].date.format("MMM YYYY")}
        </p>
      {/if}
    </div>
    <nav class="level {isMobile() && 'grid-2'}">
      <LevelItem title="Net worth" color={COLORS.primary} value={formatCurrency(networth)} />
      <LevelItem
        title="Net Investment"
        color={COLORS.secondary}
        value={formatCurrency(investment)}
      />
      <LevelItem
        title="Gain / Loss"
        color={gain >= 0 ? COLORS.gainText : COLORS.lossText}
        value={formatCurrency(gain)}
      />
      <LevelItem
        title="Contribution"
        color={COLORS.secondary}
        value={formatCurrency(contribution)}
      />
      <LevelItem
        title="Investment Return"
        color={COLORS.primary}
        value={formatCurrency(investmentReturn)}
      />
      <LevelItem title="FX Impact" color={COLORS.tertiary} value={formatCurrency(fxImpact)} />
      <LevelItem title="XIRR" value={formatFloat(xirr)} />
    </nav>
  </div>
</section>

<section class="section tab-networth">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto">
          <ZeroState item={points}>
            <strong>Oops!</strong> You have no transactions.
          </ZeroState>
          <div class="is-flex is-justify-content-flex-end pr-4 pt-2">
            <label class="checkbox is-size-7">
              <input type="checkbox" bind:checked={showFXImpact} />
              <span class="ml-2">Show FX impact overlay</span>
            </label>
          </div>

          <LegendCard {legends} clazz="ml-4" />
          <svg id="d3-networth-timeline" height="500" bind:this={svg} />
        </div>
      </div>
    </div>
    <BoxLabel text="Networth Timeline" />
  </div>
</section>
