import dayjs from "dayjs";
import type { ExpenseTrend, Posting } from "$lib/utils";

export type ExpenseTrendMode = "rolling30" | "calendarMonth";

/**
 * Returns the level-2 category for an expense account, matching Go's
 * expenseCategory() helper (e.g., "Expenses:Groceries:Supermarket" -> "Groceries",
 * "Expenses" -> "Expenses").
 */
export function expenseTrendCategory(account: string): string {
  const parts = account.split(":");
  return parts.length >= 2 && parts[1] ? parts[1] : account;
}

/**
 * Returns the calendar month immediately preceding the given "YYYY-MM" string.
 */
export function previousCalendarMonth(selectedMonth: string): string {
  return dayjs(`${selectedMonth}-01`).subtract(1, "month").format("YYYY-MM");
}

/**
 * Computes category-level spending trends comparing the selected calendar month
 * against the immediately preceding calendar month using grouped month_wise postings.
 */
export function computeCalendarMonthTrends(
  groupedExpenses: Record<string, Posting[]> | null | undefined,
  selectedMonth: string
): ExpenseTrend[] {
  if (!groupedExpenses || !selectedMonth) {
    return [];
  }

  const prevMonth = previousCalendarMonth(selectedMonth);
  const currentPostings = groupedExpenses[selectedMonth] || [];
  const previousPostings = groupedExpenses[prevMonth] || [];

  if (currentPostings.length === 0 && previousPostings.length === 0) {
    return [];
  }

  const currentByCategory = new Map<string, number>();
  for (const p of currentPostings) {
    const cat = expenseTrendCategory(p.account);
    currentByCategory.set(cat, (currentByCategory.get(cat) ?? 0) + p.amount);
  }

  const previousByCategory = new Map<string, number>();
  for (const p of previousPostings) {
    const cat = expenseTrendCategory(p.account);
    previousByCategory.set(cat, (previousByCategory.get(cat) ?? 0) + p.amount);
  }

  const categories = Array.from(
    new Set<string>([...currentByCategory.keys(), ...previousByCategory.keys()])
  ).sort((a, b) => (a < b ? -1 : a > b ? 1 : 0));

  return categories.map((category) => {
    const current = currentByCategory.get(category) ?? 0;
    const previous = previousByCategory.get(category) ?? 0;
    const variance = current - previous;
    const variance_pct =
      previous !== 0 ? Math.round((variance / previous) * 100 * 100) / 100 : null;

    return {
      category,
      current_month: current,
      previous_month: previous,
      variance,
      variance_pct
    };
  });
}

/**
 * Builds a "YYYY-MM" -> total spend map for a single expense category, optionally
 * capped at maxMonth (inclusive) so sparklines anchor on the selected month.
 */
export function buildCategorySparklineData(
  groupedExpenses: Record<string, Posting[]> | null | undefined,
  category: string,
  maxMonth?: string
): Record<string, number> {
  if (!groupedExpenses) {
    return {};
  }

  const result: Record<string, number> = {};
  for (const [ym, postings] of Object.entries(groupedExpenses)) {
    if (maxMonth && ym > maxMonth) {
      continue;
    }

    let total = 0;
    for (const p of postings) {
      if (expenseTrendCategory(p.account) === category) {
        total += p.amount;
      }
    }

    if (total > 0) {
      result[ym] = total;
    }
  }
  return result;
}
