<script lang="ts">
  import { rem } from "$lib/utils";
  import { onMount } from "svelte";
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
  let tabulator: Tabulator;

  $effect(() => {
    if (data.length > 0) {
      build();
    }
  });

  async function build() {
    if (data.length === 0) {
      return;
    }

    if (tabulator) {
      tabulator.replaceData(data);
    } else {
      tabulator = new Tabulator(tableComponent, {
        dataTree: tree,
        dataTreeStartExpanded: [true, true, false],
        dataTreeBranchElement: false,
        dataTreeChildIndent: rem(30),
        dataTreeCollapseElement:
          "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-up'></i></span>",
        dataTreeExpandElement:
          "<span class='has-text-link icon is-small mr-3'><i class='fas fa-angle-down'></i></span>",
        data: data,
        columns: columns,
        layout: "fitDataTable"
      });
    }
  }

  onMount(async () => {
    build();
  });
</script>

<div class="overflow-x-auto box py-0" style="max-width: 100%;" bind:this={tableComponent}></div>
