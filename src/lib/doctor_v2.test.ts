import { describe, expect, test } from "bun:test";
import {
  filterDoctorFindings,
  formatFindingEvidenceSummary,
  getFindingPrimaryTarget,
  groupDoctorFindings,
  normalizeDoctorQuery
} from "./doctor_v2";

describe("doctor_v2", () => {
  test("normalizes query text", () => {
    expect(normalizeDoctorQuery("  Groceries  ")).toBe("groceries");
  });

  test("groups findings by rule_id", () => {
    const findings: any[] = [
      { id: "1", rule_id: "credit_entry", title: "Credit Entry" },
      { id: "2", rule_id: "credit_entry", title: "Credit Entry" },
      { id: "3", rule_id: "negative_balance", title: "Negative Balance" }
    ];

    const groups = groupDoctorFindings(findings);
    expect(groups).toHaveLength(2);
    expect(groups.find((g) => g.rule_id === "credit_entry")?.findings).toHaveLength(2);
  });

  test("filters findings by query text across fields and evidence", () => {
    const findings: any[] = [
      {
        id: "1",
        rule_id: "credit_entry",
        title: "Credit Entry",
        why_it_matters: "Wrong sign",
        how_to_fix: "Flip sign",
        details: "Salary error",
        evidence: [{ account: "Income:Salary", payee: "Acme Corp" }]
      },
      {
        id: "2",
        rule_id: "negative_balance",
        title: "Negative Balance",
        why_it_matters: "Below zero",
        how_to_fix: "Add deposit",
        details: "Cash zero",
        evidence: [{ account: "Assets:Cash" }]
      }
    ];

    expect(filterDoctorFindings(findings, "acme")).toHaveLength(1);
    expect(filterDoctorFindings(findings, "below zero")).toHaveLength(1);
    expect(filterDoctorFindings(findings, "nonexistent")).toHaveLength(0);
  });

  test("formats evidence summary with account, date, amount, and file location", () => {
    const finding: any = {
      id: "1",
      rule_id: "negative_balance",
      kind: "rule",
      evidence: [
        {
          account: "Assets:Checking",
          amount: -150,
          date: "2024-02-05",
          file_name: "2024.ledger",
          line_number: 24,
          target_url: "/ledger/editor/2024.ledger#24"
        }
      ],
      actions: [
        { type: "open_editor", label: "Edit Transaction", url: "/ledger/editor/2024.ledger#24" }
      ]
    };

    const summary = formatFindingEvidenceSummary(finding);
    expect(summary).toContain("Assets:Checking");
    expect(summary).toContain("2024-02-05");
    expect(summary).toContain("2024.ledger:24");
    expect(getFindingPrimaryTarget(finding)).toBe("/ledger/editor/2024.ledger#24");
  });

  test("formats duplicate evidence summary with dates", () => {
    const finding: any = {
      id: "dup:1:2",
      rule_id: "duplicate_transaction",
      kind: "duplicate",
      evidence: [
        {
          account: "Expenses:Food",
          amount: 500,
          date: "2024-01-10",
          file_name: "food.ledger",
          line_number: 12
        },
        {
          account: "Expenses:Food",
          amount: 500,
          date: "2024-01-11",
          file_name: "food.ledger",
          line_number: 18
        }
      ],
      actions: []
    };

    const summary = formatFindingEvidenceSummary(finding);
    expect(summary).toContain("Expenses:Food");
    expect(summary).toContain("2024-01-10 & 2024-01-11");
    expect(summary).toContain("food.ledger:12");
  });
});
