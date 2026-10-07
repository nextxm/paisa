<script lang="ts">
  import { onMount } from "svelte";
  import HealthSummary from "$lib/components/doctor/HealthSummary.svelte";
  import FindingGroup from "$lib/components/doctor/FindingGroup.svelte";
  import {
    filterDoctorFindings,
    groupDoctorFindings,
    type DoctorFinding,
    type DoctorSummary
  } from "$lib/doctor_v2";
  import { ajax } from "$lib/utils";
  import { dataQualityIssueCount } from "../../../../store";

  let findings: DoctorFinding[] = $state([]);
  let summary: DoctorSummary = $state({
    total: 0,
    fix_count: 0,
    review_count: 0,
    info_count: 0,
    dismissed_count: 0
  });

  let isLoading = $state(true);
  let searchText = $state("");
  let activeTab: "fix" | "review" | "dismissed" | "all" = $state("fix");

  async function loadFindings() {
    isLoading = true;
    try {
      const res = await ajax("/api/doctor/findings");
      findings = res.findings || [];
      summary = res.summary || {
        total: 0,
        fix_count: 0,
        review_count: 0,
        info_count: 0,
        dismissed_count: 0
      };
      if (summary.fix_count === 0 && summary.review_count > 0 && activeTab === "fix") {
        activeTab = "review";
      } else if (summary.fix_count === 0 && summary.review_count === 0 && activeTab === "fix") {
        activeTab = "all";
      }
      dataQualityIssueCount.set(summary.fix_count + summary.review_count);
    } finally {
      isLoading = false;
    }
  }

  onMount(async () => {
    await loadFindings();
  });

  let tabFilteredFindings = $derived.by(() => {
    if (activeTab === "fix") {
      return findings.filter((f) => !f.dismissed && f.severity === "fix");
    }
    if (activeTab === "review") {
      return findings.filter(
        (f) => !f.dismissed && (f.severity === "review" || f.severity === "info")
      );
    }
    if (activeTab === "dismissed") {
      return findings.filter((f) => f.dismissed);
    }
    return findings;
  });

  let searchFilteredFindings = $derived(filterDoctorFindings(tabFilteredFindings, searchText));
  let groupedFindings = $derived(groupDoctorFindings(searchFilteredFindings));

  async function handleDismiss(finding: DoctorFinding, note: string) {
    await ajax("/api/doctor/dismiss", {
      method: "POST",
      body: JSON.stringify({
        fingerprint: finding.id,
        rule_id: finding.rule_id,
        note
      })
    });
    await loadFindings();
  }

  async function handleUndismiss(finding: DoctorFinding) {
    await ajax("/api/doctor/undismiss", {
      method: "POST",
      body: JSON.stringify({
        fingerprint: finding.id
      })
    });
    await loadFindings();
  }
</script>

<section class="section data-health-page py-5">
  <div class="container is-max-desktop">
    <HealthSummary {summary} {activeTab} onSelectTab={(tab) => (activeTab = tab)} />

    <!-- Segmented Navigation Bar -->
    <div
      class="tab-toolbar mb-4 is-flex is-justify-content-space-between is-align-items-center flex-wrap gap-3"
    >
      <div class="segmented-control">
        <button
          type="button"
          class="segmented-btn {activeTab === 'fix' ? 'is-active-fix' : ''}"
          onclick={() => (activeTab = "fix")}
        >
          <span>Fix Now</span>
          {#if summary.fix_count > 0}
            <span class="count-badge badge-fix">{summary.fix_count}</span>
          {/if}
        </button>

        <button
          type="button"
          class="segmented-btn {activeTab === 'review' ? 'is-active-review' : ''}"
          onclick={() => (activeTab = "review")}
        >
          <span>To Review</span>
          {#if summary.review_count + summary.info_count > 0}
            <span class="count-badge badge-review">{summary.review_count + summary.info_count}</span
            >
          {/if}
        </button>

        <button
          type="button"
          class="segmented-btn {activeTab === 'dismissed' ? 'is-active-dismissed' : ''}"
          onclick={() => (activeTab = "dismissed")}
        >
          <span>Dismissed</span>
          {#if summary.dismissed_count > 0}
            <span class="count-badge badge-muted">{summary.dismissed_count}</span>
          {/if}
        </button>

        <button
          type="button"
          class="segmented-btn {activeTab === 'all' ? 'is-active-all' : ''}"
          onclick={() => (activeTab = "all")}
        >
          <span>All Items</span>
          <span class="count-badge">{summary.total}</span>
        </button>
      </div>

      <div class="search-input-wrapper">
        <div class="control has-icons-left has-icons-right">
          <input
            class="input is-small is-rounded search-field"
            type="text"
            placeholder="Search findings..."
            bind:value={searchText}
          />
          <span class="icon is-small is-left">
            <i class="fas fa-search text-muted"></i>
          </span>
          {#if searchText}
            <button
              type="button"
              class="icon is-small is-right is-clear-btn"
              onclick={() => (searchText = "")}
              aria-label="Clear search"
            >
              <i class="fas fa-times"></i>
            </button>
          {/if}
        </div>
      </div>
    </div>

    <!-- Content Section -->
    {#if isLoading}
      <div class="has-text-centered py-6">
        <span class="icon is-large has-text-link">
          <i class="fas fa-spinner fa-spin fa-2x"></i>
        </span>
        <p class="has-text-grey mt-2">Checking ledger data...</p>
      </div>
    {:else if searchFilteredFindings.length === 0}
      <div class="box zero-state-card has-text-centered py-6 my-4">
        <div class="zero-state-icon-wrapper mb-3">
          <span class="icon is-large has-text-success">
            <i class="fas fa-shield-check fa-3x"></i>
          </span>
        </div>

        {#if searchText}
          <h2 class="title is-5 mb-2">No matching items found</h2>
          <p class="has-text-grey mb-3">No findings match "{searchText}".</p>
          <button
            type="button"
            class="button is-small is-rounded is-light"
            onclick={() => (searchText = "")}
          >
            Clear Search
          </button>
        {:else if activeTab === "fix"}
          <h2 class="title is-5 mb-2">Zero Critical Issues</h2>
          <p class="has-text-grey mb-0">
            Your ledger contains no negative balances or sign errors.
          </p>
        {:else if activeTab === "review"}
          <h2 class="title is-5 mb-2">No Items Needing Review</h2>
          <p class="has-text-grey mb-0">
            No duplicates, price gaps or statistical outliers detected.
          </p>
        {:else if activeTab === "dismissed"}
          <h2 class="title is-5 mb-2">No Dismissed Items</h2>
          <p class="has-text-grey mb-0">You haven't dismissed any findings yet.</p>
        {:else}
          <h2 class="title is-5 mb-2">Your Ledger is 100% Healthy</h2>
          <p class="has-text-grey mb-0">All data checks passed cleanly with no issues.</p>
        {/if}
      </div>
    {:else}
      <div class="focus-group-list">
        {#each groupedFindings as group (group.rule_id)}
          <FindingGroup
            groupTitle={group.title}
            ruleId={group.rule_id}
            findings={group.findings}
            initiallyOpen={true}
            onDismiss={handleDismiss}
            onUndismiss={handleUndismiss}
          />
        {/each}
      </div>
    {/if}
  </div>
</section>

<style>
  .data-health-page {
    min-height: 85vh;
  }

  .segmented-control {
    display: inline-flex;
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 2rem;
    padding: 3px;
    gap: 2px;
  }

  .segmented-btn {
    display: inline-flex;
    align-items: center;
    gap: 0.4rem;
    background: transparent;
    border: 0;
    border-radius: 1.5rem;
    padding: 0.35rem 0.9rem;
    font-size: 0.82rem;
    font-weight: 600;
    color: rgba(255, 255, 255, 0.7);
    cursor: pointer;
    transition: all 0.18s ease;
  }

  .segmented-btn:hover {
    color: #fff;
    background: rgba(255, 255, 255, 0.05);
  }

  .segmented-btn.is-active-fix {
    background: #f14668;
    color: #fff;
  }

  .segmented-btn.is-active-review {
    background: #ffe08a;
    color: #12161f;
  }

  .segmented-btn.is-active-dismissed {
    background: rgba(255, 255, 255, 0.15);
    color: #fff;
  }

  .segmented-btn.is-active-all {
    background: #3273dc;
    color: #fff;
  }

  .count-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: rgba(255, 255, 255, 0.18);
    border-radius: 1rem;
    padding: 0.05rem 0.45rem;
    font-size: 0.72rem;
    font-weight: 700;
    line-height: 1;
  }

  .badge-fix {
    background: rgba(255, 255, 255, 0.25);
  }

  .badge-review {
    background: rgba(0, 0, 0, 0.2);
  }

  .search-input-wrapper {
    min-width: 220px;
  }

  .search-field {
    background: rgba(255, 255, 255, 0.05);
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .search-field:focus {
    border-color: #3273dc;
  }

  .is-clear-btn {
    background: transparent;
    border: 0;
    cursor: pointer;
    pointer-events: auto;
  }

  .zero-state-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px dashed rgba(255, 255, 255, 0.12);
    border-radius: 1rem;
  }

  .text-muted {
    color: #9ca3af;
  }
</style>
