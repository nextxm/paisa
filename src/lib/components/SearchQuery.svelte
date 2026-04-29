<script lang="ts">
  import { createEditor } from "$lib/search_query_editor";
  import type { EditorView } from "codemirror";

  let editorDom: HTMLElement = $state();
  let { autocomplete }: { autocomplete: Record<string, string[]> } = $props();
  let editor: EditorView;

  $effect(() => {
    if (autocomplete && editorDom) {
      if (editor) {
        editor.destroy();
      }

      editor = createEditor("", editorDom, autocomplete);
    }
  });
</script>

<div class="search-query-editor" bind:this={editorDom}></div>
