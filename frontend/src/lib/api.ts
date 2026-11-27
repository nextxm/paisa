import dayjs from "dayjs";

export interface Posting {
  id?: number;
  date: string;
  account: string;
  amount: number;
  commodity?: string;
  payee?: string;
  narration?: string;
  tags?: string[];
  note?: string;
}

export interface Transaction {
  id: number;
  date: string;
  payee: string;
  narration: string;
  status?: string;
  tags?: string[];
  postings: Posting[];
}

export interface AssetBreakdown {
  group: string;
  marketAmount: number;
  originalBalances?: Array<{ amount: number; currency: string }>;
}

export interface DashboardResponse {
  networth?: {
    xirr?: number;
    networth?: {
      balanceAmount: number;
      netInvestmentAmount: number;
      gainAmount: number;
    };
  };
  checkingBalances?: {
    asset_breakdowns?: Record<string, AssetBreakdown>;
  };
  investmentIncome?: {
    ttm_dividend?: number;
    ttm_interest?: number;
  };
  expenses?: Record<string, Posting[]>;
  transactions?: Transaction[];
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path, {
    headers: { Accept: "application/json" }
  });
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

export function formatMoney(amount: number, currency = "INR"): string {
  try {
    return new Intl.NumberFormat(currency === "INR" ? "en-IN" : "en-US", {
      style: "currency",
      currency,
      maximumFractionDigits: 0
    }).format(amount || 0);
  } catch {
    return `${Math.round(amount || 0).toLocaleString()} ${currency}`;
  }
}

export function formatShortDate(iso: string): string {
  const d = dayjs(iso);
  return d.isValid() ? d.format("DD MMM YYYY") : iso;
}

export function getTheme(): "light" | "dark" {
  if (typeof document === "undefined") return "light";
  return document.documentElement.dataset.theme === "dark" ? "dark" : "light";
}

export function toggleTheme(): "light" | "dark" {
  const next = getTheme() === "dark" ? "light" : "dark";
  document.documentElement.dataset.theme = next;
  document.documentElement.style.colorScheme = next;
  try {
    localStorage.setItem("paisa-v2-theme", next);
  } catch {}
  return next;
}
