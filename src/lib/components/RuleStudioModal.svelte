<script lang="ts">
  import Modal from "./Modal.svelte";
  import { ajax, type ImportRule } from "$lib/utils";
  import * as toast from "bulma-toast";
  import _ from "lodash";

  let {
    open = $bindable(false),
    accounts = [],
    onruleschanged = () => {}
  }: {
    open: boolean;
    accounts?: string[];
    onruleschanged?: () => void;
  } = $props();

  let rules: ImportRule[] = $state([]);
  let loading = $state(false);
  let activeTab: "list" | "edit" = $state("list");

  let editingRule: ImportRule = $state({
    name: "",
    priority: 100,
    enabled: true,
    payee_pattern: "",
    payee_match_type: "contains",
    memo_pattern: "",
    memo_match_type: "contains",
    tx_type: "any",
    target_account: "",
    tags: [],
    flag_for_review: false
  });

  let rawTags = $state("");

  $effect(() => {
    if (open) {
      loadRules();
    }
  });

  async function loadRules() {
    loading = true;
    try {
      const res: any = await ajax("/api/rules", { background: true });
      rules = res.rules || [];
    } catch (e) {
      console.error(e);
    } finally {
      loading = false;
    }
  }

  function startCreate() {
    editingRule = {
      name: "",
      priority: 100,
      enabled: true,
      payee_pattern: "",
      payee_match_type: "contains",
      memo_pattern: "",
      memo_match_type: "contains",
      tx_type: "any",
      target_account: accounts[0] || "Expenses:Miscellaneous",
      tags: [],
      flag_for_review: false
    };
    rawTags = "";
    activeTab = "edit";
  }

  function startEdit(r: ImportRule) {
    editingRule = _.cloneDeep(r);
    rawTags = (editingRule.tags || []).join(", ");
    activeTab = "edit";
  }

  async function saveRule() {
    if (!editingRule.name.trim()) {
      toast.toast({ message: "Rule name is required", type: "is-danger" });
      return;
    }
    if (!editingRule.target_account.trim()) {
      toast.toast({ message: "Target account is required", type: "is-danger" });
      return;
    }

    editingRule.tags = rawTags
      .split(",")
      .map((t) => t.trim())
      .filter((t) => t.length > 0);

    try {
      const res: any = await ajax("/api/rules", {
        method: "POST",
        body: JSON.stringify(editingRule),
        background: true
      });
      if (res.saved) {
        toast.toast({ message: `Rule "${editingRule.name}" saved`, type: "is-success" });
        await loadRules();
        activeTab = "list";
        onruleschanged();
      }
    } catch (e: any) {
      toast.toast({ message: `Failed to save rule: ${e.message}`, type: "is-danger" });
    }
  }

  async function deleteRule(id?: number) {
    if (!id) return;
    if (!confirm("Are you sure you want to delete this rule?")) return;

    try {
      await ajax(`/api/rules/${id}`, { method: "DELETE", background: true });
      toast.toast({ message: "Rule deleted", type: "is-success" });
      await loadRules();
      onruleschanged();
    } catch (e: any) {
      toast.toast({ message: `Failed to delete rule: ${e.message}`, type: "is-danger" });
    }
  }

  async function toggleRule(r: ImportRule) {
    r.enabled = !r.enabled;
    await ajax("/api/rules", {
      method: "POST",
      body: JSON.stringify(r),
      background: true
    });
    onruleschanged();
  }
</script>

<Modal bind:active={open}>
  {#snippet head(close)}
    <div class="is-flex is-align-items-center is-justify-content-space-between w-full">
      <div class="is-flex is-align-items-center gap-2">
        <span class="icon is-medium has-text-primary">
          <i class="fas fa-wand-magic-sparkles fa-lg"></i>
        </span>
        <h3 class="title is-5 mb-0">Visual Rule Studio</h3>
      </div>
      <button class="delete" aria-label="close" onclick={() => close()}></button>
    </div>
  {/snippet}

  {#snippet body()}
    <div class="tabs is-boxed mb-4">
      <ul>
        <li class:is-active={activeTab === "list"}>
          <a
            href={"#"}
            onclick={(e) => {
              e.preventDefault();
              activeTab = "list";
            }}
          >
            <span class="icon is-small"><i class="fas fa-list"></i></span>
            <span>Rules ({rules.length})</span>
          </a>
        </li>
        <li class:is-active={activeTab === "edit"}>
          <a
            href={"#"}
            onclick={(e) => {
              e.preventDefault();
              if (activeTab !== "edit") startCreate();
            }}
          >
            <span class="icon is-small"><i class="fas fa-plus"></i></span>
            <span>{editingRule.id ? "Edit Rule" : "New Rule"}</span>
          </a>
        </li>
      </ul>
    </div>

    {#if activeTab === "list"}
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
        <p class="is-size-7 has-text-grey">
          Rules classify statement descriptions into posting accounts automatically during
          ingestion.
        </p>
        <button class="button is-small is-primary is-light" onclick={startCreate}>
          <span class="icon is-small"><i class="fas fa-plus"></i></span>
          <span>Add Rule</span>
        </button>
      </div>

      {#if loading}
        <div class="has-text-centered py-5">
          <span class="icon is-large has-text-grey"
            ><i class="fas fa-spinner fa-spin fa-2x"></i></span
          >
        </div>
      {:else if rules.length === 0}
        <div class="notification is-light has-text-centered py-5">
          <p class="has-text-grey mb-2">No classification rules defined yet.</p>
          <button class="button is-small is-primary" onclick={startCreate}>Create First Rule</button
          >
        </div>
      {:else}
        <div class="rule-list">
          {#each rules as r}
            <div
              class="box p-3 mb-2 is-flex is-align-items-center is-justify-content-space-between rule-card"
              class:has-background-light={!r.enabled}
            >
              <div style="flex: 1; min-width: 0;">
                <div class="is-flex is-align-items-center gap-2 mb-1">
                  <span class="tag is-rounded is-small is-dark font-mono">P{r.priority}</span>
                  <strong class="is-size-6">{r.name}</strong>
                  {#if !r.enabled}
                    <span class="tag is-warning is-light is-small">Disabled</span>
                  {/if}
                  {#if r.flag_for_review}
                    <span class="tag is-danger is-light is-small">Flag Review</span>
                  {/if}
                </div>

                <div class="is-size-7 has-text-grey is-flex is-flex-wrap-wrap gap-3">
                  {#if r.payee_pattern}
                    <span>Payee: <code>{r.payee_pattern}</code> ({r.payee_match_type})</span>
                  {/if}
                  {#if r.memo_pattern}
                    <span>Memo: <code>{r.memo_pattern}</code> ({r.memo_match_type})</span>
                  {/if}
                  {#if r.tx_type && r.tx_type !== "any"}
                    <span class="tag is-light is-small uppercase">{r.tx_type}</span>
                  {/if}
                  <span class="has-text-link">➔ <b>{r.target_account}</b></span>
                  {#if r.tags && r.tags.length > 0}
                    <span>Tags: {r.tags.join(" ")}</span>
                  {/if}
                </div>
              </div>

              <div class="is-flex is-align-items-center gap-2 ml-3">
                <button
                  class="button is-small is-white"
                  title={r.enabled ? "Disable" : "Enable"}
                  onclick={() => toggleRule(r)}
                >
                  <span class="icon is-small has-text-grey">
                    <i
                      class="fas {r.enabled
                        ? 'fa-toggle-on has-text-success'
                        : 'fa-toggle-off'} fa-lg"
                    ></i>
                  </span>
                </button>
                <button class="button is-small is-white" title="Edit" onclick={() => startEdit(r)}>
                  <span class="icon is-small has-text-info"><i class="fas fa-pen"></i></span>
                </button>
                <button
                  class="button is-small is-white"
                  title="Delete"
                  onclick={() => deleteRule(r.id)}
                >
                  <span class="icon is-small has-text-danger"><i class="fas fa-trash"></i></span>
                </button>
              </div>
            </div>
          {/each}
        </div>
      {/if}
    {:else if activeTab === "edit"}
      <div class="columns is-multiline">
        <div class="column is-8 py-1">
          <div class="field">
            <label class="label is-small" for="rule-name">Rule Name</label>
            <div class="control">
              <input
                id="rule-name"
                class="input is-small"
                type="text"
                bind:value={editingRule.name}
                placeholder="e.g. Uber Rides, Netflix Subscription"
              />
            </div>
          </div>
        </div>
        <div class="column is-4 py-1">
          <div class="field">
            <label class="label is-small" for="rule-priority">Priority (Lower = Higher)</label>
            <div class="control">
              <input
                id="rule-priority"
                class="input is-small"
                type="number"
                bind:value={editingRule.priority}
              />
            </div>
          </div>
        </div>

        <div class="column is-8 py-1">
          <div class="field">
            <label class="label is-small" for="payee-pattern">Payee / Description Pattern</label>
            <div class="control">
              <input
                id="payee-pattern"
                class="input is-small"
                type="text"
                bind:value={editingRule.payee_pattern}
                placeholder="e.g. UBER, AMAZON, WHOLE FOODS"
              />
            </div>
          </div>
        </div>
        <div class="column is-4 py-1">
          <div class="field">
            <label class="label is-small" for="payee-match">Match Type</label>
            <div class="control">
              <div class="select is-small is-fullwidth">
                <select id="payee-match" bind:value={editingRule.payee_match_type}>
                  <option value="contains">Contains (Default)</option>
                  <option value="exact">Exact Match</option>
                  <option value="regex">Regular Expression</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div class="column is-8 py-1">
          <div class="field">
            <label class="label is-small" for="memo-pattern"
              >Narration / Memo Pattern (Optional)</label
            >
            <div class="control">
              <input
                id="memo-pattern"
                class="input is-small"
                type="text"
                bind:value={editingRule.memo_pattern}
                placeholder="e.g. UPI/REFUND/SALARY"
              />
            </div>
          </div>
        </div>
        <div class="column is-4 py-1">
          <div class="field">
            <label class="label is-small" for="tx-type">Transaction Type</label>
            <div class="control">
              <div class="select is-small is-fullwidth">
                <select id="tx-type" bind:value={editingRule.tx_type}>
                  <option value="any">Any (Debit or Credit)</option>
                  <option value="debit">Debit (Expense/Outflow)</option>
                  <option value="credit">Credit (Income/Inflow)</option>
                </select>
              </div>
            </div>
          </div>
        </div>

        <div class="column is-12 py-1">
          <div class="field">
            <label class="label is-small" for="target-account">Target Posting Account *</label>
            <div class="control">
              <input
                id="target-account"
                class="input is-small"
                type="text"
                list="accounts-list"
                bind:value={editingRule.target_account}
                placeholder="e.g. Expenses:Transport:Taxi, Expenses:Groceries"
              />
              <datalist id="accounts-list">
                {#each accounts as acc}
                  <option value={acc}>{acc}</option>
                {/each}
              </datalist>
            </div>
          </div>
        </div>

        <div class="column is-8 py-1">
          <div class="field">
            <label class="label is-small" for="rule-tags">Append Tags (Comma separated)</label>
            <div class="control">
              <input
                id="rule-tags"
                class="input is-small"
                type="text"
                bind:value={rawTags}
                placeholder="#commute, #dining"
              />
            </div>
          </div>
        </div>
        <div class="column is-4 py-1 is-flex is-align-items-center">
          <label class="checkbox is-size-7 mt-4">
            <input type="checkbox" bind:checked={editingRule.flag_for_review} />
            Flag for Review
          </label>
        </div>
      </div>
    {/if}
  {/snippet}

  {#snippet foot(close)}
    {#if activeTab === "edit"}
      <button class="button is-primary is-small" onclick={saveRule}>Save Rule</button>
      <button class="button is-small" onclick={() => (activeTab = "list")}>Back to List</button>
    {:else}
      <button class="button is-small" onclick={() => close()}>Close</button>
    {/if}
  {/snippet}
</Modal>

<style lang="scss">
  .rule-card {
    border: 1px solid rgba(127, 127, 127, 0.2);
    border-radius: 6px;
    transition: transform 120ms ease;
  }
  .rule-card:hover {
    border-color: var(--bulma-primary, #485fc7);
  }
  .rule-list {
    max-height: 55vh;
    overflow-y: auto;
  }
</style>
