import dayjs from "dayjs";
import _ from "lodash";
import type { Posting } from "$lib/utils";

export type MoMDimension = "category" | "payee" | "account";

export interface MoMMonthlyPoint {
  month: string;
  label?: string;
  currency?: string;
  total: number;
  change: number | null;
  changePct: number | null;
  movingAverage3: number;
}

export interface MoMEntityTrend {
  key: string;
  currency?: string;
  series: Record<string, number>;
  current: number;
  previous: number;
  change: number;
  changePct: number | null;
  shareOfCurrent: number;
}

export interface MoMInsightSummary {
  latestMonth: string;
  latestTotal: number;
  previousTotal: number;
  averageMonthlySpend: number;
  volatilityPct: number | null;
  highestMonth: { month: string; total: number } | null;
  lowestMonth: { month: string; total: number } | null;
  largestIncrease: MoMEntityTrend | null;
  largestDecrease: MoMEntityTrend | null;
}

function isExpense(posting: Posting) {
  return posting.account.startsWith("Expenses:") && !posting.account.startsWith("Expenses:Tax");
}

function categoryName(account: string) {
  const segments = account.split(":");
  return segments[1] || account;
}

function dimensionValue(posting: Posting, dimension: MoMDimension) {
  if (dimension === "account") {
    return posting.account;
  }

  if (dimension === "payee") {
    const payee = posting.payee?.trim();
    return payee && payee.length > 0 ? payee : "(No payee)";
  }

  return categoryName(posting.account);
}

export function monthRange(endMonth: string, count: number) {
  if (!endMonth || count <= 0) {
    return [];
  }

  const end = dayjs(`${endMonth}-01`);
  return _.range(count)
    .map((offset) => end.add(-count + offset + 1, "month").format("YYYY-MM"))
    .sort();
}

export function availableExpenseMonths(postings: Posting[]) {
  return _.chain(postings)
    .filter(isExpense)
    .map((posting) => posting.date.format("YYYY-MM"))
    .uniq()
    .sort()
    .value();
}

function buildMonthlyPointsFromTotals(months: string[], totalsByMonth: Record<string, number>) {
  return months.map((month, index) => {
    const total = totalsByMonth[month] || 0;
    const previous = index > 0 ? totalsByMonth[months[index - 1]] || 0 : 0;
    const change = index > 0 ? total - previous : null;
    const changePct = index > 0 && previous !== 0 ? change! / previous : null;

    const windowStart = Math.max(0, index - 2);
    const rollingWindow = months.slice(windowStart, index + 1);
    const movingAverage3 =
      _.sumBy(rollingWindow, (key) => totalsByMonth[key] || 0) / rollingWindow.length;

    return {
      month,
      total,
      change,
      changePct,
      movingAverage3
    };
  });
}

export function buildMonthlyPoints(
  postings: Posting[],
  months: string[],
  currencyMode: "default" | "actual" = "default"
): MoMMonthlyPoint[] {
  if (currencyMode === "actual") {
    const monthSet = new Set(months);
    const totalsByCurrency: Record<string, Record<string, number>> = {};

    for (const posting of postings) {
      if (!isExpense(posting)) {
        continue;
      }

      const month = posting.date.format("YYYY-MM");
      if (!monthSet.has(month)) {
        continue;
      }

      const currency = posting.commodity || USER_CONFIG.default_currency;
      totalsByCurrency[currency] =
        totalsByCurrency[currency] || Object.fromEntries(months.map((m) => [m, 0]));
      totalsByCurrency[currency][month] += posting.quantity;
    }

    const currencies = Object.keys(totalsByCurrency).sort();
    if (currencies.length <= 1) {
      const currency = currencies[0] || USER_CONFIG.default_currency;
      return buildMonthlyPointsFromTotals(months, totalsByCurrency[currency] || {}).map(
        (point) => ({
          ...point,
          currency
        })
      );
    }

    return currencies
      .flatMap((currency) =>
        buildMonthlyPointsFromTotals(months, totalsByCurrency[currency]).map((point) => ({
          ...point,
          currency,
          label: `${dayjs(`${point.month}-01`).format("MMM YYYY")} (${currency})`
        }))
      )
      .sort((left, right) => {
        if (left.month === right.month) {
          return (left.currency || "").localeCompare(right.currency || "");
        }
        return left.month.localeCompare(right.month);
      });
  }

  const monthSet = new Set(months);
  const totalsByMonth: Record<string, number> = {};

  for (const month of months) {
    totalsByMonth[month] = 0;
  }

  for (const posting of postings) {
    if (!isExpense(posting)) {
      continue;
    }

    const month = posting.date.format("YYYY-MM");
    if (!monthSet.has(month)) {
      continue;
    }

    totalsByMonth[month] += posting.amount;
  }

  return buildMonthlyPointsFromTotals(months, totalsByMonth);
}

export function buildMonthlyPointsOriginal(
  postings: Posting[],
  months: string[],
  currency: string
): MoMMonthlyPoint[] {
  const monthSet = new Set(months);
  const totalsByMonth: Record<string, number> = {};

  for (const month of months) {
    totalsByMonth[month] = 0;
  }

  for (const posting of postings) {
    if (!isExpense(posting) || posting.commodity !== currency) {
      continue;
    }

    const month = posting.date.format("YYYY-MM");
    if (!monthSet.has(month)) {
      continue;
    }

    totalsByMonth[month] += posting.quantity;
  }

  return buildMonthlyPointsFromTotals(months, totalsByMonth);
}

export function buildDimensionTrends(
  postings: Posting[],
  months: string[],
  dimension: MoMDimension,
  topN = 8,
  currencyMode: "default" | "actual" = "default"
): MoMEntityTrend[] {
  const monthSet = new Set(months);
  const grouped: Record<string, Record<string, number>> = {};
  const currenciesByKey: Record<string, string> = {};

  for (const posting of postings) {
    if (!isExpense(posting)) {
      continue;
    }

    const month = posting.date.format("YYYY-MM");
    if (!monthSet.has(month)) {
      continue;
    }

    let key = dimensionValue(posting, dimension);
    let value = posting.amount;
    if (currencyMode === "actual") {
      const currency = posting.commodity || USER_CONFIG.default_currency;
      key = `${key} (${currency})`;
      value = posting.quantity;
      currenciesByKey[key] = currency;
    }

    grouped[key] = grouped[key] || {};
    grouped[key][month] = (grouped[key][month] || 0) + value;
  }

  return buildDimensionTrendsFromGrouped(grouped, months, topN, currenciesByKey);
}

export function buildDimensionTrendsOriginal(
  postings: Posting[],
  months: string[],
  dimension: MoMDimension,
  topN: number,
  currency: string
): MoMEntityTrend[] {
  const monthSet = new Set(months);
  const grouped: Record<string, Record<string, number>> = {};

  for (const posting of postings) {
    if (!isExpense(posting) || posting.commodity !== currency) {
      continue;
    }

    const month = posting.date.format("YYYY-MM");
    if (!monthSet.has(month)) {
      continue;
    }

    const key = dimensionValue(posting, dimension);
    grouped[key] = grouped[key] || {};
    grouped[key][month] = (grouped[key][month] || 0) + posting.quantity;
  }

  return buildDimensionTrendsFromGrouped(grouped, months, topN);
}

function buildDimensionTrendsFromGrouped(
  grouped: Record<string, Record<string, number>>,
  months: string[],
  topN: number,
  currenciesByKey: Record<string, string> = {}
): MoMEntityTrend[] {
  if (months.length === 0) {
    return [];
  }

  const currentMonth = months[months.length - 1];
  const previousMonth = months.length > 1 ? months[months.length - 2] : months[months.length - 1];
  const totalCurrent = _.sumBy(Object.values(grouped), (series) => series[currentMonth] || 0);

  return _.chain(grouped)
    .map((series, key) => {
      const normalizedSeries: Record<string, number> = {};
      for (const month of months) {
        normalizedSeries[month] = series[month] || 0;
      }

      const current = normalizedSeries[currentMonth] || 0;
      const previous = normalizedSeries[previousMonth] || 0;
      const change = current - previous;
      const changePct = previous !== 0 ? change / previous : null;

      return {
        key,
        currency: currenciesByKey[key],
        series: normalizedSeries,
        current,
        previous,
        change,
        changePct,
        shareOfCurrent: totalCurrent > 0 ? current / totalCurrent : 0
      };
    })
    .filter((trend) => trend.current > 0 || trend.previous > 0)
    .orderBy([(trend) => trend.current, (trend) => Math.abs(trend.change)], ["desc", "desc"])
    .take(topN)
    .value();
}

export function calculateMoMInsights(
  monthlyPoints: MoMMonthlyPoint[],
  trends: MoMEntityTrend[]
): MoMInsightSummary {
  if (monthlyPoints.length === 0) {
    return {
      latestMonth: "",
      latestTotal: 0,
      previousTotal: 0,
      averageMonthlySpend: 0,
      volatilityPct: null,
      highestMonth: null,
      lowestMonth: null,
      largestIncrease: null,
      largestDecrease: null
    };
  }

  const latest = monthlyPoints[monthlyPoints.length - 1];
  const previous = monthlyPoints.length > 1 ? monthlyPoints[monthlyPoints.length - 2] : latest;

  const averageMonthlySpend = _.sumBy(monthlyPoints, (point) => point.total) / monthlyPoints.length;

  const variance =
    _.sumBy(monthlyPoints, (point) => Math.pow(point.total - averageMonthlySpend, 2)) /
    monthlyPoints.length;
  const stdDev = Math.sqrt(variance);
  const volatilityPct = averageMonthlySpend > 0 ? stdDev / averageMonthlySpend : null;

  const highestMonth = _.maxBy(monthlyPoints, (point) => point.total) || null;
  const lowestMonth = _.minBy(monthlyPoints, (point) => point.total) || null;
  const largestIncrease = _.maxBy(trends, (trend) => trend.change) || null;
  const largestDecrease = _.minBy(trends, (trend) => trend.change) || null;

  return {
    latestMonth: latest.month,
    latestTotal: latest.total,
    previousTotal: previous.total,
    averageMonthlySpend,
    volatilityPct,
    highestMonth: highestMonth ? { month: highestMonth.month, total: highestMonth.total } : null,
    lowestMonth: lowestMonth ? { month: lowestMonth.month, total: lowestMonth.total } : null,
    largestIncrease,
    largestDecrease
  };
}
