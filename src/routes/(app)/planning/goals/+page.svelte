<script lang="ts">
  import { page } from "$app/stores";
  import { dndzone } from "svelte-dnd-action";
  import { flip } from "svelte/animate";
  import GoalSummaryCard from "$lib/components/GoalSummaryCard.svelte";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import { helpUrl, type GoalSummary } from "$lib/utils";
  import { updateConfig } from "$lib/config_client";
  import _ from "lodash";
  import * as toast from "bulma-toast";
  import { writable } from "svelte/store";
  import type { Action } from "svelte/action";
  import type { PageData } from "./$types";

  let { data }: { data: PageData } = $props();

  type GoalsTab = "tracking" | "life";
  let activeTab: GoalsTab = $state(
    $page.url.searchParams.get("tab") === "life" ? "life" : "tracking"
  );

  // Tracking (Retirement & Savings) state
  let goals: GoalSummary[] = $state([]);
  $effect(() => {
    goals = _.sortBy(data.goals, (g) => -g.priority);
  });
  let isEmpty = $derived(_.isEmpty(goals));
  let config: UserConfig = $state({} as UserConfig);
  const dragDisabled = writable(true);

  // Life Milestones state
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

  let lifeGoals: LifeGoalConfig[] = $state([]);
  let savingLifeGoals = $state(false);
  let lifeInitialized = $state(false);

  $effect(() => {
    if (!lifeInitialized && data.config) {
      const nextConfig = _.cloneDeep(data.config);
      config = nextConfig;
      lifeGoals = _.cloneDeep(nextConfig.goals?.life || []);
      lifeInitialized = true;
    }
  });

  function handleConsider(event: CustomEvent<DndEvent<GoalSummary>>) {
    goals = event.detail.items;
  }

  async function handleFinalize(event: CustomEvent<DndEvent<GoalSummary>>) {
    dragDisabled.set(true);

    goals = event.detail.items;
    for (let i = 0; i < goals.length; i++) {
      const g = goals[i];
      g.priority = goals.length - i;
      const goalConfig = _.find(config.goals[g.type] || [], { name: g.name });
      if (goalConfig) {
        goalConfig.priority = g.priority;
      }
    }
    await save(config);
  }

  async function save(newConfig: UserConfig) {
    const { success, error } = await updateConfig(newConfig, { background: true });

    if (success) {
      globalThis.USER_CONFIG = _.cloneDeep(newConfig);
      toast.toast({
        message: `Updated goal config`,
        type: "is-success"
      });
    } else {
      toast.toast({
        message: `Failed to save config: ${error}`,
        type: "is-danger",
        duration: 10000
      });
    }
  }

  function fieldId(index: number, field: string) {
    return `life-goal-${index}-${field}`;
  }

  function addLifeGoal(type: LifeGoalConfig["type"]) {
    const goal = defaultLifeGoal();
    goal.type = type;
    lifeGoals = [...lifeGoals, goal];
  }

  function removeLifeGoal(index: number) {
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

  async function saveLifeGoals() {
    savingLifeGoals = true;
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
    savingLifeGoals = false;

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

  const dragHandle: Action<HTMLElement, {}> = (node: HTMLElement) => {
    function startDrag(e: Event) {
      e.preventDefault();
      dragDisabled.set(false);
    }

    function stopDrag(_e: KeyboardEvent) {
      dragDisabled.set(true);
    }

    function handleKeyDown(e: KeyboardEvent) {
      if (e.key === "Enter" || e.key === " ") {
        dragDisabled.set(false);
      }
    }

    dragDisabled.subscribe((disabled) => {
      node.tabIndex = disabled ? 0 : -1;
      node.style.cursor = disabled ? "grab" : "grabbing";
    });

    node.addEventListener("mousedown", startDrag);
    node.addEventListener("touchstart", startDrag);
    node.addEventListener("keydown", handleKeyDown);
    node.addEventListener("mouseup", stopDrag);
    node.addEventListener("touchend", stopDrag);

    return {
      update: () => {},
      destroy: () => {
        node.removeEventListener("mousedown", startDrag);
        node.removeEventListener("touchstart", startDrag);
        node.removeEventListener("keydown", handleKeyDown);
        node.removeEventListener("mouseup", stopDrag);
        node.removeEventListener("touchend", stopDrag);
      }
    };
  };
</script>

<section class="section py-5">
  <div class="container is-fluid">
    <div
      class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-3 mb-4"
    >
      <div>
        <h1 class="title is-4 mb-1">Goals</h1>
        <p class="is-size-7 has-text-grey mb-0">
          Track retirement & savings targets alongside milestone life outflows used in Monte Carlo
          projections.
        </p>
      </div>

      <div class="tabs is-toggle is-small mb-0">
        <ul>
          <li class={activeTab === "tracking" ? "is-active" : ""}>
            <a
              href="#tracking"
              onclick={(e) => {
                e.preventDefault();
                activeTab = "tracking";
              }}
            >
              <span>Retirement & Savings</span>
              <span class="tag is-light is-small ml-2">{goals.length}</span>
            </a>
          </li>
          <li class={activeTab === "life" ? "is-active" : ""}>
            <a
              href="#life"
              onclick={(e) => {
                e.preventDefault();
                activeTab = "life";
              }}
            >
              <span>Life Milestones</span>
              <span class="tag is-light is-small ml-2">{lifeGoals.length}</span>
            </a>
          </li>
        </ul>
      </div>
    </div>

    {#if activeTab === "tracking"}
      <div
        class="columns flex-wrap"
        use:dndzone={{
          items: goals,
          dropTargetStyle: {},
          flipDurationMs: 300,
          dragDisabled: $dragDisabled
        }}
        onconsider={handleConsider}
        onfinalize={handleFinalize}
      >
        {#each goals as goal (goal.id)}
          <div animate:flip={{ duration: 300 }} class="column is-6 is-one-third-widescreen">
            <GoalSummaryCard action={dragHandle} {goal} />
          </div>
        {/each}
      </div>
      <div class="columns flex-wrap">
        <div class="column is-12">
          <ZeroState item={!isEmpty}>
            <strong>Oops!</strong> You haven't configured any retirement or savings goals yet.
            Checkout the <a href={helpUrl("goals")}>docs</a> page to get started.
          </ZeroState>
        </div>
      </div>
    {:else}
      <div
        class="is-flex is-justify-content-space-between is-align-items-center is-flex-wrap-wrap gap-2 mb-4"
      >
        <div class="buttons mb-0">
          <button class="button is-light is-small" onclick={() => addLifeGoal("milestone")}>
            Add Milestone
          </button>
          <button class="button is-light is-small" onclick={() => addLifeGoal("recurring")}>
            Add Recurring Goal
          </button>
        </div>
        <button
          class:is-loading={savingLifeGoals}
          class="button is-primary is-small"
          onclick={saveLifeGoals}
        >
          Save Life Goals
        </button>
      </div>

      {#if lifeGoals.length === 0}
        <div class="box has-text-centered py-6">
          <p class="mb-4">No life milestones configured yet.</p>
          <div class="buttons is-centered">
            <button class="button is-primary is-light" onclick={() => addLifeGoal("milestone")}>
              Add your first milestone
            </button>
            <button class="button is-light" onclick={() => addLifeGoal("recurring")}>
              Add a recurring goal
            </button>
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
                <button
                  class="button is-small is-danger is-light"
                  onclick={() => removeLifeGoal(index)}
                >
                  Remove
                </button>
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
                  <label class="label is-size-7" for={fieldId(index, "inflation")}>
                    Inflation Override %
                  </label>
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
                    <label class="label is-size-7" for={fieldId(index, "target-amount")}>
                      Target Amount
                    </label>
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
                    <label class="label is-size-7" for={fieldId(index, "target-date")}>
                      Target Month
                    </label>
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
                    <label class="label is-size-7" for={fieldId(index, "allocation")}>
                      Allocation Per Occurrence
                    </label>
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
                    <label class="label is-size-7" for={fieldId(index, "start-date")}>
                      Start Month
                    </label>
                    <input
                      id={fieldId(index, "start-date")}
                      class="input"
                      bind:value={goal.start_date}
                      placeholder="2027-01"
                      pattern="[0-9]{4}-[0-9]{2}"
                    />
                  </div>
                  <div class="column is-3">
                    <label class="label is-size-7" for={fieldId(index, "end-date")}>
                      End Month
                    </label>
                    <input
                      id={fieldId(index, "end-date")}
                      class="input"
                      bind:value={goal.end_date}
                      placeholder="2035-12"
                      pattern="[0-9]{4}-[0-9]{2}"
                    />
                  </div>
                  <div class="column is-3">
                    <label class="label is-size-7" for={fieldId(index, "frequency")}>
                      Frequency
                    </label>
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
    {/if}
  </div>
</section>

<style>
  .life-goal-card {
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.08));
  }
</style>
