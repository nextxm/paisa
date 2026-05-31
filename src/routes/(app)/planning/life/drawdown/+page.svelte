<script lang="ts">
  import { onMount } from "svelte";
  import DrawdownStrategy from "$lib/components/DrawdownStrategy.svelte";
  import { ajax, type DrawdownBucket, type DrawdownResponse } from "$lib/utils";

  let amount = $state(500000);
  let buckets = $state<DrawdownBucket[]>([
    {
      account_glob: "Assets:Equity:*",
      tax_category: "",
      override_tax_category: "equity",
      holding_period_months: 12
    },
    {
      account_glob: "Assets:*",
      tax_category: "",
      override_tax_category: "",
      holding_period_months: 0
    }
  ]);
  let response = $state<DrawdownResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let draggingIndex = $state<number | null>(null);

  const taxCategoryOptions = [
    { value: "", label: "Any" },
    { value: "equity", label: "Equity" },
    { value: "equity65", label: "Equity 65" },
    { value: "equity35", label: "Equity 35" },
    { value: "debt", label: "Debt" },
    { value: "unlisted_equity", label: "Unlisted Equity" }
  ] as const;

  function cleanBuckets(value: DrawdownBucket[]): DrawdownBucket[] {
    return value
      .map((bucket) => ({
        account_glob: bucket.account_glob.trim(),
        tax_category: bucket.tax_category,
        override_tax_category: bucket.override_tax_category,
        holding_period_months: Math.max(0, Math.round(bucket.holding_period_months || 0))
      }))
      .filter(
        (bucket) =>
          bucket.account_glob !== "" ||
          bucket.tax_category !== "" ||
          bucket.holding_period_months > 0 ||
          bucket.override_tax_category !== ""
      );
  }

  function addBucket() {
    buckets = [
      ...buckets,
      {
        account_glob: "Assets:*",
        tax_category: "",
        override_tax_category: "",
        holding_period_months: 0
      }
    ];
  }

  function removeBucket(index: number) {
    buckets = buckets.filter((_, current) => current !== index);
  }

  function moveBucket(index: number, direction: -1 | 1) {
    const next = index + direction;
    if (next < 0 || next >= buckets.length) return;
    const copy = [...buckets];
    const current = copy[index];
    copy[index] = copy[next];
    copy[next] = current;
    buckets = copy;
    void runDrawdown();
  }

  function onDragStart(index: number) {
    draggingIndex = index;
  }

  function onDrop(index: number) {
    if (draggingIndex === null || draggingIndex === index) {
      draggingIndex = null;
      return;
    }
    const copy = [...buckets];
    const [dragged] = copy.splice(draggingIndex, 1);
    copy.splice(index, 0, dragged);
    buckets = copy;
    draggingIndex = null;
    void runDrawdown();
  }

  async function runDrawdown() {
    loading = true;
    error = null;
    try {
      response = await ajax("/api/projection/drawdown", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          amount,
          buckets: cleanBuckets(buckets),
          include_projection_impact: true,
          baseline: {
            months_to_project: 360,
            iterations: 750
          }
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
        <h1 class="title is-4 is-spaced mb-2">Tax-Aware Drawdown</h1>
        <p class="subtitle is-6 has-text-grey mb-0">
          Rank taxable holdings by estimated drawdown cost using current FIFO lots and today’s
          prices.
        </p>
      </div>
      <div class="buttons">
        <button class:is-loading={loading} class="button is-primary" onclick={runDrawdown}
          >Analyze</button
        >
        <a class="button is-light" href="/planning/life/whatif">What-If Scenarios</a>
        <a class="button is-light" href="/planning/life/goals">Manage Goals</a>
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
          <div class="is-flex is-justify-content-space-between is-align-items-center mb-3">
            <h2 class="title is-6 mb-0">Strategy Buckets</h2>
            <button class="button is-small is-light" onclick={addBucket}>Add Bucket</button>
          </div>

          <div class="bucket-list">
            {#each buckets as bucket, index (`bucket-${index}`)}
              <div
                class="bucket-card"
                role="listitem"
                aria-label={`Drawdown strategy bucket ${index + 1}`}
                draggable="true"
                ondragstart={() => onDragStart(index)}
                ondragover={(event) => event.preventDefault()}
                ondrop={() => onDrop(index)}
              >
                <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                  <div class="bucket-title">Priority {index + 1}</div>
                  <div class="buttons are-small">
                    <button
                      class="button is-light"
                      onclick={() => moveBucket(index, -1)}
                      title="Move up">↑</button
                    >
                    <button
                      class="button is-light"
                      onclick={() => moveBucket(index, 1)}
                      title="Move down">↓</button
                    >
                    <button
                      class="button is-danger is-light"
                      onclick={() => removeBucket(index)}
                      disabled={buckets.length <= 1}>Remove</button
                    >
                  </div>
                </div>

                <div class="field">
                  <label class="label is-size-7" for={`bucket-account-${index}`}>Account Glob</label
                  >
                  <input
                    id={`bucket-account-${index}`}
                    class="input"
                    placeholder="Assets:Equity:*"
                    bind:value={bucket.account_glob}
                  />
                </div>
                <div class="field">
                  <label class="label is-size-7" for={`bucket-tax-${index}`}
                    >Match Tax Category</label
                  >
                  <div class="select is-fullwidth">
                    <select id={`bucket-tax-${index}`} bind:value={bucket.tax_category}>
                      {#each taxCategoryOptions as option}
                        <option value={option.value}>{option.label}</option>
                      {/each}
                    </select>
                  </div>
                </div>
                <div class="field">
                  <label class="label is-size-7" for={`bucket-tax-override-${index}`}
                    >Apply Tax As</label
                  >
                  <div class="select is-fullwidth">
                    <select
                      id={`bucket-tax-override-${index}`}
                      bind:value={bucket.override_tax_category}
                    >
                      <option value="">Use Commodity Tax Category</option>
                      {#each taxCategoryOptions.filter((option) => option.value !== "") as option}
                        <option value={option.value}>{option.label}</option>
                      {/each}
                    </select>
                  </div>
                </div>
                <div class="field mb-0">
                  <label class="label is-size-7" for={`bucket-holding-${index}`}
                    >Minimum Holding Months</label
                  >
                  <input
                    id={`bucket-holding-${index}`}
                    class="input"
                    type="number"
                    min="0"
                    step="1"
                    bind:value={bucket.holding_period_months}
                  />
                </div>
              </div>
            {/each}
          </div>
          <p class="help mt-2">
            Buckets are evaluated top-down. Use Match Tax Category as a filter and Apply Tax As as a
            rule override for how the selected assets should be taxed in the drawdown simulation.
          </p>
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

  .bucket-list {
    display: grid;
    gap: 0.75rem;
  }

  .bucket-card {
    border: 1px dashed var(--color-border, rgba(0, 0, 0, 0.14));
    border-radius: 10px;
    padding: 0.75rem;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.02));
  }

  .bucket-title {
    font-weight: 600;
    font-size: 0.85rem;
  }

  @media (max-width: 768px) {
    .drawdown-header {
      flex-direction: column;
      align-items: stretch;
    }
  }
</style>
