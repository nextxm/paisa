import { beforeEach, describe, expect, mock, test } from "bun:test";

let ajaxImpl = mock(async (..._args: any[]) => ({}));
let fetchConfigImpl = mock(async (..._args: any[]) => ({ config: {} as UserConfig }));

mock.module("$lib/utils", () => ({
  ajax: (...args: any[]) => ajaxImpl(...args)
}));

mock.module("$lib/config_client", () => ({
  fetchConfig: (...args: any[]) => fetchConfigImpl(...args)
}));

const dashboardPage = await import("./+page");
const goalsPage = await import("./planning/goals/+page");
const networthPage = await import("./assets/networth/+page");
const importPage = await import("./ledger/import/+page");
const doctorV2Page = await import("./more/doctor-v2/+page");
const doctorV3Page = await import("./more/doctor-v3/+page");
const legacyGoalsPage = await import("./more/goals/+page");
const legacyTaxPage = await import("./more/tax/+page");
const lifeProjectionPage = await import("./planning/life/+page");
const lifeGoalsRedirectPage = await import("./planning/life/goals/+page");
const cashFlowIndexPage = await import("./cash_flow/+page");
const expenseIndexPage = await import("./expense/+page");
const assetsIndexPage = await import("./assets/+page");
const liabilitiesIndexPage = await import("./liabilities/+page");

describe("route load functions", () => {
  beforeEach(() => {
    ajaxImpl = mock(async (..._args: any[]) => ({}));
    fetchConfigImpl = mock(async (..._args: any[]) => ({ config: {} as UserConfig }));
  });

  test("dashboard page load prefetches only dashboard", async () => {
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/dashboard") return { dashboard: true };
      return {};
    });

    const data = await dashboardPage.load();

    expect(ajaxImpl).toHaveBeenCalledTimes(1);
    expect((data as any).dashboard).toEqual({ dashboard: true });
  });

  test("goals page load prefetches config and goals", async () => {
    fetchConfigImpl = mock(async () => ({
      config: { goals: { savings: [] } } as unknown as UserConfig
    }));
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/goals") return { goals: [{ name: "Emergency", priority: 1 }] as any };
      return {};
    });

    const data = await goalsPage.load();

    expect(fetchConfigImpl).toHaveBeenCalledTimes(1);
    expect(ajaxImpl).toHaveBeenCalledWith("/api/goals");
    expect((data as any).goals).toEqual([{ name: "Emergency", priority: 1 }]);
    expect((data as any).config.goals.savings).toEqual([]);
    expect((data as any).config.goals.life).toEqual([]);
  });

  test("networth page load prefetches timeline and currencies", async () => {
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/networth") return { xirr: 10, networthTimeline: [] };
      if (route === "/api/price/currencies") return { currencies: ["INR", "USD"] };
      return {};
    });

    const data = await networthPage.load();

    expect(ajaxImpl).toHaveBeenCalledTimes(2);
    expect((data as any).networth.xirr).toBe(10);
    expect((data as any).currencies.currencies).toEqual(["INR", "USD"]);
  });

  test("ledger import page load prefetches templates and presets", async () => {
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/account/tf_idf") return { tf_idf: {}, index: { docs: {}, tokens: {} } };
      if (route === "/api/templates")
        return { templates: [{ id: "1", name: "T", content: "C", template_type: "custom" }] };
      if (route === "/api/import/presets")
        return {
          presets: [
            {
              id: "p",
              name: "Generic Bank CSV",
              column_mappings: {},
              date_format: "",
              default_accounts: {},
              delimiter: ",",
              preset_type: "custom"
            }
          ]
        };
      return {};
    });

    const data = await importPage.load();

    expect(ajaxImpl).toHaveBeenCalledTimes(3);
    expect((data as any).templates).toEqual([
      { id: "1", name: "T", content: "C", template_type: "custom" }
    ]);
    expect((data as any).importPresets[0].name).toEqual("Generic Bank CSV");
  });

  test("legacy and consolidated index routes redirect cleanly", () => {
    const cases: Array<[() => unknown, string]> = [
      [() => doctorV2Page.load(), "/more/doctor"],
      [() => doctorV3Page.load(), "/more/doctor"],
      [() => legacyGoalsPage.load(), "/planning/goals"],
      [() => legacyTaxPage.load(), "/planning/tax"],
      [() => lifeProjectionPage.load(), "/planning/projection?mode=montecarlo"],
      [() => lifeGoalsRedirectPage.load(), "/planning/goals?tab=life"],
      [() => cashFlowIndexPage.load(), "/cash_flow/monthly"],
      [() => expenseIndexPage.load(), "/expense/monthly"],
      [() => assetsIndexPage.load(), "/assets/networth"],
      [() => liabilitiesIndexPage.load(), "/liabilities/balance"]
    ];

    for (const [invoke, expectedLocation] of cases) {
      let caught: any;
      try {
        invoke();
      } catch (err) {
        caught = err;
      }
      expect(caught).toBeDefined();
      expect(caught.status).toBe(301);
      expect(caught.location).toBe(expectedLocation);
    }
  });

  test("ledger import page load tolerates missing templates and presets", async () => {
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/account/tf_idf") return { tf_idf: {}, index: { docs: {}, tokens: {} } };
      if (route === "/api/templates") return {};
      if (route === "/api/import/presets") return {};
      return {};
    });

    const data = await importPage.load();

    expect(ajaxImpl).toHaveBeenCalledTimes(3);
    expect((data as any).templates).toEqual([]);
    expect((data as any).importPresets).toEqual([]);
  });
});
