import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import {
  buildCategoryYoYSeries,
  buildYoYDashboardSummary,
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

  test("buildYoYDashboardSummary computes net, savings and category movers", () => {
    const summary = buildYoYDashboardSummary(
      {
        "2024": { month: { "2024-01": 80 }, total: 800 },
        "2025": { month: { "2025-01": 120, "2025-02": 90 }, total: 900 }
      },
      {
        "2024": { month: { "2024-01": 200 }, total: 1200 },
        "2025": { month: { "2025-01": 250, "2025-02": 300 }, total: 1500 }
      },
      {
        Groceries: {
          "2024": { month: {}, total: 300 },
          "2025": { month: {}, total: 500 }
        },
        Travel: {
          "2024": { month: {}, total: 100 },
          "2025": { month: {}, total: 250 }
        }
      }
    );

    expect(summary.latestYear).toBe("2025");
    expect(summary.previousYear).toBe("2024");
    expect(summary.latestNetTotal).toBe(600);
    expect(summary.previousNetTotal).toBe(400);
    expect(summary.netChangePct).toBe(50);
    expect(summary.savingsRatePct).toBe(40);
    expect(summary.monthlyNet[0].net).toBe(130);
    expect(summary.highestExpenseMonth?.month).toBe("Jan");
    expect(summary.topCategoryMovers[0].name).toBe("Groceries");
    expect(summary.topCategoryMovers[0].delta).toBe(200);
  });
});
