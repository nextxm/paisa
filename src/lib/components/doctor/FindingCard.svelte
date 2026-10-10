<script lang="ts">
  import type { DoctorFinding } from "$lib/doctor_v2";
  import { formatCurrency } from "$lib/utils";

  let {
    finding,
    onDismiss,
    onUndismiss
  }: {
    finding: DoctorFinding;
    onDismiss: (finding: DoctorFinding, note: string) => Promise<void>;
    onUndismiss: (finding: DoctorFinding) => Promise<void>;
  } = $props();

  let showDetails = $state(false);
  let showNoteInput = $state(false);
  let dismissNote = $state("");
  let isSubmitting = $state(false);

  function severityBadgeClass(severity: DoctorFinding["severity"]) {
    if (severity === "fix") return "is-danger";
    if (severity === "review") return "is-warning";
    return "is-info";
  }

  function severityLabel(severity: DoctorFinding["severity"]) {
    if (severity === "fix") return "FIX NEEDED";
    if (severity === "review") return "REVIEW";
    return "INFO";
  }

  const primaryAction = $derived(
    finding.actions.find((a) => a.type === "open_editor" || a.type === "open_url")
  );

  const evidenceSummary = $derived.by(() => {
    if (!finding.evidence || finding.evidence.length === 0) return null;
    const first = finding.evidence[0];
    if (finding.kind === "duplicate" && finding.evidence.length >= 2) {
      return `${first.account || "Account"} • ${first.amount ? formatCurrency(first.amount) : ""} • 2 matching entries`;
    }
    if (first.account || first.amount) {
      const parts = [
        first.account,
        first.payee,
        first.amount ? formatCurrency(first.amount) : null,
        first.date
      ].filter(Boolean);
      return parts.join(" • ");
    }
    return first.raw_message || null;
  });

  async function handleDismiss() {
    isSubmitting = true;
    try {
      await onDismiss(finding, dismissNote);
      showNoteInput = false;
      dismissNote = "";
    } finally {
      isSubmitting = false;
    }
  }

  async function handleUndismiss() {
    isSubmitting = true;
    try {
      await onUndismiss(finding);
    } finally {
      isSubmitting = false;
    }
  }
</script>

<article class="box minimalist-card mb-3 {finding.dismissed ? 'is-dismissed-state' : ''}">
  <div class="card-main-content">
    <div class="is-flex is-justify-content-space-between is-align-items-start gap-2 mb-2 flex-wrap">
      <div class="is-flex is-align-items-center gap-2 flex-wrap">
        {#if finding.dismissed}
          <span class="tag is-light is-rounded is-small">DISMISSED</span>
        {:else}
          <span
            class="tag {severityBadgeClass(finding.severity)} is-rounded is-small font-weight-bold"
          >
            {severityLabel(finding.severity)}
          </span>
        {/if}

        <h3 class="title is-6 mb-0 card-title-text">{finding.title}</h3>
      </div>

      {#if finding.confidence && finding.confidence > 0}
        <span class="tag is-rounded is-small is-light">
          {Math.round(finding.confidence * 100)}% match
        </span>
      {/if}
    </div>

    <p class="card-why-text mb-2">
      {finding.why_it_matters}
    </p>

    {#if evidenceSummary}
      <div class="evidence-chip-bar mb-3">
        <i class="fas fa-database mr-1 text-muted"></i>
        <span class="evidence-chip-text">{evidenceSummary}</span>
      </div>
    {/if}

    <div
      class="card-actions-bar is-flex is-justify-content-space-between is-align-items-center flex-wrap gap-2"
    >
      <div class="buttons are-small mb-0">
        {#if primaryAction}
          <a class="button is-link is-rounded action-btn-primary" href={primaryAction.url}>
            <span>{primaryAction.label}</span>
            <i class="fas fa-arrow-right ml-1"></i>
          </a>
        {/if}

        {#if finding.dismissed}
          <button
            type="button"
            class="button is-light is-rounded"
            class:is-loading={isSubmitting}
            onclick={handleUndismiss}
          >
            <i class="fas fa-rotate-left mr-1"></i>
            Restore
          </button>
        {:else if !showNoteInput}
          <button
            type="button"
            class="button is-light is-rounded"
            onclick={() => (showNoteInput = true)}
          >
            <i class="fas fa-eye-slash mr-1"></i>
            Dismiss
          </button>
        {/if}
      </div>

      <button
        type="button"
        class="button is-small is-ghost text-muted px-1"
        onclick={() => (showDetails = !showDetails)}
      >
        <span class="is-size-7">{showDetails ? "Hide details" : "Details"}</span>
        <i class="fas {showDetails ? 'fa-chevron-up' : 'fa-chevron-down'} ml-1"></i>
      </button>
    </div>

    {#if showNoteInput}
      <div class="field has-addons mt-3 mb-0">
        <div class="control is-expanded">
          <input
            class="input is-small"
            type="text"
            placeholder="Optional note (e.g. Verified manual entry)"
            bind:value={dismissNote}
          />
        </div>
        <div class="control">
          <button
            type="button"
            class="button is-small is-danger"
            class:is-loading={isSubmitting}
            onclick={handleDismiss}
          >
            Confirm
          </button>
        </div>
        <div class="control">
          <button
            type="button"
            class="button is-small is-light"
            onclick={() => (showNoteInput = false)}
          >
            Cancel
          </button>
        </div>
      </div>
    {/if}
  </div>

  {#if showDetails}
    <div class="card-details-drawer border-top p-3 mt-3">
      {#if finding.how_to_fix}
        <div class="notification is-light py-2 px-3 mb-3">
          <p class="is-size-7 mb-0">
            <i class="fas fa-lightbulb mr-1 has-text-warning"></i>
            <strong>Recommendation:</strong>
            {finding.how_to_fix}
          </p>
        </div>
      {/if}

      {#if finding.evidence && finding.evidence.length > 0}
        <p class="is-size-7 text-muted mb-1"><strong>Detailed Evidence Rows:</strong></p>
        <div class="table-container mb-3">
          <table class="table is-narrow is-fullwidth is-striped is-size-7">
            <thead>
              <tr>
                {#if finding.evidence.some((e) => e.date)}<th>Date</th>{/if}
                {#if finding.evidence.some((e) => e.payee)}<th>Payee</th>{/if}
                {#if finding.evidence.some((e) => e.account)}<th>Account</th>{/if}
                {#if finding.evidence.some((e) => e.amount)}<th>Amount</th>{/if}
                <th>Link</th>
              </tr>
            </thead>
            <tbody>
              {#each finding.evidence as ev}
                <tr>
                  {#if finding.evidence.some((e) => e.date)}<td>{ev.date || "-"}</td>{/if}
                  {#if finding.evidence.some((e) => e.payee)}<td>{ev.payee || "-"}</td>{/if}
                  {#if finding.evidence.some((e) => e.account)}<td
                      ><code>{ev.account || "-"}</code></td
                    >{/if}
                  {#if finding.evidence.some((e) => e.amount)}
                    <td class="has-text-weight-semibold"
                      >{ev.amount ? formatCurrency(ev.amount) : "-"}</td
                    >
                  {/if}
                  <td>
                    {#if ev.target_url}
                      <a href={ev.target_url} class="button is-small is-link is-light py-0">
                        {ev.file_name ? `${ev.file_name} L${ev.line_number}` : "Open"}
                      </a>
                    {/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}

      {#if finding.dismissed && finding.dismiss_note}
        <p class="is-size-7 text-muted mb-2">
          <i class="fas fa-comment-dots mr-1"></i> Note: {finding.dismiss_note}
        </p>
      {/if}

      <div class="is-size-7 text-muted">
        <span class="mr-3">Rule: <code>{finding.rule_id}</code></span>
        <span>ID: <code>{finding.id}</code></span>
      </div>
    </div>
  {/if}
</article>

<style>
  .minimalist-card {
    background: rgba(255, 255, 255, 0.02);
    border: 1px solid rgba(255, 255, 255, 0.08);
    border-radius: 0.85rem;
    padding: 1.15rem 1.35rem;
    transition:
      transform 0.18s ease,
      border-color 0.18s ease,
      box-shadow 0.18s ease;
  }

  .minimalist-card:hover {
    transform: translateY(-1px);
    border-color: rgba(255, 255, 255, 0.18);
    box-shadow: 0 4px 20px rgba(0, 0, 0, 0.12);
  }

  .is-dismissed-state {
    opacity: 0.65;
    filter: grayscale(0.2);
  }

  .card-title-text {
    font-weight: 700;
    line-height: 1.25;
  }

  .card-why-text {
    font-size: 0.92rem;
    line-height: 1.4;
    color: rgba(255, 255, 255, 0.85);
  }

  .evidence-chip-bar {
    display: inline-flex;
    align-items: center;
    background: rgba(255, 255, 255, 0.05);
    border-radius: 0.5rem;
    padding: 0.35rem 0.75rem;
    font-size: 0.82rem;
  }

  .evidence-chip-text {
    font-family: monospace;
    opacity: 0.9;
  }

  .action-btn-primary {
    font-weight: 600;
    padding-left: 1rem;
    padding-right: 1rem;
  }

  .text-muted {
    color: #9ca3af;
  }

  .border-top {
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }

  .card-details-drawer {
    background: rgba(0, 0, 0, 0.15);
    border-radius: 0.5rem;
  }
</style>
