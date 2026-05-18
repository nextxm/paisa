<script lang="ts">
  import Select from "svelte-select";
  import {
    createEditor as createTemplateEditor,
    editorState as templateEditorState
  } from "$lib/template_editor";
  import {
    createEditor as createPreviewEditor,
    updateContent as updatePreviewContent
  } from "$lib/editor";
  import Dropzone from "$lib/components/Dropzone.svelte";
  import ImportPreviewTable from "$lib/components/ImportPreviewTable.svelte";
  import PresetSelector from "$lib/components/PresetSelector.svelte";
  import { parse, asRows, render as renderJournal } from "$lib/spreadsheet";
  import _ from "lodash";
  import type { EditorView } from "codemirror";
  import { onMount } from "svelte";
  import { ajax, type ImportPreset, type ImportPreviewRow, type ImportTemplate } from "$lib/utils";
  import {
    defaultIncludedFromValidation,
    filterSelectedRows,
    toCSVContent
  } from "$lib/import_preview_utils";
  import { accountTfIdf } from "../../../../store";
  import * as toast from "bulma-toast";
  import FileModal from "$lib/components/FileModal.svelte";
  import Modal from "$lib/components/Modal.svelte";
  import type { PageData } from "./$types";

  let { data: pageData }: { data: PageData } = $props();

  let templates: ImportTemplate[] = $state(pageData.templates);
  let selectedTemplate: ImportTemplate = $state(pageData.templates[0]);
  let saveAsName: string = $state(selectedTemplate?.name || null);
  let preview = $state("");
  let parseErrorMessage: string = $state(null);
  let columnCount: number = $state(0);
  let data: any[][] = $state([]);
  let rows: Array<Record<string, any>> = $state([]);
  let previewRows: ImportPreviewRow[] = $state([]);
  let includedPreviewRows: boolean[] = $state([]);
  let importPresets: ImportPreset[] = $state(pageData.importPresets);
  let selectedPreset: ImportPreset = $state(
    _.find(pageData.importPresets, { name: "Generic Bank CSV" }) || pageData.importPresets[0]
  );
  let delimiter: string = $state(selectedPreset?.delimiter || ",");
  let options: { reverse: boolean; trim: boolean } = $state({ reverse: false, trim: true });
  let importSaving = $state(false);

  let templateEditorDom: Element = $state();
  let templateEditor: EditorView = $state();

  let previewEditorDom: Element = $state();
  let previewEditor: EditorView = $state();

  onMount(() => {
    accountTfIdf.set(pageData.accountTfIdf);
    templateEditor = createTemplateEditor(selectedTemplate?.content || "", templateEditorDom);
    previewEditor = createPreviewEditor("", preview, previewEditorDom, { readonly: true });
  });

  const saveAsNameDuplicate = $derived(
    !!_.find(templates, { name: saveAsName, template_type: "custom" })
  );
  const selectedPreviewCount = $derived(
    filterSelectedRows(previewRows, includedPreviewRows).length
  );

  async function save() {
    const { template, saved, message } = await ajax("/api/templates/upsert", {
      method: "POST",
      body: JSON.stringify({
        name: saveAsName,
        content: templateEditor.state.doc.toString()
      }),
      background: true
    });

    if (!saved) {
      toast.toast({
        message: `Failed to save ${saveAsName}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
      return;
    }

    ({ templates } = await ajax("/api/templates", { background: true }));
    selectedTemplate = _.find(templates, { id: template.id });
    saveAsName = selectedTemplate.name;
    toast.toast({
      message: `Saved ${saveAsName}`,
      type: "is-success"
    });

    $templateEditorState = _.assign({}, $templateEditorState, { hasUnsavedChanges: false });
  }

  async function remove() {
    const oldName = selectedTemplate.name;
    const confirmed = confirm(`Are you sure you want to delete ${oldName} template?`);
    if (!confirmed) {
      return;
    }
    const { success, message } = await ajax("/api/templates/delete", {
      method: "POST",
      body: JSON.stringify({
        name: selectedTemplate.name
      }),
      background: true
    });

    if (!success) {
      toast.toast({
        message: `Failed to remove ${oldName}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
      return;
    }

    ({ templates } = await ajax("/api/templates", { background: true }));
    selectedTemplate = templates[0];
    saveAsName = selectedTemplate.name;
    toast.toast({
      message: `Removed ${oldName}`,
      type: "is-success"
    });

    $templateEditorState = _.assign({}, $templateEditorState, { hasUnsavedChanges: false });
  }

  $effect(() => {
    if (!_.isEmpty(data) && $templateEditorState.template) {
      try {
        const selectedRows = filterSelectedRows(rows, includedPreviewRows);
        preview = renderJournal(selectedRows, $templateEditorState.template, {
          reverse: options.reverse,
          trim: options.trim
        });
        updatePreviewContent(previewEditor, preview);
      } catch (e) {
        console.log(e);
      }
    }
  });

  $effect(() => {
    if (selectedTemplate && templateEditor) {
      if (templateEditor.state.doc.toString() != selectedTemplate.content) {
        templateEditor.destroy();
        templateEditor = createTemplateEditor(selectedTemplate.content, templateEditorDom);
      }
    }
  });

  $effect(() => {
    if (selectedPreset) {
      delimiter = selectedPreset.delimiter || ",";
      if (!_.isEmpty(data)) {
        refreshImportPreview();
      }
    }
  });

  async function handleFilesSelect(detail: { acceptedFiles: File[] }) {
    const { acceptedFiles } = detail;

    const results = await parse(acceptedFiles[0]);
    if (results.error) {
      parseErrorMessage = results.error;
    } else {
      parseErrorMessage = null;
      data = results.data;
      rows = asRows(results);

      columnCount = _.maxBy(data, (row) => row.length).length;
      _.each(data, (row) => {
        row.length = columnCount;
      });

      await refreshImportPreview();
    }
  }

  async function refreshImportPreview() {
    if (_.isEmpty(data) || !selectedTemplate) {
      return;
    }
    const response: any = await ajax("/api/import/preview", {
      method: "POST",
      body: JSON.stringify({
        template: selectedTemplate.name,
        content: toCSVContent(data),
        delimiter,
        dry_run: true
      }),
      background: true
    });

    if (response.error) {
      parseErrorMessage = response.error.message;
      previewRows = [];
      includedPreviewRows = [];
      return;
    }

    parseErrorMessage = null;
    previewRows = response.rows;
    includedPreviewRows = defaultIncludedFromValidation(previewRows);
  }

  async function copyToClipboard() {
    try {
      await navigator.clipboard.writeText(preview);
      toast.toast({
        message: "Copied to clipboard",
        type: "is-success"
      });
    } catch (e) {
      console.log(e);
      toast.toast({
        message: "Failed to copy to clipboard",
        type: "is-danger"
      });
    }
  }

  let modalOpen = $state(false);
  function openSaveModal() {
    if (selectedPreviewCount === 0) {
      toast.toast({
        message: "Select at least one row to import",
        type: "is-danger"
      });
      return;
    }
    modalOpen = true;
  }

  async function saveToFile(destinationFile: string) {
    importSaving = true;
    const { saved, message } = await ajax("/api/editor/save", {
      method: "POST",
      body: JSON.stringify({ name: destinationFile, content: preview, operation: "overwrite" }),
      background: true
    });
    importSaving = false;

    if (saved) {
      toast.toast({
        message: `Saved <b><a href="/ledger/editor/${encodeURIComponent(
          destinationFile
        )}">${destinationFile}</a></b>`,
        type: "is-success",
        duration: 5000
      });
    } else {
      toast.toast({
        message: `Failed to save ${destinationFile}. reason: ${message}`,
        type: "is-danger",
        duration: 10000
      });
    }
  }

  async function saveCurrentAsPreset() {
    const name = prompt("Preset name");
    if (_.isEmpty(name)) {
      return;
    }

    const columnMappings = _.fromPairs(
      _.range(0, columnCount).map((index) => {
        const col = String.fromCharCode(65 + index);
        return [col, col];
      })
    );

    const response: any = await ajax("/api/import/presets", {
      method: "POST",
      body: JSON.stringify({
        name,
        column_mappings: columnMappings,
        date_format: "",
        default_accounts: {},
        delimiter
      }),
      background: true
    });

    if (response.error || !response.saved) {
      toast.toast({
        message: `Failed to save preset ${name}`,
        type: "is-danger"
      });
      return;
    }

    ({ presets: importPresets } = await ajax("/api/import/presets", { background: true }));
    selectedPreset = _.find(importPresets, { id: response.preset.id });
    toast.toast({
      message: `Saved preset ${name}`,
      type: "is-success"
    });
  }

  function builtinNotAllowed(action: string, template: ImportTemplate) {
    if (template?.template_type == "builtin") {
      return `Not allowed to ${action.toLowerCase()} builtin template`;
    }
    return action;
  }

  let templateCreateModalOpen = $state(false);
  function openTemplateCreateModal() {
    templateCreateModalOpen = true;
  }
</script>

<Modal bind:active={templateCreateModalOpen}>
  {#snippet head(close)}
    <p class="text-base font-semibold flex-1">Create Template</p>
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
      <label class="label" for="save-filename">Template Name</label>
      <div class="control" id="save-filename">
        <input class="input" type="text" bind:value={saveAsName} />
        {#if saveAsNameDuplicate}
          <p class="help is-danger">Template with the same name already exists</p>
        {/if}
      </div>
    </div>
  {/snippet}
  {#snippet foot(close)}
    <button
      class="du-btn du-btn-success du-btn-sm"
      disabled={_.isEmpty(saveAsName) || saveAsNameDuplicate}
      onclick={() => save() && close()}>Create</button
    >
    <button class="du-btn du-btn-sm" onclick={() => close()}>Cancel</button>
  {/snippet}
</Modal>

<FileModal bind:open={modalOpen} onsave={(file) => saveToFile(file)} />

<section class="section tab-import" style="padding-bottom: 0 !important">
  <div class="container is-fluid">
    <div class="columns mb-0">
      <div class="column is-5 py-0">
        <div class="box p-3 mb-3 overflow-x-auto">
          <div class="field is-grouped mb-0">
            <p class="control">
              <span data-tippy-content="Create" data-tippy-followCursor="false">
                <button
                  class="button"
                  aria-label="Create template"
                  onclick={() => openTemplateCreateModal()}
                >
                  <span class="icon">
                    <i class="fas fa-file-circle-plus"></i>
                  </span>
                </button>
              </span>

              <span
                class="ml-4"
                data-tippy-followCursor="false"
                data-tippy-content={$templateEditorState.hasUnsavedChanges == false
                  ? "No Unsaved Chagnes"
                  : builtinNotAllowed("Save", selectedTemplate)}
              >
                <button
                  class="button"
                  aria-label="Save template"
                  onclick={() => save()}
                  disabled={$templateEditorState.hasUnsavedChanges == false ||
                    selectedTemplate?.template_type == "builtin"}
                >
                  <span class="icon">
                    <i class="fas fa-floppy-disk"></i>
                  </span>
                </button>
              </span>

              <span
                data-tippy-followCursor="false"
                data-tippy-content={builtinNotAllowed("Delete", selectedTemplate)}
              >
                <button
                  class="button"
                  aria-label="Delete template"
                  onclick={() => remove()}
                  disabled={selectedTemplate?.template_type == "builtin"}
                >
                  <span class="icon">
                    <i class="fas fa-trash-can"></i>
                  </span>
                </button>
              </span>
            </p>

            <p class="control is-expanded">
              <Select
                bind:value={selectedTemplate}
                showChevron={true}
                items={templates}
                label="name"
                itemId="id"
                searchable={true}
                clearable={false}
                floatingConfig={{ strategy: "fixed" }}
                on:change={() => {
                  saveAsName = selectedTemplate.name;
                }}
              >
                <div slot="selection" let:selection>
                  {selection.name}
                  <span class="tag is-small is-link invertable is-light"
                    >{selection.template_type}</span
                  >
                </div>
                <div slot="item" let:item>
                  <span class="name">{item.name}</span>
                  <span class="tag is-small is-link invertable is-light">{item.template_type}</span>
                </div>
              </Select>
            </p>
          </div>
        </div>
        <div class="box py-0">
          <div class="field">
            <div class="control">
              <div class="template-editor" bind:this={templateEditorDom}></div>
            </div>
          </div>
        </div>
        <div class="box py-0">
          <div class="field">
            <div class="control">
              <button
                data-tippy-followCursor="false"
                data-tippy-content="Copy to Clipboard"
                class="button clipboard"
                aria-label="Copy preview to clipboard"
                disabled={_.isEmpty(preview)}
                onclick={copyToClipboard}
              >
                <span class="icon">
                  <i class="fas fa-copy"></i>
                </span>
              </button>
              <button
                data-tippy-followCursor="false"
                data-tippy-content="Confirm & Save"
                class="button save"
                class:is-loading={importSaving}
                aria-label="Confirm selected rows and save"
                disabled={_.isEmpty(preview) || selectedPreviewCount === 0 || importSaving}
                onclick={openSaveModal}
              >
                <span class="icon">
                  <i class="fas fa-floppy-disk"></i>
                </span>
              </button>
              <div class="preview-editor" bind:this={previewEditorDom}></div>
            </div>
          </div>
        </div>
      </div>
      <div class="column is-7 py-0">
        <div class="box p-3 mb-3">
          <Dropzone
            multiple={false}
            accept=".csv,.txt,.xls,.xlsx,.pdf,.CSV,.TXT,.XLS,.XLSX,.PDF"
            ondrop={handleFilesSelect}
          >
            Drag 'n' drop CSV, TXT, XLS, XLSX, PDF file here or click to select
          </Dropzone>
        </div>
        <div class="box p-3 mb-3">
          <PresetSelector
            presets={importPresets}
            bind:selectedPreset
            onsavecurrent={saveCurrentAsPreset}
          />
          <div class="field is-grouped is-align-items-center">
            <label class="label mb-0 mr-2" for="import-delimiter">Delimiter</label>
            <div class="control">
              <input
                id="import-delimiter"
                class="input is-small"
                maxlength="1"
                bind:value={delimiter}
                onblur={() => refreshImportPreview()}
              />
            </div>
          </div>
        </div>
        <div class="is-flex justify-end mb-3 gap-4">
          <div class="field color-switch">
            <input
              id="import-reverse"
              type="checkbox"
              bind:checked={options.reverse}
              class="switch is-rounded is-small"
            />
            <label for="import-reverse">Reverse</label>
          </div>
          <div class="field color-switch">
            <input
              id="trim-reverse"
              type="checkbox"
              bind:checked={options.trim}
              class="switch is-rounded is-small"
            />
            <label for="trim-reverse">Trim</label>
          </div>
        </div>
        {#if parseErrorMessage}
          <div class="message invertable is-danger">
            <div class="message-header">Failed to parse document</div>
            <div class="message-body">{parseErrorMessage}</div>
          </div>
        {/if}
        {#if !_.isEmpty(data)}
          <ImportPreviewTable rows={previewRows} bind:included={includedPreviewRows} />
        {/if}
      </div>
    </div>
    <div></div>
  </div>
</section>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  $import-full-height: calc(100vh - 205px);

  .clipboard {
    float: right;
    position: absolute;
    right: 0;
    z-index: 1;
  }

  .save {
    float: right;
    position: absolute !important;
    right: 40px;
    z-index: 1;
  }

  .color-switch {
    .switch[type="checkbox"]:checked + label::before,
    .switch[type="checkbox"]:checked + label:before {
      background: $link;
    }
  }
</style>
