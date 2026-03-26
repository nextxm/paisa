<script lang="ts">
  import { onDestroy, onMount } from "svelte";
  import _ from "lodash";
  import {
    ajax,
    formatCurrency,
    type SankeyMeta,
    type SankeyNode,
    type SankeyLink,
    type SankeyNodeKind
  } from "$lib/utils";
  import * as toast from "bulma-toast";
  import SankeyDiagram from "$lib/components/SankeyDiagram.svelte";
  import BoxLabel from "$lib/components/BoxLabel.svelte";
  import { sankeyPeriod, sankeyRefDate } from "../../../../persisted_store";
  import { derived } from "svelte/store";
  import dayjs from "dayjs";
  import quarterOfYear from "dayjs/plugin/quarterOfYear";
  dayjs.extend(quarterOfYear);

  let nodes: SankeyNode[] = [];
  let links: SankeyLink[] = [];
  let meta: SankeyMeta | null = null;
  let isLoading = true;

  let unsubscribe: (() => void) | undefined;
  let fetchId = 0;

  async function fetchSankey(period: string, refDate: string) {
    const id = ++fetchId;
    isLoading = true;
    nodes = [];
    links = [];
    meta = null;
    try {
      let url = `/api/sankey?period=${encodeURIComponent(period)}`;
      if (refDate) {
        // Compute explicit from/to dates based on the anchor
        const start = dayjs(refDate)
          .startOf(period as dayjs.OpUnitType)
          .format("YYYY-MM-DD");
        const end = dayjs(refDate)
          .endOf(period as dayjs.OpUnitType)
          .format("YYYY-MM-DD");
        url += `&from=${start}&to=${end}`;
      }
      const data = await ajax(url);
      if (id !== fetchId) return;
      nodes = data.nodes;
      links = data.links;
      meta = data.meta;

      if (meta && meta.hasUnconvertible) {
        toast.toast({
          message:
            "Unable to convert some flows due to missing FX rates. These flows have been excluded.",
          type: "is-warning",
          duration: 5000,
          position: "bottom-center"
        });
      }
    } finally {
      if (id === fetchId) isLoading = false;
    }
  }

  onMount(() => {
    const store = derived([sankeyPeriod, sankeyRefDate], ([$p, $r]) => ({
      period: $p,
      refDate: $r
    }));
    unsubscribe = store.subscribe(({ period, refDate }) => {
      fetchSankey(period, refDate);
    });
  });

  onDestroy(() => {
    unsubscribe?.();
  });

  let displayDepth = 0;
  let hideAssetTransfers = false;

  $: processedGraph = processGraph(nodes, links, displayDepth, hideAssetTransfers);
  $: processedNodes = processedGraph.nodes;
  $: processedLinks = processedGraph.links;

  function processGraph(
    rawNodes: SankeyNode[],
    rawLinks: SankeyLink[],
    depth: number,
    hideTransfers: boolean
  ) {
    if (!rawNodes || rawNodes.length === 0 || !rawLinks || rawLinks.length === 0) {
      return { nodes: [], links: [] };
    }

    const rename = (name: string) => {
      if (depth === 0) return name;
      const parts = name.split(":");
      return parts.slice(0, depth).join(":");
    };

    const linkMap = new Map<string, number>();
    for (const link of rawLinks) {
      const src = rename(link.source);
      const tgt = rename(link.target);

      if (src === tgt) continue;
      if (hideTransfers && src.startsWith("Assets:") && tgt.startsWith("Assets:")) continue;

      const key = `${src}\0${tgt}`;
      linkMap.set(key, (linkMap.get(key) || 0) + link.value);
    }

    const nodeSet = new Map<string, string>();
    const finalLinks: SankeyLink[] = [];
    for (const [key, value] of linkMap.entries()) {
      const [src, tgt] = key.split("\0");

      if (!nodeSet.has(src)) {
        nodeSet.set(src, rawNodes.find((n) => n.id.startsWith(src))?.kind || "other");
      }
      if (!nodeSet.has(tgt)) {
        nodeSet.set(tgt, rawNodes.find((n) => n.id.startsWith(tgt))?.kind || "other");
      }

      // Use 0 for txnCount as it's aggregated and we don't need it specifically
      finalLinks.push({ source: src, target: tgt, value, txnCount: 0 });
    }

    const finalNodes: SankeyNode[] = Array.from(nodeSet.entries()).map(([id, kind]) => ({
      id,
      name: id,
      kind: kind as SankeyNodeKind
    }));

    finalNodes.sort((a, b) => a.id.localeCompare(b.id));
    finalLinks.sort((a, b) => {
      if (a.source !== b.source) return a.source.localeCompare(b.source);
      return a.target.localeCompare(b.target);
    });

    return { nodes: finalNodes, links: finalLinks };
  }
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
          <div class="level mb-4">
            <div class="level-left">
              <div class="level-item">
                <div class="field is-horizontal">
                  <div class="field-label is-normal mr-2">
                    <label
                      class="label has-text-weight-normal is-size-7"
                      style="white-space: nowrap;">Account Depth</label
                    >
                  </div>
                  <div class="field-body">
                    <div class="field">
                      <div class="control">
                        <div class="select is-small">
                          <select bind:value={displayDepth}>
                            <option value={0}>All Levels</option>
                            <option value={1}>Level 1</option>
                            <option value={2}>Level 2</option>
                            <option value={3}>Level 3</option>
                            <option value={4}>Level 4</option>
                          </select>
                        </div>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div class="level-item ml-4">
                <label class="checkbox is-size-7">
                  <input type="checkbox" bind:checked={hideAssetTransfers} />
                  Hide Asset Transfers
                </label>
              </div>
            </div>
          </div>

          <SankeyDiagram
            nodes={processedNodes}
            links={processedLinks}
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
