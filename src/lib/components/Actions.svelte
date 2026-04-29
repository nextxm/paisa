<script lang="ts">
  import { sync, startPolling } from "$lib/sync";
  import { isLoggedIn, logout } from "$lib/utils";
  import { refresh } from "../../store";
  import { obscure } from "../../persisted_store";
  import { goto } from "$app/navigation";
  import { jobsList } from "$lib/stores/jobs";
  import SyncHistoryOverlay from "./SyncHistoryOverlay.svelte";

  let showHistory = false;

  async function syncWithLoader(request: Record<string, any>) {
    const jobId = await sync(request);
    if (!jobId) return;
    startPolling(jobId, () => refresh());
  }

  let last = $obscure;
  obscure.subscribe(() => {
    if ($obscure === last) return;

    refresh();
  });

  function doLogout() {
    logout();
    goto("/login");
  }

  let showLogout = isLoggedIn();

  function toggleObscure() {
    obscure.set(!$obscure);
    refresh();
  }
</script>

<div class="is-flex is-align-items-center" style="gap: 0.25rem;">
  <SyncHistoryOverlay bind:open={showHistory} />

  <button
    class="navbar-action-button sync-history-btn"
    data-tippy-content="<p>Sync History</p>"
    aria-label="Sync History"
    onclick={() => (showHistory = true)}
  >
    <span class="icon">
      <i class="fa-solid fa-clock-rotate-left"></i>
    </span>
    {#if $jobsList.length > 0}
      <span class="sync-history-badge">{$jobsList.length}</span>
    {/if}
  </button>

  <button
    class="navbar-action-button"
    data-tippy-content="<p>Sync Journal</p>"
    aria-label="Sync Journal"
    onclick={(_e) => syncWithLoader({ journal: true })}
  >
    <span class="icon">
      <i class="fa-regular fa-file-lines"></i>
    </span>
  </button>

  <button
    class="navbar-action-button"
    data-tippy-content="<p>Update Prices</p>"
    aria-label="Update Prices"
    onclick={(_e) => syncWithLoader({ prices: true })}
  >
    <span class="icon">
      <i class="fas fa-dollar-sign"></i>
    </span>
  </button>

  <button
    type="button"
    class="navbar-action-button"
    data-tippy-content="<p>Update Mutual Fund Portfolios</p>"
    aria-label="Update Mutual Fund Portfolios"
    onclick={(_e) => syncWithLoader({ portfolios: true })}
  >
    <span class="icon">
      <i class="fas fa-layer-group"></i>
    </span>
  </button>

  <button
    type="button"
    class="navbar-action-button"
    data-tippy-content="<p>{$obscure ? 'Show' : 'Hide'} numbers</p>"
    aria-label="Toggle obscure numbers"
    onclick={(_e) => toggleObscure()}
  >
    <span class="icon">
      <i class="fas {$obscure ? 'fa-eye-slash' : 'fa-eye'}"></i>
    </span>
  </button>

  {#if showLogout}
    <button
      type="button"
      class="navbar-action-button"
      data-tippy-content="<p>Logout</p>"
      aria-label="Logout"
      onclick={(_e) => doLogout()}
    >
      <span class="icon">
        <i class="fas fa-arrow-right-from-bracket"></i>
      </span>
    </button>
  {/if}
</div>

<style lang="scss">
  .navbar-action-button {
    border: none;
    background: transparent;
    color: inherit;
    width: 1.9rem;
    height: 1.9rem;
    border-radius: 0.45rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition:
      background-color 120ms ease,
      color 120ms ease;
  }

  .navbar-action-button :global(.icon) {
    font-size: 0.95rem;
  }

  .navbar-action-button:hover,
  .navbar-action-button:focus-visible {
    background: rgba(127, 127, 127, 0.14);
  }

  @media screen and (max-width: 1023px) {
    .navbar-action-button {
      width: 2rem;
      height: 2rem;
      border-radius: 0.5rem;
    }

    .navbar-action-button :global(.icon) {
      font-size: 1rem;
    }
  }

  @media screen and (max-width: 640px) {
    .navbar-action-button {
      width: 2.2rem;
      height: 2.2rem;
      border-radius: 0.5rem;
    }

    .navbar-action-button :global(.icon) {
      font-size: 1.1rem;
    }
  }

  .sync-history-btn {
    position: relative;
  }

  .sync-history-badge {
    position: absolute;
    top: 0.1rem;
    right: 0.1rem;
    min-width: 0.9rem;
    height: 0.9rem;
    padding: 0 0.2rem;
    border-radius: 0.45rem;
    background: var(--bulma-primary, #485fc7);
    color: #fff;
    font-size: 0.55rem;
    font-weight: 700;
    line-height: 0.9rem;
    text-align: center;
    pointer-events: none;
  }
</style>
