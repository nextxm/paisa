<script lang="ts">
  import { sync } from "$lib/sync";
  import { isLoggedIn, isMobile, logout } from "$lib/utils";
  import { refresh } from "../../store";
  import { obscure } from "../../persisted_store";
  import { goto } from "$app/navigation";

  async function syncWithLoader(request: Record<string, any>) {
    try {
      await sync(request);
    } finally {
      refresh();
    }
  }

  const obscureId = "obscure";
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
</script>

<div class="is-flex is-align-items-center ml-2" style="gap: 0.25rem;">
  <button
    class="button action-button is-rounded"
    data-tippy-content="<p>Sync Journal</p>"
    aria-label="Sync Journal"
    on:click={(_e) => syncWithLoader({ journal: true })}
  >
    <span class="icon">
      <i class="fas fa-file-lines" />
    </span>
  </button>

  <button
    class="button action-button is-rounded"
    data-tippy-content="<p>Update Prices</p>"
    aria-label="Update Prices"
    on:click={(_e) => syncWithLoader({ prices: true })}
  >
    <span class="icon">
      <i class="fas fa-dollar-sign" />
    </span>
  </button>

  <div class="dropdown is-hoverable {isMobile() ? 'is-left' : 'is-right'}">
    <div class="dropdown-trigger">
      <button
        class="button action-button is-rounded"
        aria-haspopup="true"
        aria-label="More actions"
      >
        <span class="icon">
          <i class="fas fa-ellipsis-vertical" />
        </span>
      </button>
    </div>
    <div class="dropdown-menu" id="dropdown-menu4" role="menu">
      <div class="dropdown-content">
        <button
          type="button"
          on:click={(_e) => syncWithLoader({ portfolios: true })}
          class="dropdown-item icon-text"
        >
          <span class="icon is-small">
            <i class="fas fa-layer-group" />
          </span>
          <span>Update Mutual Fund Portfolios</span>
        </button>
        <hr class="dropdown-divider" />
        <a class="dropdown-item icon-text">
          <label for={obscureId} class="cursor-pointer w-full inline-block">
            <input bind:checked={$obscure} id={obscureId} type="checkbox" class="is-hidden" />
            <span class="ml-0 icon is-small">
              <i class="fas {$obscure ? 'fa-eye-slash' : 'fa-eye'}" />
            </span>
            <span>{$obscure ? "Show" : "Hide"} numbers</span>
          </label>
        </a>
        {#if showLogout}
          <hr class="dropdown-divider" />
          <button type="button" on:click={(_e) => doLogout()} class="dropdown-item icon-text">
            <span class="icon is-small">
              <i class="fas fa-arrow-right-from-bracket" />
            </span>
            <span>Logout</span>
          </button>
        {/if}
      </div>
    </div>
  </div>
</div>
