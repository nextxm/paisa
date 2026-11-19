<script lang="ts">
  import type { Snippet } from "svelte";
  import { fade } from "svelte/transition";
  import { delayedLoading, delayedUnLoading } from "../../store";
  import Logo from "./Logo.svelte";

  let { children }: { children: Snippet } = $props();
  let size = 90;
</script>

<div>
  <div class="spinner-content" class:is-loading={$delayedUnLoading || $delayedLoading}>
    {@render children()}
  </div>
  {#if $delayedUnLoading || $delayedLoading}
    <div class="circle-container" transition:fade={{ duration: 200 }}>
      <Logo {size} animation />
    </div>
  {/if}
</div>

<style lang="scss">
  .spinner-content {
    transition: opacity 150ms ease;

    &.is-loading {
      opacity: 0.55;
      pointer-events: none;
    }
  }

  .circle-container {
    margin: auto;
    position: fixed;
    top: -45px;
    left: 0;
    bottom: 0;
    right: 0;
    height: 90px;
    width: 90px;
    z-index: 90;
    pointer-events: none;
  }
</style>
