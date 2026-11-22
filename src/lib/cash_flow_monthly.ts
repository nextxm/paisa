import _ from "lodash";
import type { Posting } from "$lib/utils";
import type { MonthlyCashflowGroupLevel } from "../persisted_store";

export interface AggregatedItem {
  account: string;
  commodity: string;
  amount: number;
}

/**
 * Truncates an account path to the requested depth below its top-level category
 * (e.g. "Expenses:Shopping:Amazon" -> "Expenses:Shopping" at level 1).
 */
export function groupAccountByLevel(account: string, level: MonthlyCashflowGroupLevel): string {
  if (level === 1 || level === 2) {
    return account
      .split(":")
      .slice(0, level + 1)
      .join(":");
  }
  return account;
}

/**
 * Aggregates postings by grouped account name and commodity, sorted by account.
 */
export function aggregatePostingsByLevel(
  postings: Posting[],
  negate = false,
  level: MonthlyCashflowGroupLevel = "all"
): AggregatedItem[] {
  const result: Record<string, Record<string, number>> = {};
  for (const p of postings) {
    const account = groupAccountByLevel(p.account, level);
    if (!result[account]) result[account] = {};
    if (!result[account][p.commodity]) result[account][p.commodity] = 0;
    result[account][p.commodity] += negate ? -p.quantity : p.quantity;
  }

  const flattened: AggregatedItem[] = [];
  for (const account in result) {
    for (const commodity in result[account]) {
      flattened.push({ account, commodity, amount: result[account][commodity] });
    }
  }
  return _.sortBy(flattened, "account");
}
