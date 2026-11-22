import { describe, expect, test } from "bun:test";
import { parseRawTransaction, serializeEditableTransaction } from "./transaction_patch";

describe("transaction_patch", () => {
  test("parses and serializes ledger transaction with narration and splits", () => {
    const raw = [
      "2024/05/01 * Supermarket | Weekly groceries",
      "    ; :food:",
      "    Assets:Checking                        -100.00 USD",
      "    Expenses:Food                            60.00 USD",
      "    Expenses:Household                       40.00 USD"
    ].join("\n");

    const parsed = parseRawTransaction(raw);
    expect(parsed.date).toBe("2024-05-01");
    expect(parsed.status).toBe("*");
    expect(parsed.payee).toBe("Supermarket");
    expect(parsed.narration).toBe("Weekly groceries");
    expect(parsed.commentLines).toEqual(["; :food:"]);
    expect(parsed.postings).toHaveLength(3);
    expect(parsed.postings[1].account).toBe("Expenses:Food");
    expect(parsed.postings[1].amount).toBe("60.00");
    expect(parsed.postings[1].commodity).toBe("USD");

    // Recategorize second posting
    parsed.postings[1].account = "Expenses:Groceries";
    const serialized = serializeEditableTransaction(parsed, "ledger");
    expect(serialized).toContain("2024/05/01 * Supermarket | Weekly groceries");
    expect(serialized).toContain("; :food:");
    expect(serialized).toContain("Expenses:Groceries");
    expect(serialized).toContain("60.00 USD");
  });

  test("parses and serializes beancount transaction", () => {
    const raw = [
      '2024-05-01 * "Cafe" "Latte" #coffee',
      "  Assets:Cash  -5.00 USD",
      "  Expenses:Coffee  5.00 USD"
    ].join("\n");

    const parsed = parseRawTransaction(raw);
    expect(parsed.date).toBe("2024-05-01");
    expect(parsed.status).toBe("*");
    expect(parsed.payee).toBe("Cafe");
    expect(parsed.narration).toBe("Latte");
    expect(parsed.headerSuffix).toBe("#coffee");

    const serialized = serializeEditableTransaction(parsed, "beancount");
    expect(serialized).toContain('2024-05-01 * "Cafe" "Latte" #coffee');
    expect(serialized).toContain("Expenses:Coffee");
  });
});
