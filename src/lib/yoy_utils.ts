import dayjs from "dayjs";
import _ from "lodash";
import type { Posting, YoYSeries } from "$lib/utils";

export type YoYChartType = "line" | "bar";

export interface YoYInsightSummary {
  spendingChangePct: number | null;
  incomeChangePct: number | null;
  topExpenseCategory: {
    name: string;
    latestYearTotal: number;
    changePct: number | null;
  } | null;
}

function expenseCategory(account: string) {
  const parts = account.split(":");
  return parts.length >= 2 ? parts[1] : account;
}

export function orderedYears(series: Record<string, YoYSeries>): string[] {
  return _.sortBy(Object.keys(series), (y) => Number(y));
}

export function buildMonthlyComparisonPoints(series: Record<string, YoYSeries>) {
  const years = orderedYears(series);

  return _.range(0, 12).map((idx) => {
    const month = idx + 1;
    const point: Record<string, number | string> = {
      month: dayjs(`2000-${String(month).padStart(2, "0")}-01`).format("MMM")
    };

    for (const year of years) {
      point[year] = series[year]?.month?.[`${year}-${String(month).padStart(2, "0")}`] || 0;
    }

    return point;
  });
}

export function buildCategoryYoYSeries(
  postings: Posting[],
  years: string[]
): Record<string, Record<string, YoYSeries>> {
  const yearSet = new Set(years);
  const result: Record<string, Record<string, YoYSeries>> = {};

  for (const year of years) {
    for (let month = 1; month <= 12; month++) {
      const key = `${year}-${String(month).padStart(2, "0")}`;
      for (const category of Object.keys(result)) {
        result[category][year].month[key] = result[category][year].month[key] || 0;
      }
    }
  }

  for (const posting of postings) {
    if (!posting.account.startsWith("Expenses:")) {
      continue;
    }
    const year = posting.date.year().toString();
    if (!yearSet.has(year)) {
      continue;
    }

    const category = expenseCategory(posting.account);
    const monthKey = posting.date.format("YYYY-MM");
    if (!result[category]) {
      result[category] = {};
    }

    if (!result[category][year]) {
      const month: Record<string, number> = {};
      for (let i = 1; i <= 12; i++) {
        month[`${year}-${String(i).padStart(2, "0")}`] = 0;
      }
      result[category][year] = { month, total: 0 };
    }

    result[category][year].month[monthKey] += posting.amount;
    result[category][year].total += posting.amount;
  }

  return result;
}

export function calculateYoYInsights(
  expenseSeries: Record<string, YoYSeries>,
  incomeSeries: Record<string, YoYSeries>,
  categorySeries: Record<string, Record<string, YoYSeries>>
): YoYInsightSummary {
  const years = orderedYears(expenseSeries);
  if (years.length < 2) {
    return { spendingChangePct: null, incomeChangePct: null, topExpenseCategory: null };
  }

  const previousYear = years[years.length - 2];
  const currentYear = years[years.length - 1];

  const expensePrevious = expenseSeries[previousYear]?.total || 0;
  const expenseCurrent = expenseSeries[currentYear]?.total || 0;
  const incomePrevious = incomeSeries[previousYear]?.total || 0;
  const incomeCurrent = incomeSeries[currentYear]?.total || 0;

  const spendingChangePct =
    expensePrevious === 0 ? null : ((expenseCurrent - expensePrevious) / expensePrevious) * 100;
  const incomeChangePct =
    incomePrevious === 0 ? null : ((incomeCurrent - incomePrevious) / incomePrevious) * 100;

  const topCategory = _.maxBy(Object.keys(categorySeries), (category) => {
    return categorySeries[category]?.[currentYear]?.total || 0;
  });

  if (!topCategory) {
    return { spendingChangePct, incomeChangePct, topExpenseCategory: null };
  }

  const topCurrent = categorySeries[topCategory]?.[currentYear]?.total || 0;
  const topPrevious = categorySeries[topCategory]?.[previousYear]?.total || 0;
  const topChangePct = topPrevious === 0 ? null : ((topCurrent - topPrevious) / topPrevious) * 100;

  return {
    spendingChangePct,
    incomeChangePct,
    topExpenseCategory: {
      name: topCategory,
      latestYearTotal: topCurrent,
      changePct: topChangePct
    }
  };
}

export function buildYoYExportRows(
  expenseSeries: Record<string, YoYSeries>,
  incomeSeries: Record<string, YoYSeries>
) {
  const years = _.uniq([...orderedYears(expenseSeries), ...orderedYears(incomeSeries)]);
  const rows: Record<string, string | number>[] = [];

  for (let month = 1; month <= 12; month++) {
    const monthLabel = dayjs(`2000-${String(month).padStart(2, "0")}-01`).format("MMM");
    const row: Record<string, string | number> = { Month: monthLabel };

    for (const year of years) {
      const key = `${year}-${String(month).padStart(2, "0")}`;
      row[`Expense ${year}`] = expenseSeries[year]?.month?.[key] || 0;
      row[`Income ${year}`] = incomeSeries[year]?.month?.[key] || 0;
    }

    rows.push(row);
  }

  return rows;
}
