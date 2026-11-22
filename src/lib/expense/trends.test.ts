import { describe, expect, test } from "bun:test";
import type { Posting } from "$lib/utils";
import {
  buildCategorySparklineData,
  computeCalendarMonthTrends,
  expenseTrendCategory,
  previousCalendarMonth
} from "./trends";

function makePosting(account: string, amount: number): Posting {
  return {
    account,
    amount
  } as Posting;
}

describe("expense trends helpers", () => {
  test("expenseTrendCategory extracts level-2 category with top-level fallback", () => {
    expect(expenseTrendCategory("Expenses:Groceries:Supermarket")).toBe("Groceries");
    expect(expenseTrendCategory("Expenses:Rent")).toBe("Rent");
    expect(expenseTrendCategory("Expenses")).toBe("Expenses");
    expect(expenseTrendCategory("Expenses:")).toBe("Expenses:");
  });

  test("previousCalendarMonth handles mid-year and year boundary", () => {
    expect(previousCalendarMonth("2026-11")).toBe("2026-10");
    expect(previousCalendarMonth("2026-01")).toBe("2025-12");
  });

  test("computeCalendarMonthTrends handles null/empty inputs", () => {
    expect(computeCalendarMonthTrends(null, "2026-11")).toEqual([]);
    expect(computeCalendarMonthTrends({}, "")).toEqual([]);
    expect(
      computeCalendarMonthTrends({ "2026-09": [makePosting("Expenses:Rent", 100)] }, "2026-11")
    ).toEqual([]);
  });

  test("computeCalendarMonthTrends rolls up subcategories, computes 2dp variance, and sorts alphabetically", () => {
    const grouped: Record<string, Posting[]> = {
      "2026-10": [
        makePosting("Expenses:Groceries:Vegetables", 200),
        makePosting("Expenses:Groceries:Bakery", 220),
        makePosting("Expenses:Utilities:Internet", 80)
      ],
      "2026-11": [
        makePosting("Expenses:Groceries:Supermarket", 450),
        makePosting("Expenses:Dining", 100)
      ]
    };

    const trends = computeCalendarMonthTrends(grouped, "2026-11");
    expect(trends).toEqual([
      {
        category: "Dining",
        current_month: 100,
        previous_month: 0,
        variance: 100,
        variance_pct: null
      },
      {
        category: "Groceries",
        current_month: 450,
        previous_month: 420,
        variance: 30,
        variance_pct: 7.14
      },
      {
        category: "Utilities",
        current_month: 0,
        previous_month: 80,
        variance: -80,
        variance_pct: -100
      }
    ]);
  });

  test("buildCategorySparklineData aggregates category totals and respects maxMonth", () => {
    const grouped: Record<string, Posting[]> = {
      "2026-08": [makePosting("Expenses:Food:Coffee", 50), makePosting("Expenses:Rent", 1000)],
      "2026-09": [makePosting("Expenses:Food:Lunch", 120), makePosting("Expenses:Food:Dinner", 80)],
      "2026-10": [makePosting("Expenses:Rent", 1000)],
      "2026-11": [makePosting("Expenses:Food:Coffee", 75)]
    };

    expect(buildCategorySparklineData(grouped, "Food")).toEqual({
      "2026-08": 50,
      "2026-09": 200,
      "2026-11": 75
    });

    expect(buildCategorySparklineData(grouped, "Food", "2026-09")).toEqual({
      "2026-08": 50,
      "2026-09": 200
    });
  });
});
