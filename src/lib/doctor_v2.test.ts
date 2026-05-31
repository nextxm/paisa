import { describe, expect, test } from "bun:test";
import type { DuplicatePair, Issue, OutlierTransaction, Posting } from "./utils";
import {
  buildPriorityQueue,
  countIssuesByLevel,
  filterDuplicatePairs,
  filterIssues,
  filterOutliers,
  normalizeDoctorQuery
} from "./doctor_v2";

function posting(overrides: Partial<Posting>): Posting {
  return {
    id: "1",
    date: "2026-05-01" as unknown as Posting["date"],
    payee: "Grocery Store",
    account: "Expenses:Groceries",
    commodity: "INR",
    quantity: 1,
    amount: 500,
    original_amount: 500,
    status: "",
    tag_recurring: "",
    transaction_begin_line: 12,
    transaction_end_line: 15,
    file_name: "main.ledger",
    note: "",
    transaction_note: "",
    market_amount: 500,
    balance: 0,
    ...overrides
  };
}

describe("doctor_v2", () => {
  test("normalizes query text", () => {
    expect(normalizeDoctorQuery("  Groceries  ")).toBe("groceries");
  });

  test("counts issues by normalized level", () => {
    const issues: Issue[] = [
      { level: "danger", summary: "A", description: "", details: "" },
      { level: "warning", summary: "B", description: "", details: "" },
      { level: "INFO", summary: "C", description: "", details: "" }
    ];

    expect(countIssuesByLevel(issues)).toEqual({ danger: 1, warning: 1, info: 1 });
  });

  test("filters issues by summary and details", () => {
    const issues: Issue[] = [
      {
        level: "warning",
        summary: "Missing price",
        description: "FX rate gap",
        details: "USD/INR"
      },
      { level: "info", summary: "Looks fine", description: "", details: "" }
    ];

    expect(filterIssues(issues, "usd/inr")).toHaveLength(1);
    expect(filterIssues(issues, "missing")).toHaveLength(1);
  });

  test("filters duplicates and outliers by confidence and query", () => {
    const duplicates: DuplicatePair[] = [
      {
        posting1: posting({ id: "10", payee: "Uber", account: "Expenses:Travel" }),
        posting2: posting({ id: "11", payee: "Uber", account: "Expenses:Travel" }),
        confidence: 0.92,
        reason: "same amount"
      },
      {
        posting1: posting({ id: "12", payee: "Cafe", account: "Expenses:Food" }),
        posting2: posting({ id: "13", payee: "Cafe", account: "Expenses:Food" }),
        confidence: 0.41,
        reason: "same payee"
      }
    ];

    const outliers: OutlierTransaction[] = [
      {
        posting: posting({ id: "20", payee: "Bonus", account: "Income:Bonus", amount: 250000 }),
        mean: 120000,
        std_dev: 15000,
        sigma: 8.7,
        confidence: 0.88
      }
    ];

    expect(filterDuplicatePairs(duplicates, "uber", 0.5)).toHaveLength(1);
    expect(filterDuplicatePairs(duplicates, "cafe", 0.5)).toHaveLength(0);
    expect(filterOutliers(outliers, "bonus", 0.8)).toHaveLength(1);
  });

  test("prioritizes danger issues ahead of other signals", () => {
    const issues: Issue[] = [
      {
        level: "danger",
        summary: "Negative balance",
        description: "A cash account went below zero",
        details: "Assets:Cash"
      }
    ];
    const duplicates: DuplicatePair[] = [
      {
        posting1: posting({ id: "10", payee: "Uber" }),
        posting2: posting({ id: "11", payee: "Uber" }),
        confidence: 0.95,
        reason: "Same amount and date"
      }
    ];

    const queue = buildPriorityQueue(issues, duplicates, []);

    expect(queue[0]?.section).toBe("diagnosis");
    expect(queue[0]?.title).toBe("Negative balance");
  });
});
