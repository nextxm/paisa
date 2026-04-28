<script lang="ts">
  import { createEventDispatcher } from "svelte";
  import Modal from "./Modal.svelte";
  import type { MergeView } from "@codemirror/merge";
  import { createDiffEditor } from "$lib/editor";
  import type { LedgerFile } from "$lib/utils";
  let editorDom: Element;
  let editor: MergeView;
  export let oldFiles: LedgerFile[] = [];
  export let newFiles: LedgerFile[] = [];
  export let updatedTransactionsCount = 0;
  let changedOldFiles: LedgerFile[] = [];
  let changedNewFiles: LedgerFile[] = [];
  let selectedFileIndex = 0;

  const dispatch = createEventDispatcher();
  export let open = false;

  $: {
    changedOldFiles = [];
    changedNewFiles = [];
    for (let i = 0; i < newFiles.length; i++) {
      if (oldFiles[i].content !== newFiles[i].content) {
        changedOldFiles.push(oldFiles[i]);
        changedNewFiles.push(newFiles[i]);
      }
    }
  }

  $: if (open) {
    if (editor) {
      editor.destroy();
    }

    if (changedOldFiles.length > 0) {
      editor = createDiffEditor(
        changedOldFiles[selectedFileIndex].content,
        changedNewFiles[selectedFileIndex].content,
        editorDom
      );
    }
  }
</script>

<Modal
  bind:active={open}
  width="min(1300px, 100vw)"
  bodyClass="p-0 min-h-[500px]"
  headerClass="pt-1 pb-1"
  footerClass="justify-end"
>
  <svelte:fragment slot="head" let:close>
    <p class="text-base font-semibold flex-1">
      {#if changedOldFiles.length > 0}
        {changedOldFiles[selectedFileIndex]?.name}
        [{selectedFileIndex + 1}/{changedNewFiles.length}]
      {:else}
        No Changes
      {/if}
    </p>
    <div class="flex items-center gap-2 mr-3">
      {#if changedOldFiles.length > 0}
        <span class="text-sm mt-2"><b>{updatedTransactionsCount}</b> transaction(s) changed</span>
      {/if}
      <button
        class="du-btn du-btn-sm"
        disabled={selectedFileIndex <= 0}
        onclick={() => selectedFileIndex--}
      >
        <span class="icon is-small">
          <i class="fas fa-chevron-left"></i>
        </span>
        <span>Prev</span>
      </button>
      <button
        class="du-btn du-btn-sm"
        disabled={selectedFileIndex >= changedNewFiles.length - 1}
        onclick={() => selectedFileIndex++}
      >
        <span>Next</span>
        <span class="icon is-small">
          <i class="fas fa-chevron-right"></i>
        </span>
      </button>
    </div>
    <button
      class="du-btn du-btn-sm du-btn-circle du-btn-ghost"
      aria-label="close"
      onclick={() => close()}
    >
      <i class="fas fa-times" aria-hidden="true"></i>
    </button>
  </svelte:fragment>
  <div class="field" slot="body">
    <div class="box py-0">
      <div class="diff-editor" bind:this={editorDom}></div>
      {#if changedOldFiles.length === 0}
        <div class="has-text-centered mt-6">
          <strong>Oops!</strong> No changes has been made. Make sure the bulk edit arguments are correct.
        </div>
      {/if}
    </div>
  </div>
  <svelte:fragment slot="foot" let:close>
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
    {#if changedOldFiles.length > 0}
      <button
        class="du-btn du-btn-success du-btn-sm"
        onclick={() => dispatch("save", changedNewFiles) && close()}>Save All</button
      >
    {/if}
  </svelte:fragment>
</Modal>
