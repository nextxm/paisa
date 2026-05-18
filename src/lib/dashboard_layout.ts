export const DASHBOARD_LAYOUT_STORAGE_KEY = "dashboard-layout";

export type DashboardWidgetId =
  | "assets-overview"
  | "checking-balances"
  | "investment-income"
  | "cash-flow"
  | "budget"
  | "goals"
  | "expenses"
  | "recurring"
  | "recent-transactions";

export interface DashboardWidgetSetting {
  key: string;
  label: string;
  min: number;
  max: number;
  step: number;
  defaultValue: number;
}

export interface DashboardWidgetDefinition {
  id: DashboardWidgetId;
  title: string;
  column: "left" | "right";
  defaultPosition: number;
  defaultVisible: boolean;
  settings?: DashboardWidgetSetting[];
}

export interface DashboardWidgetLayout {
  id: DashboardWidgetId;
  visible: boolean;
  config: Record<string, number>;
}

export interface StorageLike {
  getItem(key: string): string | null;
  setItem(key: string, value: string): void;
}

export const DASHBOARD_WIDGET_DEFINITIONS: DashboardWidgetDefinition[] = [
  {
    id: "assets-overview",
    title: "Assets",
    column: "left",
    defaultPosition: 1,
    defaultVisible: true
  },
  {
    id: "checking-balances",
    title: "Checking Balance",
    column: "left",
    defaultPosition: 2,
    defaultVisible: true
  },
  {
    id: "investment-income",
    title: "Investment Income",
    column: "left",
    defaultPosition: 3,
    defaultVisible: true
  },
  {
    id: "cash-flow",
    title: "Cash Flow",
    column: "left",
    defaultPosition: 4,
    defaultVisible: true
  },
  {
    id: "budget",
    title: "Budget",
    column: "left",
    defaultPosition: 5,
    defaultVisible: true,
    settings: [
      {
        key: "limit",
        label: "Accounts to show",
        min: 1,
        max: 20,
        step: 1,
        defaultValue: 10
      }
    ]
  },
  {
    id: "goals",
    title: "Goals",
    column: "left",
    defaultPosition: 6,
    defaultVisible: true,
    settings: [
      {
        key: "limit",
        label: "Goals to show",
        min: 1,
        max: 20,
        step: 1,
        defaultValue: 8
      }
    ]
  },
  {
    id: "expenses",
    title: "Expenses",
    column: "right",
    defaultPosition: 7,
    defaultVisible: true
  },
  {
    id: "recurring",
    title: "Recurring",
    column: "right",
    defaultPosition: 8,
    defaultVisible: true,
    settings: [
      {
        key: "limit",
        label: "Recurring items",
        min: 1,
        max: 40,
        step: 1,
        defaultValue: 16
      }
    ]
  },
  {
    id: "recent-transactions",
    title: "Recent Transactions",
    column: "right",
    defaultPosition: 9,
    defaultVisible: true,
    settings: [
      {
        key: "limit",
        label: "Transactions to show",
        min: 1,
        max: 40,
        step: 1,
        defaultValue: 15
      }
    ]
  }
];

function defaultConfig(widget: DashboardWidgetDefinition): Record<string, number> {
  return (widget.settings || []).reduce(
    (acc, setting) => ({
      ...acc,
      [setting.key]: setting.defaultValue
    }),
    {} as Record<string, number>
  );
}

function defaultLayout(widgets: DashboardWidgetDefinition[]): DashboardWidgetLayout[] {
  return [...widgets]
    .sort((a, b) => a.defaultPosition - b.defaultPosition)
    .map((widget) => ({
      id: widget.id,
      visible: widget.defaultVisible,
      config: defaultConfig(widget)
    }));
}

export function reconcileDashboardLayout(
  widgets: DashboardWidgetDefinition[],
  layout: DashboardWidgetLayout[] | null | undefined
): DashboardWidgetLayout[] {
  const defaults = defaultLayout(widgets);

  if (!Array.isArray(layout) || layout.length === 0) {
    return defaults;
  }

  const known = new Set(widgets.map((widget) => widget.id));
  const defaultIndex = new Map(defaults.map((entry, index) => [entry.id, index]));
  const savedById = new Map(layout.map((entry) => [entry.id, entry]));

  const orderedIds = [
    ...layout.map((entry) => entry.id).filter((id) => known.has(id)),
    ...defaults.map((entry) => entry.id).filter((id) => !savedById.has(id))
  ];

  return orderedIds
    .filter((id, index) => orderedIds.indexOf(id) === index)
    .sort((a, b) => {
      const aSaved = savedById.has(a);
      const bSaved = savedById.has(b);

      if (aSaved && bSaved) {
        return orderedIds.indexOf(a) - orderedIds.indexOf(b);
      }

      if (aSaved) {
        return -1;
      }

      if (bSaved) {
        return 1;
      }

      return (defaultIndex.get(a) || 0) - (defaultIndex.get(b) || 0);
    })
    .map((id) => {
      const widget = widgets.find((item) => item.id === id);
      const saved = savedById.get(id);
      const base = defaults.find((item) => item.id === id);

      const mergedConfig = {
        ...(base?.config || {}),
        ...(saved?.config || {})
      };

      return {
        id,
        visible: saved?.visible ?? widget?.defaultVisible ?? true,
        config: mergedConfig
      };
    });
}

export function loadDashboardLayout(
  widgets: DashboardWidgetDefinition[],
  storageKey = DASHBOARD_LAYOUT_STORAGE_KEY,
  storage: StorageLike | null = typeof localStorage === "undefined" ? null : localStorage
): DashboardWidgetLayout[] {
  if (!storage) {
    return reconcileDashboardLayout(widgets, null);
  }

  try {
    const raw = storage.getItem(storageKey);
    if (!raw) {
      return reconcileDashboardLayout(widgets, null);
    }

    const parsed = JSON.parse(raw);
    return reconcileDashboardLayout(widgets, parsed);
  } catch (_e) {
    return reconcileDashboardLayout(widgets, null);
  }
}

export function persistDashboardLayout(
  layout: DashboardWidgetLayout[],
  storageKey = DASHBOARD_LAYOUT_STORAGE_KEY,
  storage: StorageLike | null = typeof localStorage === "undefined" ? null : localStorage
) {
  if (!storage) {
    return;
  }

  storage.setItem(storageKey, JSON.stringify(layout));
}

export function reorderDashboardWidget(
  layout: DashboardWidgetLayout[],
  sourceId: DashboardWidgetId,
  targetId: DashboardWidgetId
): DashboardWidgetLayout[] {
  if (sourceId === targetId) {
    return layout;
  }

  const sourceIndex = layout.findIndex((widget) => widget.id === sourceId);
  const targetIndex = layout.findIndex((widget) => widget.id === targetId);

  if (sourceIndex < 0 || targetIndex < 0) {
    return layout;
  }

  const next = [...layout];
  const [source] = next.splice(sourceIndex, 1);
  next.splice(targetIndex, 0, source);
  return next;
}

export function stepMoveDashboardWidget(
  layout: DashboardWidgetLayout[],
  id: DashboardWidgetId,
  direction: -1 | 1
): DashboardWidgetLayout[] {
  const index = layout.findIndex((widget) => widget.id === id);
  if (index < 0) {
    return layout;
  }

  const targetIndex = index + direction;
  if (targetIndex < 0 || targetIndex >= layout.length) {
    return layout;
  }

  const next = [...layout];
  const [entry] = next.splice(index, 1);
  next.splice(targetIndex, 0, entry);
  return next;
}
