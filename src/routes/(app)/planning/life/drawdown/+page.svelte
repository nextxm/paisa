<script lang="ts">
  import { onMount } from "svelte";
  import DrawdownStrategy from "$lib/components/DrawdownStrategy.svelte";
  import { ajax, type DrawdownResponse } from "$lib/utils";

  let amount = $state(500000);
  let accountGlob = $state("");
  let response = $state<DrawdownResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);

  async function runDrawdown() {
    loading = true;
    error = null;
    try {
      response = await ajax("/api/projection/drawdown", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          amount,
          buckets: accountGlob.trim()
            ? [{ account_glob: accountGlob, tax_category: "", holding_period_months: 0 }]
            : []
        })
      });
    } catch (exception) {
      error = exception instanceof Error ? exception.message : "Drawdown analysis failed";
    } finally {
      loading = false;
    }
  }

  onMount(runDrawdown);
</script>

<section class="section">
  <div class="container is-fluid">
    <div
      class="is-flex is-justify-content-space-between is-align-items-flex-start mb-5 drawdown-header"
    >
      <div>
        <h1 class="title is-4 mb-2">Tax-Aware Drawdown</h1>
        <p class="subtitle is-6 has-text-grey mb-0">
          Rank taxable holdings by estimated drawdown cost using current FIFO lots and today’s
          prices.
        </p>
      </div>
      <div class="buttons">
        <button class:is-loading={loading} class="button is-primary" onclick={runDrawdown}
          >Analyze</button
        >
        <a class="button is-light" href="/planning/life">Back to Life Plan</a>
      </div>
    </div>

    <div class="columns">
      <div class="column is-4">
        <div class="box">
          <div class="field">
            <label class="label is-size-7" for="drawdown-amount">Withdrawal Amount</label>
            <input
              id="drawdown-amount"
              class="input"
              type="number"
              min="0"
              step="1000"
              bind:value={amount}
            />
          </div>
          <div class="field">
            <label class="label is-size-7" for="drawdown-account-filter">Account Filter</label>
            <input
              id="drawdown-account-filter"
              class="input"
              placeholder="Assets:Equity:*"
              bind:value={accountGlob}
            />
            <p class="help">Optional prefix glob to restrict the recommendation set.</p>
          </div>
        </div>
      </div>

      <div class="column is-8">
        {#if loading}
          <div class="box has-text-centered py-6">
            <span class="icon is-large"><i class="fas fa-spinner fa-pulse fa-2x"></i></span>
            <p class="mt-3 has-text-grey">Estimating drawdown order…</p>
          </div>
        {:else if error}
          <div class="notification is-danger is-light">{error}</div>
        {:else if response}
          <div class="box">
            <DrawdownStrategy {response} />
          </div>
        {/if}
      </div>
    </div>
  </div>
</section>

<style>
  .drawdown-header {
    gap: 1rem;
  }

  @media (max-width: 768px) {
    .drawdown-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
