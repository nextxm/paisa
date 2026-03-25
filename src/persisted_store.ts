import { persisted } from "svelte-local-storage-store";
import { writable, get } from "svelte/store";

export const obscure = persisted("obscure", false);

export const cashflowExpenseDepthAllowed = writable({ min: 1, max: 1 });
export const cashflowExpenseDepth = persisted("cashflowExpenseDepth", 0);
export const cashflowIncomeDepthAllowed = writable({ min: 1, max: 1 });
export const cashflowIncomeDepth = persisted("cashflowIncomeDepth", 0);

export type SankeyPeriod = "month" | "quarter" | "year";
export const sankeyPeriod = persisted<SankeyPeriod>("sankeyPeriod", "month");

// sankeyRefDate is the anchor date for the sankey diagram (YYYY-MM-DD), empty means current period
export const sankeyRefDate = persisted<string>("sankeyRefDate", "");

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
