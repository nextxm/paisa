<script lang="ts">
  import { createEventDispatcher } from "svelte";

  export let multiple: boolean = true;
  export let accept: string = "";
  export let inputElement: HTMLInputElement | undefined = undefined;
  export let disabled: boolean = false;

  const dispatch = createEventDispatcher();
  let dragging = false;
  let fileInput: HTMLInputElement;

  function getAcceptedFiles(files: FileList) {
    if (!accept) return { acceptedFiles: Array.from(files), fileRejections: [] as File[] };
    const types = accept.split(",").map((t) => t.trim().toLowerCase());
    const acceptedFiles: File[] = [];
    const fileRejections: File[] = [];
    Array.from(files).forEach((file) => {
      const ext = "." + file.name.split(".").pop()?.toLowerCase();
      if (types.includes(ext) || types.includes(file.type)) acceptedFiles.push(file);
      else fileRejections.push(file);
    });
    return { acceptedFiles, fileRejections };
  }

  function handleDrop(e: DragEvent) {
    e.preventDefault();
    dragging = false;
    if (disabled || !e.dataTransfer?.files) return;
    dispatch("drop", getAcceptedFiles(e.dataTransfer.files));
  }

  function handleChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files) return;
    dispatch("drop", getAcceptedFiles(input.files));
    input.value = "";
  }
</script>

<div
  class="dropzone"
  class:active={dragging}
  class:disabled
  role="button"
  tabindex="0"
  on:dragenter|preventDefault={() => (dragging = true)}
  on:dragleave|preventDefault={() => (dragging = false)}
  on:dragover|preventDefault
  on:drop={handleDrop}
  on:click={() => !disabled && fileInput.click()}
  on:keydown={(e) => e.key === "Enter" && !disabled && fileInput.click()}
>
  <input
    bind:this={fileInput}
    type="file"
    {multiple}
    {accept}
    style="display:none"
    on:change={handleChange}
  />
  <slot />
</div>

<style>
  .dropzone {
    border: 2px dashed #ccc;
    border-radius: 4px;
    padding: 20px;
    text-align: center;
    cursor: pointer;
    transition: border-color 0.2s;
  }
  .dropzone.active {
    border-color: #3273dc;
    background-color: rgba(50, 115, 220, 0.05);
  }
  .dropzone.disabled {
    cursor: not-allowed;
    opacity: 0.6;
  }
</style>
