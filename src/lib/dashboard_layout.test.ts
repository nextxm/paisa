import { beforeEach, describe, expect, test } from "bun:test";
import {
  DASHBOARD_LAYOUT_STORAGE_KEY,
  createDefaultDashboardLayout,
  loadDashboardLayout,
  normalizeDashboardLayout,
  saveDashboardLayout
} from "./dashboard_layout";

describe("dashboard layout", () => {
  beforeEach(() => {
    localStorage.clear();
  });

  test("normalizes stored layout and keeps known widgets in each column", () => {
    const normalized = normalizeDashboardLayout({
      left: [
        { id: "goals", visible: false, config: { maxItems: 4 } },
        { id: "unknown-widget" as any, visible: true, config: {} }
      ],
      right: [{ id: "recentTransactions", visible: true, config: { maxItems: 3 } }]
    });

    expect(normalized.left[0].id).toBe("goals");
    expect(normalized.left[0].visible).toBe(false);
    expect(normalized.left[0].config.maxItems).toBe(4);
    expect(normalized.left.some((widget) => widget.id === "networth")).toBe(true);
    expect(normalized.right[0].id).toBe("recentTransactions");
    expect(normalized.right[0].config.maxItems).toBe(3);
    expect(normalized.right.some((widget) => widget.id === "expenses")).toBe(true);
  });

  test("persists layout across load/save cycles", () => {
    const layout = createDefaultDashboardLayout();
    const left = layout.left.filter((widget) => widget.id !== "budget");
    left.unshift({ id: "budget", visible: false, config: { maxAccounts: 2 } });

    const right = layout.right.filter((widget) => widget.id !== "recurring");
    right.unshift({ id: "recurring", visible: true, config: { maxItems: 7 } });

    saveDashboardLayout({ left, right });
    const loaded = loadDashboardLayout();

    expect(loaded.left[0].id).toBe("budget");
    expect(loaded.left[0].visible).toBe(false);
    expect(loaded.left[0].config.maxAccounts).toBe(2);
    expect(loaded.right[0].id).toBe("recurring");
    expect(loaded.right[0].config.maxItems).toBe(7);
  });

  test("falls back to defaults when storage is invalid", () => {
    localStorage.setItem(DASHBOARD_LAYOUT_STORAGE_KEY, "{not json");
    const loaded = loadDashboardLayout();
    const defaults = createDefaultDashboardLayout();

    expect(loaded.left.map((widget) => widget.id)).toEqual(
      defaults.left.map((widget) => widget.id)
    );
    expect(loaded.right.map((widget) => widget.id)).toEqual(
      defaults.right.map((widget) => widget.id)
    );
  });
});
