import { describe, expect, test } from "bun:test";
import {
  formatEvidenceLabel,
  getPrimaryEditorLocation,
  matchesFindingSearch,
  parseEditorTargetUrl,
  type DoctorFinding
} from "./doctor_helpers";

describe("doctor_helpers", () => {
  test("parses editor target URLs with file and line", () => {
    expect(parseEditorTargetUrl("/ledger/editor/2024%2Fmain.ledger#42")).toEqual({
      fileName: "2024/main.ledger",
      lineNumber: 42
    });
    expect(parseEditorTargetUrl("/ledger/editor/main.ledger")).toEqual({
      fileName: "main.ledger",
      lineNumber: 0
    });
    expect(parseEditorTargetUrl("/ledger/price")).toBe(null);
  });

  test("extracts primary editor location from evidence or actions", () => {
    const finding: DoctorFinding = {
      id: "neg:Assets:Cash",
      rule_id: "negative_balance",
      severity: "fix",
      title: "Negative Account Balance",
      details: "Assets:Cash account went negative (-100.00) on 05 Feb 2024",
      evidence: [
        {
          account: "Assets:Cash",
          date: "2024-02-05",
          file_name: "2024.ledger",
          line_number: 24,
          target_url: "/ledger/editor/2024.ledger#24"
        }
      ]
    };

    expect(getPrimaryEditorLocation(finding)).toEqual({
      fileName: "2024.ledger",
      lineNumber: 24
    });
    expect(formatEvidenceLabel(finding.evidence![0])).toBe(
      "Assets:Cash • 2024-02-05 • 2024.ledger:24"
    );
  });

  test("matches search across details and evidence fields", () => {
    const finding: DoctorFinding = {
      id: "1",
      rule_id: "negative_balance",
      severity: "fix",
      title: "Negative Account Balance",
      details: "Assets:Checking went negative on 05 Feb 2024",
      evidence: [{ account: "Assets:Checking", date: "2024-02-05", file_name: "bank.ledger" }]
    };

    expect(matchesFindingSearch(finding, "checking")).toBe(true);
    expect(matchesFindingSearch(finding, "2024-02-05")).toBe(true);
    expect(matchesFindingSearch(finding, "bank.ledger")).toBe(true);
    expect(matchesFindingSearch(finding, "nonexistent")).toBe(false);
  });
});
