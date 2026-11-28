import { describe, expect, test } from "bun:test";
import {
  asRows,
  cleanPayee,
  columnName,
  extractShortNote,
  findMatch,
  generateLedgerText,
  parseCSVText,
  renderTemplatePreview,
  toCSVContent,
  type AccountTfIdfData
} from "./import_helpers";

describe("import_helpers", () => {
  test("columnName converts 0-based indices to spreadsheet letters", () => {
    expect(columnName(0)).toBe("A");
    expect(columnName(25)).toBe("Z");
    expect(columnName(26)).toBe("AA");
    expect(columnName(27)).toBe("AB");
    expect(columnName(701)).toBe("ZZ");
    expect(columnName(702)).toBe("AAA");
  });

  test("parseCSVText handles quotes, escaped quotes, custom delimiters, and padding", () => {
    const csv =
      'Date,Payee,Amount\r\n2026-01-01,"Coffee, ""Shop""",150\r\n\r\n2026-01-02,Bakery\r\n';
    expect(parseCSVText(csv)).toEqual([
      ["Date", "Payee", "Amount"],
      ["2026-01-01", 'Coffee, "Shop"', "150"],
      ["2026-01-02", "Bakery", ""]
    ]);

    expect(parseCSVText("a;b;c\n1;2;3", ";")).toEqual([
      ["a", "b", "c"],
      ["1", "2", "3"]
    ]);
  });

  test("asRows and toCSVContent convert rows cleanly", () => {
    const data = [
      ["2026-01-01", "Cafe, Inc", "100"],
      ["2026-01-02", 'Book "Store"', "200"]
    ];
    expect(asRows(data)).toEqual([
      { index: 0, A: "2026-01-01", B: "Cafe, Inc", C: "100" },
      { index: 1, A: "2026-01-02", B: 'Book "Store"', C: "200" }
    ]);

    expect(toCSVContent(data)).toBe('2026-01-01,"Cafe, Inc",100\n2026-01-02,"Book ""Store""",200');
  });

  test("cleanPayee and extractShortNote normalize bank narration", () => {
    expect(cleanPayee("UPI/Swiggy 123456789")).toBe("Swiggy");
    expect(cleanPayee("IMPS-Rent Transfer   987654")).toBe("Rent Transfer");
    expect(cleanPayee("")).toBe("");
    expect(extractShortNote("NEFT/Acme Corp 123456", "Fallback")).toBe("Acme Corp");
    expect(extractShortNote("", "POS/Grocery 555666")).toBe("Grocery");
  });

  test("findMatch ranks accounts by TF-IDF similarity", () => {
    const tfIdf: AccountTfIdfData = {
      tf_idf: {
        "Expenses:Food:Coffee": { coffee: 1.5, starbucks: 1.2 },
        "Expenses:Transport": { uber: 1.4, ride: 1.1 }
      },
      index: {
        docs: {
          "Expenses:Food:Coffee": 1,
          "Expenses:Transport": 1
        },
        tokens: {
          coffee: ["Expenses:Food:Coffee"],
          starbucks: ["Expenses:Food:Coffee"],
          uber: ["Expenses:Transport"]
        }
      }
    };

    const matches = findMatch("Starbucks Coffee", tfIdf);
    expect(matches.length).toBe(1);
    expect(matches[0][0]).toBe("Expenses:Food:Coffee");
    expect(matches[0][1]).toBeGreaterThan(0);
    expect(findMatch("", tfIdf)).toEqual([]);
    expect(findMatch("coffee", null)).toEqual([]);
  });

  test("renderTemplatePreview renders Handlebars templates with helpers", () => {
    const rows = asRows([
      ["Date", "Payee", "Amount"],
      ["2026-01-05", "Coffee", "120.50"],
      ["2026-01-06", "Tea", "45.00"]
    ]);
    const tpl =
      '{{#if (isDate ROW.A "YYYY-MM-DD")}}{{ROW.A}} * "{{ROW.B}}"\n  Expenses:Food  {{amount ROW.C "INR"}}\n  Assets:Bank  {{negate ROW.C}} INR\n{{/if}}';

    const out = renderTemplatePreview(tpl, rows, { reverse: true, trim: true }, null);
    expect(out).toContain('2026-01-06 * "Tea"');
    expect(out).toContain('2026-01-05 * "Coffee"');
    expect(out.indexOf("2026-01-06")).toBeLessThan(out.indexOf("2026-01-05"));
    expect(out).toContain("Expenses:Food  120.5");
    expect(out).toContain("-120.5 INR");
  });

  test("generateLedgerText formats single and consolidated entries", () => {
    const txs = [
      {
        date: "2026/02/01",
        payee: "UPI/Coffee 123456",
        amount: 150,
        is_debit: true,
        selected_account: "Expenses:Coffee",
        tags: ["morning"]
      },
      {
        date: "2026/02/01",
        payee: "Bakery",
        amount: 250,
        is_debit: true,
        selected_account: "Expenses:Food"
      },
      {
        date: "2026/02/02",
        payee: "Salary",
        amount: 5000,
        is_debit: false,
        selected_account: "Income:Salary"
      }
    ];

    const single = generateLedgerText(txs, "Assets:Bank", "INR", false);
    expect(single).toContain('2026-02-01 * "Coffee"  ; morning');
    expect(single).toContain("Assets:Bank                               -150.00 INR");
    expect(single).toContain("Expenses:Coffee                            150.00 INR");
    expect(single).toContain('2026-02-02 * "Salary"');
    expect(single).toContain("Income:Salary                             -5000.00 INR");

    const clubbed = generateLedgerText(txs, "Assets:Bank", "INR", true);
    expect(clubbed).toContain('2026-02-01 * "Consolidated Debits (2 items)"');
    expect(clubbed).toContain("Assets:Bank                               -400.00 INR");
    expect(clubbed).toContain('2026-02-02 * "Salary"');
  });
});
