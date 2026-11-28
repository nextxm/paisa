<script lang="ts">
  import { filterSuggestions } from "$lib/autocomplete";

  let {
    id,
    value = "",
    suggestions = [],
    placeholder = "",
    disabled = false,
    inputClass = "",
    onChange
  }: {
    id?: string;
    value?: string;
    suggestions?: ReadonlyArray<string>;
    placeholder?: string;
    disabled?: boolean;
    inputClass?: string;
    onChange: (next: string) => void;
  } = $props();

  let open = $state(false);
  let activeIndex = $state(0);

  const filtered = $derived(filterSuggestions(suggestions, value));

  function selectItem(item: string) {
    onChange(item);
    open = false;
    activeIndex = 0;
  }

  function handleKeydown(e: KeyboardEvent) {
    if (!open && (e.key === "ArrowDown" || e.key === "ArrowUp") && filtered.length > 0) {
      open = true;
      activeIndex = 0;
      e.preventDefault();
      return;
    }
    if (!open || filtered.length === 0) return;

    if (e.key === "ArrowDown") {
      e.preventDefault();
      activeIndex = (activeIndex + 1) % filtered.length;
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      activeIndex = (activeIndex - 1 + filtered.length) % filtered.length;
    } else if (e.key === "Enter") {
      e.preventDefault();
      selectItem(filtered[activeIndex] ?? filtered[0]);
    } else if (e.key === "Escape") {
      open = false;
    }
  }
</script>

<div class="relative w-full">
  <input
    {id}
    type="text"
    {disabled}
    {placeholder}
    {value}
    autocomplete="off"
    onfocus={() => {
      if (filtered.length > 0) {
        open = true;
        activeIndex = 0;
      }
    }}
    onblur={() => {
      open = false;
    }}
    oninput={(e) => {
      const next = (e.currentTarget as HTMLInputElement).value;
      onChange(next);
      open = true;
      activeIndex = 0;
    }}
    onkeydown={handleKeydown}
    class={inputClass}
  />

  {#if open && !disabled && filtered.length > 0}
    <ul
      role="listbox"
      class="absolute left-0 right-0 z-50 mt-1 max-h-52 overflow-y-auto rounded-lg border border-[var(--border)] bg-[var(--surface)] shadow-lg py-1 text-xs font-mono"
    >
      {#each filtered as item, idx (item)}
        <li role="option" aria-selected={idx === activeIndex}>
          <button
            type="button"
            onmousedown={(e) => {
              e.preventDefault();
              selectItem(item);
            }}
            onmouseenter={() => (activeIndex = idx)}
            class="w-full text-left px-3 py-1.5 cursor-pointer truncate {idx === activeIndex
              ? 'bg-[var(--accent)]/15 text-[var(--accent)] font-medium'
              : 'text-[var(--text)] hover:bg-[var(--surface-elevated)]'}"
          >
            {item}
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</div>
