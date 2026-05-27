<script lang="ts">
  import { onMount } from "svelte";
  import COLORS from "$lib/colors";
  import { ajax, formatCurrency } from "$lib/utils";
  import type { DuplicatePair, OutlierTransaction, Issue } from "$lib/utils";
  import { renderIssues } from "$lib/doctor";
  import { dataQualityIssueCount } from "../../../../store";

  let issues: Issue[] = $state([]);
  let duplicates: DuplicatePair[] = $state([]);
  let outliers: OutlierTransaction[] = $state([]);
  let suppressLoading: Record<string, boolean> = $state({});
  let activeView: "all" | "duplicates" | "outliers" = $state("all");
  let searchText = $state("");
  let minConfidence = $state(0);
  let pageSize = $state(25);
  let duplicatePage = $state(1);
  let outlierPage = $state(1);

  let filteredDuplicates = $derived.by(() =>
    duplicates.filter(
      (pair) => pair.confidence >= minConfidence && matchesDuplicateQuery(pair, searchText)
    )
  );

  let filteredOutliers = $derived.by(() =>
    outliers.filter(
      (outlier) => outlier.confidence >= minConfidence && matchesOutlierQuery(outlier, searchText)
    )
  );

  let duplicatePageCount = $derived(Math.max(1, Math.ceil(filteredDuplicates.length / pageSize)));
  let outlierPageCount = $derived(Math.max(1, Math.ceil(filteredOutliers.length / pageSize)));

  let pagedDuplicates = $derived.by(() => {
    const start = (duplicatePage - 1) * pageSize;
    return filteredDuplicates.slice(start, start + pageSize);
  });

  let pagedOutliers = $derived.by(() => {
    const start = (outlierPage - 1) * pageSize;
    return filteredOutliers.slice(start, start + pageSize);
  });

  onMount(async () => {
    ({ issues } = await ajax("/api/diagnosis"));
    renderIssues(issues);
    const dq = await ajax("/api/diagnosis/duplicates");
    duplicates = dq.duplicates || [];
    outliers = dq.outliers || [];
  });

  $effect(() => {
    dataQualityIssueCount.set(duplicates.length + outliers.length);
  });

  $effect(() => {
    if (duplicatePage > duplicatePageCount) {
      duplicatePage = duplicatePageCount;
    }
    if (duplicatePage < 1) {
      duplicatePage = 1;
    }
  });

  $effect(() => {
    if (outlierPage > outlierPageCount) {
      outlierPage = outlierPageCount;
    }
    if (outlierPage < 1) {
      outlierPage = 1;
    }
  });

  async function suppress(pair: DuplicatePair) {
    const key = `${pair.posting1.id}-${pair.posting2.id}`;
    suppressLoading[key] = true;
    try {
      await ajax("/api/diagnosis/duplicates/suppress", {
        method: "POST",
        body: JSON.stringify({
          posting_id_1: Number(pair.posting1.id),
          posting_id_2: Number(pair.posting2.id)
        })
      });
      duplicates = duplicates.filter(
        (p) => !(p.posting1.id === pair.posting1.id && p.posting2.id === pair.posting2.id)
      );
    } finally {
      suppressLoading[key] = false;
    }
  }

  function pct(confidence: number) {
    return Math.round(confidence * 100);
  }

  function normalize(value: string) {
    return value.trim().toLowerCase();
  }

  function matchesDuplicateQuery(pair: DuplicatePair, q: string) {
    const query = normalize(q);
    if (!query) return true;
    const haystack = [
      pair.reason,
      pair.posting1.date,
      pair.posting1.payee,
      pair.posting1.account,
      String(pair.posting1.amount),
      pair.posting2.date,
      pair.posting2.payee,
      pair.posting2.account,
      String(pair.posting2.amount)
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(query);
  }

  function matchesOutlierQuery(outlier: OutlierTransaction, q: string) {
    const query = normalize(q);
    if (!query) return true;
    const haystack = [
      outlier.posting.date,
      outlier.posting.payee,
      outlier.posting.account,
      String(outlier.posting.amount)
    ]
      .join(" ")
      .toLowerCase();
    return haystack.includes(query);
  }

  function applyFilters() {
    duplicatePage = 1;
    outlierPage = 1;
  }

  function setView(view: "all" | "duplicates" | "outliers") {
    activeView = view;
  }

  function updateSearchText(event: Event) {
    searchText = (event.currentTarget as HTMLInputElement).value;
    applyFilters();
  }

  function updateMinConfidence(event: Event) {
    minConfidence = Number((event.currentTarget as HTMLSelectElement).value);
    applyFilters();
  }

  function updatePageSize(event: Event) {
    pageSize = Number((event.currentTarget as HTMLSelectElement).value) || 25;
    applyFilters();
  }

  function confidenceColor(confidence: number) {
    if (confidence >= 0.8) return COLORS.lossText;
    if (confidence >= 0.5) return "#e6a817";
    return COLORS.gainText;
  }
</script>

<section class="section tab-doctor">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12 has-text-centered">
        <div>
          <b
            class="p-1 has-text-white"
            style="background-color: {issues.length > 0 ? COLORS.lossText : COLORS.gainText}"
            >{issues.length}</b
          > potential issue(s) found.
        </div>
      </div>
    </div>
    <div class="columns is-flex-wrap-wrap" id="d3-diagnosis"></div>

    <!-- Data Quality Section -->
    <hr />
    <h2 class="title is-5">
      Data Quality
      {#if duplicates.length + outliers.length > 0}
        <span class="tag is-danger is-rounded ml-2">{duplicates.length + outliers.length}</span>
      {/if}
    </h2>

    <div class="doctor-summary-grid mb-4">
      <button
        class="box doctor-summary-card"
        class:is-focused={activeView === "all"}
        onclick={() => setView("all")}
      >
        <p class="heading">All data quality signals</p>
        <p class="title is-4 mb-1">{duplicates.length + outliers.length}</p>
        <p class="is-size-7 has-text-grey">Combined duplicates and outliers</p>
      </button>

      <button
        class="box doctor-summary-card"
        class:is-focused={activeView === "duplicates"}
        onclick={() => setView("duplicates")}
      >
        <p class="heading">Potential duplicates</p>
        <p class="title is-4 mb-1">{duplicates.length}</p>
        <p class="is-size-7 has-text-grey">High confidence pairs first</p>
      </button>

      <button
        class="box doctor-summary-card"
        class:is-focused={activeView === "outliers"}
        onclick={() => setView("outliers")}
      >
        <p class="heading">Outlier transactions</p>
        <p class="title is-4 mb-1">{outliers.length}</p>
        <p class="is-size-7 has-text-grey">Large deviations from account norms</p>
      </button>
    </div>

    <div class="box doctor-controls mb-5">
      <div class="columns is-multiline is-vcentered">
        <div class="column is-12-tablet is-5-desktop">
          <label class="label is-small mb-1" for="doctor-search">Search</label>
          <input
            id="doctor-search"
            class="input"
            type="text"
            placeholder="Payee, account, date, amount, reason"
            value={searchText}
            oninput={updateSearchText}
          />
        </div>
        <div class="column is-6-tablet is-3-desktop">
          <label class="label is-small mb-1" for="doctor-confidence">Minimum confidence</label>
          <div class="select is-fullwidth">
            <select id="doctor-confidence" value={minConfidence} onchange={updateMinConfidence}>
              <option value="0">All results</option>
              <option value="0.5">50%+</option>
              <option value="0.8">80%+</option>
            </select>
          </div>
        </div>
        <div class="column is-6-tablet is-2-desktop">
          <label class="label is-small mb-1" for="doctor-page-size">Page size</label>
          <div class="select is-fullwidth">
            <select id="doctor-page-size" value={pageSize} onchange={updatePageSize}>
              <option value="10">10</option>
              <option value="25">25</option>
              <option value="50">50</option>
            </select>
          </div>
        </div>
        <div class="column is-12-tablet is-2-desktop">
          <p class="label is-small mb-1">Focus</p>
          <div class="buttons has-addons is-fullwidth doctor-focus-toggle">
            <button
              class="button is-small"
              class:is-link={activeView === "all"}
              onclick={() => setView("all")}>All</button
            >
            <button
              class="button is-small"
              class:is-link={activeView === "duplicates"}
              onclick={() => setView("duplicates")}>Dup</button
            >
            <button
              class="button is-small"
              class:is-link={activeView === "outliers"}
              onclick={() => setView("outliers")}>Out</button
            >
          </div>
        </div>
      </div>
    </div>

    <!-- Duplicate Pairs -->
    {#if activeView !== "outliers"}
      <section id="doctor-duplicates" class="mb-5">
        <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
          <h3 class="title is-6 mb-0">Potential Duplicates</h3>
          {#if duplicates.length > 0}
            <span class="tag is-warning is-light is-rounded"
              >{filteredDuplicates.length} / {duplicates.length} shown</span
            >
          {/if}
        </div>

        {#if duplicates.length === 0}
          <p class="has-text-grey">No duplicate transactions detected.</p>
        {:else if filteredDuplicates.length === 0}
          <p class="has-text-grey">No duplicates match your current filters.</p>
        {:else}
          <div class="doctor-pagination mb-3">
            <button
              class="button is-small"
              disabled={duplicatePage === 1}
              onclick={() => (duplicatePage = Math.max(1, duplicatePage - 1))}
            >
              Previous
            </button>
            <span class="is-size-7 has-text-grey">Page {duplicatePage} of {duplicatePageCount}</span
            >
            <button
              class="button is-small"
              disabled={duplicatePage === duplicatePageCount}
              onclick={() => (duplicatePage = Math.min(duplicatePageCount, duplicatePage + 1))}
            >
              Next
            </button>
          </div>

          {#each pagedDuplicates as pair}
            {@const key = `${pair.posting1.id}-${pair.posting2.id}`}
            <div class="box mb-4">
              <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                <div>
                  <!-- svelte-ignore a11y_no_static_element_interactions -->
                  <span class="tag is-warning is-light mr-2 invertable">Duplicate</span>
                  <span class="has-text-grey is-size-7">{@html pair.reason}</span>
                </div>
                <div class="is-flex is-align-items-center">
                  <span
                    class="tag is-rounded mr-3"
                    style="background-color: {confidenceColor(pair.confidence)}; color: white"
                  >
                    {pct(pair.confidence)}% confidence
                  </span>
                  <button
                    class="button is-small is-light"
                    class:is-loading={suppressLoading[key]}
                    onclick={() => suppress(pair)}
                    title="Mark as not a duplicate"
                  >
                    Dismiss
                  </button>
                </div>
              </div>
              <div class="columns">
                <div class="column is-6">
                  <div class="notification is-light invertable py-2 px-3">
                    <p class="is-size-7 has-text-grey mb-1">Posting 1</p>
                    <p>
                      <strong>{pair.posting1.date}</strong>
                      &nbsp;
                      <a
                        href="/ledger/editor/{encodeURIComponent(pair.posting1.file_name)}#{pair
                          .posting1.transaction_begin_line}"
                      >
                        {pair.posting1.payee}
                      </a>
                    </p>
                    <p class="is-size-7">{pair.posting1.account}</p>
                    <p class="has-text-weight-semibold">{formatCurrency(pair.posting1.amount)}</p>
                  </div>
                </div>
                <div class="column is-6">
                  <div class="notification is-light invertable py-2 px-3">
                    <p class="is-size-7 has-text-grey mb-1">Posting 2</p>
                    <p>
                      <strong>{pair.posting2.date}</strong>
                      &nbsp;
                      <a
                        href="/ledger/editor/{encodeURIComponent(pair.posting2.file_name)}#{pair
                          .posting2.transaction_begin_line}"
                      >
                        {pair.posting2.payee}
                      </a>
                    </p>
                    <p class="is-size-7">{pair.posting2.account}</p>
                    <p class="has-text-weight-semibold">{formatCurrency(pair.posting2.amount)}</p>
                  </div>
                </div>
              </div>
            </div>
          {/each}
        {/if}
      </section>
    {/if}

    <!-- Outlier Transactions -->
    {#if activeView !== "duplicates"}
      <section id="doctor-outliers">
        <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
          <h3 class="title is-6 mb-0">Outlier Transactions</h3>
          {#if outliers.length > 0}
            <span class="tag is-danger is-light is-rounded"
              >{filteredOutliers.length} / {outliers.length} shown</span
            >
          {/if}
        </div>

        {#if outliers.length === 0}
          <p class="has-text-grey">No statistical outliers detected.</p>
        {:else if filteredOutliers.length === 0}
          <p class="has-text-grey">No outliers match your current filters.</p>
        {:else}
          <div class="doctor-pagination mb-3">
            <button
              class="button is-small"
              disabled={outlierPage === 1}
              onclick={() => (outlierPage = Math.max(1, outlierPage - 1))}
            >
              Previous
            </button>
            <span class="is-size-7 has-text-grey">Page {outlierPage} of {outlierPageCount}</span>
            <button
              class="button is-small"
              disabled={outlierPage === outlierPageCount}
              onclick={() => (outlierPage = Math.min(outlierPageCount, outlierPage + 1))}
            >
              Next
            </button>
          </div>

          <div class="columns is-flex-wrap-wrap">
            {#each pagedOutliers as outlier}
              <div class="column is-6">
                <div class="box">
                  <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
                    <span class="tag is-danger is-light invertable">Outlier</span>
                    <span
                      class="tag is-rounded"
                      style="background-color: {confidenceColor(outlier.confidence)}; color: white"
                    >
                      {pct(outlier.confidence)}% confidence
                    </span>
                  </div>
                  <p>
                    <strong>{outlier.posting.date}</strong>
                    &nbsp;
                    <a
                      href="/ledger/editor/{encodeURIComponent(outlier.posting.file_name)}#{outlier
                        .posting.transaction_begin_line}"
                    >
                      {outlier.posting.payee}
                    </a>
                  </p>
                  <p class="is-size-7 has-text-grey">{outlier.posting.account}</p>
                  <p class="has-text-weight-semibold">{formatCurrency(outlier.posting.amount)}</p>
                  <p class="is-size-7 has-text-grey mt-1">
                    {outlier.sigma.toFixed(1)}σ above mean &nbsp;(mean:
                    {formatCurrency(outlier.mean)}, σ: {formatCurrency(outlier.std_dev)})
                  </p>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </section>
    {/if}
  </div>
</section>

<style>
  .doctor-summary-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 0.75rem;
  }

  .doctor-summary-card {
    text-align: left;
    width: 100%;
    border: 1px solid #dbdbdb;
    transition:
      border-color 0.2s ease,
      box-shadow 0.2s ease;
  }

  .doctor-summary-card:hover {
    border-color: #b5b5b5;
  }

  .doctor-summary-card.is-focused {
    border-color: #3273dc;
    box-shadow: 0 0 0 1px #3273dc inset;
  }

  .doctor-controls {
    position: sticky;
    top: 0.75rem;
    z-index: 5;
  }

  .doctor-focus-toggle {
    width: 100%;
  }

  .doctor-focus-toggle .button {
    flex: 1;
  }

  .doctor-pagination {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  @media (max-width: 768px) {
    .doctor-controls {
      position: static;
    }
  }
</style>
