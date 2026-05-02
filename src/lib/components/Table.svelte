<script lang="ts">
  import { rem } from "$lib/utils";
  import { onDestroy } from "svelte";
  import { TabulatorFull as Tabulator, type ColumnDefinition } from "tabulator-tables";

  let {
    data,
    columns,
    tree = false
  }: {
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
    const currentData = data;
    if (tabulator && isBuilt) {
      tabulator.setData(currentData || []).catch(() => {});
    } else if (tabulator && !isBuilt) {
      pendingData = currentData;
    }
  });

  $effect(() => {
    // React to columns changing
    const currentColumns = columns;
    if (tabulator && isBuilt) {
      tabulator.setColumns(currentColumns);
    }
  });

  $effect(() => {
    const el = tableComponent;
    if (!el || !isTableMounted() || tabulator) {
      return;
    }

    tabulator = new Tabulator(el, {
      dataTree: tree,
      dataTreeStartExpanded: [true, true, false],
      dataTreeBranchElement: false,
      dataTreeChildIndent: rem(30),
      dataTreeCollapseElement:
        "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-up'></i></span>",
      dataTreeExpandElement:
        "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-down'></i></span>",
      data: data || [],
      columns: columns,
      layout: "fitDataTable"
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

<div class="overflow-x-auto box py-0" style="max-width: 100%;" bind:this={tableComponent}></div>
