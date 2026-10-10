<script lang="ts">
  import type { DoctorSummary } from "$lib/doctor_v2";
  import HealthScoreRing from "./HealthScoreRing.svelte";

  let {
    summary,
    activeTab,
    onSelectTab
  }: {
    summary: DoctorSummary;
    activeTab: "fix" | "review" | "dismissed" | "all";
    onSelectTab: (tab: "fix" | "review" | "dismissed" | "all") => void;
  } = $props();

  const healthScore = $derived.by(() => {
    let penalty = summary.fix_count * 15 + summary.review_count * 5 + summary.info_count * 2;
    return Math.max(0, Math.min(100, 100 - penalty));
  });

  const isAllClear = $derived(summary.fix_count === 0 && summary.review_count === 0);
</script>

<div class="health-hero-card box mb-5">
  <div class="is-flex is-align-items-center gap-5 flex-wrap">
    <HealthScoreRing score={healthScore} size={110} strokeWidth={9} />

    <div class="hero-copy-column">
      <div class="is-flex is-align-items-center gap-2 mb-1">
        <h1 class="title is-3 mb-0">Data Health Studio</h1>
        {#if isAllClear}
          <span class="tag is-success is-light is-rounded">All Systems Operational</span>
        {:else if summary.fix_count > 0}
          <span class="tag is-danger is-rounded"
            >{summary.fix_count} Action Item{summary.fix_count > 1 ? "s" : ""}</span
          >
        {:else}
          <span class="tag is-warning is-rounded"
            >{summary.review_count} Review Item{summary.review_count > 1 ? "s" : ""}</span
          >
        {/if}
      </div>

      <p class="subtitle is-6 has-text-grey mb-3">
        {#if isAllClear}
          Your financial ledger passes all integrity checks cleanly with zero detected anomalies.
        {:else if summary.fix_count > 0}
          Found {summary.fix_count} critical issue{summary.fix_count > 1 ? "s" : ""} that require manual
          correction in your journal files.
        {:else}
          Found {summary.review_count} item{summary.review_count > 1 ? "s" : ""} to review for duplicates,
          price gaps, or unusual amounts.
        {/if}
      </p>

      <div class="hero-stats-row is-flex gap-4">
        <div class="stat-item">
          <span class="stat-value has-text-danger">{summary.fix_count}</span>
          <span class="stat-label">To Fix</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value has-text-warning"
            >{summary.review_count + summary.info_count}</span
          >
          <span class="stat-label">To Review</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value has-text-grey">{summary.dismissed_count}</span>
          <span class="stat-label">Dismissed</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value">{summary.total}</span>
          <span class="stat-label">Total Scanned</span>
        </div>
      </div>
    </div>
  </div>
</div>

<style>
  .health-hero-card {
    background:
      radial-gradient(circle at top left, rgba(50, 115, 220, 0.08), transparent 70%),
      rgba(18, 22, 31, 0.6);
    border: 1px solid rgba(255, 255, 255, 0.08);
    backdrop-filter: blur(12px);
    border-radius: 1rem;
    padding: 1.5rem 1.75rem;
  }

  .hero-copy-column {
    flex: 1;
    min-width: 260px;
  }

  .hero-stats-row {
    align-items: center;
  }

  .stat-item {
    display: flex;
    flex-direction: column;
  }

  .stat-value {
    font-size: 1.25rem;
    font-weight: 800;
    line-height: 1;
  }

  .stat-label {
    font-size: 0.65rem;
    font-weight: 700;
    letter-spacing: 0.05em;
    color: #9ca3af;
    text-transform: uppercase;
    margin-top: 3px;
  }

  .stat-divider {
    width: 1px;
    height: 24px;
    background: rgba(255, 255, 255, 0.1);
  }
</style>
