<script lang="ts">
  import { sync, startPolling } from "$lib/sync";
  import { isLoggedIn, logout } from "$lib/utils";
  import { refresh } from "../../store";
  import { obscure } from "../../persisted_store";
  import { goto } from "$app/navigation";

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
  <button
    class="navbar-action-button"
    data-tippy-content="<p>Sync Journal</p>"
    aria-label="Sync Journal"
    on:click={(_e) => syncWithLoader({ journal: true })}
  >
    <span class="icon">
      <i class="fa-regular fa-file-lines" />
    </span>
  </button>

  <button
    class="navbar-action-button"
    data-tippy-content="<p>Update Prices</p>"
    aria-label="Update Prices"
    on:click={(_e) => syncWithLoader({ prices: true })}
  >
    <span class="icon">
      <i class="fas fa-dollar-sign" />
    </span>
  </button>

  <button
    type="button"
    class="navbar-action-button"
    data-tippy-content="<p>Update Mutual Fund Portfolios</p>"
    aria-label="Update Mutual Fund Portfolios"
    on:click={(_e) => syncWithLoader({ portfolios: true })}
  >
    <span class="icon">
      <i class="fas fa-layer-group" />
    </span>
  </button>

  <button
    type="button"
    class="navbar-action-button"
    data-tippy-content="<p>{$obscure ? 'Show' : 'Hide'} numbers</p>"
    aria-label="Toggle obscure numbers"
    on:click={(_e) => toggleObscure()}
  >
    <span class="icon">
      <i class="fas {$obscure ? 'fa-eye-slash' : 'fa-eye'}" />
    </span>
  </button>

  {#if showLogout}
    <button
      type="button"
      class="navbar-action-button"
      data-tippy-content="<p>Logout</p>"
      aria-label="Logout"
      on:click={(_e) => doLogout()}
    >
      <span class="icon">
        <i class="fas fa-arrow-right-from-bracket" />
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
</style>
