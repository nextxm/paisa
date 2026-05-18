import dayjs from "dayjs";
import type { DailyExpenseDay } from "$lib/utils";

export interface ExpenseHeatmapCell {
  date: dayjs.Dayjs;
  total: number;
  inRange: boolean;
}

export interface ExpensePatternPoint {
  label: string;
  value: number;
  count: number;
  total: number;
}

const weekdayLabels = ["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"];
const monthLabels = [
  "Jan",
  "Feb",
  "Mar",
  "Apr",
  "May",
  "Jun",
  "Jul",
  "Aug",
  "Sep",
  "Oct",
  "Nov",
  "Dec"
];

export function getDailyExpenseAmount(day: DailyExpenseDay, category = "") {
  return category ? day.by_category?.[category] || 0 : day.total;
}

function weekdayIndex(date: dayjs.Dayjs) {
  return (date.day() + 6) % 7;
}

function buildDailyLookup(days: DailyExpenseDay[], category = "") {
  const lookup = new Map<string, number>();
  for (const day of days) {
    const key = day.date.format("YYYY-MM-DD");
    lookup.set(key, (lookup.get(key) || 0) + getDailyExpenseAmount(day, category));
  }
  return lookup;
}

export function buildHeatmapWeeks(
  days: DailyExpenseDay[],
  from: dayjs.Dayjs,
  to: dayjs.Dayjs,
  category = ""
) {
  const lookup = buildDailyLookup(days, category);
  const start = from.startOf("day");
  const end = to.startOf("day");
  const paddedStart = start.subtract(weekdayIndex(start), "day");
  const paddedEnd = end.add(6 - weekdayIndex(end), "day");

  const weeks: ExpenseHeatmapCell[][] = [];
  let week: ExpenseHeatmapCell[] = [];

  for (let cursor = paddedStart; !cursor.isAfter(paddedEnd, "day"); cursor = cursor.add(1, "day")) {
    week.push({
      date: cursor,
      total: lookup.get(cursor.format("YYYY-MM-DD")) || 0,
      inRange: !cursor.isBefore(start, "day") && !cursor.isAfter(end, "day")
    });

    if (week.length === 7) {
      weeks.push(week);
      week = [];
    }
  }

  return weeks;
}

export function buildWeekdayPattern(
  days: DailyExpenseDay[],
  from: dayjs.Dayjs,
  to: dayjs.Dayjs,
  category = ""
) {
  const lookup = buildDailyLookup(days, category);
  const totals = Array.from({ length: 7 }, () => 0);
  const counts = Array.from({ length: 7 }, () => 0);

  for (
    let cursor = from.startOf("day");
    !cursor.isAfter(to, "day");
    cursor = cursor.add(1, "day")
  ) {
    const index = weekdayIndex(cursor);
    counts[index] += 1;
    totals[index] += lookup.get(cursor.format("YYYY-MM-DD")) || 0;
  }

  return weekdayLabels.map((label, index) => ({
    label,
    value: counts[index] ? totals[index] / counts[index] : 0,
    count: counts[index],
    total: totals[index]
  }));
}

export function buildSeasonalityPattern(
  days: DailyExpenseDay[],
  from: dayjs.Dayjs,
  to: dayjs.Dayjs,
  category = ""
) {
  const lookup = buildDailyLookup(days, category);
  const totals = Array.from({ length: 12 }, () => 0);
  const counts = Array.from({ length: 12 }, () => 0);

  for (
    let monthCursor = from.startOf("month");
    !monthCursor.isAfter(to, "month");
    monthCursor = monthCursor.add(1, "month")
  ) {
    const monthStart = monthCursor.isSame(from, "month")
      ? from.startOf("day")
      : monthCursor.startOf("month");
    const monthEnd = monthCursor.isSame(to, "month")
      ? to.startOf("day")
      : monthCursor.endOf("month");
    let monthlyTotal = 0;

    for (let cursor = monthStart; !cursor.isAfter(monthEnd, "day"); cursor = cursor.add(1, "day")) {
      monthlyTotal += lookup.get(cursor.format("YYYY-MM-DD")) || 0;
    }

    const monthIndex = monthCursor.month();
    totals[monthIndex] += monthlyTotal;
    counts[monthIndex] += 1;
  }

  return monthLabels.map((label, index) => ({
    label,
    value: counts[index] ? totals[index] / counts[index] : 0,
    count: counts[index],
    total: totals[index]
  }));
}
