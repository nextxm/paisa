import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import {
  buildHeatmapWeeks,
  buildHeatmapWeeksOfMonth,
  buildSeasonalityPattern,
  buildWeekdayPattern,
  getDailyExpenseAmount
} from "./expense_heatmap";

describe("expense heatmap utils", () => {
  const days = [
    {
      date: dayjs("2025-01-01T00:00:00Z"),
      total: 100,
      by_category: { Groceries: 60, Dining: 40 }
    },
    {
      date: dayjs("2025-01-03T00:00:00Z"),
      total: 80,
      by_category: { Groceries: 80 }
    },
    {
      date: dayjs("2025-02-10T00:00:00Z"),
      total: 50,
      by_category: { Dining: 50 }
    },
    {
      date: dayjs("2026-01-15T00:00:00Z"),
      total: 200,
      by_category: { Groceries: 200 }
    }
  ];

  test("getDailyExpenseAmount respects the category filter", () => {
    expect(getDailyExpenseAmount(days[0] as any)).toBe(100);
    expect(getDailyExpenseAmount(days[0] as any, "Groceries")).toBe(60);
    expect(getDailyExpenseAmount(days[0] as any, "Travel")).toBe(0);
  });

  test("buildHeatmapWeeks pads partial weeks for arbitrary date ranges", () => {
    const weeks = buildHeatmapWeeks(
      days as any,
      dayjs("2025-01-01T00:00:00Z"),
      dayjs("2025-01-10T00:00:00Z"),
      "Groceries"
    );

    expect(weeks).toHaveLength(2);
    expect(weeks[0]).toHaveLength(7);
    expect(weeks[0][0].date.format("YYYY-MM-DD")).toBe("2024-12-30");
    expect(weeks[0][2].total).toBe(60);
    expect(weeks[0][4].total).toBe(80);
    expect(weeks[1][4].date.format("YYYY-MM-DD")).toBe("2025-01-10");
    expect(weeks[1][4].inRange).toBe(true);
    expect(weeks[1][5].inRange).toBe(false);
  });

  test("buildWeekdayPattern averages across all weekdays in range, including zero-spend days", () => {
    const pattern = buildWeekdayPattern(
      days as any,
      dayjs("2025-01-01T00:00:00Z"),
      dayjs("2025-01-14T00:00:00Z"),
      "Dining"
    );

    expect(pattern.map((point) => point.label)).toEqual([
      "Mon",
      "Tue",
      "Wed",
      "Thu",
      "Fri",
      "Sat",
      "Sun"
    ]);
    expect(pattern.find((point) => point.label === "Wed")?.value).toBe(20);
    expect(pattern.find((point) => point.label === "Fri")?.value).toBe(0);
  });

  test("buildSeasonalityPattern averages monthly totals by month-of-year", () => {
    const pattern = buildSeasonalityPattern(
      days as any,
      dayjs("2025-01-01T00:00:00Z"),
      dayjs("2026-01-31T00:00:00Z"),
      "Groceries"
    );

    expect(pattern[0].label).toBe("Jan");
    expect(pattern[0].count).toBe(2);
    expect(pattern[0].total).toBe(340);
    expect(pattern[0].value).toBe(170);
    expect(pattern[1].count).toBe(1);
    expect(pattern[1].value).toBe(0);
  });

  test("buildHeatmapWeeksOfMonth groups daily expenses by month and week of month", () => {
    const weeksOfMonth = buildHeatmapWeeksOfMonth(
      days as any,
      dayjs("2025-01-01T00:00:00Z"),
      dayjs("2025-02-28T00:00:00Z"),
      "Groceries"
    );

    // 2 months in range: Jan and Feb
    expect(weeksOfMonth).toHaveLength(2);

    // Month 1: Jan 2025
    const jan = weeksOfMonth[0];
    expect(jan).toHaveLength(5);
    // Week 1: 1st-7th. Date: Jan 3 has 80, Jan 1 has 60. Total: 140
    expect(jan[0].total).toBe(140);
    expect(jan[0].inRange).toBe(true);
    expect(jan[0].tooltipRange).toBe("Jan 1 - 7, 2025");

    // Week 2: 8th-14th. Total: 0
    expect(jan[1].total).toBe(0);
    expect(jan[1].inRange).toBe(true);

    // Month 2: Feb 2025 (non-leap, 28 days)
    const feb = weeksOfMonth[1];
    expect(feb).toHaveLength(5);
    // Week 5 should be empty and out of range because startDay (29) > endDay (28)
    expect(feb[4].inRange).toBe(false);
    expect(feb[4].total).toBe(0);
  });
});
