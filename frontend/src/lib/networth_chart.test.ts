import { describe, expect, test } from "bun:test";
import {
  buildBandPath,
  buildLinePath,
  computeNetworthBounds,
  getInvestmentReturn,
  type NetworthPoint
} from "./networth_chart";

describe("networth_chart helpers", () => {
  const sample: NetworthPoint[] = [
    {
      date: "2026-01-01",
      balanceAmount: 120000,
      netInvestmentAmount: 100000,
      gainAmount: 20000,
      investment_return: 15000,
      fx_impact: 5000
    },
    {
      date: "2026-02-01",
      balanceAmount: 110000,
      netInvestmentAmount: 115000,
      gainAmount: -5000,
      investment_return: -7000,
      fx_impact: 2000
    }
  ];

  test("getInvestmentReturn uses explicit investment_return or derives from gain - fx_impact", () => {
    expect(getInvestmentReturn(sample[0])).toBe(15000);
    expect(
      getInvestmentReturn({
        date: "2026-03-01",
        balanceAmount: 100,
        netInvestmentAmount: 80,
        gainAmount: 20,
        fx_impact: 6
      })
    ).toBe(14);
  });

  test("computeNetworthBounds tracks balance, invested, zero, and ex-FX values", () => {
    const withoutFx = computeNetworthBounds(sample, false);
    expect(withoutFx.min).toBe(0);
    expect(withoutFx.max).toBeCloseTo(120000 * 1.06, 2);

    const withFx = computeNetworthBounds(
      [
        {
          date: "2026-01-01",
          balanceAmount: 100000,
          netInvestmentAmount: 90000,
          gainAmount: 10000,
          investment_return: 30000,
          fx_impact: -20000
        }
      ],
      true
    );
    expect(withFx.max).toBeCloseTo(120000 * 1.06, 2);
  });

  test("buildLinePath and buildBandPath produce valid SVG paths", () => {
    expect(buildLinePath([], [])).toBe("");
    expect(buildLinePath([0, 50], [100, 40])).toBe("M0.0,100.0 L50.0,40.0");

    expect(buildBandPath([], [], [])).toBe("");
    expect(buildBandPath([0, 50], [100, 100], [60, 40])).toBe(
      "M0.0,60.0 L50.0,40.0 L50.0,100.0 L0.0,100.0 Z"
    );
  });
});
