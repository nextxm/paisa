import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import {
  buildCategoryYoYSeries,
  buildMonthlyComparisonPoints,
  calculateYoYInsights
} from "./yoy_utils";

describe("yoy utils", () => {
  test("buildMonthlyComparisonPoints aligns months across years", () => {
    const points = buildMonthlyComparisonPoints({
      "2024": { month: { "2024-01": 50 }, total: 50 },
      "2025": { month: { "2025-01": 100, "2025-02": 20 }, total: 120 }
    });

    expect(points).toHaveLength(12);
    expect(points[0].month).toBe("Jan");
    expect(points[0]["2024"]).toBe(50);
    expect(points[0]["2025"]).toBe(100);
    expect(points[1]["2025"]).toBe(20);
    expect(points[1]["2024"]).toBe(0);
  });

  test("calculateYoYInsights computes spending, income and top category trends", () => {
    const insights = calculateYoYInsights(
      {
        "2024": { month: {}, total: 100 },
        "2025": { month: {}, total: 120 }
      },
      {
        "2024": { month: {}, total: 200 },
        "2025": { month: {}, total: 260 }
      },
      {
        Groceries: {
          "2024": { month: {}, total: 30 },
          "2025": { month: {}, total: 60 }
        }
      }
    );

    expect(insights.spendingChangePct).toBe(20);
    expect(insights.incomeChangePct).toBe(30);
    expect(insights.topExpenseCategory?.name).toBe("Groceries");
    expect(insights.topExpenseCategory?.changePct).toBe(100);
  });

  test("buildCategoryYoYSeries groups postings by category and year", () => {
    const grouped = buildCategoryYoYSeries(
      [
        {
          date: dayjs("2025-01-01"),
          account: "Expenses:Groceries:Store",
          amount: 50
        } as any,
        {
          date: dayjs("2024-02-01"),
          account: "Expenses:Dining",
          amount: 30
        } as any
      ],
      ["2024", "2025"]
    );

    expect(grouped.Groceries["2025"].month["2025-01"]).toBe(50);
    expect(grouped.Dining["2024"].month["2024-02"]).toBe(30);
  });
});
