<script lang="ts">
  import Modal from "./Modal.svelte";
  import { jobsList, jobs } from "$lib/stores/jobs";
  import { statusTagClass, statusIconClass, formatTs, formatDuration } from "./sync_history_utils";

  export let open = false;

  /** Jobs displayed newest-first. */
  $: displayJobs = [...$jobsList].reverse();
</script>

<Modal bind:active={open} width="min(680px, 100vw)" footerClass="is-justify-content-right">
  <svelte:fragment slot="head" let:close>
    <p class="modal-card-title">
      <span class="icon is-small mr-1"><i class="fa-solid fa-clock-rotate-left" /></span>
      Sync History
    </p>
    <button class="delete" aria-label="Close sync history" on:click={(e) => close(e)} />
  </svelte:fragment>

  <div slot="body">
    {#if displayJobs.length === 0}
      <div class="has-text-centered has-text-grey py-6">
        <span class="icon is-large">
          <i class="fa-regular fa-clock fa-2x" />
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
                    <i class={statusIconClass(job.status)} aria-hidden="true" />
                  </span>
                  <span>&nbsp;{job.status}</span>
                </span>
                <span class="has-text-grey is-size-7" title={job.id}>
                  #{job.id.slice(0, 8)}
                </span>
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
                  <i class="fa-solid fa-triangle-exclamation" aria-hidden="true" />
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

  <svelte:fragment slot="foot" let:close>
    <button
      class="button is-light"
      disabled={displayJobs.length === 0}
      on:click={() => jobs.reset()}
    >
      Clear history
    </button>
    <button class="button" on:click={(e) => close(e)}>Close</button>
  </svelte:fragment>
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
