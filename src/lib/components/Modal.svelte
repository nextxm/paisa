<script lang="ts">
  export let active = false;
  export let width = "min(640px, 100vw)";
  export let bodyClass = "";
  export let headerClass = "";
  export let footerClass = "";

  function close() {
    active = false;
  }
</script>

<div class="du-modal" class:du-modal-open={active}>
  <div class="du-modal-box p-0 max-w-none overflow-visible" style:width>
    <header class="flex items-center px-4 py-3 border-b border-base-300 {headerClass}">
      <slot name="head" {close} />
    </header>
    <section class="p-4 overflow-y-auto {bodyClass}">
      <slot name="body" />
    </section>
    <footer class="flex items-center px-4 py-3 border-t border-base-300 {footerClass}">
      <slot name="foot" {close} />
    </footer>
  </div>
  <button type="button" class="du-modal-backdrop" aria-label="Close modal" onclick={() => close()}
  ></button>
</div>
