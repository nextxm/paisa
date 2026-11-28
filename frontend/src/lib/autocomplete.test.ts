import { describe, expect, test } from "bun:test";
import { filterSuggestions, getConfigSuggestionKind } from "./autocomplete";

describe("autocomplete helpers", () => {
  test("detects account and commodity config fields", () => {
    expect(getConfigSuggestionKind("accounts #1")).toBe("account");
    expect(getConfigSuggestionKind("commodities #1")).toBe("commodity");
    expect(getConfigSuggestionKind("name", "commodities #1")).toBe("commodity");
    expect(getConfigSuggestionKind("name", "accounts #1")).toBe("account");
    expect(getConfigSuggestionKind("target", "allocation_targets #1")).toBe(null);
  });

  test("filters suggestions with prefix matches prioritized", () => {
    const items = ["Expenses:Food", "Assets:Bank:HDFC", "Assets:Equity:NIFTY", "Income:Salary"];
    expect(filterSuggestions(items, "assets")).toEqual(["Assets:Bank:HDFC", "Assets:Equity:NIFTY"]);
    expect(filterSuggestions(items, "nifty")).toEqual(["Assets:Equity:NIFTY"]);
    expect(filterSuggestions(items, "", 2)).toEqual(["Expenses:Food", "Assets:Bank:HDFC"]);
  });
});
