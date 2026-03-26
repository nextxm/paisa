import { describe, expect, test } from "bun:test";
import type { SankeyLink, SankeyMeta, SankeyNode, SankeyResponse } from "$lib/utils";

// ---------------------------------------------------------------------------
// Helpers mirrored from SankeyDiagram.svelte (extracted for unit testing)
// ---------------------------------------------------------------------------

const kindColors: Record<string, string> = {
  income: "#aeea00",
  asset: "#00b0ff",
  liability: "#ffa000",
  expense: "#ff1744",
  equity: "#d500f9",
  other: "hsl(0, 0%, 48%)"
};

function nodeColor(kind: string): string {
  return kindColors[kind] ?? kindColors["other"];
}

const LABEL_MAX_CHARS = 20;

function truncateLabel(name: string): string {
  const parts = name.split(":");
  const short = parts[parts.length - 1];
  return short.length > LABEL_MAX_CHARS ? short.slice(0, LABEL_MAX_CHARS) + "…" : short;
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("SankeyDiagram – nodeColor", () => {
  test("returns the expected color for each known kind", () => {
    expect(nodeColor("income")).toBe(kindColors["income"]);
    expect(nodeColor("asset")).toBe(kindColors["asset"]);
    expect(nodeColor("liability")).toBe(kindColors["liability"]);
    expect(nodeColor("expense")).toBe(kindColors["expense"]);
    expect(nodeColor("equity")).toBe(kindColors["equity"]);
    expect(nodeColor("other")).toBe(kindColors["other"]);
  });

  test("falls back to 'other' color for unknown kind", () => {
    expect(nodeColor("unknown")).toBe(kindColors["other"]);
    expect(nodeColor("")).toBe(kindColors["other"]);
  });
});

describe("SankeyDiagram – truncateLabel", () => {
  test("returns the last segment of a colon-separated name", () => {
    expect(truncateLabel("Income:Salary:Tech")).toBe("Tech");
    expect(truncateLabel("Expenses:Food")).toBe("Food");
    expect(truncateLabel("Assets")).toBe("Assets");
  });

  test("truncates labels longer than 20 chars", () => {
    const long = "AVeryLongAccountName123";
    expect(truncateLabel(long).endsWith("…")).toBe(true);
    expect(truncateLabel(long).length).toBeLessThanOrEqual(LABEL_MAX_CHARS + 1); // +1 for "…"
  });

  test("does not truncate labels of exactly 20 chars", () => {
    const exact = "ExactlyTwentyCharsXX";
    expect(exact.length).toBe(20);
    expect(truncateLabel(exact)).toBe(exact);
    expect(truncateLabel(exact).endsWith("…")).toBe(false);
  });
});

describe("SankeyDiagram – TypeScript interfaces", () => {
  test("SankeyNode has required fields", () => {
    const node: SankeyNode = { id: "Income:Salary", name: "Income:Salary", kind: "income" };
    expect(node.id).toBe("Income:Salary");
    expect(node.kind).toBe("income");
  });

  test("SankeyLink has required fields", () => {
    const link: SankeyLink = {
      source: "Income:Salary",
      target: "Assets:Checking",
      value: 5000,
      txnCount: 3
    };
    expect(link.value).toBe(5000);
    expect(link.txnCount).toBe(3);
  });

  test("SankeyResponse has nodes, links and meta", () => {
    const meta: SankeyMeta = {
      from: "2024-01-01",
      to: "2024-01-31",
      period: "month",
      currency: "USD",
      totalInflow: 5000,
      totalOutflow: 3000,
      hasUnconvertible: false
    };
    const response: SankeyResponse = {
      nodes: [{ id: "Income", name: "Income", kind: "income" }],
      links: [{ source: "Income", target: "Assets", value: 5000, txnCount: 1 }],
      meta
    };
    expect(response.nodes).toHaveLength(1);
    expect(response.links).toHaveLength(1);
    expect(response.meta.currency).toBe("USD");
  });

  test("empty nodes/links represent the empty state", () => {
    const response: SankeyResponse = {
      nodes: [],
      links: [],
      meta: {
        from: "2024-01-01",
        to: "2024-01-31",
        period: "month",
        currency: "USD",
        totalInflow: 0,
        totalOutflow: 0,
        hasUnconvertible: false
      }
    };
    expect(response.nodes).toHaveLength(0);
    expect(response.links).toHaveLength(0);
  });
});
