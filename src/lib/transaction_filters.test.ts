import { beforeEach, describe, expect, test } from "bun:test";
import {
  TRANSACTION_SAVED_SEARCHES_STORAGE_KEY,
  buildTransactionFiltersQuery,
  emptyTransactionFilters,
  loadSavedTransactionSearches,
  parseTransactionFiltersFromSearchParams,
  saveSavedTransactionSearches,
  upsertSavedTransactionSearch
} from "./transaction_filters";

describe("transaction filters", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  test("builds backend query string using issue filter parameter names", () => {
    const query = buildTransactionFiltersQuery({
      q: "amazon",
      amountMin: "10000",
      amountMax: "25000",
      account: "expenses:shopping",
      commodity: "INR",
      dateFrom: "2024-01-01",
      dateTo: "2024-12-31"
    });

    expect(query).toContain("q=amazon");
    expect(query).toContain("amount_min=10000");
    expect(query).toContain("amount_max=25000");
    expect(query).toContain("account=expenses%3Ashopping");
    expect(query).toContain("commodity=INR");
    expect(query).toContain("date_from=2024-01-01");
    expect(query).toContain("date_to=2024-12-31");
  });

  test("parses filters from URL search params", () => {
    const filters = parseTransactionFiltersFromSearchParams(
      new URLSearchParams(
        "q=rent&amount_min=1000&amount_max=5000&account=expenses:rent&commodity=INR&date_from=2024-01-01&date_to=2024-01-31"
      )
    );
    expect(filters.q).toBe("rent");
    expect(filters.amountMin).toBe("1000");
    expect(filters.amountMax).toBe("5000");
    expect(filters.account).toBe("expenses:rent");
    expect(filters.commodity).toBe("INR");
    expect(filters.dateFrom).toBe("2024-01-01");
    expect(filters.dateTo).toBe("2024-01-31");
  });

  test("loads and upserts saved searches", () => {
    const initial = upsertSavedTransactionSearch([], "Amazon 2024", {
      ...emptyTransactionFilters(),
      q: "amazon",
      dateFrom: "2024-01-01",
      dateTo: "2024-12-31"
    });
    saveSavedTransactionSearches(initial);

    const loaded = loadSavedTransactionSearches();
    expect(loaded).toHaveLength(1);
    expect(loaded[0].name).toBe("Amazon 2024");
    expect(loaded[0].filters.q).toBe("amazon");

    localStorage.setItem(TRANSACTION_SAVED_SEARCHES_STORAGE_KEY, "{invalid");
    expect(loadSavedTransactionSearches()).toEqual([]);
  });
});
