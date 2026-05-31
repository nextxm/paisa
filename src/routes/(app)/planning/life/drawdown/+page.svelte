<script lang="ts">
  import { onMount } from "svelte";
  import DrawdownStrategy from "$lib/components/DrawdownStrategy.svelte";
  import { ajax, type DrawdownBucket, type DrawdownResponse } from "$lib/utils";

  let amount = $state(500000);
  let buckets = $state<DrawdownBucket[]>([
    {
      name: "Equity First",
      accounts: [],
      account_glob: "Assets:Equity:*",
      tax_category: "",
      override_tax_category: "equity",
      holding_period_months: 12
    },
    {
      name: "Fallback",
      accounts: [],
      account_glob: "Assets:*",
      tax_category: "",
      override_tax_category: "",
      holding_period_months: 0
    }
  ]);
  let response = $state<DrawdownResponse | null>(null);
  let loading = $state(true);
  let error = $state<string | null>(null);
  let draggingAccount = $state<string | null>(null);
  let dragFromBucketIndex = $state<number | null>(null);

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
        name: bucket.name.trim(),
        accounts: Array.from(
          new Set(bucket.accounts.map((account) => account.trim()).filter(Boolean))
        ),
        account_glob: bucket.account_glob.trim(),
        tax_category: bucket.tax_category,
        override_tax_category: bucket.override_tax_category,
        holding_period_months: Math.max(0, Math.round(bucket.holding_period_months || 0))
      }))
      .filter(
        (bucket) =>
          bucket.account_glob !== "" ||
          bucket.accounts.length > 0 ||
          bucket.tax_category !== "" ||
          bucket.holding_period_months > 0 ||
          bucket.override_tax_category !== ""
      );
  }

  function addBucket() {
    buckets = [
      ...buckets,
      {
        name: `Bucket ${buckets.length + 1}`,
        accounts: [],
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

  function onAccountDragStart(account: string, fromBucketIndex: number | null) {
    draggingAccount = account;
    dragFromBucketIndex = fromBucketIndex;
  }

  function removeAccountFromAllBuckets(account: string) {
    buckets = buckets.map((bucket) => ({
      ...bucket,
      accounts: bucket.accounts.filter((entry) => entry !== account)
    }));
  }

  function dropAccountToBucket(targetBucketIndex: number) {
    if (!draggingAccount) {
      return;
    }

    const copy = [...buckets];
    for (let i = 0; i < copy.length; i++) {
      copy[i] = {
        ...copy[i],
        accounts: copy[i].accounts.filter((account) => account !== draggingAccount)
      };
    }
    copy[targetBucketIndex] = {
      ...copy[targetBucketIndex],
      accounts: [...copy[targetBucketIndex].accounts, draggingAccount]
    };
    buckets = copy;

    draggingAccount = null;
    dragFromBucketIndex = null;
  }

  function dropAccountToUnassigned() {
    if (!draggingAccount) {
      return;
    }
    removeAccountFromAllBuckets(draggingAccount);
    draggingAccount = null;
    dragFromBucketIndex = null;
  }

  function clearBucketAccounts(index: number) {
    buckets = buckets.map((bucket, currentIndex) =>
      currentIndex === index
        ? {
            ...bucket,
            accounts: []
          }
        : bucket
    );
  }

  function unassignedAccounts(): string[] {
    const allAccounts = response?.available_assets.map((asset) => asset.account) || [];
    const assigned = new Set(buckets.flatMap((bucket) => bucket.accounts));
    return allAccounts.filter((account) => !assigned.has(account));
  }

  function bucketAccounts(index: number): string[] {
    const allAccounts = response?.available_assets.map((asset) => asset.account) || [];
    const assigned = new Set(allAccounts);
    return buckets[index].accounts.filter((account) => assigned.has(account));
  }

  function onAccountDragEnd() {
    draggingAccount = null;
    dragFromBucketIndex = null;
  }

  function dragSourceLabel(index: number | null): string {
    if (index === null) {
      return "Unassigned";
    }
    return buckets[index]?.name?.trim() || `Bucket ${index + 1}`;
  }

  function accountChipClass(isDragging: boolean): string {
    return isDragging ? "asset-chip is-dragging" : "asset-chip";
  }

  function shouldShowAssignmentBoard(): boolean {
    return !!response && response.available_assets.length > 0;
  }

  function availableAssets() {
    return response?.available_assets || [];
  }

  function groupNameFromAccount(account: string): string {
    const parts = account.split(":").filter(Boolean);
    if (parts.length >= 3) {
      return parts[1];
    }
    if (parts.length >= 2) {
      return parts[0];
    }
    return "Other";
  }

  function titleCase(value: string): string {
    return value
      .split(/[_\s-]+/)
      .filter(Boolean)
      .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
      .join(" ");
  }

  function resetBucketRuleFields(bucket: DrawdownBucket): DrawdownBucket {
    return {
      ...bucket,
      account_glob: "",
      tax_category: "",
      holding_period_months: 0
    };
  }

  function buildAutoBucketsFromGroups(groups: Map<string, string[]>): DrawdownBucket[] {
    const sortedGroups = Array.from(groups.entries())
      .filter(([, accounts]) => accounts.length > 0)
      .sort((a, b) => a[0].localeCompare(b[0]));

    if (sortedGroups.length === 0) {
      return buckets;
    }

    return sortedGroups.map(([name, accounts], index) => {
      const existing = buckets[index] || {
        name,
        accounts: [],
        account_glob: "",
        tax_category: "",
        override_tax_category: "",
        holding_period_months: 0
      };
      return {
        ...resetBucketRuleFields(existing),
        name,
        accounts: [...accounts].sort((a, b) => a.localeCompare(b))
      };
    });
  }

  function autoGroupByAccountFamily() {
    const assets = availableAssets();
    if (assets.length === 0) {
      return;
    }

    const groups = new Map<string, string[]>();
    for (const asset of assets) {
      const group = titleCase(groupNameFromAccount(asset.account));
      const list = groups.get(group) || [];
      list.push(asset.account);
      groups.set(group, list);
    }

    buckets = buildAutoBucketsFromGroups(groups);
    void runDrawdown();
  }

  function autoGroupByTaxCategory() {
    const assets = availableAssets();
    if (assets.length === 0) {
      return;
    }

    const groups = new Map<string, string[]>();
    for (const asset of assets) {
      const key = asset.tax_category || "other";
      const group = titleCase(key);
      const list = groups.get(group) || [];
      list.push(asset.account);
      groups.set(group, list);
    }

    buckets = buildAutoBucketsFromGroups(groups).map((bucket) => {
      const lower = bucket.name.toLowerCase().replace(/\s+/g, "_");
      const override = taxCategoryOptions.find((option) => option.value === lower)?.value || "";
      return {
        ...bucket,
        override_tax_category: override
      };
    });
    void runDrawdown();
  }

  function matchesAccountGlob(account: string, glob: string): boolean {
    if (!glob || glob === "*") {
      return true;
    }
    if (glob.endsWith("*")) {
      return account.startsWith(glob.slice(0, -1));
    }
    return account === glob;
  }

  function matchesBucketRule(
    asset: { account: string; tax_category: string },
    bucket: DrawdownBucket
  ) {
    if (bucket.tax_category !== "" && asset.tax_category !== bucket.tax_category) {
      return false;
    }
    if (bucket.account_glob !== "" && !matchesAccountGlob(asset.account, bucket.account_glob)) {
      return false;
    }
    return true;
  }

  function autoGroupByCurrentRules() {
    const assets = availableAssets();
    if (assets.length === 0 || buckets.length === 0) {
      return;
    }

    const assigned = new Set<string>();
    const nextBuckets = buckets.map((bucket) => ({
      ...bucket,
      accounts: [] as string[]
    }));

    for (let i = 0; i < nextBuckets.length; i++) {
      const matches = assets
        .filter((asset) => !assigned.has(asset.account) && matchesBucketRule(asset, nextBuckets[i]))
        .map((asset) => asset.account)
        .sort((a, b) => a.localeCompare(b));
      nextBuckets[i].accounts = matches;
      for (const account of matches) {
        assigned.add(account);
      }
    }

    buckets = nextBuckets;
    void runDrawdown();
  }

  function accountFamilyGroupCount(): number {
    const assets = availableAssets();
    if (assets.length === 0) {
      return 0;
    }
    const groups = new Set<string>();
    for (const asset of assets) {
      groups.add(titleCase(groupNameFromAccount(asset.account)));
    }
    return groups.size;
  }

  function taxCategoryGroupCount(): number {
    const assets = availableAssets();
    if (assets.length === 0) {
      return 0;
    }
    const groups = new Set<string>();
    for (const asset of assets) {
      const key = asset.tax_category || "other";
      groups.add(titleCase(key));
    }
    return groups.size;
  }

  function currentRulesGroupCount(): number {
    const assets = availableAssets();
    if (assets.length === 0 || buckets.length === 0) {
      return 0;
    }

    const assigned = new Set<string>();
    let matchedBuckets = 0;
    for (const bucket of buckets) {
      const matches = assets.filter(
        (asset) => !assigned.has(asset.account) && matchesBucketRule(asset, bucket)
      );
      if (matches.length > 0) {
        matchedBuckets++;
      }
      for (const asset of matches) {
        assigned.add(asset.account);
      }
    }
    return matchedBuckets;
  }

  function onBucketDrop(index: number) {
    dropAccountToBucket(index);
    void runDrawdown();
  }

  function onPoolDrop() {
    dropAccountToUnassigned();
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

          {#if shouldShowAssignmentBoard()}
            <div class="buttons mb-3">
              <button class="button is-small is-light" onclick={autoGroupByAccountFamily}
                >Auto-group: Account Family ({accountFamilyGroupCount()} buckets)</button
              >
              <button class="button is-small is-light" onclick={autoGroupByTaxCategory}
                >Auto-group: Tax Category ({taxCategoryGroupCount()} buckets)</button
              >
              <button class="button is-small is-light" onclick={autoGroupByCurrentRules}
                >Auto-group: Current Rules ({currentRulesGroupCount()} buckets)</button
              >
            </div>
          {/if}

          {#if shouldShowAssignmentBoard()}
            <div
              class="asset-pool mb-3"
              role="list"
              aria-label="Unassigned assets"
              ondragover={(event) => event.preventDefault()}
              ondrop={onPoolDrop}
            >
              <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                <p class="bucket-subtitle mb-0">Unassigned Assets</p>
                {#if draggingAccount}
                  <span class="help is-size-7 mb-0"
                    >Dragging from {dragSourceLabel(dragFromBucketIndex)}</span
                  >
                {/if}
              </div>
              <div class="asset-chip-list">
                {#if unassignedAccounts().length === 0}
                  <span class="asset-empty">All assets assigned to buckets</span>
                {:else}
                  {#each unassignedAccounts() as account (account)}
                    <button
                      class={accountChipClass(draggingAccount === account)}
                      draggable="true"
                      ondragstart={() => onAccountDragStart(account, null)}
                      ondragend={onAccountDragEnd}
                    >
                      {account}
                    </button>
                  {/each}
                {/if}
              </div>
            </div>
          {/if}

          <div class="bucket-list">
            {#each buckets as bucket, index (`bucket-${index}`)}
              <div
                class="bucket-card"
                role="listitem"
                aria-label={`Drawdown strategy bucket ${index + 1}`}
                ondragover={(event) => event.preventDefault()}
                ondrop={() => onBucketDrop(index)}
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
                  <label class="label is-size-7" for={`bucket-name-${index}`}>Bucket Name</label>
                  <input
                    id={`bucket-name-${index}`}
                    class="input"
                    placeholder={`Bucket ${index + 1}`}
                    bind:value={bucket.name}
                  />
                </div>

                {#if shouldShowAssignmentBoard()}
                  <div class="field">
                    <div
                      class="is-flex is-justify-content-space-between is-align-items-center mb-2"
                    >
                      <p class="label is-size-7 mb-0">Assigned Assets</p>
                      <button
                        class="button is-small is-ghost"
                        onclick={() => clearBucketAccounts(index)}
                        disabled={bucketAccounts(index).length === 0}>Clear</button
                      >
                    </div>
                    <div class="asset-chip-list">
                      {#if bucketAccounts(index).length === 0}
                        <span class="asset-empty">Drop assets here</span>
                      {:else}
                        {#each bucketAccounts(index) as account (account)}
                          <button
                            class={accountChipClass(draggingAccount === account)}
                            draggable="true"
                            ondragstart={() => onAccountDragStart(account, index)}
                            ondragend={onAccountDragEnd}
                          >
                            {account}
                          </button>
                        {/each}
                      {/if}
                    </div>
                  </div>
                {/if}

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
            Buckets are evaluated top-down. Drag assets into named buckets for explicit assignment,
            or use Account Glob + tax filters when you prefer rule-based matching.
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

  .asset-pool {
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.12));
    border-radius: 10px;
    padding: 0.75rem;
    background: var(--color-background-overlay, rgba(0, 0, 0, 0.02));
  }

  .bucket-subtitle {
    font-size: 0.78rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
    color: var(--color-text-muted, rgba(0, 0, 0, 0.55));
  }

  .asset-chip-list {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    min-height: 2rem;
  }

  .asset-chip {
    border: 1px solid var(--color-border, rgba(0, 0, 0, 0.12));
    border-radius: 999px;
    padding: 0.2rem 0.55rem;
    background: var(--color-background-card, #fff);
    font-size: 0.72rem;
    line-height: 1.3;
    color: var(--color-text, inherit);
    cursor: grab;
  }

  .asset-chip.is-dragging {
    opacity: 0.55;
  }

  .asset-empty {
    font-size: 0.72rem;
    color: var(--color-text-muted, rgba(0, 0, 0, 0.55));
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
