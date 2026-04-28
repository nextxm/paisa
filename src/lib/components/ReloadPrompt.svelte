<script lang="ts">
  import { useRegisterSW } from "virtual:pwa-register/svelte";

  const { needRefresh, updateServiceWorker } = useRegisterSW();

  function reload() {
    updateServiceWorker(true);
  }

  function close() {
    needRefresh.set(false);
  }
</script>

{#if $needRefresh}
  <div class="reload-prompt notification is-info is-light invertable">
    <button class="delete" aria-label="Dismiss update prompt" on:click={close}></button>
    <p>New content available, click on reload button to update.</p>
    <div class="mt-2">
      <button class="button is-info is-small" on:click={reload}>Reload</button>
    </div>
  </div>
{/if}

<style>
  .reload-prompt {
    position: fixed;
    bottom: 1rem;
    right: 1rem;
    z-index: 9999;
    max-width: 24rem;
  }
</style>
