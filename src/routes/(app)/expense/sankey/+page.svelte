<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import _ from "lodash";
  import { ajax, formatCurrency, type SankeyMeta, type SankeyNode, type SankeyLink } from "$lib/utils";
  import SankeyDiagram from "$lib/components/SankeyDiagram.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import { dateRange } from "../../../../store";
  import type { Dayjs } from "dayjs";

  let nodes: SankeyNode[] = [];
  let links: SankeyLink[] = [];
  let meta: SankeyMeta | null = null;
  let isLoading = true;

  let unsubscribe: (() => void) | undefined;
  let fetchId = 0;

  async function fetchSankey(from: Dayjs, to: Dayjs) {
    const id = ++fetchId;
    isLoading = true;
    nodes = [];
    links = [];
    meta = null;
    try {
      const fromStr = from.format("YYYY-MM-DD");
      const toStr = to.format("YYYY-MM-DD");
      const data = await ajax(
        `/api/sankey?from=${encodeURIComponent(fromStr)}&to=${encodeURIComponent(toStr)}`
      );
      if (id !== fetchId) return;
      nodes = data.nodes;
      links = data.links;
      meta = data.meta;
    } finally {
      if (id === fetchId) isLoading = false;
    }
  }

  onMount(() => {
    unsubscribe = dateRange.subscribe((range) => {
      fetchSankey(range.from, range.to);
    });
  });

  onDestroy(() => {
    unsubscribe?.();
  });
</script>

<section class="section tab-expense">
  <div class="container is-fluid">
    {#if meta}
      <div class="level mb-3">
        <div class="level-left">
          <div class="level-item">
            <span class="has-text-grey is-size-7">
              {meta.from} – {meta.to}
            </span>
          </div>
        </div>
        <div class="level-right">
          <div class="level-item has-text-centered mr-4">
            <div>
              <p class="heading">Total Inflow</p>
              <p class="title is-6 has-text-success">{formatCurrency(meta.totalInflow)}</p>
            </div>
          </div>
          <div class="level-item has-text-centered">
            <div>
              <p class="heading">Total Outflow</p>
              <p class="title is-6 has-text-danger">{formatCurrency(meta.totalOutflow)}</p>
            </div>
          </div>
        </div>
      </div>
    {/if}

    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto">
          <SankeyDiagram
            {nodes}
            {links}
            height={600}
            loading={isLoading}
            emptyMessage="No flow data available for this period."
          />
        </div>
      </div>
    </div>
    <BoxLabel text="Expense Flow" />
  </div>
</section>
