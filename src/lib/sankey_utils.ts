import type { Graph, SankeyLink, SankeyNode, SankeyNodeKind } from "$lib/utils";

/**
 * Minimum absolute flow value included in the Sankey output.
 * Links whose netted value is below this threshold are dropped.
 */
const SANKEY_EPSILON = 0.001;

/**
 * Classifies a ledger account name into a SankeyNodeKind based on its
 * top-level prefix (e.g. "Income:Salary" → "income").
 *
 * Mirrors the Go `classifySankeyAccount` function in internal/server/sankey.go.
 */
export function classifyAccount(account: string): SankeyNodeKind {
  const top = account.split(":")[0];
  switch (top) {
    case "Income":
      return "income";
    case "Expenses":
      return "expense";
    case "Assets":
      return "asset";
    case "Liabilities":
      return "liability";
    case "Equity":
      return "equity";
    default:
      return "other";
  }
}

/**
 * Transforms a legacy Graph payload (from GET /api/expense `graph` field) into
 * Sankey-compatible { nodes, links } arrays suitable for SankeyDiagram.svelte.
 *
 * The legacy Graph uses numeric node IDs; SankeyNode uses the account name as
 * the string ID.  The transformer:
 *
 *  1. Builds a numeric-ID → account-name lookup from the nodes array.
 *  2. Aggregates duplicate edges (same source/target pair) by summing values.
 *  3. Nets bidirectional pairs (A→B and B→A): keeps only the dominant direction
 *     with value = |dominant| − |reverse|.  Exactly-equal pairs cancel out.
 *  4. Drops links with:
 *     - Unknown node IDs (source or target not present in nodes array).
 *     - Self-links (source account === target account).
 *     - Zero, negative, or NaN values (before and after netting).
 *  5. Returns deterministically ordered (alphabetical) nodes and links.
 */
export function graphToSankey(graph: Graph): { nodes: SankeyNode[]; links: SankeyLink[] } {
  // Build numeric-id → account-name lookup.
  const idToName = new Map<number, string>();
  for (const node of graph.nodes) {
    idToName.set(node.id, node.name);
  }

  // First pass: aggregate link values by directed (sourceName, targetName) pair.
  // Use null byte as separator – safe because account names are plain text.
  const rawValues = new Map<string, number>();
  for (const link of graph.links) {
    const sourceName = idToName.get(link.source);
    const targetName = idToName.get(link.target);

    // Drop links with unknown node references.
    if (sourceName === undefined || targetName === undefined) {
      continue;
    }

    // Drop self-links.
    if (sourceName === targetName) {
      continue;
    }

    const value = link.value;

    // Drop NaN, Infinity, zero, or negative values.
    if (!Number.isFinite(value) || value <= 0) {
      continue;
    }

    const key = `${sourceName}\0${targetName}`;
    rawValues.set(key, (rawValues.get(key) ?? 0) + value);
  }

  // Second pass: net bidirectional pairs.
  const nettedValues = new Map<string, number>();
  const processed = new Set<string>();

  for (const [key, value] of rawValues) {
    if (processed.has(key)) continue;

    const [sourceName, targetName] = key.split("\0");
    const reverseKey = `${targetName}\0${sourceName}`;
    const reverseValue = rawValues.get(reverseKey) ?? 0;

    processed.add(key);
    processed.add(reverseKey);

    const net = value - reverseValue;
    if (net > SANKEY_EPSILON) {
      nettedValues.set(key, net);
    } else if (net < -SANKEY_EPSILON) {
      nettedValues.set(reverseKey, -net);
    }
    // Values within ±SANKEY_EPSILON of zero cancel out and are discarded.
  }

  // Collect all account names referenced by surviving links.
  const nodeSet = new Map<string, SankeyNodeKind>();
  for (const key of nettedValues.keys()) {
    const [sourceName, targetName] = key.split("\0");
    nodeSet.set(sourceName, classifyAccount(sourceName));
    nodeSet.set(targetName, classifyAccount(targetName));
  }

  // Build deterministically ordered node array (alphabetical by name).
  const nodeNames = Array.from(nodeSet.keys()).sort();
  const nodes: SankeyNode[] = nodeNames.map((name) => ({
    id: name,
    name,
    kind: nodeSet.get(name)!
  }));

  // Build deterministically ordered link array (alphabetical by source, then target).
  const sortedKeys = Array.from(nettedValues.keys()).sort();
  const links: SankeyLink[] = sortedKeys.map((key) => {
    const [sourceName, targetName] = key.split("\0");
    return {
      source: sourceName,
      target: targetName,
      value: nettedValues.get(key)!,
      txnCount: 0 // transaction count is not available in the legacy Graph format
    };
  });

  return { nodes, links };
}
