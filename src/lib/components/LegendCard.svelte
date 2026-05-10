<script lang="ts">
  import * as d3 from "d3";
  import type { Action } from "svelte/action";
  import type { Legend } from "$lib/utils";

  let { clazz = "", legends }: { clazz?: string; legends: Legend[] } = $props();

  const textureScale = 14;
  const texture: Action<SVGSVGElement, { texture: any }> = (element, props) => {
    const svg = d3.select(element);
    svg.call(props.texture);
    svg
      .append("rect")
      .attr("x", 0)
      .attr("y", 0)
      .attr("height", textureScale)
      .attr("width", textureScale)
      .attr("fill", props.texture.url());

    return {};
  };

  let selectedLegend: Legend = $state(null);

  function onClick(legend: Legend) {
    if (!legend.onClick) {
      return;
    }

    legend.onClick(legend);
    if (selectedLegend == legend) {
      // toggle
      legend.selected = false;
      selectedLegend = null;
    } else {
      selectedLegend && (selectedLegend.selected = false);
      legend.selected = true;
      selectedLegend = legend;
    }
  }
</script>

<div class="legend-list flex flex-wrap items-start justify-center gap-x-4 gap-y-3 {clazz}">
  {#each legends as legend}
    <button
      type="button"
      class="flex flex-col items-center p-2 gap-1.5 legend-box {legend.onClick && 'cursor-pointer'}"
      onclick={(_e) => onClick(legend)}
      class:selected={selectedLegend == legend}
      disabled={!legend.onClick}
      aria-label={legend.label}
      style="flex: 0 0 auto;"
    >
      {#if legend.texture}
        <svg
          use:texture={{ texture: legend.texture }}
          height="1.1rem"
          width="1.1rem"
          viewBox="0 0 {textureScale} {textureScale}"
        ></svg>
      {:else if legend.shape == "square"}
        <div
          style="background-color: {legend.color}; height: 1.1rem; width: 1.1rem; border-radius: 3px;"
        ></div>
      {:else if legend.shape == "line"}
        <div style="border-top: 3px solid {legend.color}; height: 0.1rem; width: 2rem;"></div>
      {/if}
      <div
        class="legend-label is-size-7 has-text-grey has-text-centered"
        style="word-break: break-word; max-width: 120px;"
      >
        {legend.label}
      </div>
    </button>
  {/each}
</div>

<style lang="scss">
  .legend-list {
    min-width: 0;
  }

  .legend-box {
    min-width: 90px;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    color: inherit;
    transition: all 0.2s ease;

    &:hover:not(:disabled) {
      background-color: rgba(127, 127, 127, 0.1);
      border-color: rgba(127, 127, 127, 0.2);

      :global(html[data-theme="dark"]) & {
        background-color: rgba(0, 0, 0, 0.2);
        border-color: rgba(0, 0, 0, 0.3);
      }
    }

    &.selected {
      background-color: rgba(127, 127, 127, 0.15);
      border-color: var(--bulma-link, #485fc7);

      :global(html[data-theme="dark"]) & {
        background-color: rgba(0, 0, 0, 0.4);
      }
    }

    &:disabled {
      cursor: default;
    }
  }

  @media screen and (max-width: 768px) {
    .legend-list {
      justify-content: flex-start;
      gap: 0.5rem;
      margin-left: 0 !important;
    }

    .legend-box {
      min-width: 72px;
      padding: 0.35rem;
      gap: 0.3rem;
    }

    .legend-label {
      max-width: 88px !important;
      font-size: 0.68rem !important;
      line-height: 1.15;
    }

    .legend-list:global(.ml-4),
    .legend-list:global(.ml-5) {
      margin-left: 0 !important;
    }
  }
</style>
