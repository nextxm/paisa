import { describe, expect, test } from "bun:test";
import dayjs from "dayjs";
import type { Posting } from "$lib/utils";
import { aggregatePostingsByLevel, groupAccountByLevel } from "./cash_flow_monthly";

function makePosting(account: string, quantity: number, commodity = "INR"): Posting {
  return {
    id: "1",
    date: dayjs("2026-01-15"),
    payee: "Test",
    account,
    commodity,
    quantity,
    amount: quantity,
    original_amount: quantity,
    status: "",
    tag_recurring: "",
    transaction_begin_line: 1,
    transaction_end_line: 2,
    file_name: "test.ledger",
    note: "",
    transaction_note: "",
    market_amount: quantity,
    balance: 0
  };
}

describe("cash_flow_monthly helpers", () => {
  test("groupAccountByLevel groups accounts to level 1, level 2, or all", () => {
    expect(groupAccountByLevel("Expenses:Shopping:Amazon", 1)).toBe("Expenses:Shopping");
    expect(groupAccountByLevel("Expenses:Shopping:Amazon:Prime", 2)).toBe(
      "Expenses:Shopping:Amazon"
    );
    expect(groupAccountByLevel("Expenses:Shopping:Amazon:Prime", "all")).toBe(
      "Expenses:Shopping:Amazon:Prime"
    );
    expect(groupAccountByLevel("Expenses:Rent", 1)).toBe("Expenses:Rent");
    expect(groupAccountByLevel("Expenses:Rent", 2)).toBe("Expenses:Rent");
  });

  test("aggregatePostingsByLevel rolls up subcategories at level 1 and keeps commodities separate", () => {
    const postings = [
      makePosting("Expenses:Shopping:Amazon", 120, "INR"),
      makePosting("Expenses:Shopping:Jewellery", 380, "INR"),
      makePosting("Expenses:Shopping:Amazon", 25, "USD"),
      makePosting("Expenses:Food:Dining", 90, "INR")
    ];

    expect(aggregatePostingsByLevel(postings, false, 1)).toEqual([
      { account: "Expenses:Food", commodity: "INR", amount: 90 },
      { account: "Expenses:Shopping", commodity: "INR", amount: 500 },
      { account: "Expenses:Shopping", commodity: "USD", amount: 25 }
    ]);

    expect(aggregatePostingsByLevel(postings, false, 2)).toEqual([
      { account: "Expenses:Food:Dining", commodity: "INR", amount: 90 },
      { account: "Expenses:Shopping:Amazon", commodity: "INR", amount: 120 },
      { account: "Expenses:Shopping:Amazon", commodity: "USD", amount: 25 },
      { account: "Expenses:Shopping:Jewellery", commodity: "INR", amount: 380 }
    ]);
  });

  test("aggregatePostingsByLevel negates inflows when requested", () => {
    const postings = [
      makePosting("Income:Salary:Base", -5000, "INR"),
      makePosting("Income:Salary:Bonus", -1000, "INR")
    ];

    expect(aggregatePostingsByLevel(postings, true, 1)).toEqual([
      { account: "Income:Salary", commodity: "INR", amount: 6000 }
    ]);
  });
});
