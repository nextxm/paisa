<script lang="ts">
  import COLORS, { generateColorScheme, genericBarColor } from "$lib/colors";
  import { renderAccountOverview, buildLegends } from "$lib/gain";
  import { filterCommodityBreakdowns, renderPortfolioBreakdown } from "$lib/portfolio";
  import {
    ajax,
    type Posting,
    formatCurrency,
    formatFloat,
    type AccountGain,
    type Networth,
    type PortfolioAggregate,
    type AssetBreakdown,
    formatPercentage,
    formatFloatUptoPrecision,
    now
  } from "$lib/utils";
  import {
    buildTrendPath,
    filterTrendPoints,
    trendRangeFromMonths
  } from "$lib/account_balance_trend";
  import _ from "lodash";
  import dayjs from "dayjs";
  import { onMount, onDestroy } from "svelte";
  import type { PageData } from "./$types";
  import PostingCard from "$lib/components/PostingCard.svelte";
  import LevelItem from "$lib/components/LevelItem.svelte";
  import { iconify } from "$lib/icon";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let commodities: string[] = $state([]);
  let selectedCommodities: string[] = $state([]);
  let security_type: PortfolioAggregate[] = $state([]);
  let name_and_security_type: PortfolioAggregate[] = $state([]);
  let rating: PortfolioAggregate[] = $state([]);
  let industry: PortfolioAggregate[] = $state([]);
  let color: any = $state(null);

  let securityTypeEmpty: boolean = $state(false);
  let nameAndSecurityTypeEmpty: boolean = $state(false);
  let ratingEmpty: boolean = $state(false);
  let industryEmpty: boolean = $state(false);

  let { data }: { data: PageData } = $props();
  let gain: AccountGain = $state(null);
  let overview: Networth = $state(null);
  let assetBreakdown: AssetBreakdown = $state(null);
  let legends = $state(buildLegends());

  let destroyCallback = () => {};
  let postings: Posting[] = $state([]);

  let securityTypeR: any = $state(null),
    portfolioR: any = $state(null),
    industryR: any = $state(null),
    ratingR: any = $state(null);
  let asOfDate = $state(now().format("YYYY-MM-DD"));
  let showTrend = $state(false);
  let trendMonths = $state(6);
  const initialTrendRange = trendRangeFromMonths(now().format("YYYY-MM-DD"), 6);
  let trendStart = $state(initialTrendRange.start);
  let trendEnd = $state(initialTrendRange.end);
  const trendSvgWidth = 900;
  const trendSvgHeight = 260;

  function applyTrendPreset(months: number) {
    trendMonths = months;
    const range = trendRangeFromMonths(asOfDate, months);
    trendStart = range.start;
    trendEnd = range.end;
  }

  async function fetchAccountGain() {
    const gainResult = await ajax(
      `/api/gain/${encodeURIComponent(data.name)}?as_of_date=${encodeURIComponent(asOfDate)}`
    );
    gain = gainResult.gain_timeline_breakdown;
    assetBreakdown = gainResult.asset_breakdown;
    ({ name_and_security_type, security_type, rating, industry, commodities } =
      gainResult.portfolio_allocation);

    overview = _.last(gain.networthTimeline);
    postings = _.chain(gain.postings)
      .sortBy((p) => p.date)
      .reverse()
      .take(100)
      .value();
    destroyCallback();
    destroyCallback = renderAccountOverview(
      gain.networthTimeline,
      gain.postings,
      "d3-account-timeline-breakdown"
    );

    selectedCommodities = [...commodities];
    ({ renderer: securityTypeR } = renderPortfolioBreakdown(
      "#d3-portfolio-security-type",
      security_type,
      {
        small: true
      }
    ));
    ({ renderer: ratingR } = renderPortfolioBreakdown("#d3-portfolio-security-rating", rating, {
      small: true
    }));
    ({ renderer: industryR } = renderPortfolioBreakdown(
      "#d3-portfolio-security-industry",
      industry,
      {
        small: true,
        z: [genericBarColor()]
      }
    ));
    ({ renderer: portfolioR } = renderPortfolioBreakdown("#d3-portfolio", name_and_security_type, {
      small: true
    }));

    if (commodities.length !== 0) {
      color = generateColorScheme(commodities);
    }

    securityTypeEmpty = security_type.length === 0;
    nameAndSecurityTypeEmpty = name_and_security_type.length === 0;
    ratingEmpty = rating.length === 0;
    industryEmpty = industry.length === 0;
  }

  onDestroy(async () => {
    destroyCallback();
  });

  onMount(async () => {
    await fetchAccountGain();
  });

  $effect(() => {
    if (securityTypeR) {
      securityTypeR(filterCommodityBreakdowns(security_type, selectedCommodities), color);
      ratingR(filterCommodityBreakdowns(rating, selectedCommodities), color);
      industryR(filterCommodityBreakdowns(industry, selectedCommodities), color);
      portfolioR(filterCommodityBreakdowns(name_and_security_type, selectedCommodities), color);
    }
  });

  let trendPoints = $derived(
    gain?.networthTimeline ? filterTrendPoints(gain.networthTimeline, trendStart, trendEnd) : []
  );
  let trendPath = $derived(buildTrendPath(trendPoints, trendSvgWidth, trendSvgHeight));
  let trendCurrentBalance = $derived(_.last(trendPoints)?.balanceAmount ?? null);
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="columns is-flex-wrap-wrap">
      <div class="column is-3">
        <div class="columns is-flex-wrap-wrap">
          {#if overview}
            <div class="column is-full">
              <div>
                <nav class="level grid-2">
                  <LevelItem
                    narrow
                    title="Balance"
                    color={COLORS.primary}
                    value={formatCurrency(overview.balanceAmount)}
                    href="/accounts/{encodeURIComponent(data.name)}/transactions"
                  />
                  <LevelItem
                    narrow
                    title="Net Investment"
                    color={COLORS.secondary}
                    value={formatCurrency(overview.netInvestmentAmount)}
                  />
                </nav>
              </div>
            </div>
            <div class="column is-full">
              <div>
                <nav class="level grid-2">
                  <LevelItem
                    narrow
                    title="Gain / Loss"
                    color={overview.gainAmount >= 0 ? COLORS.gainText : COLORS.lossText}
                    value={formatCurrency(overview.gainAmount)}
                  />

                  <LevelItem
                    narrow
                    title="XIRR"
                    value={formatFloat(gain.xirr)}
                    subtitle="{formatPercentage(assetBreakdown.absoluteReturn, 2)} absolute return"
                  />
                </nav>
              </div>
            </div>
          {/if}

          <div class="column is-full">
            Postings
            {#each postings as posting}
              <PostingCard
                {posting}
                color={posting.amount >= 0
                  ? posting.account.startsWith("Income:CapitalGains")
                    ? COLORS.loss
                    : COLORS.secondary
                  : posting.account.startsWith("Income:CapitalGains")
                    ? COLORS.gain
                    : COLORS.tertiary}
              />
            {/each}
          </div>
        </div>
      </div>
      <div class="column is-9">
        {#if overview}
          <div class="box py-2 mb-4 mt-0">
            <div class="is-flex mr-2 is-align-items-baseline" style="min-width: fit-content">
              <div class="ml-3 custom-icon is-size-5">
                <span>{iconify(data.name)}</span>
              </div>
              <div class="ml-3">
                <span class="mr-1 is-size-7 has-text-grey">Investment</span>
                <span class="has-text-weight-bold">{formatCurrency(overview.investmentAmount)}</span
                >
              </div>
              <div class="ml-3">
                <span class="mr-1 is-size-7 has-text-grey">As of</span>
                <span class="has-text-weight-bold">{dayjs(asOfDate).format("MMM D, YYYY")}</span>
              </div>
              <div class="ml-3">
                <span class="mr-1 is-size-7 has-text-grey">Withdrawal</span>
                <span class="has-text-weight-bold">{formatCurrency(overview.withdrawalAmount)}</span
                >
              </div>
              {#if overview.balanceUnits > 0}
                <div class="ml-3">
                  <span class="mr-1 is-size-7 has-text-grey">Balance Units</span>
                  <span class="has-text-weight-bold">
                    <a
                      href="/accounts/{encodeURIComponent(data.name)}/transactions"
                      class="has-text-grey-darker"
                    >
                      {formatFloatUptoPrecision(overview.balanceUnits, 4)}
                    </a>
                  </span>
                </div>
              {/if}
            </div>
          </div>
        {/if}
        <LegendCard {legends} clazz="mb-2" />
        <div class="box p-3 mb-2">
          <div class="field is-grouped is-grouped-multiline mb-2">
            <div class="control">
              <label class="label is-size-7 mb-1" for="account-balance-as-of">View as of</label>
              <input
                id="account-balance-as-of"
                class="input is-small"
                type="date"
                bind:value={asOfDate}
                onchange={async () => {
                  applyTrendPreset(trendMonths);
                  await fetchAccountGain();
                }}
              />
            </div>
            <div class="control">
              <button
                class="button is-small is-light mt-5"
                onclick={() => (showTrend = !showTrend)}
              >
                {showTrend ? "Hide Trend" : "View Trend"}
              </button>
            </div>
          </div>
          {#if showTrend}
            <div class="field is-grouped is-grouped-multiline mb-2">
              <div class="control">
                <button
                  class="button is-small {trendMonths === 6 ? 'is-link' : 'is-light'}"
                  onclick={() => applyTrendPreset(6)}
                >
                  6M
                </button>
              </div>
              <div class="control">
                <button
                  class="button is-small {trendMonths === 12 ? 'is-link' : 'is-light'}"
                  onclick={() => applyTrendPreset(12)}
                >
                  12M
                </button>
              </div>
              <div class="control">
                <input class="input is-small" type="date" bind:value={trendStart} />
              </div>
              <div class="control">
                <input class="input is-small" type="date" bind:value={trendEnd} />
              </div>
            </div>
            {#if trendPoints.length > 1}
              <svg viewBox="0 0 {trendSvgWidth} {trendSvgHeight}" width="100%" height="280">
                <path d={trendPath.path} fill="none" stroke={COLORS.primary} stroke-width="2" />
                {#if trendPath.marker}
                  <circle
                    cx={trendPath.marker.x}
                    cy={trendPath.marker.y}
                    r="4"
                    fill={COLORS.gain}
                  />
                {/if}
              </svg>
              {#if trendCurrentBalance !== null}
                <p class="is-size-7 has-text-grey">
                  Current balance marker: {formatCurrency(trendCurrentBalance)}
                </p>
              {/if}
            {:else}
              <p class="is-size-7 has-text-grey">
                No trend data available for selected date range.
              </p>
            {/if}
          {/if}
        </div>
        <div class="box">
          <svg id="d3-account-timeline-breakdown" width="100%" height="450" />
        </div>
        <BoxLabel text="Timeline" />

        <div class="columns">
          <div class="column is-6">
            <div class="mt-5" class:is-hidden={securityTypeEmpty}>
              <div class="box overflow-x-auto">
                <div
                  id="d3-portfolio-security-type-treemap"
                  style="width: 100%; position: relative"
                ></div>
                <svg id="d3-portfolio-security-type" width="100%" />
              </div>
              <BoxLabel text="Security Type" />
            </div>

            <div class="mt-5" class:is-hidden={ratingEmpty}>
              <div class="box overflow-x-auto">
                <div
                  id="d3-portfolio-security-rating-treemap"
                  style="width: 100%; position: relative"
                ></div>
                <svg id="d3-portfolio-security-rating" width="100%" />
              </div>
              <BoxLabel text="Security Rating" />
            </div>

            <div class="mt-5" class:is-hidden={industryEmpty}>
              <div class="box overflow-x-auto">
                <div
                  id="d3-portfolio-security-industry-treemap"
                  style="width: 100%; position: relative"
                ></div>
                <svg id="d3-portfolio-security-industry" width="100%" />
              </div>
              <BoxLabel text="Industry" />
            </div>
          </div>
          <div class="column is-6 mt-5">
            <div class:is-hidden={nameAndSecurityTypeEmpty}>
              <div class="box overflow-x-auto">
                <div id="d3-portfolio-treemap" style="width: 100%; position: relative"></div>
                <svg id="d3-portfolio" width="100%" />
              </div>
              <BoxLabel text="Security" />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</section>
