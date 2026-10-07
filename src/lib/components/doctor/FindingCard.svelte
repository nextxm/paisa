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

  function severityTagClass(severity: DoctorFinding["severity"]) {
    if (severity === "fix") return "is-danger";
    if (severity === "review") return "is-warning";
    return "is-info";
  }

  function severityLabel(severity: DoctorFinding["severity"]) {
    if (severity === "fix") return "Needs Fix";
    if (severity === "review") return "To Review";
    return "Info";
  }

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

<article class="box finding-card mb-4 {finding.dismissed ? 'is-dismissed-card' : ''}">
  <header
    class="is-flex is-justify-content-space-between is-align-items-start gap-2 mb-3 flex-wrap"
  >
    <div class="is-flex is-align-items-center gap-2 flex-wrap">
      {#if finding.dismissed}
        <span class="tag is-light is-rounded">Dismissed</span>
      {:else}
        <span class="tag {severityTagClass(finding.severity)} is-rounded">
          {severityLabel(finding.severity)}
        </span>
      {/if}

      {#if finding.kind === "duplicate"}
        <span class="tag is-warning is-light is-rounded">Duplicate Signal</span>
      {:else if finding.kind === "outlier"}
        <span class="tag is-danger is-light is-rounded">Outlier Signal</span>
      {/if}

      <h3 class="title is-5 mb-0 finding-title">{finding.title}</h3>
    </div>

    {#if finding.confidence && finding.confidence > 0}
      <span class="tag is-rounded is-light">
        {Math.round(finding.confidence * 100)}% confidence
      </span>
    {/if}
  </header>

  <p class="finding-why mb-3">
    <strong>Why it matters:</strong>
    {finding.why_it_matters}
  </p>

  {#if finding.how_to_fix}
    <div class="notification is-light py-2 px-3 mb-3 fix-guidance">
      <p class="is-size-7 mb-0">
        <i class="fas fa-lightbulb mr-1 has-text-warning"></i>
        <strong>How to fix:</strong>
        {finding.how_to_fix}
      </p>
    </div>
  {/if}

  {#if finding.evidence && finding.evidence.length > 0}
    <div class="evidence-container mb-3">
      <p class="is-size-7 has-text-grey mb-1"><strong>Evidence / Entries:</strong></p>
      <div class="table-container mb-0">
        <table class="table is-narrow is-fullwidth is-striped is-hoverable is-size-7">
          <thead>
            <tr>
              {#if finding.evidence.some((e) => e.date)}<th>Date</th>{/if}
              {#if finding.evidence.some((e) => e.payee)}<th>Payee</th>{/if}
              {#if finding.evidence.some((e) => e.account)}<th>Account</th>{/if}
              {#if finding.evidence.some((e) => e.amount)}<th>Amount</th>{/if}
              {#if finding.evidence.some((e) => e.sigma)}<th>Metrics</th>{/if}
              <th>Location / Link</th>
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
                  <td class="has-text-weight-semibold">
                    {ev.amount ? formatCurrency(ev.amount) : "-"}
                  </td>
                {/if}
                {#if finding.evidence.some((e) => e.sigma)}
                  <td>
                    {#if ev.sigma}
                      <span class="tag is-small is-light">{ev.sigma.toFixed(1)}σ above mean</span>
                    {:else}
                      -
                    {/if}
                  </td>
                {/if}
                <td>
                  {#if ev.target_url}
                    <a href={ev.target_url} class="button is-small is-link is-light py-0">
                      <i class="fas fa-arrow-up-right-from-square mr-1"></i>
                      {ev.file_name ? `${ev.file_name} L${ev.line_number}` : "Open"}
                    </a>
                  {:else if ev.raw_message}
                    <span>{@html ev.raw_message}</span>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    </div>
  {/if}

  {#if finding.dismissed && finding.dismiss_note}
    <p class="is-size-7 has-text-grey mb-3 italic">
      <i class="fas fa-sticky-note mr-1"></i> Dismiss note: {finding.dismiss_note}
    </p>
  {/if}

  <footer
    class="is-flex is-justify-content-space-between is-align-items-center flex-wrap gap-2 pt-2 border-top"
  >
    <div class="buttons are-small mb-0">
      {#each finding.actions as action}
        {#if action.type === "open_editor" || action.type === "open_url"}
          <a class="button is-link" href={action.url}>
            <i class="fas fa-pen-to-square mr-1"></i>
            {action.label}
          </a>
        {/if}
      {/each}

      {#if finding.dismissed}
        <button
          type="button"
          class="button is-light"
          class:is-loading={isSubmitting}
          onclick={handleUndismiss}
        >
          <i class="fas fa-rotate-left mr-1"></i>
          Undo Dismissal
        </button>
      {:else if !showNoteInput}
        <button type="button" class="button is-light" onclick={() => (showNoteInput = true)}>
          <i class="fas fa-eye-slash mr-1"></i>
          Dismiss / Ignore
        </button>
      {/if}
    </div>

    <button
      type="button"
      class="button is-small is-ghost px-1"
      onclick={() => (showDetails = !showDetails)}
    >
      <span class="is-size-7">
        {showDetails ? "Hide details" : "Show details"}
      </span>
      <i class="fas {showDetails ? 'fa-chevron-up' : 'fa-chevron-down'} ml-1"></i>
    </button>
  </footer>

  {#if showNoteInput}
    <div class="field has-addons mt-3 mb-0">
      <div class="control is-expanded">
        <input
          class="input is-small"
          type="text"
          placeholder="Optional note (e.g. Known reimbursement, verified sign)"
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
          Confirm Dismiss
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

  {#if showDetails}
    <div class="box is-shadowless border mt-3 mb-0 py-2 px-3 is-size-7 details-panel">
      <p class="mb-1"><strong>Rule ID:</strong> <code>{finding.rule_id}</code></p>
      <p class="mb-1"><strong>Fingerprint:</strong> <code>{finding.id}</code></p>
      <p class="mb-0"><strong>Technical Details:</strong> {@html finding.details}</p>
    </div>
  {/if}
</article>

<style>
  .finding-card {
    border: 1px solid rgba(255, 255, 255, 0.08);
    transition: border-color 0.18s ease;
  }

  .finding-card:hover {
    border-color: rgba(255, 255, 255, 0.16);
  }

  .is-dismissed-card {
    opacity: 0.75;
    background: rgba(0, 0, 0, 0.02);
  }

  .finding-title {
    line-height: 1.2;
  }

  .finding-why {
    line-height: 1.35;
  }

  .fix-guidance {
    border-left: 3px solid #ffdd57;
  }

  .border-top {
    border-top: 1px solid rgba(255, 255, 255, 0.08);
  }

  .details-panel {
    background: rgba(0, 0, 0, 0.04);
  }
</style>
