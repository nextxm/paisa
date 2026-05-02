<script lang="ts">
  import type { Snippet } from "svelte";

  let {
    active = $bindable(false),
    width = "min(640px, 100vw)",
    bodyClass = "",
    headerClass = "",
    footerClass = "",
    head,
    body,
    foot
  }: {
    active?: boolean;
    width?: string;
    bodyClass?: string;
    headerClass?: string;
    footerClass?: string;
    head?: Snippet<[any]>;
    body?: Snippet;
    foot?: Snippet<[any]>;
  } = $props();

  function close() {
    active = false;
  }
</script>

<div class="du-modal" class:du-modal-open={active}>
  <div class="du-modal-box p-0 max-w-none overflow-visible" style:width>
    {#if head}
      <header class="flex items-center px-4 py-3 border-b border-base-300 {headerClass}">
        {@render head(close)}
      </header>
    {/if}
    <section class="p-4 overflow-y-auto {bodyClass}">
      {#if body}
        {@render body()}
      {/if}
    </section>
    {#if foot}
      <footer class="flex items-center px-4 py-3 border-t border-base-300 {footerClass}">
        {@render foot(close)}
      </footer>
    {/if}
  </div>
  <button type="button" class="du-modal-backdrop" aria-label="Close modal" onclick={() => close()}
  ></button>
</div>
