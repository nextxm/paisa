import { describe, expect, test } from "bun:test";
import {
  DASHBOARD_WIDGET_DEFINITIONS,
  loadDashboardLayout,
  persistDashboardLayout,
  reconcileDashboardLayout,
  reorderDashboardWidget,
  stepMoveDashboardWidget,
  type DashboardWidgetDefinition,
  type StorageLike
} from "./dashboard_layout";

class MemoryStorage implements StorageLike {
  private values = new Map<string, string>();

  getItem(key: string): string | null {
    return this.values.has(key) ? this.values.get(key) || null : null;
  }

  setItem(key: string, value: string): void {
    this.values.set(key, value);
  }
}

const widgets: DashboardWidgetDefinition[] = [
  {
    id: "assets-overview",
    title: "Assets",
    column: "left",
    defaultPosition: 1,
    defaultVisible: true
  },
  {
    id: "budget",
    title: "Budget",
    column: "left",
    defaultPosition: 2,
    defaultVisible: false,
    settings: [
      {
        key: "limit",
        label: "Items",
        min: 1,
        max: 20,
        step: 1,
        defaultValue: 8
      }
    ]
  },
  {
    id: "recent-transactions",
    title: "Recent",
    column: "right",
    defaultPosition: 3,
    defaultVisible: true,
    settings: [
      {
        key: "limit",
        label: "Items",
        min: 1,
        max: 40,
        step: 1,
        defaultValue: 15
      }
    ]
  }
];

describe("dashboard_layout", () => {
  test("loads defaults when storage is empty", () => {
    const storage = new MemoryStorage();
    const layout = loadDashboardLayout(widgets, "dashboard-layout", storage);

    expect(layout.map((entry) => entry.id)).toEqual([
      "assets-overview",
      "budget",
      "recent-transactions"
    ]);
    expect(layout.find((entry) => entry.id === "budget")?.visible).toBe(false);
    expect(layout.find((entry) => entry.id === "recent-transactions")?.config.limit).toBe(15);
  });

  test("persists and restores layout order and settings", () => {
    const storage = new MemoryStorage();
    const current = loadDashboardLayout(widgets, "dashboard-layout", storage);

    const reordered = reorderDashboardWidget(current, "recent-transactions", "assets-overview").map(
      (entry) => {
        if (entry.id === "budget") {
          return {
            ...entry,
            visible: true,
            config: {
              ...entry.config,
              limit: 12
            }
          };
        }

        return entry;
      }
    );

    persistDashboardLayout(reordered, "dashboard-layout", storage);

    const loaded = loadDashboardLayout(widgets, "dashboard-layout", storage);
    expect(loaded.map((entry) => entry.id)).toEqual([
      "recent-transactions",
      "assets-overview",
      "budget"
    ]);
    expect(loaded.find((entry) => entry.id === "budget")?.visible).toBe(true);
    expect(loaded.find((entry) => entry.id === "budget")?.config.limit).toBe(12);
  });

  test("reconcile keeps saved order while adding new widgets", () => {
    const saved = [
      { id: "recent-transactions", visible: true, config: { limit: 18 } },
      { id: "assets-overview", visible: true, config: {} }
    ] as const;

    const reconciled = reconcileDashboardLayout(
      DASHBOARD_WIDGET_DEFINITIONS.filter((widget) =>
        ["assets-overview", "budget", "recent-transactions"].includes(widget.id)
      ),
      [...saved]
    );

    expect(reconciled.map((entry) => entry.id)).toEqual([
      "recent-transactions",
      "assets-overview",
      "budget"
    ]);
    expect(reconciled.find((entry) => entry.id === "budget")?.config.limit).toBe(10);
  });

  test("supports step-based move for keyboard controls", () => {
    const storage = new MemoryStorage();
    const current = loadDashboardLayout(widgets, "dashboard-layout", storage);
    const moved = stepMoveDashboardWidget(current, "budget", -1);

    expect(moved.map((entry) => entry.id)).toEqual([
      "budget",
      "assets-overview",
      "recent-transactions"
    ]);
  });
});
