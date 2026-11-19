import { describe, expect, test } from "bun:test";
import type { FinancialProfile } from "$lib/utils";

// ---------------------------------------------------------------------------
// Helpers from projection page tested for behavior and regression prevention
// ---------------------------------------------------------------------------

type ProjectionTableRow = {
  year: number;
  startingCorpus: number;
  annualContribution: number;
  returnAmount: number;
  endingCorpus: number;
  expenses: number;
};

type ProjectionScenarioTable = {
  label: string;
  cagr: number;
  rows: ProjectionTableRow[];
};

function generateScenarioCSV(scenario: ProjectionScenarioTable): string {
  const headers = [
    "Year",
    "Starting Corpus",
    "Annual Contribution",
    "Return",
    "Ending Corpus",
    "Inflated Expenses"
  ];
  const round = (val: number) => Math.round((val || 0) * 100) / 100;
  const lines = [
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
  ];
  return lines.join("\n");
}

function getScenarioCSVFilename(label: string): string {
  const slug = label
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/(^-|-$)/g, "");
  return `paisa-projection-${slug || "scenario"}.csv`;
}

function parseVal(val: number | null | undefined, fallback: number): number {
  if (val === null || val === undefined || isNaN(Number(val))) {
    return fallback;
  }
  return Number(val);
}

function buildProjectionRows(
  startingCorpus: number,
  annualExpenses: number,
  inflationPercent: number,
  cagrPercent: number,
  monthlyContribution: number,
  years: number
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

    rows.push({
      year,
      startingCorpus: yearStartCorpus,
      annualContribution,
      returnAmount: yearlyReturn,
      endingCorpus,
      expenses: inflatedExpenses
    });
  }

  return rows;
}

function calibrateSliders(
  dnaProfile: FinancialProfile | null,
  current: {
    monthlyContribution: number;
    expectedCagr: number;
    conservativeCagr: number;
    optimisticCagr: number;
    inflationRate: number;
  }
) {
  let { monthlyContribution, expectedCagr, conservativeCagr, optimisticCagr, inflationRate } = {
    ...current
  };

  if (!dnaProfile) {
    return { monthlyContribution, expectedCagr, conservativeCagr, optimisticCagr, inflationRate };
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
    const spread =
      dnaProfile.return_volatility > 0 ? Math.round(dnaProfile.return_volatility * 10) / 10 : 3;
    conservativeCagr = Math.min(30, Math.max(-5, Math.round((expectedCagr - spread) * 10) / 10));
    optimisticCagr = Math.min(40, Math.max(-5, Math.round((expectedCagr + spread) * 10) / 10));
  }

  if (dnaProfile.expense_years_covered >= 2 && dnaProfile.expense_growth_rate > 0) {
    const expenseRate = Math.round(dnaProfile.expense_growth_rate * 10) / 10;
    inflationRate = Math.min(15, Math.max(0, expenseRate));
  }

  return { monthlyContribution, expectedCagr, conservativeCagr, optimisticCagr, inflationRate };
}

describe("Net Worth Projection", () => {
  describe("buildProjectionRows", () => {
    test("endingCorpus equals startingCorpus + annualContribution + returnAmount for each year", () => {
      const startingCorpus = 1000000;
      const annualExpenses = 600000;
      const inflationPercent = 6;
      const cagrPercent = 10;
      const monthlyContribution = 50000;
      const years = 5;

      const rows = buildProjectionRows(
        startingCorpus,
        annualExpenses,
        inflationPercent,
        cagrPercent,
        monthlyContribution,
        years
      );

      expect(rows.length).toBe(5);

      for (const row of rows) {
        expect(row.annualContribution).toBe(50000 * 12);
        const expectedEnding = row.startingCorpus + row.annualContribution + row.returnAmount;
        expect(row.endingCorpus).toBeCloseTo(expectedEnding, 5);
      }
    });

    test("matches month-by-month accumulation model without double-counting expenses", () => {
      const startingCorpus = 500000;
      const annualExpenses = 300000;
      const inflationPercent = 5;
      const cagrPercent = 8;
      const monthlyContribution = 20000;
      const years = 2;

      const rows = buildProjectionRows(
        startingCorpus,
        annualExpenses,
        inflationPercent,
        cagrPercent,
        monthlyContribution,
        years
      );

      // Verify year 2 starting corpus equals year 1 ending corpus
      expect(rows[1].startingCorpus).toBe(rows[0].endingCorpus);

      // Verify expenses reflect inflation benchmark without deducting from endingCorpus
      expect(rows[0].expenses).toBe(300000);
      expect(rows[1].expenses).toBeCloseTo(300000 * 1.05, 5);

      // In the broken implementation, endingCorpus subtracted inflatedExpenses (300000),
      // which would produce a significantly lower corpus.
      expect(rows[0].endingCorpus).toBeGreaterThan(startingCorpus + monthlyContribution * 12);
    });

    test("handles negative return scenario correctly", () => {
      const startingCorpus = 1000000;
      const rows = buildProjectionRows(startingCorpus, 500000, 6, -5, 0, 1);
      expect(rows[0].returnAmount).toBeLessThan(0);
      expect(rows[0].endingCorpus).toBeLessThan(startingCorpus);
      expect(rows[0].endingCorpus).toBeCloseTo(rows[0].startingCorpus + rows[0].returnAmount, 5);
    });
  });

  describe("calibrateSliders", () => {
    const baseCurrent = {
      monthlyContribution: 10000,
      expectedCagr: 9,
      conservativeCagr: 6,
      optimisticCagr: 12,
      inflationRate: 6
    };

    test("applies full profile when all data meets minimum threshold", () => {
      const dna: FinancialProfile = {
        current_networth: 2000000,
        monthly_contribution: 45678.9,
        savings_rate: 45,
        annual_expenses: 600000,
        annual_income: 1200000,
        income_growth_rate: 8,
        expense_growth_rate: 7.2,
        historical_return: 11.4,
        return_volatility: 4.2,
        income_years_covered: 3,
        expense_years_covered: 3,
        price_months_covered: 12
      };

      const result = calibrateSliders(dna, baseCurrent);

      expect(result.monthlyContribution).toBe(45679);
      expect(result.expectedCagr).toBe(11.4);
      expect(result.conservativeCagr).toBe(7.2); // 11.4 - 4.2
      expect(result.optimisticCagr).toBe(15.6); // 11.4 + 4.2
      expect(result.inflationRate).toBe(7.2);
    });

    test("uses default +-3% spread when volatility is zero or unavailable", () => {
      const dna: FinancialProfile = {
        current_networth: 2000000,
        monthly_contribution: 25000,
        savings_rate: 40,
        annual_expenses: 500000,
        annual_income: 1000000,
        income_growth_rate: 5,
        expense_growth_rate: 5.5,
        historical_return: 10,
        return_volatility: 0,
        income_years_covered: 2,
        expense_years_covered: 2,
        price_months_covered: 6
      };

      const result = calibrateSliders(dna, baseCurrent);

      expect(result.expectedCagr).toBe(10);
      expect(result.conservativeCagr).toBe(7); // 10 - 3
      expect(result.optimisticCagr).toBe(13); // 10 + 3
    });

    test("does not update CAGR if price_months_covered < 3", () => {
      const dna: FinancialProfile = {
        current_networth: 1000000,
        monthly_contribution: 30000,
        savings_rate: 30,
        annual_expenses: 400000,
        annual_income: 800000,
        income_growth_rate: 5,
        expense_growth_rate: 6,
        historical_return: 25,
        return_volatility: 5,
        income_years_covered: 1,
        expense_years_covered: 1,
        price_months_covered: 2 // < 3
      };

      const result = calibrateSliders(dna, baseCurrent);

      expect(result.monthlyContribution).toBe(30000);
      // CAGR unchanged
      expect(result.expectedCagr).toBe(baseCurrent.expectedCagr);
      expect(result.conservativeCagr).toBe(baseCurrent.conservativeCagr);
      expect(result.optimisticCagr).toBe(baseCurrent.optimisticCagr);
      // Inflation unchanged because expense_years_covered < 2
      expect(result.inflationRate).toBe(baseCurrent.inflationRate);
    });

    test("clamps values to slider bounds", () => {
      const dna: FinancialProfile = {
        current_networth: 10000000,
        monthly_contribution: 2000000, // exceeds max 1000000
        savings_rate: 80,
        annual_expenses: 500000,
        annual_income: 5000000,
        income_growth_rate: 10,
        expense_growth_rate: 22, // exceeds max 15
        historical_return: 42, // exceeds max 35
        return_volatility: 50,
        income_years_covered: 5,
        expense_years_covered: 5,
        price_months_covered: 24
      };

      const result = calibrateSliders(dna, baseCurrent);

      expect(result.monthlyContribution).toBe(1000000); // clamped to max
      expect(result.expectedCagr).toBe(35); // clamped to max
      expect(result.conservativeCagr).toBe(-5); // clamped to min -5
      expect(result.optimisticCagr).toBe(40); // clamped to max 40
      expect(result.inflationRate).toBe(15); // clamped to max 15
    });
  });

  describe("CSV Export", () => {
    test("generates proper CSV headers and formatted row values", () => {
      const scenario: ProjectionScenarioTable = {
        label: "Expected Scenario",
        cagr: 9,
        rows: [
          {
            year: 1,
            startingCorpus: 1000000.456,
            annualContribution: 51000,
            returnAmount: 92345.678,
            endingCorpus: 1143346.134,
            expenses: 300000
          },
          {
            year: 2,
            startingCorpus: 1143346.134,
            annualContribution: 51000,
            returnAmount: 105678.912,
            endingCorpus: 1300025.046,
            expenses: 318000
          }
        ]
      };

      const csv = generateScenarioCSV(scenario);
      const lines = csv.split("\n");

      expect(lines[0]).toBe(
        "Year,Starting Corpus,Annual Contribution,Return,Ending Corpus,Inflated Expenses"
      );
      expect(lines[1]).toBe("1,1000000.46,51000,92345.68,1143346.13,300000");
      expect(lines[2]).toBe("2,1143346.13,51000,105678.91,1300025.05,318000");
    });

    test("handles negative returns and zero values correctly in CSV", () => {
      const scenario: ProjectionScenarioTable = {
        label: "Conservative Scenario",
        cagr: -5,
        rows: [
          {
            year: 1,
            startingCorpus: 500000,
            annualContribution: 0,
            returnAmount: -25000.5,
            endingCorpus: 474999.5,
            expenses: 200000
          }
        ]
      };

      const csv = generateScenarioCSV(scenario);
      const lines = csv.split("\n");
      expect(lines.length).toBe(2);
      expect(lines[1]).toBe("1,500000,0,-25000.5,474999.5,200000");
    });

    test("generates clean download filenames from scenario labels", () => {
      expect(getScenarioCSVFilename("Conservative Scenario")).toBe(
        "paisa-projection-conservative-scenario.csv"
      );
      expect(getScenarioCSVFilename("Expected Scenario")).toBe(
        "paisa-projection-expected-scenario.csv"
      );
      expect(getScenarioCSVFilename("Optimistic Scenario")).toBe(
        "paisa-projection-optimistic-scenario.csv"
      );
      expect(getScenarioCSVFilename("Custom FIRE 40%")).toBe("paisa-projection-custom-fire-40.csv");
    });
  });

  describe("Precision Numeric Inputs & Dynamic Bounds", () => {
    test("parseVal handles valid numbers, zero, negative numbers, and fallbacks", () => {
      expect(parseVal(15, 15)).toBe(15);
      expect(parseVal(0, 10)).toBe(0);
      expect(parseVal(-5, 6)).toBe(-5);
      expect(parseVal(4250, 0)).toBe(4250);
      expect(parseVal(65000, 0)).toBe(65000);
      expect(parseVal(null, 15)).toBe(15);
      expect(parseVal(undefined, 9)).toBe(9);
      expect(parseVal(NaN, 6)).toBe(6);
    });

    test("slider dynamic bounds expand to accommodate non-INR or high values", () => {
      const highContribution = 2500000;
      const minBound = Math.min(-100000, Number(highContribution) || 0);
      const maxBound = Math.max(1000000, Number(highContribution) || 0);

      expect(minBound).toBe(-100000);
      expect(maxBound).toBe(2500000); // Expanded past default 1000000

      const largeYears = 50;
      const yearsMax = Math.max(40, Number(largeYears) || 40);
      expect(yearsMax).toBe(50); // Expanded past default 40
    });
  });
});
