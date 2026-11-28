import { afterEach, beforeEach, describe, expect, mock, test } from "bun:test";
import {
  apiGet,
  appPath,
  authHeaders,
  formatMoney,
  formatShortDate,
  getTheme,
  initObscureFromStorage,
  loadRuntimeConfig,
  runtime,
  toggleObscure,
  toggleTheme,
  triggerSync
} from "./api";

const originalFetch = globalThis.fetch;

describe("frontend api helpers", () => {
  beforeEach(() => {
    localStorage.clear();
    runtime.defaultCurrency = "INR";
    runtime.locale = "en-IN";
    runtime.readonly = false;
    runtime.accounts = [];
    runtime.obscure = false;
    runtime.loaded = false;
    document.documentElement.dataset.theme = "light";
    document.documentElement.style.colorScheme = "light";
  });

  afterEach(() => {
    globalThis.fetch = originalFetch;
  });

  test("appPath prefixes paths with base", () => {
    expect(appPath("/")).toBe("/");
    expect(appPath("/transactions")).toBe("/transactions");
    expect(appPath("/login")).toBe("/login");
  });

  test("formatMoney respects currency, locale, and obscure mode", () => {
    runtime.defaultCurrency = "USD";
    runtime.locale = "en-US";
    expect(formatMoney(1234)).toBe("$1,234");
    expect(formatMoney(1234, "EUR")).toContain("1,234");

    runtime.obscure = true;
    expect(formatMoney(1234)).toBe("••••••");

    runtime.obscure = false;
    expect(formatMoney(500, "INVALID_CURR")).toContain("500");
  });

  test("formatShortDate formats valid ISO dates and preserves invalid ones", () => {
    expect(formatShortDate("2026-04-15")).toBe("15 Apr 2026");
    expect(formatShortDate("not-a-date")).toBe("not-a-date");
  });

  test("authHeaders reads token from localStorage", () => {
    expect(authHeaders()).toEqual({});
    localStorage.setItem("token", "abc-123");
    expect(authHeaders()).toEqual({ "X-Auth": "abc-123" });
  });

  test("initObscureFromStorage and toggleObscure persist state", () => {
    expect(initObscureFromStorage()).toBe(false);
    expect(toggleObscure()).toBe(true);
    expect(localStorage.getItem("paisa-v2-obscure")).toBe("true");
    expect(initObscureFromStorage()).toBe(true);
    expect(toggleObscure()).toBe(false);
    expect(localStorage.getItem("paisa-v2-obscure")).toBe("false");
  });

  test("getTheme and toggleTheme update document dataset and localStorage", () => {
    expect(getTheme()).toBe("light");
    expect(toggleTheme()).toBe("dark");
    expect(document.documentElement.dataset.theme).toBe("dark");
    expect(localStorage.getItem("paisa-v2-theme")).toBe("dark");
    expect(toggleTheme()).toBe("light");
    expect(document.documentElement.dataset.theme).toBe("light");
  });

  test("loadRuntimeConfig uses Connect-RPC response when ok", async () => {
    localStorage.setItem("token", "tok-1");
    const calls: string[] = [];
    globalThis.fetch = mock(async (input: RequestInfo | URL, init?: RequestInit) => {
      const url = String(input);
      calls.push(url);
      expect((init?.headers as Record<string, string>)["X-Auth"]).toBe("tok-1");
      return new Response(
        JSON.stringify({
          config: { default_currency: "EUR", locale: "de-DE", readonly: true },
          accounts: ["Assets:Bank", "Expenses:Food"]
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      );
    }) as unknown as typeof fetch;

    const state = await loadRuntimeConfig();
    expect(calls).toEqual(["/connect/paisa.v1.PaisaService/GetConfig"]);
    expect(state.defaultCurrency).toBe("EUR");
    expect(state.locale).toBe("de-DE");
    expect(state.readonly).toBe(true);
    expect(state.accounts).toEqual(["Assets:Bank", "Expenses:Food"]);
    expect(state.loaded).toBe(true);
  });

  test("loadRuntimeConfig falls back to /api/config when Connect-RPC fails", async () => {
    const calls: string[] = [];
    globalThis.fetch = mock(async (input: RequestInfo | URL) => {
      const url = String(input);
      calls.push(url);
      if (url.startsWith("/connect/")) {
        return new Response("not found", { status: 404 });
      }
      return new Response(
        JSON.stringify({
          config: { default_currency: "USD", locale: "en-US", readonly: false },
          accounts: ["Assets:Checking"]
        }),
        { status: 200, headers: { "Content-Type": "application/json" } }
      );
    }) as unknown as typeof fetch;

    const state = await loadRuntimeConfig();
    expect(calls).toEqual(["/connect/paisa.v1.PaisaService/GetConfig", "/api/config"]);
    expect(state.defaultCurrency).toBe("USD");
    expect(state.accounts).toEqual(["Assets:Checking"]);
    expect(state.loaded).toBe(true);
  });

  test("apiGet sends X-Auth header and parses JSON, or throws on error", async () => {
    localStorage.setItem("token", "my-token");
    globalThis.fetch = mock(async (_input: RequestInfo | URL, init?: RequestInit) => {
      expect((init?.headers as Record<string, string>)["X-Auth"]).toBe("my-token");
      return new Response(JSON.stringify({ ok: true }), { status: 200 });
    }) as unknown as typeof fetch;

    await expect(apiGet<{ ok: boolean }>("/api/ping")).resolves.toEqual({ ok: true });

    globalThis.fetch = mock(
      async () => new Response("Server Error", { status: 500, statusText: "Internal Server Error" })
    ) as unknown as typeof fetch;
    await expect(apiGet("/api/ping")).rejects.toThrow("500 Internal Server Error");
  });

  test("triggerSync polls job status until completed or failed", async () => {
    let pollCount = 0;
    globalThis.fetch = mock(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/api/sync") {
        return new Response(JSON.stringify({ job_id: "job-42" }), { status: 202 });
      }
      pollCount++;
      const status = pollCount >= 2 ? "completed" : "running";
      return new Response(JSON.stringify({ status }), { status: 200 });
    }) as unknown as typeof fetch;

    expect(await triggerSync({ journal: true })).toBe(true);
    expect(pollCount).toBe(2);

    globalThis.fetch = mock(async (input: RequestInfo | URL) => {
      const url = String(input);
      if (url === "/api/sync") {
        return new Response(JSON.stringify({ job_id: "job-99" }), { status: 202 });
      }
      return new Response(JSON.stringify({ status: "failed" }), { status: 200 });
    }) as unknown as typeof fetch;

    expect(await triggerSync({ journal: true })).toBe(false);
  });
});
