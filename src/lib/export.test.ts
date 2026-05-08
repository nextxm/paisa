import { describe, expect, test } from "bun:test";

import { buildAssetBalanceExportRows } from "./export";
import type { AssetBreakdown } from "./utils";

function breakdown(group: string): AssetBreakdown {
  return {
    group,
    investmentAmount: 10,
    withdrawalAmount: 2,
    balanceUnits: 1,
    marketAmount: 12,
    xirr: 0.1,
    gainAmount: 4,
    absoluteReturn: 0.4,
    originalBalances: [{ currency: "USD", amount: 12 }],
    inactive: false
  };
}

describe("buildAssetBalanceExportRows", () => {
  const breakdowns: Record<string, AssetBreakdown> = {
    "Assets:Bank:Checking": breakdown("Assets:Bank:Checking"),
    "Assets:Bank": breakdown("Assets:Bank"),
    Assets: breakdown("Assets")
  };

  test("builds hierarchy rows in tree order with indented labels", () => {
    const rows = buildAssetBalanceExportRows(breakdowns, false);

    expect(rows.map((row) => row.Account)).toEqual(["Assets", "  Bank", "    Checking"]);
  });

  test("builds flat rows in sorted account order", () => {
    const rows = buildAssetBalanceExportRows(breakdowns, true);

    expect(rows.map((row) => row.Account)).toEqual([
      "Assets",
      "Assets:Bank",
      "Assets:Bank:Checking"
    ]);
  });
});
