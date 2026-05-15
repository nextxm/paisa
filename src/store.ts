import { writable, derived, get } from "svelte/store";
import * as d3 from "d3";
import { cashflowExpenseDepth, cashflowIncomeDepth } from "./persisted_store";

import dayjs from "dayjs";
import type { AccountTfIdf, LedgerFileError, SheetFileError, SheetLineResult } from "$lib/utils";
import _ from "lodash";

export function now() {
  if (globalThis.__now) {
    return globalThis.__now;
  }
  return dayjs();
}

interface EditorState {
  hasUnsavedChanges: boolean;
  undoDepth: number;
  redoDepth: number;
  errors: LedgerFileError[];
  output: string;
  fileName: string;
}

export const initialEditorState: EditorState = {
  hasUnsavedChanges: false,
  undoDepth: 0,
  redoDepth: 0,
  errors: [],
  output: "",
  fileName: ""
};

interface SheetEditorState {
  hasUnsavedChanges: boolean;
  undoDepth: number;
  redoDepth: number;
  doc: string;
  pendingEval: boolean;
  evalDuration: number;
  currentLine: number;
  errors: SheetFileError[];
  results: SheetLineResult[];
}

export const initialSheetEditorState: SheetEditorState = {
  hasUnsavedChanges: false,
  undoDepth: 0,
  redoDepth: 0,
  currentLine: 0,
  doc: "",
  pendingEval: false,
  evalDuration: 0,
  errors: [],
  results: []
};

export const editorState = writable(initialEditorState);
export const sheetEditorState = writable(initialSheetEditorState);

export const month = writable(now().format("YYYY-MM"));
export const year = writable<string>("");
export const dateRangeOption = writable<number>(3);

export const dateMin = writable(dayjs("1980", "YYYY"));
export const dateMax = writable(now());

export const dateRange = derived(
  [dateMin, dateMax, dateRangeOption],
  ([$dateMin, $dateMax, $dateRangeOption]) => {
    if ($dateRangeOption === -1) {
      return { from: $dateMin, to: $dateMax };
    } else {
      return {
        from: $dateMax.subtract($dateRangeOption, "year"),
        to: $dateMax
      };
    }
  }
);

export const theme = writable("light");

function createLoadingStore() {
  const { subscribe, set, update } = writable(0);
  return {
    subscribe,
    set: (v: boolean) => update((n) => (v ? n + 1 : Math.max(0, n - 1))),
    reset: () => set(0)
  };
}

export const loading = createLoadingStore();

const DELAY = 200;
const DEBOUNCE_DELAY = 200;

let timeoutId: NodeJS.Timeout;
export const delayedLoading = derived(
  [loading],
  ([$l], set) => {
    if (timeoutId) {
      clearTimeout(timeoutId);
    }

    const isLoading = $l > 0;
    timeoutId = setTimeout(
      () => {
        return set(isLoading);
      },
      isLoading ? DELAY : DEBOUNCE_DELAY
    );
  },
  false
);

let swithcTimeoutId: NodeJS.Timeout;
export const delayedUnLoading = derived(
  [loading],
  ([$l], set) => {
    if (swithcTimeoutId) {
      clearTimeout(swithcTimeoutId);
    }

    const isLoading = $l > 0;
    if (isLoading) {
      set(isLoading);
    } else {
      swithcTimeoutId = setTimeout(() => {
        return set(isLoading);
      }, DEBOUNCE_DELAY);
    }
  },
  false
);

export const willClearTippy = writable(0);

export const accountTfIdf = writable<AccountTfIdf | null>(null);

export function setAllowedDateRange(dates: dayjs.Dayjs[]) {
  const [start, end] = d3.extent(dates);
  if (start) {
    dateMin.set(start);
    dateMax.set(end);
  }
}

// Transient UI state for cashflow depth controls (not persisted).
// The allowed range is set by the page after loading data; the selected depth is persisted.
export const cashflowExpenseDepthAllowed = writable({ min: 1, max: 1 });
export const cashflowIncomeDepthAllowed = writable({ min: 1, max: 1 });

export function setCashflowDepthAllowed(expense: number, income: number) {
  cashflowExpenseDepthAllowed.set({ min: 1, max: expense });
  if (get(cashflowExpenseDepth) == 0 || get(cashflowExpenseDepth) > expense) {
    cashflowExpenseDepth.set(expense);
  }

  cashflowIncomeDepthAllowed.set({ min: 1, max: income });
  if (get(cashflowIncomeDepth) == 0 || get(cashflowIncomeDepth) > income) {
    cashflowIncomeDepth.set(income);
  }
}

export const willRefresh = writable(0);
export const commandPaletteOpen = writable(false);
export async function refresh() {
  if (get(editorState).hasUnsavedChanges) {
    const confirmed = confirm("You have unsaved changes. Are you sure you want to leave?");
    if (!confirmed) {
      return false;
    } else {
      editorState.update((current) => _.assign({}, current, { hasUnsavedChanges: false }));
    }
  }
  try {
    const { invalidateAll } = await import("$app/navigation");
    await invalidateAll();
  } catch (e) {
    // Ignore in environments where SvelteKit modules are not available
  }
  willRefresh.update((n) => n + 1);
  return true;
}

export { jobs, jobsList, isJobRunning } from "$lib/stores/jobs";
