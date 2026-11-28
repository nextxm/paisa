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

  test("computeNetworthBounds scales to the windowed data range instead of anchoring min at 0", () => {
    const withoutFx = computeNetworthBounds(sample, false);
    // sample spans 100000..120000 (span = 20000, 6% pad = 1200)
    expect(withoutFx.min).toBeCloseTo(100000 - 1200, 2);
    expect(withoutFx.max).toBeCloseTo(120000 + 1200, 2);

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
    // exFx = 120000, min = 90000, span = 30000, 6% pad = 1800
    expect(withFx.min).toBeCloseTo(90000 - 1800, 2);
    expect(withFx.max).toBeCloseTo(120000 + 1800, 2);

    // Non-negative series near 0 should clamp min at 0
    const nearZero = computeNetworthBounds(
      [
        { date: "2026-01-01", balanceAmount: 100, netInvestmentAmount: 0, gainAmount: 100 },
        { date: "2026-02-01", balanceAmount: 1000, netInvestmentAmount: 800, gainAmount: 200 }
      ],
      false
    );
    expect(nearZero.min).toBe(0);
    expect(nearZero.max).toBeCloseTo(1060, 2);
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
