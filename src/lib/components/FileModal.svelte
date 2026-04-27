<script lang="ts">
  import Modal from "$lib/components/Modal.svelte";
  import _ from "lodash";
  import { createEventDispatcher } from "svelte";

  export let label = "Save As";
  export let help = "Create or overwrite existing file";
  export let placeholder = "expense.ledger";
  export let open = false;
  let destinationFile = "";

  const dispatch = createEventDispatcher();
</script>

<Modal bind:active={open}>
  <svelte:fragment slot="head" let:close>
    <p class="text-base font-semibold flex-1">{label}</p>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true" />
    </button>
  </svelte:fragment>
  <div class="field" slot="body">
    <label class="label" for="save-filename">File Name</label>
    <div class="control" id="save-filename">
      <input class="input" type="text" {placeholder} bind:value={destinationFile} />
      <p class="help">{help}</p>
    </div>
  </div>
  <svelte:fragment slot="foot" let:close>
    <button
      class="du-btn du-btn-success du-btn-sm"
      disabled={_.isEmpty(destinationFile)}
      onclick={() => dispatch("save", destinationFile) && close()}>{label}</button
    >
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
  </svelte:fragment>
</Modal>
