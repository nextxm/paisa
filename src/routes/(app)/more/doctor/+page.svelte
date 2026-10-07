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
      // Auto-switch to 'review' or 'all' if no fix items exist
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

<section class="section data-health-page">
  <div class="container is-fluid">
    <HealthSummary {summary} {activeTab} onSelectTab={(tab) => (activeTab = tab)} />

    <div class="box controls-box mb-4">
      <div class="columns is-vcentered">
        <div class="column is-8-desktop is-12-tablet">
          <div class="field mb-0">
            <label class="label is-small" for="health-search-input">Search Findings</label>
            <div class="control has-icons-left">
              <input
                id="health-search-input"
                class="input"
                type="text"
                placeholder="Search by title, payee, account, date or details..."
                bind:value={searchText}
              />
              <span class="icon is-small is-left">
                <i class="fas fa-search"></i>
              </span>
            </div>
          </div>
        </div>

        <div class="column is-4-desktop is-12-tablet">
          <label class="label is-small" for="tab-filter-buttons">Category Filter</label>
          <div id="tab-filter-buttons" class="buttons has-addons is-fullwidth mb-0">
            <button
              type="button"
              class="button is-small {activeTab === 'fix' ? 'is-danger' : ''}"
              onclick={() => (activeTab = "fix")}
            >
              Fix ({summary.fix_count})
            </button>
            <button
              type="button"
              class="button is-small {activeTab === 'review' ? 'is-warning' : ''}"
              onclick={() => (activeTab = "review")}
            >
              Review ({summary.review_count + summary.info_count})
            </button>
            <button
              type="button"
              class="button is-small {activeTab === 'dismissed' ? 'is-dark' : ''}"
              onclick={() => (activeTab = "dismissed")}
            >
              Dismissed ({summary.dismissed_count})
            </button>
            <button
              type="button"
              class="button is-small {activeTab === 'all' ? 'is-link' : ''}"
              onclick={() => (activeTab = "all")}
            >
              All ({summary.total})
            </button>
          </div>
        </div>
      </div>
    </div>

    {#if isLoading}
      <div class="has-text-centered py-6">
        <span class="icon is-large has-text-link">
          <i class="fas fa-spinner fa-spin fa-2x"></i>
        </span>
        <p class="has-text-grey mt-2">Scanning financial ledger for issues...</p>
      </div>
    {:else if searchFilteredFindings.length === 0}
      <div class="box empty-state-box has-text-centered py-6">
        <span class="icon is-large has-text-success mb-3">
          <i class="fas fa-shield-check fa-3x"></i>
        </span>
        {#if searchText}
          <h2 class="title is-5 mb-2">No matching findings</h2>
          <p class="has-text-grey mb-3">No findings match your search query "{searchText}".</p>
          <button type="button" class="button is-small is-light" onclick={() => (searchText = "")}>
            Clear Search
          </button>
        {:else if activeTab === "fix"}
          <h2 class="title is-5 mb-2">No items need fixing</h2>
          <p class="has-text-grey mb-0">
            Great news! There are no critical data errors or negative balances in your ledger.
          </p>
        {:else if activeTab === "review"}
          <h2 class="title is-5 mb-2">No items to review</h2>
          <p class="has-text-grey mb-0">
            No potential duplicates, price gaps, or statistical outliers detected.
          </p>
        {:else if activeTab === "dismissed"}
          <h2 class="title is-5 mb-2">No dismissed items</h2>
          <p class="has-text-grey mb-0">You have not dismissed any data health findings.</p>
        {:else}
          <h2 class="title is-5 mb-2">All Clear!</h2>
          <p class="has-text-grey mb-0">
            Your financial ledger passed all Data Health checks cleanly.
          </p>
        {/if}
      </div>
    {:else}
      <div class="findings-list">
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
  .controls-box {
    border: 1px solid rgba(255, 255, 255, 0.08);
  }

  .empty-state-box {
    border: 1px dashed rgba(255, 255, 255, 0.15);
  }

  .findings-list {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }
</style>
