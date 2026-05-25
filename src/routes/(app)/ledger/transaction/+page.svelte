<script lang="ts">
  import { ajax, isMobile, type LedgerFile, type Transaction as T } from "$lib/utils";
  import _ from "lodash";
  import { onDestroy, onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";
  import Transaction from "$lib/components/Transaction.svelte";
  import TransactionHeader from "$lib/components/TransactionHeader.svelte";
  import BulkEditForm from "$lib/components/BulkEditForm.svelte";
  import { slide } from "svelte/transition";
  import * as bulkEdit from "$lib/bulk_edit";
  import * as toast from "bulma-toast";
  import DiffViewModal from "$lib/components/DiffViewModal.svelte";
  import SearchQuery from "$lib/components/SearchQuery.svelte";
  import { editorState } from "$lib/search_query_editor";
  import { get } from "svelte/store";
  import { download } from "$lib/export";
  import { sync, startPolling } from "$lib/sync";

  let buldEditOpen = $state(false);
  let transactions: T[] = $state(null);
  let filtered: T[] = $state([]);
  let files: LedgerFile[] = $state([]);
  let newFiles: LedgerFile[] = $state([]);
  let updatedTransactionsCount = $state(0);
  let openPreviewModal = $state(false);
  let accounts: string[] = $state([]);
  let commodities: string[] = $state([]);
  let selectedTagFilters: string[] = $state([]);
  let autocompleteTags: string[] = $state([]);
  let selectedTransactions: Record<string, boolean> = $state({});
  let bulkTag = $state("");
  let undoBulk: { tag: string; ids: string[] } | null = $state(null);
  let activePredicate: (t: T) => boolean = $state(() => true);

  function handleInputRaw(predicate: (t: T) => boolean) {
    activePredicate = predicate;
    applyFilters();
  }

  function applyFilters() {
    filtered = _.filter(transactions, (t) => {
      if (!activePredicate(t)) {
        return false;
      }
      if (selectedTagFilters.length === 0) {
        return true;
      }
      return _.some(t.tags || [], (tag) => selectedTagFilters.includes(tag));
    });
  }

  const handleInput = _.debounce(handleInputRaw, 100);

  const unsubscribe = editorState.subscribe((state) => {
    handleInput(state.predicate);
  });

  onDestroy(async () => {
    unsubscribe();
  });

  const mobile = isMobile();

  const itemSize = (i: number) => {
    const t = filtered[i];
    const count = t.postings.length;
    const tagHeight = (t.tags?.length || 0) > 0 ? 26 : 0;
    return 14 + count * 22 + tagHeight + (mobile ? 25 : 0);
  };

  async function loadTransactions() {
    ({ files, accounts, commodities } = await ajax("/api/editor/files"));
    ({ transactions } = await ajax("/api/transaction"));
    ({ tags: autocompleteTags } = await ajax("/api/tags/autocomplete", { background: true }));
    selectedTransactions = {};
    handleInputRaw(get(editorState).predicate);

    newFiles = files;
  }

  async function downloadTransactions() {
    const { balancedPostings } = await ajax("/api/transaction/balanced");
    download(balancedPostings);
  }

  function showPreview(detail: any) {
    ({ newFiles, updatedTransactionsCount } = bulkEdit.applyChanges(
      files,
      filtered,
      detail.operation,
      detail.args
    ));
    openPreviewModal = true;
  }

  async function saveAll(newFiles: LedgerFile[]) {
    for (const newFile of newFiles) {
      const { saved, message } = await ajax("/api/editor/save", {
        method: "POST",
        body: JSON.stringify({ name: newFile.name, content: newFile.content }),
        background: true
      });

      if (!saved) {
        toast.toast({
          message: `Failed to save ${newFile.name}. reason: ${message}`,
          type: "is-danger",
          duration: 10000
        });
      } else {
        toast.toast({
          message: `Saved ${newFile.name}`,
          type: "is-success"
        });
      }
    }
    await loadTransactions();
  }

  onMount(async () => {
    await loadTransactions();
  });

  function toggleTagFilter(tag: string) {
    if (selectedTagFilters.includes(tag)) {
      selectedTagFilters = selectedTagFilters.filter((t) => t !== tag);
    } else {
      selectedTagFilters = [...selectedTagFilters, tag];
    }
    applyFilters();
  }

  function clearTagFilter(tag: string) {
    selectedTagFilters = selectedTagFilters.filter((t) => t !== tag);
    applyFilters();
  }

  function updateSelected(detail: { id: string; selected: boolean }) {
    selectedTransactions = {
      ...selectedTransactions,
      [detail.id]: detail.selected
    };
  }

  const selectedCount = $derived(
    _.size(_.pickBy(selectedTransactions, (isSelected) => isSelected === true))
  );

  async function applyBulkTag() {
    const tag = bulkTag.trim();
    if (!tag) {
      return;
    }
    const ids = _.keys(_.pickBy(selectedTransactions, (isSelected) => isSelected === true));
    if (ids.length === 0) {
      return;
    }

    await Promise.all(
      ids.map((id) =>
        ajax(
          "/api/transactions/:id/tags",
          {
            method: "POST",
            body: JSON.stringify({ tag }),
            background: true
          },
          { id: encodeURIComponent(id) }
        )
      )
    );
    toast.toast({
      message: `Added tag "${tag}" to ${ids.length} transaction(s).`,
      type: "is-success"
    });
    undoBulk = { tag, ids };
    bulkTag = "";
    await loadTransactions();
    const undo = undoBulk;
    setTimeout(() => {
      if (undoBulk === undo) {
        undoBulk = null;
      }
    }, 10000);
  }

  async function undoBulkTag() {
    if (!undoBulk) {
      return;
    }
    await Promise.all(
      undoBulk.ids.map((id) =>
        ajax(
          "/api/transactions/:id/tags/:tag",
          { method: "DELETE", background: true },
          {
            id: encodeURIComponent(id),
            tag: encodeURIComponent(undoBulk.tag)
          }
        )
      )
    );
    toast.toast({
      message: `Removed tag "${undoBulk.tag}" from ${undoBulk.ids.length} transaction(s).`,
      type: "is-info"
    });
    undoBulk = null;
    await loadTransactions();
  }

  async function forceFullSync() {
    const jobId = await sync({ journal: true, force_journal: true });
    if (!jobId) return;

    startPolling(jobId, async () => {
      await loadTransactions();
    });
  }
</script>

<DiffViewModal
  onsave={(files) => saveAll(files)}
  bind:open={openPreviewModal}
  oldFiles={files}
  {newFiles}
  {updatedTransactionsCount}
/>

{#if transactions}
  <section class="section tab-journal">
    <div class="container is-fluid">
      <div class="columns">
        <div class="column is-12">
          <nav class="level">
            <div class="level-left">
              <div class="level-item">
                <div class="field">
                  <div class="control">
                    <SearchQuery
                      autocomplete={{
                        account: accounts,
                        commodity: commodities,
                        filename: files.map((f) => f.name)
                      }}
                    />
                  </div>
                </div>
              </div>
              <div class="level-item">
                <div class="field">
                  <div class="control">
                    <button
                      class="button is-link is-light invertable"
                      onclick={(_e) => (buldEditOpen = !buldEditOpen)}
                    >
                      <span>Bulk Edit</span>
                      <span class="icon is-small">
                        <i class="fas {buldEditOpen ? 'fa-angle-up' : 'fa-angle-down'}"></i>
                      </span>
                    </button>
                  </div>
                </div>
              </div>
            </div>
            <div class="level-right">
              <div class="level-item">
                <p class="is-6"><b>{filtered.length}</b> transaction(s)</p>
              </div>
              {#if selectedCount > 0}
                <div class="level-item">
                  <input class="input is-small" placeholder="Bulk tag" bind:value={bulkTag} />
                </div>
                <div class="level-item">
                  <button
                    class="button is-small is-link is-light"
                    type="button"
                    onclick={applyBulkTag}
                  >
                    Add Tag to {selectedCount}
                  </button>
                </div>
              {/if}
              {#if undoBulk}
                <div class="level-item">
                  <button class="button is-small is-text" type="button" onclick={undoBulkTag}>
                    Undo "{undoBulk.tag}"
                  </button>
                </div>
              {/if}
              <div class="level-item">
                <button
                  type="button"
                  class="button is-small is-link invertable is-light"
                  title="Re-parse the journal and replace all postings in the database, bypassing the incremental-sync cache."
                  onclick={(_e) => forceFullSync()}
                >
                  <span class="icon is-small">
                    <i class="fas fa-rotate"></i>
                  </span>
                  Force Full Sync
                </button>
              </div>
              <div class="level-item">
                <button
                  type="button"
                  class="button is-small is-text"
                  onclick={(_e) => downloadTransactions()}
                >
                  <span class="icon is-small">
                    <i class="fa-solid fa-file-arrow-down"></i>
                  </span>
                  download
                </button>
              </div>
            </div>
          </nav>
        </div>
      </div>

      {#if buldEditOpen}
        <div class="columns">
          <div class="column is-12" transition:slide>
            <BulkEditForm {accounts} onpreview={(detail) => showPreview(detail)} />
          </div>
        </div>
      {/if}

      {#if selectedTagFilters.length > 0}
        <div class="columns">
          <div class="column is-12">
            <div class="box py-2 px-3 mb-3">
              <span class="has-text-grey mr-2">Tag filters:</span>
              {#each selectedTagFilters as tag}
                <span class="tag is-light mr-1">
                  {tag}
                  <button
                    class="delete is-small ml-1"
                    type="button"
                    aria-label={"Remove tag filter " + tag}
                    onclick={() => clearTagFilter(tag)}
                  ></button>
                </span>
              {/each}
            </div>
          </div>
        </div>
      {/if}

      <div class="columns">
        <div class="column is-12">
          <div class="box">
            <TransactionHeader showExtraColumns={false} />
            <VirtualList
              width="100%"
              height={window.innerHeight - 150}
              itemCount={filtered.length}
              {itemSize}
            >
              <div slot="item" let:index let:style {style}>
                {@const t = filtered[index]}
                <Transaction
                  {t}
                  selectable={true}
                  selected={!!selectedTransactions[t.id]}
                  {autocompleteTags}
                  onselect={(detail) => updateSelected(detail)}
                  ontagclick={(detail) => toggleTagFilter(detail.tag)}
                  ontagchanged={async (_detail) => {
                    ({ tags: autocompleteTags } = await ajax("/api/tags/autocomplete", {
                      background: true
                    }));
                    applyFilters();
                  }}
                />
              </div>
            </VirtualList>
          </div>
        </div>
      </div>
    </div>
  </section>
{/if}
