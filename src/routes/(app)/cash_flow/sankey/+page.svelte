<script lang="ts">
  import {
    ajax,
    formatCurrency,
    type SankeyMeta,
    type SankeyNode,
    type SankeyLink,
    type SankeyResponse
  } from "$lib/utils";
  import SankeyDiagram from "$lib/components/SankeyDiagram.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import BoxedTabs from "$lib/components/BoxedTabs.svelte";

  let nodes: SankeyNode[] = [];
  let links: SankeyLink[] = [];
  let meta: SankeyMeta | null = null;
  let isLoading = true;

  type SankeyPeriod = "month" | "quarter" | "year";
  let period: SankeyPeriod = "month";

  const periodOptions: { label: string; value: SankeyPeriod }[] = [
    { label: "Month", value: "month" },
    { label: "Quarter", value: "quarter" },
    { label: "Year", value: "year" }
  ];

  async function fetchSankey(p: SankeyPeriod) {
    isLoading = true;
    try {
      const params = new URLSearchParams({ period: p });
      const data: SankeyResponse = await ajax(`/api/sankey?${params}`);
      nodes = data.nodes;
      links = data.links;
      meta = data.meta;
    } finally {
      isLoading = false;
    }
  }

  $: fetchSankey(period);
</script>

<section class="section">
  <div class="container is-fluid">
    <div class="level mb-3">
      <div class="level-left">
        <div class="level-item">
          {#if meta}
            <span class="has-text-grey is-size-7">
              {meta.from} – {meta.to}
            </span>
          {/if}
        </div>
      </div>
      <div class="level-right">
        {#if meta}
          <div class="level-item has-text-centered mr-4">
            <div>
              <p class="heading">Total Inflow</p>
              <p class="title is-6 has-text-success">{formatCurrency(meta.totalInflow)}</p>
            </div>
          </div>
          <div class="level-item has-text-centered mr-4">
            <div>
              <p class="heading">Total Outflow</p>
              <p class="title is-6 has-text-danger">{formatCurrency(meta.totalOutflow)}</p>
            </div>
          </div>
        {/if}
        <div class="level-item">
          <BoxedTabs bind:value={period} options={periodOptions} />
        </div>
      </div>
    </div>

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
