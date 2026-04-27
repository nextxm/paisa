import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import type { Transaction, Posting } from "$lib/utils";
import type { Filter } from "./query_builder_utils";
import {
  buildPredicate,
  combinePredicate,
  filterLabel,
  operatorLabel,
  defaultOperator,
  OPERATORS_BY_TYPE
} from "./query_builder_utils";

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

function makePosting(overrides: Partial<Posting> = {}): Posting {
  return {
    id: "p1",
    date: dayjs("2024-01-15"),
    payee: "Test Payee",
    account: "Expenses:Food",
    commodity: "INR",
    quantity: 500,
    amount: 500,
    status: "",
    tag_recurring: "",
    transaction_begin_line: 1,
    transaction_end_line: 5,
    file_name: "test.ledger",
    note: "",
    transaction_note: "",
    market_amount: 500,
    balance: 500,
    ...overrides
  };
}

function makeTransaction(overrides: Partial<Transaction> = {}): Transaction {
  return {
    id: "t1",
    date: dayjs("2024-01-15"),
    payee: "Test Payee",
    beginLine: 1,
    endLine: 5,
    fileName: "test.ledger",
    note: "",
    postings: [makePosting()],
    ...overrides
  };
}

// ---------------------------------------------------------------------------
// buildPredicate – account
// ---------------------------------------------------------------------------

describe("buildPredicate – account", () => {
  test("= matches exact account", () => {
    const f: Filter = { id: "1", type: "account", operator: "=", value: "Expenses:Food" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("= does not match partial account", () => {
    const f: Filter = { id: "1", type: "account", operator: "=", value: "Expenses" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("!= excludes exact account", () => {
    const f: Filter = { id: "1", type: "account", operator: "!=", value: "Expenses:Food" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("!= passes when account differs", () => {
    const f: Filter = { id: "1", type: "account", operator: "!=", value: "Income:Salary" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("contains matches substring (case-insensitive)", () => {
    const f: Filter = { id: "1", type: "account", operator: "contains", value: "food" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("contains does not match non-substring", () => {
    const f: Filter = { id: "1", type: "account", operator: "contains", value: "income" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("starts_with matches prefix (case-insensitive)", () => {
    const f: Filter = { id: "1", type: "account", operator: "starts_with", value: "expenses" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("starts_with does not match non-prefix", () => {
    const f: Filter = { id: "1", type: "account", operator: "starts_with", value: "income" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// buildPredicate – amount
// ---------------------------------------------------------------------------

describe("buildPredicate – amount", () => {
  test("= matches exact amount", () => {
    const f: Filter = { id: "1", type: "amount", operator: "=", value: "500" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("!= excludes exact amount", () => {
    const f: Filter = { id: "1", type: "amount", operator: "!=", value: "500" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("> passes when amount is greater", () => {
    const f: Filter = { id: "1", type: "amount", operator: ">", value: "100" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("> fails when amount is less", () => {
    const f: Filter = { id: "1", type: "amount", operator: ">", value: "1000" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("< passes when amount is less", () => {
    const f: Filter = { id: "1", type: "amount", operator: "<", value: "1000" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test(">= matches exact amount", () => {
    const f: Filter = { id: "1", type: "amount", operator: ">=", value: "500" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("<= matches exact amount", () => {
    const f: Filter = { id: "1", type: "amount", operator: "<=", value: "500" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("invalid number value passes (no-op filter)", () => {
    const f: Filter = { id: "1", type: "amount", operator: "=", value: "abc" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// buildPredicate – date
// ---------------------------------------------------------------------------

describe("buildPredicate – date", () => {
  test("= matches same day", () => {
    const f: Filter = { id: "1", type: "date", operator: "=", value: "2024-01-15" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("= does not match different day", () => {
    const f: Filter = { id: "1", type: "date", operator: "=", value: "2024-01-16" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("> passes when transaction is after", () => {
    const f: Filter = { id: "1", type: "date", operator: ">", value: "2024-01-14" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("> fails when transaction is on same day", () => {
    const f: Filter = { id: "1", type: "date", operator: ">", value: "2024-01-15" };
    expect(buildPredicate(f)(makeTransaction())).toBe(false);
  });

  test("< passes when transaction is before", () => {
    const f: Filter = { id: "1", type: "date", operator: "<", value: "2024-01-16" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test(">= passes on the same day", () => {
    const f: Filter = { id: "1", type: "date", operator: ">=", value: "2024-01-15" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("<= passes on the same day", () => {
    const f: Filter = { id: "1", type: "date", operator: "<=", value: "2024-01-15" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });

  test("invalid date value passes (no-op filter)", () => {
    const f: Filter = { id: "1", type: "date", operator: "=", value: "not-a-date" };
    expect(buildPredicate(f)(makeTransaction())).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// buildPredicate – tag
// ---------------------------------------------------------------------------

describe("buildPredicate – tag", () => {
  const withTag = makeTransaction({
    postings: [makePosting({ tag_recurring: "monthly" })]
  });
  const noTag = makeTransaction();

  test("= matches exact tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "=", value: "monthly" };
    expect(buildPredicate(f)(withTag)).toBe(true);
  });

  test("= does not match different tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "=", value: "weekly" };
    expect(buildPredicate(f)(withTag)).toBe(false);
  });

  test("!= passes when tag differs", () => {
    const f: Filter = { id: "1", type: "tag", operator: "!=", value: "weekly" };
    expect(buildPredicate(f)(withTag)).toBe(true);
  });

  test("!= fails when tag matches", () => {
    const f: Filter = { id: "1", type: "tag", operator: "!=", value: "monthly" };
    expect(buildPredicate(f)(withTag)).toBe(false);
  });

  test("is_set passes when any posting has a tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_set", value: "" };
    expect(buildPredicate(f)(withTag)).toBe(true);
  });

  test("is_set fails when no posting has a tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_set", value: "" };
    expect(buildPredicate(f)(noTag)).toBe(false);
  });

  test("is_not_set passes when no posting has a tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_not_set", value: "" };
    expect(buildPredicate(f)(noTag)).toBe(true);
  });

  test("is_not_set fails when a posting has a tag", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_not_set", value: "" };
    expect(buildPredicate(f)(withTag)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// combinePredicate
// ---------------------------------------------------------------------------

describe("combinePredicate", () => {
  const t = makeTransaction();

  test("empty filter list passes everything", () => {
    expect(combinePredicate([], "AND")(t)).toBe(true);
    expect(combinePredicate([], "OR")(t)).toBe(true);
  });

  test("AND: all filters must pass", () => {
    const filters: Filter[] = [
      { id: "1", type: "account", operator: "=", value: "Expenses:Food" },
      { id: "2", type: "amount", operator: "=", value: "500" }
    ];
    expect(combinePredicate(filters, "AND")(t)).toBe(true);
  });

  test("AND: fails when one filter fails", () => {
    const filters: Filter[] = [
      { id: "1", type: "account", operator: "=", value: "Expenses:Food" },
      { id: "2", type: "amount", operator: "=", value: "999" }
    ];
    expect(combinePredicate(filters, "AND")(t)).toBe(false);
  });

  test("OR: passes when at least one filter passes", () => {
    const filters: Filter[] = [
      { id: "1", type: "account", operator: "=", value: "Expenses:Food" },
      { id: "2", type: "amount", operator: "=", value: "999" }
    ];
    expect(combinePredicate(filters, "OR")(t)).toBe(true);
  });

  test("OR: fails when no filter passes", () => {
    const filters: Filter[] = [
      { id: "1", type: "account", operator: "=", value: "Income:Salary" },
      { id: "2", type: "amount", operator: "=", value: "999" }
    ];
    expect(combinePredicate(filters, "OR")(t)).toBe(false);
  });
});

// ---------------------------------------------------------------------------
// filterLabel / operatorLabel
// ---------------------------------------------------------------------------

describe("filterLabel", () => {
  test("renders is_set label without value", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_set", value: "" };
    expect(filterLabel(f)).toBe("tag is set");
  });

  test("renders is_not_set label without value", () => {
    const f: Filter = { id: "1", type: "tag", operator: "is_not_set", value: "" };
    expect(filterLabel(f)).toBe("tag not set");
  });

  test("renders operator and value for regular filters", () => {
    const f: Filter = { id: "1", type: "account", operator: "contains", value: "Food" };
    expect(filterLabel(f)).toBe("account contains Food");
  });

  test("renders unicode operator symbols", () => {
    const f: Filter = { id: "1", type: "amount", operator: ">=", value: "100" };
    expect(filterLabel(f)).toBe("amount ≥ 100");
  });
});

describe("operatorLabel", () => {
  test("maps = to =", () => expect(operatorLabel("=")).toBe("="));
  test("maps != to ≠", () => expect(operatorLabel("!=")).toBe("≠"));
  test("maps >= to ≥", () => expect(operatorLabel(">=")).toBe("≥"));
  test("maps <= to ≤", () => expect(operatorLabel("<=")).toBe("≤"));
  test("maps contains to contains", () => expect(operatorLabel("contains")).toBe("contains"));
  test("maps starts_with to starts with", () =>
    expect(operatorLabel("starts_with")).toBe("starts with"));
  test("maps is_set to is set", () => expect(operatorLabel("is_set")).toBe("is set"));
  test("maps is_not_set to not set", () => expect(operatorLabel("is_not_set")).toBe("not set"));
});

// ---------------------------------------------------------------------------
// defaultOperator / OPERATORS_BY_TYPE sanity checks
// ---------------------------------------------------------------------------

describe("defaultOperator", () => {
  test("returns first operator for each type", () => {
    expect(defaultOperator("account")).toBe("=");
    expect(defaultOperator("amount")).toBe("=");
    expect(defaultOperator("date")).toBe("=");
    expect(defaultOperator("tag")).toBe("=");
  });
});

describe("OPERATORS_BY_TYPE", () => {
  test("each type has at least one operator", () => {
    for (const type of ["account", "amount", "date", "tag"] as const) {
      expect(OPERATORS_BY_TYPE[type].length).toBeGreaterThan(0);
    }
  });
});
