import { fromStore } from "svelte/store";
import {
  obscure,
  cashflowExpenseDepth,
  cashflowIncomeDepth,
  cashflowShowTransfers,
  sankeyPeriod,
  sankeyRefDate,
  editorLeftWidth,
  editorRightWidth,
  editorLeftCollapsed,
  editorRightCollapsed,
  configSidebarCollapsed
} from "../../persisted_store";

/**
 * Class-based reactive wrapper around the persisted stores in persisted_store.ts.
 * Use `persistedState.<property>.current` for reading and writing store values
 * in Svelte 5 rune-based components. The underlying `$store` syntax in
 * legacy components continues to work unchanged.
 */
class PersistedState {
  readonly obscure = fromStore(obscure);
  readonly cashflowExpenseDepth = fromStore(cashflowExpenseDepth);
  readonly cashflowIncomeDepth = fromStore(cashflowIncomeDepth);
  readonly cashflowShowTransfers = fromStore(cashflowShowTransfers);
  readonly sankeyPeriod = fromStore(sankeyPeriod);
  readonly sankeyRefDate = fromStore(sankeyRefDate);
  readonly editorLeftWidth = fromStore(editorLeftWidth);
  readonly editorRightWidth = fromStore(editorRightWidth);
  readonly editorLeftCollapsed = fromStore(editorLeftCollapsed);
  readonly editorRightCollapsed = fromStore(editorRightCollapsed);
  readonly configSidebarCollapsed = fromStore(configSidebarCollapsed);
}

export const persistedState = new PersistedState();
