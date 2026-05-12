import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import {
  availableExpenseMonths,
  buildDimensionTrends,
  buildMonthlyPoints,
  calculateMoMInsights,
  monthRange
} from "./mom_utils";

describe("mom utils", () => {
  test("monthRange returns ascending range ending at selected month", () => {
    expect(monthRange("2026-03", 4)).toEqual(["2025-12", "2026-01", "2026-02", "2026-03"]);
  });

  test("buildMonthlyPoints computes totals, deltas, and moving average", () => {
    const points = buildMonthlyPoints(
      [
        { account: "Expenses:Food", amount: 100, date: dayjs("2026-01-12") } as any,
        { account: "Expenses:Food", amount: 50, date: dayjs("2026-02-12") } as any,
        { account: "Expenses:Rent", amount: 300, date: dayjs("2026-02-01") } as any
      ],
      ["2026-01", "2026-02", "2026-03"]
    );

    expect(points.map((point) => point.total)).toEqual([100, 350, 0]);
    expect(points[1].change).toBe(250);
    expect(points[1].changePct).toBe(2.5);
    expect(points[2].movingAverage3).toBe(150);
  });

  test("buildDimensionTrends aggregates per category with shares", () => {
    const trends = buildDimensionTrends(
      [
        {
          account: "Expenses:Food:Groceries",
          payee: "Store A",
          amount: 120,
          date: dayjs("2026-02-02")
        } as any,
        {
          account: "Expenses:Food:Dining",
          payee: "Store B",
          amount: 180,
          date: dayjs("2026-03-10")
        } as any,
        {
          account: "Expenses:Rent",
          payee: "Landlord",
          amount: 700,
          date: dayjs("2026-03-01")
        } as any
      ],
      ["2026-02", "2026-03"],
      "category",
      10
    );

    const rent = trends.find((trend) => trend.key === "Rent");
    const food = trends.find((trend) => trend.key === "Food");

    expect(rent?.current).toBe(700);
    expect(food?.previous).toBe(120);
    expect(food?.change).toBe(60);
    expect(rent?.shareOfCurrent).toBeCloseTo(700 / 880, 4);
  });

  test("calculateMoMInsights returns month extremes and movers", () => {
    const points = buildMonthlyPoints(
      [
        { account: "Expenses:Food", amount: 100, date: dayjs("2026-01-01") } as any,
        { account: "Expenses:Food", amount: 300, date: dayjs("2026-02-01") } as any,
        { account: "Expenses:Food", amount: 200, date: dayjs("2026-03-01") } as any
      ],
      ["2026-01", "2026-02", "2026-03"]
    );

    const trends = [
      {
        key: "Rent",
        current: 500,
        previous: 300,
        change: 200,
        changePct: 200 / 300,
        shareOfCurrent: 0.5,
        series: {}
      },
      {
        key: "Food",
        current: 100,
        previous: 250,
        change: -150,
        changePct: -150 / 250,
        shareOfCurrent: 0.1,
        series: {}
      }
    ];

    const insights = calculateMoMInsights(points, trends);

    expect(insights.latestMonth).toBe("2026-03");
    expect(insights.highestMonth?.month).toBe("2026-02");
    expect(insights.lowestMonth?.month).toBe("2026-01");
    expect(insights.largestIncrease?.key).toBe("Rent");
    expect(insights.largestDecrease?.key).toBe("Food");
  });

  test("availableExpenseMonths returns sorted unique expense months", () => {
    const months = availableExpenseMonths([
      { account: "Expenses:Food", amount: 30, date: dayjs("2026-02-01") } as any,
      { account: "Assets:Checking", amount: 30, date: dayjs("2026-01-01") } as any,
      { account: "Expenses:Food", amount: 10, date: dayjs("2026-02-20") } as any,
      { account: "Expenses:Rent", amount: 100, date: dayjs("2026-01-20") } as any
    ]);

    expect(months).toEqual(["2026-01", "2026-02"]);
  });
});
