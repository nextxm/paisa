<script lang="ts">
  import _ from "lodash";
  import * as toast from "bulma-toast";
  import { updateConfig } from "$lib/config_client";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  const defaultLifeGoal = (): LifeGoalConfig => ({
    name: "",
    icon: "flag",
    type: "milestone",
    target_amount: 0,
    target_date: "",
    start_date: "",
    end_date: "",
    frequency: "yearly",
    inflation_rate: null,
    priority: 0,
    funded_by: [],
    monthly_allocation: 0
  });

  let config: UserConfig = $state({} as UserConfig);
  let lifeGoals: LifeGoalConfig[] = $state([]);
  let saving = $state(false);

  $effect(() => {
    if (lifeGoals.length === 0 && !config.goals) {
      const nextConfig = _.cloneDeep(data.config);
      config = nextConfig;
      lifeGoals = _.cloneDeep(nextConfig.goals?.life || []);
    }
  });

  function fieldId(index: number, field: string) {
    return `life-goal-${index}-${field}`;
  }

  function addGoal(type: LifeGoalConfig["type"]) {
    const goal = defaultLifeGoal();
    goal.type = type;
    lifeGoals = [...lifeGoals, goal];
  }

  function removeGoal(index: number) {
    lifeGoals = lifeGoals.filter((_, currentIndex) => currentIndex !== index);
  }

  function fundedByText(goal: LifeGoalConfig) {
    return (goal.funded_by || []).join(", ");
  }

  function setFundedBy(goal: LifeGoalConfig, value: string) {
    goal.funded_by = value
      .split(",")
      .map((entry) => entry.trim())
      .filter(Boolean);
  }

  async function save() {
    saving = true;
    const nextConfig = _.cloneDeep(config);
    nextConfig.goals = {
      retirement: nextConfig.goals?.retirement || [],
      savings: nextConfig.goals?.savings || [],
      life: lifeGoals.map((goal, index) => ({
        ...goal,
        funded_by: goal.funded_by || [],
        priority: lifeGoals.length - index,
        inflation_rate:
          goal.inflation_rate === null || goal.inflation_rate === undefined
            ? null
            : Number(goal.inflation_rate),
        target_amount: Number(goal.target_amount || 0),
        monthly_allocation: Number(goal.monthly_allocation || 0)
      }))
    };

    const { success, error } = await updateConfig(nextConfig);
    saving = false;

    if (!success) {
      toast.toast({
        message: `Failed to save life goals: ${error}`,
        type: "is-danger",
        duration: 8000
      });
      return;
    }

    config = nextConfig;
    globalThis.USER_CONFIG = _.cloneDeep(nextConfig);
    toast.toast({
      message: "Life goals updated",
      type: "is-success"
    });
  }
</script>

<section class="section">
  <div class="container is-fluid">
    <div
      class="is-flex is-justify-content-space-between is-align-items-flex-start mb-5 life-goals-header"
    >
      <div>
        <h1 class="title is-4 mb-2">Life Goals</h1>
        <p class="subtitle is-6 has-text-grey mb-0">
          Define one-off milestones and recurring outflows that feed the life projection simulation.
        </p>
      </div>
      <button class:is-loading={saving} class="button is-primary" onclick={save}>Save Goals</button>
    </div>

    <div class="buttons mb-4">
      <button class="button is-light" onclick={() => addGoal("milestone")}>Add Milestone</button>
      <button class="button is-light" onclick={() => addGoal("recurring")}
        >Add Recurring Goal</button
      >
      <a class="button is-ghost" href="/planning/life">Back to Life Projection</a>
    </div>

    {#if lifeGoals.length === 0}
      <div class="box has-text-centered py-6">
        <p class="mb-4">No life goals configured yet.</p>
        <div class="buttons is-centered">
          <button class="button is-primary is-light" onclick={() => addGoal("milestone")}
            >Add your first milestone</button
          >
          <button class="button is-light" onclick={() => addGoal("recurring")}
            >Add a recurring goal</button
          >
        </div>
      </div>
    {/if}

    <div class="columns is-multiline">
      {#each lifeGoals as goal, index (`${goal.name}-${index}`)}
        <div class="column is-12">
          <div class="box life-goal-card">
            <div class="is-flex is-justify-content-space-between is-align-items-center mb-4">
              <div>
                <h2 class="title is-6 mb-1">{goal.name || `Goal ${index + 1}`}</h2>
                <p class="is-size-7 has-text-grey mb-0">Priority {lifeGoals.length - index}</p>
              </div>
              <button class="button is-small is-danger is-light" onclick={() => removeGoal(index)}
                >Remove</button
              >
            </div>

            <div class="columns is-multiline">
              <div class="column is-4">
                <label class="label is-size-7" for={fieldId(index, "name")}>Name</label>
                <input
                  id={fieldId(index, "name")}
                  class="input"
                  bind:value={goal.name}
                  placeholder="House Down Payment"
                />
              </div>
              <div class="column is-2">
                <label class="label is-size-7" for={fieldId(index, "icon")}>Icon</label>
                <input
                  id={fieldId(index, "icon")}
                  class="input"
                  bind:value={goal.icon}
                  placeholder="home"
                />
              </div>
              <div class="column is-3">
                <label class="label is-size-7" for={fieldId(index, "type")}>Type</label>
                <div class="select is-fullwidth">
                  <select id={fieldId(index, "type")} bind:value={goal.type}>
                    <option value="milestone">Milestone</option>
                    <option value="recurring">Recurring</option>
                  </select>
                </div>
              </div>
              <div class="column is-3">
                <label class="label is-size-7" for={fieldId(index, "inflation")}
                  >Inflation Override %</label
                >
                <input
                  id={fieldId(index, "inflation")}
                  class="input"
                  bind:value={goal.inflation_rate}
                  type="number"
                  min="0"
                  step="0.1"
                  placeholder="Use global"
                />
              </div>

              {#if goal.type === "milestone"}
                <div class="column is-4">
                  <label class="label is-size-7" for={fieldId(index, "target-amount")}
                    >Target Amount</label
                  >
                  <input
                    id={fieldId(index, "target-amount")}
                    class="input"
                    bind:value={goal.target_amount}
                    type="number"
                    min="0"
                    step="1000"
                  />
                </div>
                <div class="column is-4">
                  <label class="label is-size-7" for={fieldId(index, "target-date")}
                    >Target Month</label
                  >
                  <input
                    id={fieldId(index, "target-date")}
                    class="input"
                    bind:value={goal.target_date}
                    placeholder="2030-06"
                    pattern="[0-9]{4}-[0-9]{2}"
                  />
                </div>
              {:else}
                <div class="column is-3">
                  <label class="label is-size-7" for={fieldId(index, "allocation")}
                    >Allocation Per Occurrence</label
                  >
                  <input
                    id={fieldId(index, "allocation")}
                    class="input"
                    bind:value={goal.monthly_allocation}
                    type="number"
                    min="0"
                    step="1000"
                  />
                </div>
                <div class="column is-3">
                  <label class="label is-size-7" for={fieldId(index, "start-date")}
                    >Start Month</label
                  >
                  <input
                    id={fieldId(index, "start-date")}
                    class="input"
                    bind:value={goal.start_date}
                    placeholder="2027-01"
                    pattern="[0-9]{4}-[0-9]{2}"
                  />
                </div>
                <div class="column is-3">
                  <label class="label is-size-7" for={fieldId(index, "end-date")}>End Month</label>
                  <input
                    id={fieldId(index, "end-date")}
                    class="input"
                    bind:value={goal.end_date}
                    placeholder="2035-12"
                    pattern="[0-9]{4}-[0-9]{2}"
                  />
                </div>
                <div class="column is-3">
                  <label class="label is-size-7" for={fieldId(index, "frequency")}>Frequency</label>
                  <div class="select is-fullwidth">
                    <select id={fieldId(index, "frequency")} bind:value={goal.frequency}>
                      <option value="monthly">Monthly</option>
                      <option value="quarterly">Quarterly</option>
                      <option value="yearly">Yearly</option>
                    </select>
                  </div>
                </div>
              {/if}

              <div class="column is-12">
                <label class="label is-size-7" for={fieldId(index, "funded-by")}>Funded By</label>
                <input
                  id={fieldId(index, "funded-by")}
                  class="input"
                  value={fundedByText(goal)}
                  oninput={(event) =>
                    setFundedBy(goal, (event.currentTarget as HTMLInputElement).value)}
                  placeholder="Assets:Investments:*, Assets:Checking"
                />
                <p class="help">Comma-separated account globs used for planning context.</p>
              </div>
            </div>
          </div>
        </div>
      {/each}
    </div>
  </div>
</section>

<style>
  .life-goals-header {
    gap: 1rem;
  }

  .life-goal-card {
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }

  @media (max-width: 768px) {
    .life-goals-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
