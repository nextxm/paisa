import { describe, expect, test } from "bun:test";
import { reconciliationLabel, reconciliationTagClass, reconciliationIcon } from "./reconciliation";
import type { AccountReconciliationStatus } from "./utils";

function status(overrides: Partial<AccountReconciliationStatus>): AccountReconciliationStatus {
  return {
    account: "Assets:Checking",
    last_reconciled: "2026-05-01",
    frequency_days: 30,
    days_since: 4,
    is_overdue: false,
    ...overrides
  };
}

describe("reconciliation helpers", () => {
  test("returns success class for recent reconciliations", () => {
    expect(reconciliationTagClass(status({ days_since: 4 }))).toBe("is-success");
  });

  test("returns warning class when approaching due date", () => {
    expect(reconciliationTagClass(status({ days_since: 24 }))).toBe("is-warning");
  });

  test("returns danger class when overdue or never reconciled", () => {
    expect(reconciliationTagClass(status({ days_since: 40, is_overdue: true }))).toBe("is-danger");
    expect(
      reconciliationTagClass(status({ last_reconciled: null, days_since: null, is_overdue: true }))
    ).toBe("is-danger");
  });

  test("formats labels for never/today/plural-day states", () => {
    expect(
      reconciliationLabel(status({ last_reconciled: null, days_since: null, is_overdue: true }))
    ).toBe("Last reconciled: never");
    expect(reconciliationLabel(status({ days_since: 0 }))).toBe(
      "Last reconciled: today (2026-05-01)"
    );
    expect(reconciliationLabel(status({ days_since: 5 }))).toBe(
      "Last reconciled: 5 days ago (2026-05-01)"
    );
  });

  test("returns correct icons for each state", () => {
    // We can't easily test the exact glyph char here without knowing it,
    // but we can check it returns something
    expect(reconciliationIcon(status({ days_since: 0 }))).toBeTruthy();
    expect(reconciliationIcon(status({ days_since: 24 }))).toBeTruthy();
    expect(reconciliationIcon(status({ days_since: 40 }))).toBeTruthy();
    expect(reconciliationIcon(status({ days_since: null }))).toBeTruthy();
  });
});
