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
  let isBuilt = false;
  let pendingData: any[] | null = null;

  function isTableMounted() {
    return !!tableComponent && tableComponent.isConnected;
  }

  $effect(() => {
    // Capture reactive dependency on data and tableComponent
    const currentData = data;
    const el = tableComponent;

    if (!el) return;

    if (tabulator && isBuilt) {
      tabulator.setData(currentData || []);
    } else if (tabulator && !isBuilt) {
      // Table is initializing — defer until tableBuilt fires
      pendingData = currentData;
    } else {
      // First render: create the table
      tabulator = new Tabulator(el, {
        dataTree: tree,
        dataTreeStartExpanded: [true, true, false],
        dataTreeBranchElement: false,
        dataTreeChildIndent: rem(30),
        dataTreeCollapseElement:
          "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-up'></i></span>",
        dataTreeExpandElement:
          "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-down'></i></span>",
        data: currentData || [],
        columns: columns,
        layout: "fitDataTable"
      });

      tabulator.on("tableBuilt", () => {
        isBuilt = true;
        if (pendingData !== null) {
          tabulator!.setData(pendingData);
          pendingData = null;
        }
      });
    }
  });

  onDestroy(() => {
    if (tabulator) {
      tabulator.destroy();
      tabulator = null;
      isBuilt = false;
      pendingData = null;
    }
  });
</script>

<div class="overflow-x-auto box py-0" style="max-width: 100%;" bind:this={tableComponent}></div>
