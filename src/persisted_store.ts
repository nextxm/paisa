import { persisted } from "svelte-local-storage-store";

export const obscure = persisted("obscure", false);

export const cashflowExpenseDepth = persisted("cashflowExpenseDepth", 0);
export const cashflowIncomeDepth = persisted("cashflowIncomeDepth", 0);
export const cashflowShowTransfers = persisted("cashflowShowTransfers", true);

export type SankeyPeriod = "month" | "quarter" | "year";
export const sankeyPeriod = persisted<SankeyPeriod>("sankeyPeriod", "month");

// sankeyRefDate is the anchor date for the sankey diagram (YYYY-MM-DD), empty means current period
export const sankeyRefDate = persisted<string>("sankeyRefDate", "");

export const editorLeftWidth = persisted("editorLeftWidth", 250);
export const editorRightWidth = persisted("editorRightWidth", 350);
export const editorLeftCollapsed = persisted("editorLeftCollapsed", false);
export const editorRightCollapsed = persisted("editorRightCollapsed", false);
export const configSidebarCollapsed = persisted("configSidebarCollapsed", false);
