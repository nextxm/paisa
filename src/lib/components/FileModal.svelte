<script lang="ts">
  import Modal from "$lib/components/Modal.svelte";
  import _ from "lodash";

  let {
    label = "Save As",
    help = "Create or overwrite existing file",
    placeholder = "expense.ledger",
    open = $bindable(false),
    onsave
  }: {
    label?: string;
    help?: string;
    placeholder?: string;
    open?: boolean;
    onsave?: (file: string) => void;
  } = $props();
  let destinationFile = $state("");
</script>

<Modal bind:active={open}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">{label}</p>
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
      <label class="label" for="save-filename">File Name</label>
      <div class="control" id="save-filename">
        <input class="input" type="text" {placeholder} bind:value={destinationFile} />
        <p class="help">{help}</p>
      </div>
    </div>
  {/snippet}
  {#snippet foot(close)}
    <button
      class="du-btn du-btn-success du-btn-sm"
      disabled={_.isEmpty(destinationFile)}
      onclick={() => {
        onsave?.(destinationFile);
        close();
      }}>{label}</button
    >
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
  {/snippet}
</Modal>
