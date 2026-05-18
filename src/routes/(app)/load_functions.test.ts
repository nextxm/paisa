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

describe("route load functions", () => {
  beforeEach(() => {
    ajaxImpl = mock(async (..._args: any[]) => ({}));
    fetchConfigImpl = mock(async (..._args: any[]) => ({ config: {} as UserConfig }));
  });

  test("dashboard page load prefetches dashboard and investment income", async () => {
    ajaxImpl = mock(async (route: string) => {
      if (route === "/api/dashboard") return { dashboard: true };
      if (route === "/api/income/investment") return { income: true };
      return {};
    });

    const data = await dashboardPage.load();

    expect(ajaxImpl).toHaveBeenCalledTimes(2);
    expect((data as any).dashboard).toEqual({ dashboard: true });
    expect((data as any).income).toEqual({ income: true });
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
});
