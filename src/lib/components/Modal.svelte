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
    head?: Snippet<[() => void]>;
    body?: Snippet;
    foot?: Snippet<[() => void]>;
  } = $props();

  function close() {
    active = false;
  }
</script>

<div class="du-modal" class:du-modal-open={active}>
  <div class="du-modal-box p-0 max-w-none overflow-visible" style:width>
    <header class="flex items-center px-4 py-3 border-b border-base-300 {headerClass}">
      {@render head?.(close)}
    </header>
    <section class="p-4 overflow-y-auto {bodyClass}">
      {@render body?.()}
    </section>
    <footer class="flex items-center px-4 py-3 border-t border-base-300 {footerClass}">
      {@render foot?.(close)}
    </footer>
  </div>
  <button type="button" class="du-modal-backdrop" aria-label="Close modal" onclick={() => close()}
  ></button>
</div>
