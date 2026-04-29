<script lang="ts">
  import Modal from "./Modal.svelte";
  import { jobsList, jobs } from "$lib/stores/jobs";
  import { statusTagClass, statusIconClass, formatTs, formatDuration } from "./sync_history_utils";

  let { open = $bindable(false) } = $props();

  /** Jobs displayed newest-first. */
  const displayJobs = $derived([...$jobsList].reverse());
</script>

<Modal bind:active={open} width="min(680px, 100vw)" footerClass="justify-end">
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">
      <span class="icon is-small mr-1"><i class="fa-solid fa-clock-rotate-left"></i></span>
      Sync History
    </p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="Close sync history"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}

  {#snippet body()}
    <div>
      {#if displayJobs.length === 0}
        <div class="has-text-centered has-text-grey py-6">
          <span class="icon is-large">
            <i class="fa-regular fa-clock fa-2x"></i>
          </span>
          <p class="mt-3">No sync jobs yet.</p>
        </div>
      {:else}
        <div class="sync-history-list">
          {#each displayJobs as job (job.id)}
            <div class="sync-history-item box p-3 mb-3">
              <div class="is-flex is-justify-content-space-between is-align-items-flex-start">
                <div class="is-flex is-align-items-center" style="gap: 0.5rem;">
                  <span class="tag {statusTagClass(job.status)} is-light">
                    <span class="icon is-small">
                      <i class={statusIconClass(job.status)} aria-hidden="true"></i>
                    </span>
                    <span>&nbsp;{job.status}</span>
                  </span>
                  <span class="has-text-grey is-size-7" title={job.id}>
                    #{job.id.slice(0, 8)}
                  </span>
                  {#if job.metadata}
                    <span class="has-text-weight-semibold is-size-7 ml-2">
                      {#if job.metadata.journal}Journal{/if}
                      {#if job.metadata.prices}{job.metadata.journal ? " + " : ""}Prices{/if}
                      {#if job.metadata.portfolios}
                        {(job.metadata.journal || job.metadata.prices) ? " + " : ""}Portfolios
                      {/if}
                    </span>
                  {/if}
                </div>
                <span class="has-text-grey is-size-7">{formatTs(job.created_at)}</span>
              </div>

              <div class="mt-2 is-size-7 has-text-grey">
                <div class="columns is-mobile is-gapless mb-0">
                  <div class="column">
                    <span class="has-text-weight-semibold">Started:</span>
                    {formatTs(job.started_at)}
                  </div>
                  <div class="column">
                    <span class="has-text-weight-semibold">Finished:</span>
                    {formatTs(job.finished_at)}
                  </div>
                  <div class="column is-narrow">
                    <span class="has-text-weight-semibold">Duration:</span>
                    {formatDuration(job)}
                  </div>
                </div>
              </div>

              {#if job.error}
                <div
                  class="mt-2 p-2 has-background-danger-light has-text-danger-dark is-size-7 sync-error-snippet"
                >
                  <span class="icon is-small mr-1">
                    <i class="fa-solid fa-triangle-exclamation" aria-hidden="true"></i>
                  </span>
                  {job.error}
                </div>
              {/if}

              {#if job.details && job.details.length > 0}
                <div class="mt-2">
                  <details>
                    <summary
                      class="is-size-7 has-text-grey has-text-weight-semibold"
                      style="cursor:pointer;"
                    >
                      {job.details.length} detail{job.details.length !== 1 ? "s" : ""}
                    </summary>
                    <ul class="mt-1 ml-3">
                      {#each job.details as detail}
                        <li class="is-size-7 has-text-grey">{detail}</li>
                      {/each}
                    </ul>
                  </details>
                </div>
              {/if}
            </div>
          {/each}
        </div>
      {/if}
    </div>
  {/snippet}

  {#snippet foot(close)}
    <button
      class="du-btn du-btn-ghost du-btn-sm"
      disabled={displayJobs.length === 0}
      onclick={() => jobs.reset()}
    >
      Clear history
    </button>
    <button class="du-btn du-btn-sm" onclick={() => close()}>Close</button>
  {/snippet}
</Modal>

<style lang="scss">
  .sync-history-list {
    max-height: 60vh;
    overflow-y: auto;
  }

  .sync-error-snippet {
    border-radius: 4px;
    word-break: break-word;
    white-space: pre-wrap;
  }
</style>
