<script lang="ts">
  import { onMount } from "svelte";
  import COLORS from "$lib/colors";
  import {
    buildPriorityQueue,
    countIssuesByLevel,
    filterDuplicatePairs,
    filterIssues,
    filterOutliers,
    issueTone,
    normalizeDoctorQuery,
    type DoctorPriorityItem
  } from "$lib/doctor_v2";
  import { ajax, formatCurrency } from "$lib/utils";
  import type { DuplicatePair, OutlierTransaction, Issue } from "$lib/utils";
  import { dataQualityIssueCount } from "../../../../store";

  let issues: Issue[] = $state([]);
  let duplicates: DuplicatePair[] = $state([]);
  let outliers: OutlierTransaction[] = $state([]);
  let suppressLoading: Record<string, boolean> = $state({});
  let activeView: "all" | "duplicates" | "outliers" = $state("all");
  let issueSearchText = $state("");
  let issuePageSize = $state(25);
  let issuePage = $state(1);
  let searchText = $state("");
  let minConfidence = $state(0);
  let pageSize = $state(25);
  let duplicatePage = $state(1);
  let outlierPage = $state(1);

  let issueCounts = $derived(countIssuesByLevel(issues));
  let filteredIssues = $derived(filterIssues(issues, issueSearchText));
  let priorityQueue = $derived(buildPriorityQueue(issues, duplicates, outliers).slice(0, 6));

  let issuePageCount = $derived(Math.max(1, Math.ceil(filteredIssues.length / issuePageSize)));

  let pagedIssues = $derived.by(() => {
    const start = (issuePage - 1) * issuePageSize;
    return filteredIssues.slice(start, start + issuePageSize);
  });

  let filteredDuplicates = $derived(filterDuplicatePairs(duplicates, searchText, minConfidence));

  let filteredOutliers = $derived(filterOutliers(outliers, searchText, minConfidence));

  let duplicatePageCount = $derived(Math.max(1, Math.ceil(filteredDuplicates.length / pageSize)));
  let outlierPageCount = $derived(Math.max(1, Math.ceil(filteredOutliers.length / pageSize)));

  let visibleQueue = $derived(priorityQueue);

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
    const dq = await ajax("/api/diagnosis/duplicates");
    duplicates = dq.duplicates || [];
    outliers = dq.outliers || [];
  });

  $effect(() => {
    if (issuePage > issuePageCount) {
      issuePage = issuePageCount;
    }
    if (issuePage < 1) {
      issuePage = 1;
    }
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

  function applyFilters() {
    duplicatePage = 1;
    outlierPage = 1;
  }

  function setView(view: "all" | "duplicates" | "outliers") {
    activeView = view;
  }

  function updateIssueSearchText(event: Event) {
    issueSearchText = (event.currentTarget as HTMLInputElement).value;
    issuePage = 1;
  }

  function updateIssuePageSize(event: Event) {
    issuePageSize = Number((event.currentTarget as HTMLSelectElement).value) || 25;
    issuePage = 1;
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

  function toneColor(level: string) {
    const tone = issueTone(level);
    if (tone === "danger") return COLORS.lossText;
    if (tone === "warning") return "#e6a817";
    if (tone === "success") return COLORS.gainText;
    return "#4a90e2";
  }

  function queueBadgeColor(tone: string) {
    if (tone === "danger") return COLORS.lossText;
    if (tone === "warning") return "#e6a817";
    if (tone === "success") return COLORS.gainText;
    return "#4a90e2";
  }

  function toneClass(item: DoctorPriorityItem) {
    if (item.tone === "danger") return "is-danger";
    if (item.tone === "warning") return "is-warning";
    if (item.tone === "success") return "is-success";
    return "is-info";
  }

  function snippet(value: string, limit = 150) {
    const text = normalizeDoctorQuery(value.replace(/<[^>]+>/g, " "));
    if (text.length <= limit) return text;
    return `${text.slice(0, limit - 1)}…`;
  }
</script>

<section class="section tab-doctor">
  <div class="container is-fluid">
    <div class="notification is-light doctor-v2-banner">
      <div>
        <strong>Doctor V2 is available.</strong>
        <p class="mb-0">Use the new triage-first layout for a less cluttered review flow.</p>
      </div>
      <a class="button is-dark is-small" href="/more/doctor-v2">Open Doctor V2</a>
    </div>

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

    <section class="mb-5">
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
        <h2 class="title is-5 mb-0">Priority queue</h2>
        {#if visibleQueue.length > 0}
          <span class="tag is-light is-rounded">{visibleQueue.length} items shown</span>
        {/if}
      </div>

      {#if visibleQueue.length === 0}
        <p class="has-text-grey mb-0">No items in the current review queue.</p>
      {:else}
        <div class="doctor-queue-grid mb-5">
          {#each visibleQueue as item}
            <article class="box doctor-queue-card">
              <div class="is-flex is-justify-content-space-between is-align-items-start mb-2 gap-2">
                <div>
                  <span
                    class="tag is-light is-rounded"
                    style="background-color: {queueBadgeColor(item.tone)}; color: white"
                  >
                    {item.section}
                  </span>
                  <p class="doctor-queue-title mt-2">{item.title}</p>
                </div>
                <span class="tag is-rounded doctor-score-tag">{Math.round(item.score)} score</span>
              </div>
              <p class="is-size-7 has-text-grey">{item.subtitle}</p>
              <p class="doctor-queue-meta">{item.meta}</p>
            </article>
          {/each}
        </div>
      {/if}
    </section>

    <section class="mb-5">
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-2">
        <h2 class="title is-5 mb-0">Diagnosis findings</h2>
        {#if issues.length > 0}
          <span class="tag is-info is-light is-rounded">{issueCounts.danger || 0} critical</span>
        {/if}
      </div>

      <div class="box mb-3">
        <div class="columns is-multiline is-vcentered">
          <div class="column is-12-tablet is-8-desktop">
            <label class="label is-small mb-1" for="doctor-issue-search">Search findings</label>
            <input
              id="doctor-issue-search"
              class="input"
              type="text"
              placeholder="Summary, description, details"
              value={issueSearchText}
              oninput={updateIssueSearchText}
            />
          </div>
          <div class="column is-6-tablet is-4-desktop">
            <label class="label is-small mb-1" for="doctor-issue-page-size">Page size</label>
            <div class="select is-fullwidth">
              <select
                id="doctor-issue-page-size"
                value={issuePageSize}
                onchange={updateIssuePageSize}
              >
                <option value="10">10</option>
                <option value="25">25</option>
                <option value="50">50</option>
              </select>
            </div>
          </div>
        </div>
      </div>

      {#if issues.length === 0}
        <p class="has-text-grey">No diagnosis findings.</p>
      {:else if filteredIssues.length === 0}
        <p class="has-text-grey">No findings match your search.</p>
      {:else}
        <div class="doctor-pagination mb-3">
          <button
            class="button is-small"
            disabled={issuePage === 1}
            onclick={() => (issuePage = Math.max(1, issuePage - 1))}
          >
            Previous
          </button>
          <span class="is-size-7 has-text-grey">Page {issuePage} of {issuePageCount}</span>
          <button
            class="button is-small"
            disabled={issuePage === issuePageCount}
            onclick={() => (issuePage = Math.min(issuePageCount, issuePage + 1))}
          >
            Next
          </button>
        </div>

        <div class="doctor-issue-list">
          {#each pagedIssues as issue}
            <details class="box doctor-issue-card" open={issue.level === "danger"}>
              <summary class="doctor-issue-summary">
                <span
                  class="tag is-rounded issue-tone"
                  style="background-color: {toneColor(issue.level)}; color: white"
                >
                  {issueTone(issue.level)}
                </span>
                <div class="doctor-issue-summary-copy">
                  <p class="doctor-issue-title">{issue.summary}</p>
                  <p class="is-size-7 has-text-grey">
                    {snippet(issue.description || issue.details)}
                  </p>
                </div>
              </summary>
              <div class="doctor-issue-body issue-details">
                {@html `${issue.description} <br/> <br/> ${issue.details}`}
              </div>
            </details>
          {/each}
        </div>
      {/if}
    </section>

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
    {#if activeView === "duplicates"}
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
    {#if activeView === "outliers"}
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
  .doctor-v2-banner {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 1rem;
    margin-bottom: 1rem;
  }

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

  .doctor-queue-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
    gap: 0.75rem;
  }

  .doctor-queue-card {
    min-height: 100%;
    border: 1px solid rgba(255, 255, 255, 0.08);
    transition:
      transform 0.18s ease,
      border-color 0.18s ease;
  }

  .doctor-queue-card:hover {
    transform: translateY(-1px);
    border-color: rgba(255, 255, 255, 0.16);
  }

  .doctor-queue-title {
    font-weight: 700;
    line-height: 1.15;
  }

  .doctor-queue-meta {
    margin-top: 0.6rem;
    color: #9ca3af;
    font-size: 0.88rem;
  }

  .doctor-issue-list {
    display: grid;
    gap: 0.75rem;
  }

  .doctor-issue-card {
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 0.85rem;
    padding: 0;
    overflow: hidden;
  }

  .doctor-issue-summary {
    display: flex;
    align-items: flex-start;
    gap: 0.85rem;
    cursor: pointer;
    padding: 1rem;
    list-style: none;
  }

  .doctor-issue-summary::-webkit-details-marker {
    display: none;
  }

  .doctor-issue-summary-copy {
    min-width: 0;
  }

  .doctor-issue-title {
    font-weight: 700;
    line-height: 1.2;
    margin-bottom: 0.2rem;
  }

  .doctor-issue-body {
    padding: 0 1rem 1rem 1rem;
    overflow-wrap: anywhere;
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }

  .issue-tone {
    flex-shrink: 0;
  }

  .doctor-score-tag {
    background: rgba(255, 255, 255, 0.06);
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
    .doctor-v2-banner {
      flex-direction: column;
      align-items: flex-start;
    }

    .doctor-queue-grid {
      grid-template-columns: 1fr;
    }

    .doctor-issue-summary {
      flex-direction: column;
    }

    .doctor-controls {
      position: static;
    }
  }
</style>
