<script lang="ts">
  import { onMount } from "svelte";
  import ScenarioComparison from "$lib/components/ScenarioComparison.svelte";
  import {
    ajax,
    formatCurrency,
    formatFloat,
    type WhatIfResponse,
    type WhatIfScenarioRequest
  } from "$lib/utils";

  type BaselineForm = {
    projectionYears: number;
    monthlyContribution: number;
    contributionGrowthRate: number;
    expectedReturn: number;
    returnVolatility: number;
    inflationRate: number;
    swr: number;
    iterations: number;
  };

  type ScenarioTemplate = {
    label: string;
    description: string;
    scenario: WhatIfScenarioRequest;
  };

  const templates: ScenarioTemplate[] = [
    {
      label: "Increase SIP by 10k",
      description: "Model a higher monthly contribution with the same return assumptions.",
      scenario: { name: "Increase SIP", overrides: { monthly_contribution: 10000 } }
    },
    {
      label: "Job loss for 6 months",
      description:
        "Approximate a temporary income shock as paused contributions and slower growth.",
      scenario: {
        name: "Job Loss",
        overrides: { monthly_contribution: 0, contribution_growth_rate: 0, expected_return: 8 }
      }
    },
    {
      label: "Early retirement",
      description:
        "Stress test the plan with lower withdrawal flexibility and weaker contributions.",
      scenario: {
        name: "Early Retirement",
        overrides: { monthly_contribution: 0, swr: 3.5, inflation_rate: 7 }
      }
    }
  ];

  let baseline = $state<BaselineForm>({
    projectionYears: 30,
    monthlyContribution: 0,
    contributionGrowthRate: 5,
    expectedReturn: 12,
    returnVolatility: 18,
    inflationRate: 6,
    swr: 4,
    iterations: 1000
  });

  let scenarios = $state<WhatIfScenarioRequest[]>([
    structuredClone(templates[0].scenario),
    structuredClone(templates[1].scenario)
  ]);
  let comparison = $state<WhatIfResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function runComparison() {
    loading = true;
    error = null;

    try {
      comparison = await ajax("/api/projection/whatif", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          baseline: {
            iterations: baseline.iterations,
            months_to_project: baseline.projectionYears * 12,
            monthly_contribution: baseline.monthlyContribution,
            contribution_growth_rate: baseline.contributionGrowthRate,
            expected_return: baseline.expectedReturn,
            return_volatility: baseline.returnVolatility,
            inflation_rate: baseline.inflationRate,
            swr: baseline.swr
          },
          scenarios: scenarios.slice(0, 5)
        })
      });

      if (
        comparison?.profile &&
        baseline.monthlyContribution === 0 &&
        comparison.profile.monthly_contribution > 0
      ) {
        baseline = {
          ...baseline,
          monthlyContribution: Math.round(comparison.profile.monthly_contribution),
          contributionGrowthRate:
            comparison.profile.income_growth_rate > 0
              ? Math.round(comparison.profile.income_growth_rate * 10) / 10
              : baseline.contributionGrowthRate,
          expectedReturn:
            comparison.profile.historical_return > 0
              ? Math.round(comparison.profile.historical_return * 10) / 10
              : baseline.expectedReturn,
          returnVolatility:
            comparison.profile.return_volatility > 0
              ? Math.round(comparison.profile.return_volatility * 10) / 10
              : baseline.returnVolatility
        };
      }
    } catch (exception) {
      error = exception instanceof Error ? exception.message : "What-if comparison failed";
    } finally {
      loading = false;
    }
  }

  function addTemplate(template: ScenarioTemplate) {
    if (scenarios.length >= 5) return;
    scenarios = [...scenarios, structuredClone(template.scenario)];
  }

  function addBlankScenario() {
    if (scenarios.length >= 5) return;
    scenarios = [
      ...scenarios,
      {
        name: `Scenario ${scenarios.length + 1}`,
        overrides: {}
      }
    ];
  }

  function removeScenario(index: number) {
    scenarios = scenarios.filter((_, current) => current !== index);
  }

  function fieldId(index: number, field: string) {
    return `whatif-scenario-${index}-${field}`;
  }

  onMount(runComparison);
</script>

<section class="section">
  <div class="container is-fluid">
    <div
      class="is-flex is-justify-content-space-between is-align-items-flex-start mb-5 whatif-header"
    >
      <div>
        <h1 class="title is-4 is-spaced mb-2">What-If Scenarios</h1>
        <p class="subtitle is-6 has-text-grey mb-0">
          Compare your baseline plan against up to five alternate trajectories using the same
          goal-aware simulation engine.
        </p>
      </div>
      <div class="buttons">
        <button class:is-loading={loading} class="button is-primary" onclick={runComparison}
          >Run Comparison</button
        >
        <a class="button is-light" href="/planning/life/drawdown">Tax-Aware Drawdown</a>
        <a class="button is-light" href="/planning/life/goals">Manage Goals</a>
        <a class="button is-light" href="/planning/life">Back to Life Plan</a>
      </div>
    </div>

    <div class="columns">
      <div class="column is-4">
        <div class="box">
          <h2 class="title is-6 mb-3">Baseline</h2>
          <div class="field">
            <label class="label is-size-7" for="baseline-years">Projection Years</label>
            <input
              id="baseline-years"
              class="input"
              type="number"
              min="5"
              max="50"
              bind:value={baseline.projectionYears}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-contribution">Monthly Contribution</label>
            <input
              id="baseline-contribution"
              class="input"
              type="number"
              min="0"
              step="1000"
              bind:value={baseline.monthlyContribution}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-growth">Contribution Growth %</label>
            <input
              id="baseline-growth"
              class="input"
              type="number"
              min="0"
              step="0.1"
              bind:value={baseline.contributionGrowthRate}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-return">Expected Return %</label>
            <input
              id="baseline-return"
              class="input"
              type="number"
              min="0"
              step="0.1"
              bind:value={baseline.expectedReturn}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-volatility">Volatility %</label>
            <input
              id="baseline-volatility"
              class="input"
              type="number"
              min="0"
              step="0.1"
              bind:value={baseline.returnVolatility}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-inflation">Inflation %</label>
            <input
              id="baseline-inflation"
              class="input"
              type="number"
              min="0"
              step="0.1"
              bind:value={baseline.inflationRate}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-swr">SWR %</label>
            <input
              id="baseline-swr"
              class="input"
              type="number"
              min="2"
              step="0.1"
              bind:value={baseline.swr}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="baseline-iterations">Iterations</label>
            <input
              id="baseline-iterations"
              class="input"
              type="number"
              min="100"
              max="10000"
              step="100"
              bind:value={baseline.iterations}
            />
          </div>
        </div>

        <div class="box">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
            <h2 class="title is-6 mb-0">Templates</h2>
            <span class="is-size-7 has-text-grey">{scenarios.length}/5 scenarios</span>
          </div>
          <div class="template-list">
            {#each templates as template}
              <button
                class="button template-button"
                onclick={() => addTemplate(template)}
                disabled={scenarios.length >= 5}
              >
                <span>
                  <strong>{template.label}</strong>
                  <small>{template.description}</small>
                </span>
              </button>
            {/each}
          </div>
          <button
            class="button is-light is-fullwidth mt-3"
            onclick={addBlankScenario}
            disabled={scenarios.length >= 5}>Add Blank Scenario</button
          >
        </div>
      </div>

      <div class="column is-8">
        <div class="box">
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
            <h2 class="title is-6 mb-0">Scenarios</h2>
            <span class="is-size-7 has-text-grey"
              >Each scenario overrides only the fields you set.</span
            >
          </div>

          <div class="scenario-list">
            {#each scenarios as scenario, index (`${scenario.name}-${index}`)}
              <div class="scenario-editor">
                <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
                  <input
                    class="input scenario-name"
                    bind:value={scenario.name}
                    placeholder="Scenario name"
                  />
                  <button
                    class="button is-small is-danger is-light"
                    onclick={() => removeScenario(index)}>Remove</button
                  >
                </div>
                <div class="columns is-multiline">
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "monthly-contribution")}
                      >Monthly Contribution</label
                    >
                    <input
                      id={fieldId(index, "monthly-contribution")}
                      class="input"
                      type="number"
                      step="1000"
                      bind:value={scenario.overrides.monthly_contribution}
                    />
                  </div>
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "expected-return")}
                      >Expected Return %</label
                    >
                    <input
                      id={fieldId(index, "expected-return")}
                      class="input"
                      type="number"
                      step="0.1"
                      bind:value={scenario.overrides.expected_return}
                    />
                  </div>
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "volatility")}
                      >Volatility %</label
                    >
                    <input
                      id={fieldId(index, "volatility")}
                      class="input"
                      type="number"
                      step="0.1"
                      bind:value={scenario.overrides.return_volatility}
                    />
                  </div>
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "contribution-growth")}
                      >Contribution Growth %</label
                    >
                    <input
                      id={fieldId(index, "contribution-growth")}
                      class="input"
                      type="number"
                      step="0.1"
                      bind:value={scenario.overrides.contribution_growth_rate}
                    />
                  </div>
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "inflation")}
                      >Inflation %</label
                    >
                    <input
                      id={fieldId(index, "inflation")}
                      class="input"
                      type="number"
                      step="0.1"
                      bind:value={scenario.overrides.inflation_rate}
                    />
                  </div>
                  <div class="column is-4">
                    <label class="label is-size-7" for={fieldId(index, "swr")}>SWR %</label>
                    <input
                      id={fieldId(index, "swr")}
                      class="input"
                      type="number"
                      step="0.1"
                      bind:value={scenario.overrides.swr}
                    />
                  </div>
                </div>
              </div>
            {/each}
          </div>
        </div>

        {#if loading}
          <div class="box has-text-centered py-6">
            <span class="icon is-large"><i class="fas fa-spinner fa-pulse fa-2x"></i></span>
            <p class="mt-3 has-text-grey">Running comparison simulations…</p>
          </div>
        {:else if error}
          <div class="notification is-danger is-light">{error}</div>
        {:else if comparison}
          <div class="box">
            <div class="columns is-multiline mb-1">
              <div class="column is-3">
                <div class="metric-card">
                  <div class="metric-label">Baseline FIRE</div>
                  <div class="metric-value">
                    {formatFloat(comparison.baseline.simulation.fire_probability * 100)}%
                  </div>
                </div>
              </div>
              <div class="column is-3">
                <div class="metric-card">
                  <div class="metric-label">Target Corpus</div>
                  <div class="metric-value">
                    {formatCurrency(comparison.baseline.simulation.target_corpus)}
                  </div>
                </div>
              </div>
              <div class="column is-3">
                <div class="metric-card">
                  <div class="metric-label">Median FIRE Year</div>
                  <div class="metric-value">
                    {comparison.baseline.simulation.fire_year_p50 > 0
                      ? `${comparison.baseline.simulation.fire_year_p50}y`
                      : "—"}
                  </div>
                </div>
              </div>
              <div class="column is-3">
                <div class="metric-card">
                  <div class="metric-label">Active Goals</div>
                  <div class="metric-value">{comparison.baseline.goals.length}</div>
                </div>
              </div>
            </div>

            <ScenarioComparison {comparison} />
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>

<style>
  .whatif-header {
    gap: 1rem;
  }

  .template-list,
  .scenario-list {
    display: grid;
    gap: 0.75rem;
  }

  .template-button {
    height: auto;
    justify-content: flex-start;
    padding: 0.8rem 0.9rem;
    text-align: left;
    white-space: normal;
  }

  .template-button span {
    display: grid;
    gap: 0.25rem;
  }

  .template-button small {
    font-size: 0.72rem;
    opacity: 0.7;
  }

  .scenario-editor {
    padding: 0.9rem;
    border-radius: 8px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  .scenario-name {
    max-width: 320px;
  }

  .metric-card {
    padding: 0.8rem;
    border-radius: 8px;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.03));
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  .metric-label {
    font-size: 0.72rem;
    font-weight: 600;
    letter-spacing: 0.03em;
    opacity: 0.65;
    text-transform: uppercase;
  }

  .metric-value {
    font-size: 1.2rem;
    font-weight: 700;
    margin-top: 0.25rem;
  }

  @media (max-width: 768px) {
    .whatif-header {
      flex-direction: column;
      align-items: stretch;
    }

    .scenario-name {
      max-width: none;
    }
  }
</style>
