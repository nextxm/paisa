<script lang="ts">
  import { rem } from "$lib/utils";
  import { onDestroy } from "svelte";
  import { TabulatorFull as Tabulator, type ColumnDefinition } from "tabulator-tables";

  let props: {
    data: any[];
    columns: ColumnDefinition[];
    tree?: boolean;
  } = $props();

  let tableComponent: HTMLElement = $state();
  let tabulator: Tabulator | null = null;
  let isBuilt = $state(false);
  let pendingData: any[] | null = null;

  function isTableMounted() {
    return !!tableComponent && tableComponent.isConnected;
  }

  $effect(() => {
    // React to data changing
    const currentData = props.data;
    if (tabulator && isBuilt) {
      tabulator.setData(currentData || []).catch(() => {});
    } else if (tabulator && !isBuilt) {
      pendingData = currentData;
    }
  });

  $effect(() => {
    // React to columns changing
    const currentColumns = props.columns;
    if (tabulator && isBuilt) {
      tabulator.setColumns(currentColumns);
      tabulator.redraw(true);
    }
  });

  $effect(() => {
    const el = tableComponent;
    if (!el || !isTableMounted() || tabulator) {
      return;
    }

    const layout =
      typeof window !== "undefined" && window.innerWidth <= 768 ? "fitColumns" : "fitDataTable";

    tabulator = new Tabulator(el, {
      dataTree: props.tree,
      dataTreeStartExpanded: [true, true, false],
      dataTreeBranchElement: false,
      dataTreeChildIndent: rem(30),
      dataTreeCollapseElement:
        "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-up'></i></span>",
      dataTreeExpandElement:
        "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-down'></i></span>",
      data: props.data || [],
      columns: props.columns,
      layout
    });

    tabulator.on("tableBuilt", () => {
      if (!tabulator) return;
      isBuilt = true;
      if (pendingData !== null) {
        tabulator.setData(pendingData).catch(() => {});
        pendingData = null;
      }
    });
  });

  onDestroy(() => {
    if (tabulator) {
      const t = tabulator;
      tabulator = null;
      isBuilt = false;
      pendingData = null;
      try {
        t.destroy();
      } catch {
        // Ignore destruction errors
      }
    }
  });
</script>

<div class="responsive-table box py-0" bind:this={tableComponent}></div>

<style lang="scss">
  .responsive-table {
    max-width: 100%;
    min-width: 0;
    overflow-x: auto;
    overflow-y: hidden;
    -webkit-overflow-scrolling: touch;
    overscroll-behavior-x: contain;
  }

  .responsive-table :global(.tabulator) {
    min-width: 100%;
  }

  @media screen and (max-width: 768px) {
    .responsive-table {
      padding-bottom: 0.35rem;
    }
  }
</style>
