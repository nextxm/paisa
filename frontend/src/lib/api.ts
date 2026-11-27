import dayjs from "dayjs";

import { base } from "$app/paths";

export function appPath(path: string): string {
  return `${base}${path}`;
}

export interface Posting {
  id?: number;
  date: string;
  account: string;
  amount: number;
  quantity?: number;
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
  fileName?: string;
  beginLine?: number;
  endLine?: number;
  postings: Posting[];
}

export interface AssetBreakdown {
  group: string;
  marketAmount: number;
  investmentAmount?: number;
  withdrawalAmount?: number;
  balanceUnits?: number;
  gainAmount?: number;
  xirr?: number;
  absoluteReturn?: number;
  inactive?: boolean;
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
  cashFlows?: Array<{
    date: string;
    income: number;
    expenses: number;
    investment: number;
    balance: number;
  }>;
  budget?: {
    budgetsByMonth?: Record<
      string,
      {
        accounts: Array<{
          account: string;
          forecast: number;
          actual: number;
          available: number;
        }>;
      }
    >;
  };
  goalSummaries?: Array<{
    id: string;
    name: string;
    type: string;
    current: number;
    target: number;
    targetDate: string;
  }>;
  transactionSequences?: Array<{
    name?: string;
    payee?: string;
    narration?: string;
    account?: string;
    frequency?: string;
    next_date?: string;
  }>;
  expenses?: Record<string, Posting[]>;
  transactions?: Transaction[];
}

export interface RuntimeState {
  defaultCurrency: string;
  locale: string;
  readonly: boolean;
  accounts: string[];
  obscure: boolean;
  loaded: boolean;
}

export const runtime: RuntimeState = {
  defaultCurrency: "INR",
  locale: "en-IN",
  readonly: false,
  accounts: [],
  obscure: false,
  loaded: false
};

export function initObscureFromStorage(): boolean {
  try {
    runtime.obscure = localStorage.getItem("paisa-v2-obscure") === "true";
  } catch {
    runtime.obscure = false;
  }
  return runtime.obscure;
}

export function toggleObscure(): boolean {
  runtime.obscure = !runtime.obscure;
  try {
    localStorage.setItem("paisa-v2-obscure", String(runtime.obscure));
  } catch { }
  return runtime.obscure;
}

export async function loadRuntimeConfig(): Promise<RuntimeState> {
  initObscureFromStorage();
  try {
    const res = await fetch("/connect/paisa.v1.PaisaService/GetConfig", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json", ...authHeaders() },
      body: "{}"
    });
    if (res.ok) {
      const data = (await res.json()) as {
        config?: { default_currency?: string; locale?: string; readonly?: boolean };
        accounts?: string[];
      };
      if (data.config?.default_currency) {
        runtime.defaultCurrency = data.config.default_currency;
      }
      if (data.config?.locale) {
        runtime.locale = data.config.locale;
      }
      runtime.readonly = Boolean(data.config?.readonly);
      runtime.accounts = data.accounts || [];
      runtime.loaded = true;
    }
  } catch {
    // Keep defaults if backend config endpoint is unavailable
  }
  return runtime;
}

export function authHeaders(): Record<string, string> {
  try {
    const token = localStorage.getItem("token");
    return token ? { "X-Auth": token } : {};
  } catch {
    return {};
  }
}

export async function logout(): Promise<void> {
  try {
    const headers = authHeaders();
    if (headers["X-Auth"]) {
      await fetch("/api/auth/logout", {
        method: "POST",
        headers: { "Content-Type": "application/json", ...headers }
      }).catch(() => { });
    }
    localStorage.removeItem("token");
  } catch { }
  if (typeof window !== "undefined" && !window.location.pathname.endsWith("/login")) {
    window.location.href = appPath("/login");
  }
}

export async function apiGet<T>(path: string): Promise<T> {
  const res = await fetch(path, {
    headers: { Accept: "application/json", ...authHeaders() }
  });
  if (res.status === 401 && typeof window !== "undefined" && !window.location.pathname.endsWith("/login")) {
    window.location.href = appPath("/login");
    throw new Error("Unauthorized");
  }
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
  }
  return (await res.json()) as T;
}

export async function getFullConfig(): Promise<Record<string, unknown>> {
  const res = await fetch("/api/config", {
    headers: { Accept: "application/json", ...authHeaders() }
  });
  if (!res.ok) {
    throw new Error(`${res.status} ${res.statusText}`);
  }
  const data = (await res.json()) as { config?: Record<string, unknown> };
  return data.config || (data as Record<string, unknown>);
}

export async function updateConfig(
  config: Record<string, unknown>
): Promise<{ success: boolean; error?: string }> {
  try {
    const res = await fetch("/api/config", {
      method: "POST",
      headers: { "Content-Type": "application/json", Accept: "application/json", ...authHeaders() },
      body: JSON.stringify(config)
    });
    if (!res.ok) {
      const errText = await res.text().catch(() => "");
      return { success: false, error: errText || `${res.status} ${res.statusText}` };
    }
    const data = (await res.json()) as { success?: boolean };
    return { success: Boolean(data.success) };
  } catch (e) {
    return { success: false, error: e instanceof Error ? e.message : "Save failed" };
  }
}

export async function triggerSync(
  opts: { journal?: boolean; prices?: boolean; portfolios?: boolean } = { journal: true }
): Promise<boolean> {
  try {
    const res = await fetch("/api/sync", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(opts)
    });
    return res.ok;
  } catch {
    return false;
  }
}

export function formatMoney(amount: number, currency?: string): string {
  if (runtime.obscure) {
    return "••••••";
  }
  const curr = currency || runtime.defaultCurrency || "INR";
  const locale = runtime.locale || (curr === "INR" ? "en-IN" : "en-US");
  try {
    return new Intl.NumberFormat(locale, {
      style: "currency",
      currency: curr,
      maximumFractionDigits: 0
    }).format(amount || 0);
  } catch {
    return `${Math.round(amount || 0).toLocaleString()} ${curr}`;
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
  } catch { }
  return next;
}
