<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    multiple = true,
    accept = "",
    disabled = false,
    ondrop: onDropCallback = null,
    children
  }: {
    multiple?: boolean;
    accept?: string;
    disabled?: boolean;
    ondrop?: ((detail: { acceptedFiles: File[]; fileRejections: File[] }) => void) | null;
    children?: Snippet;
  } = $props();

  let dragging = $state(false);
  let fileInput: HTMLInputElement = $state();

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
    onDropCallback?.(getAcceptedFiles(e.dataTransfer.files));
  }

  function handleChange(e: Event) {
    const input = e.target as HTMLInputElement;
    if (!input.files) return;
    onDropCallback?.(getAcceptedFiles(input.files));
    input.value = "";
  }
</script>

<div
  class="dropzone"
  class:active={dragging}
  class:disabled
  role="button"
  tabindex="0"
  ondragenter={(e) => {
    e.preventDefault();
    dragging = true;
  }}
  ondragleave={(e) => {
    e.preventDefault();
    dragging = false;
  }}
  ondragover={(e) => e.preventDefault()}
  ondrop={handleDrop}
  onclick={() => !disabled && fileInput.click()}
  onkeydown={(e) => e.key === "Enter" && !disabled && fileInput.click()}
>
  <input
    bind:this={fileInput}
    type="file"
    {multiple}
    {accept}
    style="display:none"
    onchange={handleChange}
  />
  {@render children?.()}
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
