<script lang="ts">
  import { ajax, postingUrl, type Transaction } from "$lib/utils";
  import Postings from "$lib/components/Postings.svelte";
  import PostingStatus from "$lib/components/PostingStatus.svelte";
  import TransactionNote from "./TransactionNote.svelte";

  let {
    compact = false,
    t,
    highlightAccount = "",
    selectable = false,
    selected = false,
    autocompleteTags = [],
    onselect = (_detail: { id: string; selected: boolean }) => {},
    ontagclick = (_detail: { tag: string }) => {},
    ontagchanged = (_detail: { id: string }) => {}
  }: {
    compact?: boolean;
    t: Transaction;
    highlightAccount?: string;
    selectable?: boolean;
    selected?: boolean;
    autocompleteTags?: string[];
    onselect?: (detail: { id: string; selected: boolean }) => void;
    ontagclick?: (detail: { tag: string }) => void;
    ontagchanged?: (detail: { id: string }) => void;
  } = $props();

  let addTagOpen = $state(false);
  let inputTag = $state("");
  let loadingTag = $state(false);

  const tagListId = $derived(`tags-${t.id}`);

  async function addTag() {
    const tag = inputTag.trim();
    if (!tag) return;
    loadingTag = true;
    try {
      const { tags } = await ajax(
        "/api/transactions/:id/tags",
        {
          method: "POST",
          body: JSON.stringify({ tag }),
          background: true
        },
        { id: encodeURIComponent(t.id) }
      );
      t.tags = tags;
      inputTag = "";
      addTagOpen = false;
      ontagchanged({ id: t.id });
    } finally {
      loadingTag = false;
    }
  }

  async function removeTag(tag: string) {
    if (!confirm(`Remove tag "${tag}" from this transaction?`)) {
      return;
    }
    loadingTag = true;
    try {
      const { tags } = await ajax(
        "/api/transactions/:id/tags/:tag",
        { method: "DELETE", background: true },
        { id: encodeURIComponent(t.id), tag: encodeURIComponent(tag) }
      );
      t.tags = tags;
      ontagchanged({ id: t.id });
    } finally {
      loadingTag = false;
    }
  }

  function emitTagClick(tag: string) {
    ontagclick({ tag });
  }
</script>

<div class="column is-12">
  {#if compact}
    <div class="columns is-flex-wrap-wrap transaction">
      <div class="column is-12 py-0 truncate">
        <div class="description is-size-7">
          <b>{t.date.format("DD MMM YYYY")}</b>
          <span title={t.payee}>
            <PostingStatus posting={t.postings[0]} />
            <TransactionNote transaction={t} />
            <a class="secondary-link" href={postingUrl(t.postings[0])}>{t.payee}</a></span
          >
        </div>
      </div>
      <div class="column is-12 py-0">
        <Postings postings={t.postings} {highlightAccount} />
      </div>
    </div>
  {:else}
    <div class="columns is-flex-wrap-wrap transaction bordered">
      <div class="column py-0 truncate" style="flex: 0 0 30%; max-width: 30%;">
        <div class="description mt-2 is-size-7">
          {#if selectable}
            <label class="checkbox mr-2">
              <input
                type="checkbox"
                checked={selected}
                onchange={(e) =>
                  onselect({
                    id: t.id,
                    selected: (e.currentTarget as HTMLInputElement).checked
                  })}
              />
            </label>
          {/if}
          <b>{t.date.format("DD MMM YYYY")}</b>
          <span title={t.payee}
            ><PostingStatus posting={t.postings[0]} />
            <TransactionNote transaction={t} />
            <a class="secondary-link" href={postingUrl(t.postings[0])}>{t.payee}</a></span
          >
        </div>
        <div class="tags-container mt-1">
          {#each t.tags || [] as tag}
            <span class="tag is-light is-small mr-1 mb-1 tag-chip">
              <button type="button" class="tag-label" onclick={() => emitTagClick(tag)}
                >{tag}</button
              >
              <button
                type="button"
                class="delete is-small"
                aria-label="remove tag"
                disabled={loadingTag}
                onclick={() => removeTag(tag)}
              ></button>
            </span>
          {/each}
          {#if addTagOpen}
            <div class="is-flex is-align-items-center mb-1">
              <input
                class="input is-small"
                style="max-width: 180px"
                list={tagListId}
                bind:value={inputTag}
                onkeydown={(e) => e.key === "Enter" && addTag()}
                placeholder="Add tag"
              />
              <button
                class="button is-small is-link is-light ml-1"
                type="button"
                disabled={loadingTag}
                onclick={addTag}>Add</button
              >
              <button
                class="button is-small is-text ml-1"
                type="button"
                disabled={loadingTag}
                onclick={() => {
                  addTagOpen = false;
                  inputTag = "";
                }}>Cancel</button
              >
            </div>
          {:else}
            <button
              type="button"
              class="button is-small is-text px-1"
              disabled={loadingTag}
              onclick={() => (addTagOpen = true)}
            >
              + Add Tag
            </button>
          {/if}
          <datalist id={tagListId}>
            {#each autocompleteTags as tag}
              <option value={tag}></option>
            {/each}
          </datalist>
        </div>
      </div>
      <div class="column py-0" style="flex: 0 0 70%; max-width: 70%;">
        <Postings postings={t.postings} {highlightAccount} />
      </div>
    </div>
  {/if}
</div>

<style lang="scss">
  @import "bulma/sass/utilities/_all.sass";

  .description {
    display: inline-block;
    white-space: nowrap;
    overflow: hidden;
  }

  .tags-container {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
  }

  .tag-chip {
    display: inline-flex;
    align-items: center;
    gap: 0.2rem;
    background: var(--p-surface-2);
  }

  .tag-label {
    border: none;
    background: transparent;
    color: inherit;
    cursor: pointer;
    padding: 0;
  }
</style>
