export const DASHBOARD_LAYOUT_STORAGE_KEY = "dashboard-layout";

export type DashboardWidgetId =
  | "networth"
  | "checkingBalances"
  | "investmentIncome"
  | "cashFlow"
  | "budget"
  | "goals"
  | "expenses"
  | "recurring"
  | "recentTransactions";

export type DashboardWidgetColumn = "left" | "right";

export interface DashboardWidgetConfigOption {
  key: string;
  label: string;
  min: number;
  max: number;
  step?: number;
  defaultValue: number;
}

export interface DashboardWidgetMeta {
  id: DashboardWidgetId;
  title: string;
  defaultColumn: DashboardWidgetColumn;
  defaultVisible: boolean;
  configOptions?: DashboardWidgetConfigOption[];
}

export interface DashboardWidgetSetting {
  id: DashboardWidgetId;
  visible: boolean;
  config: Record<string, number>;
}

export interface DashboardLayout {
  left: DashboardWidgetSetting[];
  right: DashboardWidgetSetting[];
}

export const dashboardWidgetRegistry: DashboardWidgetMeta[] = [
  { id: "networth", title: "Assets", defaultColumn: "left", defaultVisible: true },
  {
    id: "checkingBalances",
    title: "Checking Balance",
    defaultColumn: "left",
    defaultVisible: true,
    configOptions: [
      {
        key: "maxItems",
        label: "Accounts to show",
        min: 1,
        max: 12,
        defaultValue: 6
      }
    ]
  },
  {
    id: "investmentIncome",
    title: "Investment Income",
    defaultColumn: "left",
    defaultVisible: true
  },
  { id: "cashFlow", title: "Cash Flow", defaultColumn: "left", defaultVisible: true },
  {
    id: "budget",
    title: "Budget",
    defaultColumn: "left",
    defaultVisible: true,
    configOptions: [
      {
        key: "maxAccounts",
        label: "Budget accounts to show",
        min: 1,
        max: 20,
        defaultValue: 10
      }
    ]
  },
  {
    id: "goals",
    title: "Goals",
    defaultColumn: "left",
    defaultVisible: true,
    configOptions: [
      {
        key: "maxItems",
        label: "Goals to show",
        min: 1,
        max: 12,
        defaultValue: 6
      }
    ]
  },
  { id: "expenses", title: "Expenses", defaultColumn: "right", defaultVisible: true },
  {
    id: "recurring",
    title: "Recurring",
    defaultColumn: "right",
    defaultVisible: true,
    configOptions: [
      {
        key: "maxItems",
        label: "Recurring items to show",
        min: 1,
        max: 30,
        defaultValue: 16
      }
    ]
  },
  {
    id: "recentTransactions",
    title: "Recent Transactions",
    defaultColumn: "right",
    defaultVisible: true,
    configOptions: [
      {
        key: "maxItems",
        label: "Recent transactions to show",
        min: 1,
        max: 30,
        defaultValue: 10
      }
    ]
  }
];

function defaultConfigForWidget(meta: DashboardWidgetMeta): Record<string, number> {
  return Object.fromEntries(
    (meta.configOptions || []).map((option) => [option.key, option.defaultValue])
  );
}

export function createDefaultDashboardLayout(): DashboardLayout {
  const left: DashboardWidgetSetting[] = [];
  const right: DashboardWidgetSetting[] = [];

  dashboardWidgetRegistry.forEach((meta) => {
    const setting = {
      id: meta.id,
      visible: meta.defaultVisible,
      config: defaultConfigForWidget(meta)
    };
    if (meta.defaultColumn === "left") {
      left.push(setting);
    } else {
      right.push(setting);
    }
  });

  return { left, right };
}

function normalizeWidgetSetting(
  setting: Partial<DashboardWidgetSetting>
): DashboardWidgetSetting | null {
  const meta = dashboardWidgetRegistry.find((widget) => widget.id === setting.id);
  if (!meta) {
    return null;
  }

  const defaultConfig = defaultConfigForWidget(meta);
  const mergedConfig = { ...defaultConfig, ...(setting.config || {}) };

  Object.keys(mergedConfig).forEach((key) => {
    const option = (meta.configOptions || []).find((configOption) => configOption.key === key);
    if (!option) {
      delete mergedConfig[key];
      return;
    }

    const value = Number(mergedConfig[key]);
    if (Number.isNaN(value)) {
      mergedConfig[key] = option.defaultValue;
      return;
    }

    const step = option.step || 1;
    const roundedValue = Math.round(value / step) * step;
    mergedConfig[key] = Math.max(option.min, Math.min(option.max, roundedValue));
  });

  return {
    id: meta.id,
    visible: typeof setting.visible === "boolean" ? setting.visible : meta.defaultVisible,
    config: mergedConfig
  };
}

export function normalizeDashboardLayout(
  layout: Partial<DashboardLayout> | null | undefined
): DashboardLayout {
  const defaults = createDefaultDashboardLayout();
  const seen = new Set<DashboardWidgetId>();

  const normalizeColumn = (column: DashboardWidgetColumn): DashboardWidgetSetting[] => {
    const rawColumn = Array.isArray(layout?.[column]) ? layout?.[column] : [];
    const normalized = rawColumn
      .map((setting) => normalizeWidgetSetting(setting))
      .filter((setting): setting is DashboardWidgetSetting => Boolean(setting))
      .filter((setting) => {
        if (seen.has(setting.id)) {
          return false;
        }
        seen.add(setting.id);
        return true;
      });

    defaults[column].forEach((fallback) => {
      if (!seen.has(fallback.id)) {
        normalized.push(fallback);
        seen.add(fallback.id);
      }
    });

    return normalized.filter((setting) => {
      const meta = dashboardWidgetRegistry.find((widget) => widget.id === setting.id);
      return meta?.defaultColumn === column;
    });
  };

  return {
    left: normalizeColumn("left"),
    right: normalizeColumn("right")
  };
}

export function loadDashboardLayout(): DashboardLayout {
  if (typeof localStorage === "undefined") {
    return createDefaultDashboardLayout();
  }

  const raw = localStorage.getItem(DASHBOARD_LAYOUT_STORAGE_KEY);
  if (!raw) {
    return createDefaultDashboardLayout();
  }

  try {
    return normalizeDashboardLayout(JSON.parse(raw));
  } catch {
    return createDefaultDashboardLayout();
  }
}

export function saveDashboardLayout(layout: DashboardLayout) {
  if (typeof localStorage === "undefined") {
    return;
  }

  localStorage.setItem(
    DASHBOARD_LAYOUT_STORAGE_KEY,
    JSON.stringify(normalizeDashboardLayout(layout))
  );
}

export function getWidgetMeta(id: DashboardWidgetId): DashboardWidgetMeta {
  return dashboardWidgetRegistry.find((widget) => widget.id === id)!;
}
