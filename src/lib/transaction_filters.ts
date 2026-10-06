export interface TransactionFilters {
  q: string;
  amountMin: string;
  amountMax: string;
  account: string;
  commodity: string;
  dateFrom: string;
  dateTo: string;
}

export interface SavedTransactionSearch {
  name: string;
  filters: TransactionFilters;
}

export const TRANSACTION_SAVED_SEARCHES_STORAGE_KEY = "transaction-saved-searches";

export function emptyTransactionFilters(): TransactionFilters {
  return {
    q: "",
    amountMin: "",
    amountMax: "",
    account: "",
    commodity: "",
    dateFrom: "",
    dateTo: ""
  };
}

export function buildTransactionFiltersQuery(filters: TransactionFilters): string {
  const params = new URLSearchParams();
  if (filters.q.trim()) params.set("q", filters.q.trim());
  if (filters.amountMin.trim()) params.set("amount_min", filters.amountMin.trim());
  if (filters.amountMax.trim()) params.set("amount_max", filters.amountMax.trim());
  if (filters.account.trim()) params.set("account", filters.account.trim());
  if (filters.commodity.trim()) params.set("commodity", filters.commodity.trim());
  if (filters.dateFrom.trim()) params.set("date_from", filters.dateFrom.trim());
  if (filters.dateTo.trim()) params.set("date_to", filters.dateTo.trim());
  return params.toString();
}

export function parseTransactionFiltersFromSearchParams(
  searchParams: URLSearchParams
): TransactionFilters {
  const filters = emptyTransactionFilters();
  filters.q = searchParams.get("q") || "";
  filters.amountMin = searchParams.get("amount_min") || "";
  filters.amountMax = searchParams.get("amount_max") || "";
  filters.account = searchParams.get("account") || "";
  filters.commodity = searchParams.get("commodity") || "";
  filters.dateFrom = searchParams.get("date_from") || "";
  filters.dateTo = searchParams.get("date_to") || "";
  return filters;
}

export function loadSavedTransactionSearches(): SavedTransactionSearch[] {
  if (typeof localStorage === "undefined") {
    return [];
  }
  const raw = localStorage.getItem(TRANSACTION_SAVED_SEARCHES_STORAGE_KEY);
  if (!raw) {
    return [];
  }
  try {
    const parsed = JSON.parse(raw) as SavedTransactionSearch[];
    if (!Array.isArray(parsed)) {
      return [];
    }
    return parsed
      .filter((entry) => entry && typeof entry.name === "string" && entry.filters)
      .map((entry) => ({
        name: entry.name,
        filters: {
          ...emptyTransactionFilters(),
          ...entry.filters
        }
      }));
  } catch {
    return [];
  }
}

export function saveSavedTransactionSearches(searches: SavedTransactionSearch[]) {
  if (typeof localStorage === "undefined") {
    return;
  }
  localStorage.setItem(TRANSACTION_SAVED_SEARCHES_STORAGE_KEY, JSON.stringify(searches));
}

export function upsertSavedTransactionSearch(
  searches: SavedTransactionSearch[],
  name: string,
  filters: TransactionFilters
): SavedTransactionSearch[] {
  const trimmedName = name.trim();
  if (!trimmedName) {
    return searches;
  }
  const withoutExisting = searches.filter((s) => s.name !== trimmedName);
  return [{ name: trimmedName, filters: { ...filters } }, ...withoutExisting];
}
