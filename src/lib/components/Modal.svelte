<script lang="ts">
  let {
    active = $bindable(false),
    width = "min(640px, 100vw)",
    bodyClass = "",
    headerClass = "",
    footerClass = ""
  } = $props();

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
  <label class="du-modal-backdrop" onclick={() => close()}></label>
</div>
