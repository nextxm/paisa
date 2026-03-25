import { describe, expect, test } from "bun:test";
import { classifyAccount, graphToSankey } from "./sankey_utils";
import type { Graph } from "$lib/utils";

// ---------------------------------------------------------------------------
// classifyAccount
// ---------------------------------------------------------------------------

describe("classifyAccount", () => {
  test("classifies known top-level prefixes", () => {
    expect(classifyAccount("Income:Salary")).toBe("income");
    expect(classifyAccount("Income:Salary:Tech")).toBe("income");
    expect(classifyAccount("Expenses:Food")).toBe("expense");
    expect(classifyAccount("Expenses:Tax:Federal")).toBe("expense");
    expect(classifyAccount("Assets:Checking")).toBe("asset");
    expect(classifyAccount("Assets:Savings:Emergency")).toBe("asset");
    expect(classifyAccount("Liabilities:CreditCard")).toBe("liability");
    expect(classifyAccount("Equity:OpeningBalance")).toBe("equity");
  });

  test("falls back to 'other' for unknown or empty prefixes", () => {
    expect(classifyAccount("Unknown:Account")).toBe("other");
    expect(classifyAccount("")).toBe("other");
    expect(classifyAccount("income")).toBe("other"); // lowercase doesn't match
    expect(classifyAccount("expenses")).toBe("other");
  });
});

// ---------------------------------------------------------------------------
// graphToSankey – simple linear flow
// ---------------------------------------------------------------------------

describe("graphToSankey – simple linear flow", () => {
  test("transforms a two-node graph to SankeyNode and SankeyLink arrays", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [{ source: 1, target: 2, value: 5000 }]
    };
    const { nodes, links } = graphToSankey(graph);

    expect(nodes).toHaveLength(2);
    expect(links).toHaveLength(1);

    const incomeNode = nodes.find((n) => n.id === "Income:Salary");
    const assetNode = nodes.find((n) => n.id === "Assets:Checking");
    expect(incomeNode?.kind).toBe("income");
    expect(assetNode?.kind).toBe("asset");

    expect(links[0].source).toBe("Income:Salary");
    expect(links[0].target).toBe("Assets:Checking");
    expect(links[0].value).toBe(5000);
    expect(links[0].txnCount).toBe(0);
  });

  test("output nodes are ordered deterministically (alphabetical by name)", () => {
    const graph: Graph = {
      nodes: [
        { id: 3, name: "Income:Salary" },
        { id: 1, name: "Assets:Checking" },
        { id: 2, name: "Expenses:Food" }
      ],
      links: [
        { source: 3, target: 1, value: 5000 },
        { source: 1, target: 2, value: 1000 }
      ]
    };
    const { nodes } = graphToSankey(graph);
    const names = nodes.map((n) => n.name);
    expect(names).toEqual([...names].sort());
  });

  test("output links are ordered deterministically (alphabetical by source then target)", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" },
        { id: 3, name: "Expenses:Food" }
      ],
      links: [
        { source: 2, target: 3, value: 1000 },
        { source: 1, target: 2, value: 5000 }
      ]
    };
    const { links } = graphToSankey(graph);
    expect(links[0].source).toBe("Assets:Checking");
    expect(links[1].source).toBe("Income:Salary");
  });
});

// ---------------------------------------------------------------------------
// graphToSankey – bidirectional flow netting
// ---------------------------------------------------------------------------

describe("graphToSankey – bidirectional flow", () => {
  test("nets A→B and B→A, keeping only the dominant direction", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Assets:Checking" },
        { id: 2, name: "Assets:Savings" }
      ],
      links: [
        { source: 1, target: 2, value: 3000 },
        { source: 2, target: 1, value: 1000 }
      ]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(1);
    expect(links[0].source).toBe("Assets:Checking");
    expect(links[0].target).toBe("Assets:Savings");
    expect(links[0].value).toBeCloseTo(2000);
  });

  test("drops exactly-cancelling bidirectional pairs (both nodes and link removed)", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Assets:Checking" },
        { id: 2, name: "Assets:Savings" }
      ],
      links: [
        { source: 1, target: 2, value: 1000 },
        { source: 2, target: 1, value: 1000 }
      ]
    };
    const { links, nodes } = graphToSankey(graph);

    expect(links).toHaveLength(0);
    expect(nodes).toHaveLength(0);
  });

  test("reverses direction when the reverse flow is dominant", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Assets:Checking" },
        { id: 2, name: "Income:Salary" }
      ],
      links: [
        { source: 1, target: 2, value: 500 },
        { source: 2, target: 1, value: 4000 }
      ]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(1);
    expect(links[0].source).toBe("Income:Salary");
    expect(links[0].target).toBe("Assets:Checking");
    expect(links[0].value).toBeCloseTo(3500);
  });
});

// ---------------------------------------------------------------------------
// graphToSankey – duplicate edge aggregation
// ---------------------------------------------------------------------------

describe("graphToSankey – duplicate edge aggregation", () => {
  test("sums multiple links with the same source and target", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [
        { source: 1, target: 2, value: 2000 },
        { source: 1, target: 2, value: 3000 }
      ]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(1);
    expect(links[0].value).toBeCloseTo(5000);
  });

  test("aggregates duplicates then nets against reverse direction", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Assets:Checking" },
        { id: 2, name: "Assets:Savings" }
      ],
      links: [
        { source: 1, target: 2, value: 1500 },
        { source: 1, target: 2, value: 1000 }, // duplicate: total 2500
        { source: 2, target: 1, value: 800 }
      ]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(1);
    expect(links[0].source).toBe("Assets:Checking");
    expect(links[0].target).toBe("Assets:Savings");
    expect(links[0].value).toBeCloseTo(1700);
  });
});

// ---------------------------------------------------------------------------
// graphToSankey – missing and invalid value filtering
// ---------------------------------------------------------------------------

describe("graphToSankey – missing and invalid value filtering", () => {
  test("drops links referencing unknown source node IDs", () => {
    const graph: Graph = {
      nodes: [{ id: 2, name: "Assets:Checking" }],
      links: [{ source: 99, target: 2, value: 5000 }]
    };
    const { links, nodes } = graphToSankey(graph);

    expect(links).toHaveLength(0);
    expect(nodes).toHaveLength(0);
  });

  test("drops links referencing unknown target node IDs", () => {
    const graph: Graph = {
      nodes: [{ id: 1, name: "Income:Salary" }],
      links: [{ source: 1, target: 99, value: 5000 }]
    };
    const { links, nodes } = graphToSankey(graph);

    expect(links).toHaveLength(0);
    expect(nodes).toHaveLength(0);
  });

  test("drops links with zero value", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [{ source: 1, target: 2, value: 0 }]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(0);
  });

  test("drops links with negative value", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [{ source: 1, target: 2, value: -500 }]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(0);
  });

  test("drops links with NaN value", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [{ source: 1, target: 2, value: NaN }]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(0);
  });

  test("drops links with Infinity value", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: [{ source: 1, target: 2, value: Infinity }]
    };
    const { links } = graphToSankey(graph);

    expect(links).toHaveLength(0);
  });

  test("drops self-links (source === target by name)", () => {
    const graph: Graph = {
      nodes: [{ id: 1, name: "Assets:Checking" }],
      links: [{ source: 1, target: 1, value: 1000 }]
    };
    const { links, nodes } = graphToSankey(graph);

    expect(links).toHaveLength(0);
    expect(nodes).toHaveLength(0);
  });

  test("handles empty graph (no nodes or links)", () => {
    const { nodes, links } = graphToSankey({ nodes: [], links: [] });

    expect(nodes).toHaveLength(0);
    expect(links).toHaveLength(0);
  });

  test("handles graph with nodes but no links", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" }
      ],
      links: []
    };
    const { nodes, links } = graphToSankey(graph);

    expect(nodes).toHaveLength(0);
    expect(links).toHaveLength(0);
  });

  test("only includes nodes that are referenced by surviving links", () => {
    const graph: Graph = {
      nodes: [
        { id: 1, name: "Income:Salary" },
        { id: 2, name: "Assets:Checking" },
        { id: 3, name: "Expenses:Food" }
      ],
      links: [
        { source: 1, target: 2, value: 5000 }
        // node 3 (Expenses:Food) is not referenced by any link
      ]
    };
    const { nodes } = graphToSankey(graph);

    expect(nodes).toHaveLength(2);
    expect(nodes.map((n) => n.id)).not.toContain("Expenses:Food");
  });
});
