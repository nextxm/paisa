<script lang="ts">
  import type { DoctorSummary } from "$lib/doctor_v2";

  let {
    summary,
    activeTab,
    onSelectTab
  }: {
    summary: DoctorSummary;
    activeTab: "fix" | "review" | "dismissed" | "all";
    onSelectTab: (tab: "fix" | "review" | "dismissed" | "all") => void;
  } = $props();

  const isAllClear = $derived(summary.fix_count === 0 && summary.review_count === 0);
</script>

<div class="box health-summary-box mb-5">
  <div class="is-flex is-justify-content-space-between is-align-items-center mb-4 flex-wrap gap-3">
    <div>
      <div class="is-flex is-align-items-center gap-2">
        <span
          class="icon is-medium {isAllClear
            ? 'has-text-success'
            : summary.fix_count > 0
              ? 'has-text-danger'
              : 'has-text-warning'}"
        >
          <i
            class="fas {isAllClear
              ? 'fa-circle-check'
              : summary.fix_count > 0
                ? 'fa-triangle-exclamation'
                : 'fa-circle-info'} fa-2x"
          ></i>
        </span>
        <div>
          <h1 class="title is-4 mb-0">Data Health Overview</h1>
          <p class="subtitle is-6 has-text-grey mb-0">
            {#if isAllClear}
              All financial data checks passed cleanly. No issues detected.
            {:else if summary.fix_count > 0}
              {summary.fix_count} issue{summary.fix_count > 1 ? "s" : ""} require fixing to ensure ledger
              accuracy.
            {:else}
              {summary.review_count} item{summary.review_count > 1 ? "s" : ""} ready for review.
            {/if}
          </p>
        </div>
      </div>
    </div>
  </div>

  <div class="summary-pills-grid">
    <button
      type="button"
      class="box summary-pill-card {activeTab === 'fix' ? 'is-active' : ''}"
      onclick={() => onSelectTab("fix")}
    >
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
        <span class="tag is-danger is-rounded">Needs Fixing</span>
        <span class="title is-5 mb-0 {summary.fix_count > 0 ? 'has-text-danger' : 'has-text-grey'}"
          >{summary.fix_count}</span
        >
      </div>
      <p class="is-size-7 has-text-grey">Data errors that impact financial reports</p>
    </button>

    <button
      type="button"
      class="box summary-pill-card {activeTab === 'review' ? 'is-active' : ''}"
      onclick={() => onSelectTab("review")}
    >
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
        <span class="tag is-warning is-rounded">To Review</span>
        <span
          class="title is-5 mb-0 {summary.review_count > 0 ? 'has-text-warning' : 'has-text-grey'}"
          >{summary.review_count + summary.info_count}</span
        >
      </div>
      <p class="is-size-7 has-text-grey">Potential duplicates, price gaps & outliers</p>
    </button>

    <button
      type="button"
      class="box summary-pill-card {activeTab === 'dismissed' ? 'is-active' : ''}"
      onclick={() => onSelectTab("dismissed")}
    >
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
        <span class="tag is-light is-rounded">Dismissed</span>
        <span class="title is-5 mb-0 has-text-grey">{summary.dismissed_count}</span>
      </div>
      <p class="is-size-7 has-text-grey">Ignored or confirmed false positives</p>
    </button>

    <button
      type="button"
      class="box summary-pill-card {activeTab === 'all' ? 'is-active' : ''}"
      onclick={() => onSelectTab("all")}
    >
      <div class="is-flex is-justify-content-space-between is-align-items-center mb-1">
        <span class="tag is-info is-light is-rounded">Total Scanned</span>
        <span class="title is-5 mb-0">{summary.total}</span>
      </div>
      <p class="is-size-7 has-text-grey">All data quality rules & signals</p>
    </button>
  </div>
</div>

<style>
  .summary-pills-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: 0.75rem;
  }

  .summary-pill-card {
    text-align: left;
    width: 100%;
    margin-bottom: 0;
    padding: 0.85rem 1rem;
    border: 1px solid rgba(255, 255, 255, 0.08);
    cursor: pointer;
    transition:
      border-color 0.18s ease,
      transform 0.18s ease;
  }

  .summary-pill-card:hover {
    transform: translateY(-1px);
    border-color: rgba(255, 255, 255, 0.2);
  }

  .summary-pill-card.is-active {
    border-color: #3273dc;
    box-shadow: 0 0 0 1px #3273dc inset;
  }
</style>
