<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import _ from "lodash";
  import {
    ajax,
    formatCurrency,
    type SankeyMeta,
    type SankeyNode,
    type SankeyLink
  } from "$lib/utils";
  import SankeyDiagram from "$lib/components/SankeyDiagram.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import { sankeyPeriod } from "../../../../persisted_store";

  let nodes: SankeyNode[] = [];
  let links: SankeyLink[] = [];
  let meta: SankeyMeta | null = null;
  let isLoading = true;

  let unsubscribe: (() => void) | undefined;
  let fetchId = 0;

  async function fetchSankey(period: string) {
    const id = ++fetchId;
    isLoading = true;
    nodes = [];
    links = [];
    meta = null;
    try {
      const data = await ajax(`/api/sankey?period=${encodeURIComponent(period)}`);
      if (id !== fetchId) return;
      nodes = data.nodes;
      links = data.links;
      meta = data.meta;
    } finally {
      if (id === fetchId) isLoading = false;
    }
  }

  onMount(() => {
    unsubscribe = sankeyPeriod.subscribe((period) => {
      fetchSankey(period);
    });
  });

  onDestroy(() => {
    unsubscribe?.();
  });
</script>

<section class="section">
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
            onLinkClick={(link) => {
              // TODO: implement drill-down navigation in a future iteration
              void link;
            }}
          />
        </div>
      </div>
    </div>
    <BoxLabel text="Money Flow" />
  </div>
</section>
