import dayjs from "dayjs";
import { describe, expect, it } from "bun:test";
import { buildTrendPath, filterTrendPoints, trendRangeFromMonths } from "./account_balance_trend";
import type { Networth } from "./utils";

function point(date: string, balanceAmount: number): Networth {
  return {
    date: dayjs(date),
    investmentAmount: 0,
    withdrawalAmount: 0,
    gainAmount: 0,
    balanceAmount,
    balanceUnits: 0,
    netInvestmentAmount: 0
  };
}

describe("account balance trend helpers", () => {
  it("builds a months-based trend range ending on as-of date", () => {
    const range = trendRangeFromMonths("2026-03-01", 6);
    expect(range.end).toBe("2026-03-01");
    expect(range.start).toBe("2025-09-01");
  });

  it("filters trend points within selected start/end dates", () => {
    const points = [point("2026-01-01", 100), point("2026-02-01", 200), point("2026-03-01", 250)];
    const filtered = filterTrendPoints(points, "2026-02-01", "2026-03-01");
    expect(filtered).toHaveLength(2);
    expect(filtered[0].date.format("YYYY-MM-DD")).toBe("2026-02-01");
    expect(filtered[1].date.format("YYYY-MM-DD")).toBe("2026-03-01");
  });

  it("creates an svg path and last-point marker", () => {
    const points = [point("2026-01-01", 100), point("2026-01-02", 120), point("2026-01-03", 90)];
    const output = buildTrendPath(points, 600, 240);
    expect(output.path.startsWith("M ")).toBe(true);
    expect(output.path.includes(" L ")).toBe(true);
    expect(output.marker).not.toBeNull();
  });
});
