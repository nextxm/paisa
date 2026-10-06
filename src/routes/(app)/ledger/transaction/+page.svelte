<script lang="ts">
  import { goto } from "$app/navigation";
  import TransactionFilterBar from "$lib/components/TransactionFilterBar.svelte";
  import {
    buildTransactionFiltersQuery,
    emptyTransactionFilters,
    loadSavedTransactionSearches,
    parseTransactionFiltersFromSearchParams,
    saveSavedTransactionSearches,
    upsertSavedTransactionSearch,
    type SavedTransactionSearch,
    type TransactionFilters
  } from "$lib/transaction_filters";
  import { ajax, isMobile, type LedgerFile, type Transaction as T } from "$lib/utils";
  import _ from "lodash";
  import { onMount } from "svelte";
  import VirtualList from "svelte-tiny-virtual-list";
  import Transaction from "$lib/components/Transaction.svelte";
  import TransactionHeader from "$lib/components/TransactionHeader.svelte";
  import BulkEditForm from "$lib/components/BulkEditForm.svelte";
  import { slide } from "svelte/transition";
  import * as bulkEdit from "$lib/bulk_edit";
  import * as toast from "bulma-toast";
  import DiffViewModal from "$lib/components/DiffViewModal.svelte";
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
  let filters: TransactionFilters = $state(emptyTransactionFilters());
  let savedSearches: SavedTransactionSearch[] = $state([]);
  let focusSearch = $state(false);
  let routeSeededFromFilter = $state(false);

  const mobile = isMobile();

  const itemSize = (i: number) => {
    const t = filtered[i];
    const count = t.postings.length;
    return 8 + count * 22 + (mobile ? 25 : 0);
  };

  async function fetchTransactions(currentFilters = filters) {
    const query = buildTransactionFiltersQuery(currentFilters);
    const route = query ? `/api/transaction?${query}` : "/api/transaction";
    ({ transactions: filtered } = await ajax(route));
    transactions = filtered;
  }

  async function loadTransactions() {
    ({ files, accounts, commodities } = await ajax("/api/editor/files"));
    await fetchTransactions();
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

  const debouncedFetchTransactions = _.debounce((currentFilters: TransactionFilters) => {
    fetchTransactions(currentFilters);
  }, 200);

  function handleFiltersChange(next: TransactionFilters) {
    filters = { ...next };
    debouncedFetchTransactions(filters);
  }

  function saveSearch(name: string) {
    savedSearches = upsertSavedTransactionSearch(savedSearches, name, filters);
    saveSavedTransactionSearches(savedSearches);
  }

  function loadSavedSearch(name: string) {
    const found = savedSearches.find((search) => search.name === name);
    if (!found) return;
    filters = { ...found.filters };
    fetchTransactions(filters);
  }

  onMount(async () => {
    const searchParams = new URLSearchParams(window.location.search);
    const seededFilters = parseTransactionFiltersFromSearchParams(searchParams);
    const hasSeededFilter = buildTransactionFiltersQuery(seededFilters).length > 0;
    if (hasSeededFilter) {
      filters = seededFilters;
      routeSeededFromFilter = true;
    }
    focusSearch = searchParams.get("focus") === "search";
    if (focusSearch) {
      const nextParams = new URLSearchParams(window.location.search);
      nextParams.delete("focus");
      const route = nextParams.toString()
        ? `/ledger/transaction?${nextParams.toString()}`
        : "/ledger/transaction";
      goto(route, { replaceState: true, keepFocus: true, noScroll: true });
    }
    savedSearches = loadSavedTransactionSearches();
    await loadTransactions();
    if (routeSeededFromFilter) {
      routeSeededFromFilter = false;
    }
  });

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
                <TransactionFilterBar
                  {filters}
                  {accounts}
                  {commodities}
                  {savedSearches}
                  showSavedSearches={true}
                  autoFocusSearch={focusSearch}
                  onFiltersChange={handleFiltersChange}
                  onSaveSearch={saveSearch}
                  onLoadSavedSearch={loadSavedSearch}
                />
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
                <Transaction {t} />
              </div>
            </VirtualList>
          </div>
        </div>
      </div>
    </div>
  </section>
{/if}
