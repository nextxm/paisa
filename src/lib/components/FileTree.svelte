<script lang="ts">
  import type { Directory, LedgerFile } from "$lib/utils";
  import _ from "lodash";
  import FileTree from "./FileTree.svelte";

  let {
    files = [],
    path,
    selectedFileName,
    hasUnsavedChanges,
    root = true,
    onselect
  }: {
    files?: Array<Directory | LedgerFile>;
    path: string;
    selectedFileName: string;
    hasUnsavedChanges: boolean;
    root?: boolean;
    onselect?: (file: LedgerFile) => void;
  } = $props();

  function fileName(path: string) {
    return _.last(path.split("/"));
  }

  function join(paths: string[]) {
    return _.filter(paths, (p) => !_.isEmpty(p)).join("/");
  }

  function isOpen(file: Directory | LedgerFile) {
    const fullPath = join([path, file.name]);
    return selectedFileName?.startsWith(fullPath);
  }
</script>

<ul class={root && "du-menu du-menu-sm w-full p-0"}>
  {#each files as file}
    {#if file.type != "directory"}
      <li>
        <button
          type="button"
          onclick={() => onselect?.(file as LedgerFile)}
          class={file.name == selectedFileName ? "du-active" : ""}
        >
          <span class="icon is-small">
            <i class="fa-regular fa-file-lines"></i>
          </span>
          <span title={fileName(file.name)} class="truncate">{fileName(file.name)}</span>
          {#if file.name == selectedFileName && hasUnsavedChanges}
            <span class="ml-1 tag is-danger">unsaved</span>
          {/if}
        </button>
      </li>
    {:else}
      <li>
        <details open={isOpen(file)}>
          <summary>
            <span class="icon is-small">
              <i class="fa-regular fa-folder"></i>
            </span>
            <span title={file.name} class="truncate">{file.name}</span>
          </summary>
          <FileTree
            path={join([path, file.name])}
            {onselect}
            root={false}
            files={file.children}
            {selectedFileName}
            {hasUnsavedChanges}
          />
        </details>
      </li>
    {/if}
  {/each}
</ul>
