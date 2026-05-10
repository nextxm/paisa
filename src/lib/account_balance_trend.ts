import dayjs from "dayjs";
import type { Networth } from "./utils";

export type TrendMarker = { x: number; y: number };

export function trendRangeFromMonths(
  asOfDate: string,
  months: number
): { start: string; end: string } {
  const end = dayjs(asOfDate).startOf("day");
  const start = end.subtract(months, "month").startOf("day");
  return { start: start.format("YYYY-MM-DD"), end: end.format("YYYY-MM-DD") };
}

export function filterTrendPoints(points: Networth[], start: string, end: string): Networth[] {
  const startDate = dayjs(start).startOf("day");
  const endDate = dayjs(end).endOf("day");
  if (!startDate.isValid() || !endDate.isValid() || endDate.isBefore(startDate)) {
    return [];
  }
  return points.filter((point) => !point.date.isBefore(startDate) && !point.date.isAfter(endDate));
}

export function buildTrendPath(
  points: Networth[],
  width: number,
  height: number,
  padding = 20
): { path: string; marker: TrendMarker | null } {
  if (points.length === 0) {
    return { path: "", marker: null };
  }

  const start = points[0].date.startOf("day");
  const end = points[points.length - 1].date.endOf("day");
  const spanDays = Math.max(end.diff(start, "day"), 1);

  const balances = points.map((point) => point.balanceAmount);
  const minBalance = Math.min(...balances);
  const maxBalance = Math.max(...balances);
  const balanceSpan = maxBalance - minBalance || 1;

  const x = (date: dayjs.Dayjs) =>
    padding + (date.diff(start, "day") / spanDays) * (width - padding * 2);
  const y = (balance: number) =>
    height - padding - ((balance - minBalance) / balanceSpan) * (height - padding * 2);

  const segments = points.map(
    (point, index) => `${index === 0 ? "M" : "L"} ${x(point.date)} ${y(point.balanceAmount)}`
  );
  const last = points[points.length - 1];
  return { path: segments.join(" "), marker: { x: x(last.date), y: y(last.balanceAmount) } };
}
