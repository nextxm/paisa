<script lang="ts">
  import { onMount } from "svelte";
  import { ajax, type ReconcileItem } from "$lib/utils";
  import { updateConfig } from "$lib/config_client";
  import COLORS from "$lib/colors";

  let items = $state<ReconcileItem[]>([]);
  let loading = $state(true);
  let error = $state("");

  async function load() {
    loading = true;
    error = "";
    try {
      const res = await ajax("/api/firefly/reconcile");
      items = res.items;
    } catch (e: any) {
      error = e.message;
    } finally {
      loading = false;
    }
  }

  async function toggleIgnore(item: ReconcileItem) {
    const newIgnored = !item.ignored;
    // Update local state
    item.ignored = newIgnored;

    // Update config
    const fireflyConfig = { ...USER_CONFIG.firefly };
    if (newIgnored) {
      fireflyConfig.ignore_accounts = [
        ...(fireflyConfig.ignore_accounts || []),
        item.firefly_account
      ];
    } else {
      fireflyConfig.ignore_accounts = (fireflyConfig.ignore_accounts || []).filter(
        (a: string) => a !== item.firefly_account
      );
    }

    const newConfig = { ...USER_CONFIG, firefly: fireflyConfig };
    try {
      const result = await updateConfig(newConfig);
      if (!result.success) {
        throw new Error(result.error || "Failed to save config");
      }
      USER_CONFIG.firefly = fireflyConfig;
    } catch (e: any) {
      alert("Failed to save config: " + e.message);
      // Revert local state
      item.ignored = !newIgnored;
    }
  }

  onMount(load);
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="level">
      <div class="level-left">
        <h1 class="title">Firefly III Reconciliation</h1>
      </div>
      <div class="level-right">
        <button class="button is-small" onclick={load} disabled={loading}>
          <span class="icon is-small">
            <i class="fas fa-sync"></i>
          </span>
          <span>Refresh</span>
        </button>
      </div>
    </div>

    {#if loading}
      <div class="has-text-centered p-6">
        <span class="icon is-large">
          <i class="fas fa-spinner fa-pulse fa-2x"></i>
        </span>
      </div>
    {:else if error}
      <div class="notification is-danger">
        {error}
      </div>
    {:else}
      <div class="table-container">
        <table class="table is-fullwidth is-hoverable is-narrow">
          <thead>
            <tr>
              <th>Firefly Account</th>
              <th>Paisa Account</th>
              <th class="has-text-right">Firefly Balance</th>
              <th class="has-text-right">Paisa Balance</th>
              <th class="has-text-right">Diff</th>
              <th class="has-text-centered">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each items as item}
              <tr class={item.ignored ? "has-text-grey-light" : ""}>
                <td>
                  {item.firefly_account}
                  {#if item.ignored}
                    <span class="tag is-small ml-2">Ignored</span>
                  {/if}
                </td>
                <td>
                  {#if item.paisa_account}
                    {item.paisa_account}
                  {:else}
                    <span class="has-text-danger">Not linked</span>
                  {/if}
                </td>
                <td class="has-text-right">
                  {item.firefly_balance}
                  {item.currency}
                </td>
                <td class="has-text-right">
                  {item.paisa_balance}
                  {item.currency}
                </td>
                <td
                  class="has-text-right"
                  style="color: {parseFloat(item.diff) == 0 ? COLORS.gainText : COLORS.lossText}"
                >
                  {item.diff}
                  {item.currency}
                </td>
                <td class="has-text-centered">
                  <button class="button is-small is-ghost" onclick={() => toggleIgnore(item)}>
                    {item.ignored ? "Unignore" : "Ignore"}
                  </button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</section>
