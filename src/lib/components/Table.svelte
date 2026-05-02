<script lang="ts">
  import { rem } from "$lib/utils";
  import { onDestroy, onMount } from "svelte";
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
  let isBuilt = false;

  function isTableMounted() {
    return !!tableComponent && tableComponent.isConnected;
  }

  $effect(() => {
    // React to data or columns changing
    if (tabulator && isBuilt) {
      tabulator.setData(data || []);
    }
  });

  $effect(() => {
    if (!isTableMounted() || tabulator) {
      return;
    }

    tabulator = new Tabulator(tableComponent, {
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
      isBuilt = true;
    });
  });

  onDestroy(() => {
    if (tabulator) {
      tabulator.destroy();
      tabulator = null;
      isBuilt = false;
    }
  });
</script>

<div class="overflow-x-auto box py-0" style="max-width: 100%;" bind:this={tableComponent}></div>
