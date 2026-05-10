import { describe, expect, test } from "bun:test";
import {
  applySuggestionSelection,
  buildQuickAddSubmitRequest,
  clearedParserState,
  parserFormOverrides,
  selectedSuggestionUsed,
  type QuickAddFormValues
} from "./quick_add_parser_utils";

function baseValues(): QuickAddFormValues {
  return {
    date: "2026-05-10",
    payee: "",
    narration: "",
    fromAccount: "",
    toAccount: "",
    amount: "",
    commodity: "INR"
  };
}

describe("quick_add_parser_utils", () => {
  test("parserFormOverrides maps parser fields into quick add form", () => {
    const overrides = parserFormOverrides(
      {
        payee: "No Frills",
        from_account: "Liabilities:BMO:CC",
        to_account: "Expenses:Groceries",
        amount: "15",
        currency: "CAD"
      },
      "INR"
    );

    expect(overrides.payee).toBe("No Frills");
    expect(overrides.fromAccount).toBe("Liabilities:BMO:CC");
    expect(overrides.toAccount).toBe("Expenses:Groceries");
    expect(overrides.amount).toBe("15");
    expect(overrides.commodity).toBe("CAD");
  });

  test("applySuggestionSelection updates account and records selected index", () => {
    const selection = applySuggestionSelection(
      "from_account",
      "Liabilities:CAD:BMO:CC",
      1,
      baseValues(),
      {}
    );

    expect(selection.values.fromAccount).toBe("Liabilities:CAD:BMO:CC");
    expect(selection.selectedSuggestionIndex.from_account).toBe(1);
  });

  test("buildQuickAddSubmitRequest uses parser endpoint and payload when parser text exists", () => {
    const values = {
      ...baseValues(),
      payee: "Lunch",
      fromAccount: "Assets:Checking",
      toAccount: "Expenses:Dining",
      amount: "20",
      commodity: "USD"
    };

    const request = buildQuickAddSubmitRequest({
      parserText: "paid $20 for lunch using debit",
      values,
      selectedSuggestionIndex: { to_account: 2 },
      parseStartedAt: 1000,
      nowMs: 2300
    });

    expect(request.endpoint).toBe("/api/parser/create-transaction");
    expect(request.payload.text).toBe("paid $20 for lunch using debit");
    expect(request.payload.suggestion_used).toBe(2);
    expect(request.payload.time_to_confirm_ms).toBe(1300);
  });

  test("buildQuickAddSubmitRequest uses manual endpoint without parser text", () => {
    const values = {
      ...baseValues(),
      payee: "Manual",
      fromAccount: "Assets:Checking",
      toAccount: "Expenses:Groceries",
      amount: "50",
      commodity: "INR"
    };

    const request = buildQuickAddSubmitRequest({
      parserText: " ",
      values,
      selectedSuggestionIndex: {},
      parseStartedAt: null,
      nowMs: 5000
    });

    expect(request.endpoint).toBe("/api/transaction/add");
    expect(request.payload.payee).toBe("Manual");
    expect((request.payload as any).text).toBeUndefined();
  });

  test("selectedSuggestionUsed prioritizes from_account then to_account", () => {
    expect(selectedSuggestionUsed({ from_account: 0, to_account: 2 })).toBe(0);
    expect(selectedSuggestionUsed({ to_account: 2 })).toBe(2);
    expect(selectedSuggestionUsed({})).toBe(-1);
  });

  test("clearedParserState resets parser related state", () => {
    const cleared = clearedParserState();
    expect(cleared.parserText).toBe("");
    expect(cleared.parserResult).toBeNull();
    expect(cleared.parserWarnings).toHaveLength(0);
    expect(cleared.requiresConfirmation).toBe(false);
    expect(cleared.parseStartedAt).toBeNull();
  });
});
