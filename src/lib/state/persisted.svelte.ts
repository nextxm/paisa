import { fromStore } from "svelte/store";
import {
  obscure,
  cashflowExpenseDepthAllowed,
  cashflowExpenseDepth,
  cashflowIncomeDepthAllowed,
  cashflowIncomeDepth,
  cashflowShowTransfers,
  sankeyPeriod,
  sankeyRefDate
} from "../../persisted_store";

/**
 * Class-based reactive wrapper around the persisted stores in persisted_store.ts.
 * Use `persistedState.<property>.current` for reading and writing store values
 * in Svelte 5 rune-based components. The underlying `$store` syntax in
 * legacy components continues to work unchanged.
 */
class PersistedState {
  readonly obscure = fromStore(obscure);
  readonly cashflowExpenseDepthAllowed = fromStore(cashflowExpenseDepthAllowed);
  readonly cashflowExpenseDepth = fromStore(cashflowExpenseDepth);
  readonly cashflowIncomeDepthAllowed = fromStore(cashflowIncomeDepthAllowed);
  readonly cashflowIncomeDepth = fromStore(cashflowIncomeDepth);
  readonly cashflowShowTransfers = fromStore(cashflowShowTransfers);
  readonly sankeyPeriod = fromStore(sankeyPeriod);
  readonly sankeyRefDate = fromStore(sankeyRefDate);
}

export const persistedState = new PersistedState();
