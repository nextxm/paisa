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

  let svg: Element = $state();
  let destroy: () => void;
  let legends: Legend[] = $state([]);
  let points: Networth[] = $state([]);
  let projection: NetworthProjectionResponse | null = $state(null);

  let years = $state(15);
  let conservativeCagr = $state(8);
  let expectedCagr = $state(12);
  let optimisticCagr = $state(16);
  let monthlyContribution = $state(0);
  let swr = $state(4);
  let controlsInitialized = $state(false);

  async function fetchProjection() {
    const params = new URLSearchParams();
    params.set("years", `${years}`);
    params.set("conservative_cagr", `${conservativeCagr}`);
    params.set("expected_cagr", `${expectedCagr}`);
    params.set("optimistic_cagr", `${optimisticCagr}`);
    params.set("monthly_contribution", `${monthlyContribution}`);
    params.set("swr", `${swr}`);
    projection = (await ajax(
      `/api/networth/projection?${params.toString()}`
    )) as NetworthProjectionResponse;

    if (!controlsInitialized && projection) {
      years = Math.round(projection.projection.expected.length / 12);
      conservativeCagr = projection.conservative_cagr;
      expectedCagr = projection.expected_cagr;
      optimisticCagr = projection.optimistic_cagr;
      monthlyContribution = projection.monthly_contribution;
      swr = projection.swr;
      controlsInitialized = true;
    }
  }

  async function refreshProjection() {
    if (controlsInitialized) {
      await fetchProjection();
    }
  }

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
    await fetchProjection();
  });

  onDestroy(() => {
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
          <nav class="level grid-1">
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
              oninput={refreshProjection}
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
              oninput={refreshProjection}
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
              oninput={refreshProjection}
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
              oninput={refreshProjection}
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
              oninput={refreshProjection}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="projection-swr"
              >Safe Withdrawal Rate (SWR): {formatFloat(swr)}%</label
            >
            <input
              id="projection-swr"
              type="range"
              min="2"
              max="8"
              step="0.1"
              bind:value={swr}
              oninput={refreshProjection}
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
  </div>
</section>
