<script lang="ts">
  import { onMount } from "svelte";
  import _ from "lodash";
  import { renderFlow } from "$lib/cash_flow";
  import { ajax, depth, firstName, rem, type Graph, type Legend, type Posting } from "$lib/utils";
  import { dateMin, year } from "../../../../store";
  import {
    setCashflowDepthAllowed,
    cashflowExpenseDepth,
    cashflowIncomeDepth,
    cashflowShowTransfers
  } from "../../../../persisted_store";
  import ZeroState from "$lib/components/ZeroState.svelte";
  import LegendCard from "$lib/components/LegendCard.svelte";

  let legends: Legend[] = [];
  let graph: Record<string, Graph>, expenses: Posting[];
  let isEmpty = false;

  function maxDepth(prefix: string) {
    if (!graph) return 1;
    const max = _.chain(graph)
      .flatMap((g) => g.nodes)
      .filter((n) => n.name.startsWith(prefix))
      .map((n) => depth(n.name))
      .max()
      .value();

    return max || 1;
  }

  function filter(graph: Graph, incomeDepth: number, expenseDepth: number) {
    if (!graph) return graph;

    const [removed, allowed] = _.partition(graph.nodes, (n) => {
      const account = firstName(n.name);
      if (account === "Income") return depth(n.name) > incomeDepth;
      if (account === "Expenses") return depth(n.name) > expenseDepth;
      return false;
    });

    const removedIds = removed.map((n) => n.id);
    return {
      nodes: allowed,
      links: graph.links.filter(
        (l) => !removedIds.includes(l.source) && !removedIds.includes(l.target)
      )
    };
  }

  function filterTransfers(graph: Graph): Graph {
    if (!graph) return graph;

    const nodeById = new Map(graph.nodes.map((n) => [n.id, n]));
    const isIncomeOrExpenses = (nodeId: number) => {
      const node = nodeById.get(nodeId);
      return node != null && (node.name.startsWith("Income") || node.name.startsWith("Expenses"));
    };

    const filteredLinks = graph.links.filter(
      (l) => isIncomeOrExpenses(l.source) || isIncomeOrExpenses(l.target)
    );

    const usedIds = new Set<number>();
    for (const l of filteredLinks) {
      usedIds.add(l.source);
      usedIds.add(l.target);
    }

    return {
      nodes: graph.nodes.filter((n) => usedIds.has(n.id)),
      links: filteredLinks
    };
  }

  $: if (graph) {
    if (graph[$year] == null) {
      isEmpty = true;
    } else {
      let g = filter(_.cloneDeep(graph[$year]), $cashflowIncomeDepth, $cashflowExpenseDepth);
      if (!$cashflowShowTransfers) {
        g = filterTransfers(g);
      }
      legends = renderFlow(g);
      isEmpty = false;
    }
  }

  onMount(async () => {
    ({ expenses, graph } = await ajax("/api/expense"));
    let firstExpense = _.minBy(expenses, (e) => e.date);
    if (firstExpense) {
      dateMin.set(firstExpense.date);
    }

    setCashflowDepthAllowed(maxDepth("Expenses"), maxDepth("Income"));
  });
</script>

<section class="section" style="padding-bottom: 0 !important">
  <div class="container is-fluid">
    <div class="columns">
      <div class="column is-12">
        <div class="box overflow-x-auto">
          <ZeroState item={!isEmpty}
            ><strong>Oops!</strong> You have not made any transactions for the selected year.</ZeroState
          >

          <LegendCard {legends} clazz="ml-5 mb-2" />
          <svg
            class:is-not-visible={isEmpty}
            id="d3-expense-flow"
            height={window.innerHeight - rem(210)}
          />
        </div>
      </div>
    </div>
  </div>
</section>
