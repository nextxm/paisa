<script lang="ts">
  import Modal from "$lib/components/Modal.svelte";

  let {
    open = $bindable(false),
    filterBase = "",
    onexport
  }: {
    open?: boolean;
    filterBase?: string;
    onexport: (options: { format: string; scope: string; zip: boolean }) => void;
  } = $props();

  let format = $state("hledger");
  let scope = $state("all");
  let zip = $state(false);

  $effect(() => {
    if (open) {
      scope = filterBase ? "filtered" : "all";
      format = "hledger";
      zip = false;
    }
  });

  function handleExport(close: () => void) {
    onexport({ format, scope, zip });
    close();
  }
</script>

<Modal bind:active={open}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">Export Prices</p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  {/snippet}

  {#snippet body()}
    <div class="field">
      <label class="label" for="export-format">Format</label>
      <div class="control">
        <div class="select is-fullwidth">
          <select id="export-format" bind:value={format}>
            <option value="ledger">Ledger</option>
            <option value="hledger">hLedger</option>
            <option value="beancount">Beancount</option>
          </select>
        </div>
      </div>
    </div>

    <div class="field">
      <label class="label" for="">Scope</label>
      <div class="control">
        <label class="radio">
          <input type="radio" name="scope" value="all" bind:group={scope} />
          All Commodities
        </label>
        {#if filterBase}
          <label class="radio">
            <input type="radio" name="scope" value="filtered" bind:group={scope} />
            Only {filterBase}
          </label>
        {/if}
      </div>
    </div>

    <div class="field">
      <label class="label" for="">Output Structure</label>
      <div class="control">
        <label class="radio">
          <input type="radio" name="zip" value={false} bind:group={zip} />
          Single File
        </label>
        <label class="radio">
          <input type="radio" name="zip" value={true} bind:group={zip} />
          One file per commodity (ZIP)
        </label>
      </div>
    </div>
  {/snippet}

  {#snippet foot(close)}
    <div class="flex gap-2 justify-end w-full">
      <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
      <button class="du-btn du-btn-primary du-btn-sm" onclick={() => handleExport(close)}>
        Export
      </button>
    </div>
  {/snippet}
</Modal>
