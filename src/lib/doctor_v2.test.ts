import { describe, expect, test } from "bun:test";
import { filterDoctorFindings, groupDoctorFindings, normalizeDoctorQuery } from "./doctor_v2";

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
});
