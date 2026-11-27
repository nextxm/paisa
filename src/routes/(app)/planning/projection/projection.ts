import dayjs from "dayjs";
import type { FinancialProfile } from "$lib/utils";

export type ProjectionTableRow = {
  year: number;
  startingCorpus: number;
  annualContribution: number;
  returnAmount: number;
  endingCorpus: number;
  expenses: number;
  fireTarget: number;
  isFireReached: boolean;
};

export type ProjectionScenarioKey = "expected" | "conservative" | "optimistic";

export type ProjectionScenarioTable = {
  key: ProjectionScenarioKey;
  label: string;
  cagr: number;
  rows: ProjectionTableRow[];
};

export function parseVal(val: number | null | undefined, fallback: number): number {
  if (val === null || val === undefined || isNaN(Number(val))) {
    return fallback;
  }
  return Number(val);
}

/**
 * Computes real (inflation-adjusted) CAGR from nominal CAGR and inflation rate.
 */
export function toRealCagr(nominalCagrPercent: number, inflationPercent: number): number {
  const nominal = nominalCagrPercent / 100;
  const inflation = inflationPercent / 100;
  if (1 + inflation <= 0) return nominalCagrPercent;
  return ((1 + nominal) / (1 + inflation) - 1) * 100;
}

export function projectNetworth(
  startDate: dayjs.Dayjs,
  currentNetworth: number,
  monthlyContribution: number,
  cagrPercent: number,
  months: number
) {
  if (months <= 0) return [];
  const cagr = cagrPercent / 100;
  const monthlyRate = Math.pow(1 + cagr, 1 / 12) - 1;
  const points = [];
  let current = currentNetworth;
  for (let i = 1; i <= months; i++) {
    current = current * (1 + monthlyRate) + monthlyContribution;
    points.push({
      date: startDate.add(i, "month"),
      balanceAmount: Math.round(current * 100) / 100
    });
  }
  return points;
}

/**
 * Finds the first month where projected balance reaches the FIRE target.
 * When inflationPercent > 0 (nominal mode), the target corpus grows with inflation over time.
 */
export function firstFireCrossingDate(
  points: { date: dayjs.Dayjs; balanceAmount: number }[],
  baseTargetCorpus: number,
  inflationPercent = 0
): { date: dayjs.Dayjs; targetAtCrossing: number } | null {
  if (baseTargetCorpus <= 0) return null;
  for (let i = 0; i < points.length; i++) {
    const yearsElapsed = (i + 1) / 12;
    const inflatedTarget =
      inflationPercent > 0
        ? baseTargetCorpus * Math.pow(1 + inflationPercent / 100, yearsElapsed)
        : baseTargetCorpus;
    if (points[i].balanceAmount >= inflatedTarget) {
      return {
        date: points[i].date,
        targetAtCrossing: Math.round(inflatedTarget * 100) / 100
      };
    }
  }
  return null;
}

export function monthDiff(from: dayjs.Dayjs, to: dayjs.Dayjs): number {
  const months = (to.year() - from.year()) * 12 + (to.month() - from.month());
  return months < 0 ? 0 : months;
}

export function buildProjectionRows(
  startingCorpus: number,
  annualExpenses: number,
  inflationPercent: number,
  cagrPercent: number,
  monthlyContribution: number,
  years: number,
  swrPercent = 4
): ProjectionTableRow[] {
  const rows: ProjectionTableRow[] = [];
  let currentCorpus = startingCorpus;
  const annualContribution = monthlyContribution * 12;
  const cagr = cagrPercent / 100;
  const monthlyRate = Math.pow(1 + cagr, 1 / 12) - 1;

  for (let year = 1; year <= years; year++) {
    const inflatedExpenses =
      annualExpenses * Math.pow(1 + inflationPercent / 100, Math.max(year - 1, 0));
    const yearStartCorpus = currentCorpus;
    let yearlyReturn = 0;

    for (let m = 0; m < 12; m++) {
      const monthReturn = currentCorpus * monthlyRate;
      yearlyReturn += monthReturn;
      currentCorpus = currentCorpus + monthReturn + monthlyContribution;
    }

    const endingCorpus = currentCorpus;
    const fireTarget = swrPercent > 0 ? inflatedExpenses / (swrPercent / 100) : 0;

    rows.push({
      year,
      startingCorpus: yearStartCorpus,
      annualContribution,
      returnAmount: yearlyReturn,
      endingCorpus,
      expenses: inflatedExpenses,
      fireTarget,
      isFireReached: fireTarget > 0 && endingCorpus >= fireTarget
    });
  }

  return rows;
}

export function calibrateFromDna(
  dnaProfile: FinancialProfile | null,
  current: {
    monthlyContribution: number;
    expectedCagr: number;
    cagrSpread: number;
    inflationRate: number;
  }
) {
  let { monthlyContribution, expectedCagr, cagrSpread, inflationRate } = { ...current };

  if (!dnaProfile) {
    return { monthlyContribution, expectedCagr, cagrSpread, inflationRate };
  }

  if (dnaProfile.monthly_contribution !== undefined && dnaProfile.monthly_contribution !== null) {
    monthlyContribution = Math.min(
      1000000,
      Math.max(-100000, Math.round(dnaProfile.monthly_contribution))
    );
  }

  if (dnaProfile.price_months_covered >= 3 && dnaProfile.historical_return > 0) {
    const returnVal = Math.round(dnaProfile.historical_return * 10) / 10;
    expectedCagr = Math.min(35, Math.max(-5, returnVal));
    const rawSpread =
      dnaProfile.return_volatility > 0 ? Math.round(dnaProfile.return_volatility * 10) / 10 : 3;
    cagrSpread = Math.min(15, Math.max(0.5, rawSpread));
  }

  if (dnaProfile.expense_years_covered >= 2 && dnaProfile.expense_growth_rate > 0) {
    const expenseRate = Math.round(dnaProfile.expense_growth_rate * 10) / 10;
    inflationRate = Math.min(15, Math.max(0, expenseRate));
  }

  return { monthlyContribution, expectedCagr, cagrSpread, inflationRate };
}

export function generateScenarioCSV(scenario: ProjectionScenarioTable): string {
  const headers = [
    "Year",
    "Starting Corpus",
    "Annual Contribution",
    "Return",
    "Ending Corpus",
    "Annual Expenses"
  ];
  const round = (val: number) => Math.round((val || 0) * 100) / 100;
  return [
    headers.join(","),
    ...scenario.rows.map((row) =>
      [
        row.year,
        round(row.startingCorpus),
        round(row.annualContribution),
        round(row.returnAmount),
        round(row.endingCorpus),
        round(row.expenses)
      ].join(",")
    )
  ].join("\n");
}

export function getScenarioCSVFilename(label: string): string {
  const slug = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `paisa-projection-${slug || "scenario"}.csv`;
}
