import type { AssetBreakdown, BalancedPosting } from "./utils";
import Papa from "papaparse";
import * as XLSX from "xlsx";

export function download(balancedPostings: BalancedPosting[]) {
  const rows = balancedPostings.map((balancedPosting) => {
    return {
      Date: balancedPosting.from.date.toISOString(),
      Payee: balancedPosting.from.payee,
      FromAccount: balancedPosting.from.account,
      FromQuantity: balancedPosting.from.quantity,
      FromAmount: balancedPosting.from.amount,
      FromCommodity: balancedPosting.from.commodity,
      ToAccount: balancedPosting.to.account,
      ToQuantity: balancedPosting.to.quantity,
      ToAmount: balancedPosting.to.amount,
      ToCommodity: balancedPosting.to.commodity
    };
  });

  const csv = Papa.unparse(rows);
  const downloadLink = document.createElement("a");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  downloadLink.href = window.URL.createObjectURL(blob);
  downloadLink.download = "paisa-transactions.csv";
  downloadLink.click();
}

type AssetBalanceExportRow = {
  Account: string;
  InvestmentAmount: number;
  WithdrawalAmount: number;
  BalanceUnits: number;
  OriginalValue: string;
  MarketValue: number;
  Change: number;
  XIRR: number;
  AbsoluteReturn: number;
};

type AssetBreakdownTreeNode = AssetBreakdown & { _children?: AssetBreakdownTreeNode[] };

function buildAssetBreakdownTree(items: AssetBreakdown[]): AssetBreakdownTreeNode[] {
  const result: AssetBreakdownTreeNode[] = [];
  const sorted = [...items].sort((a, b) => a.group.localeCompare(b.group));

  for (const item of sorted) {
    const parts = item.group.split(":");
    let current = result;
    for (let i = 0; i < parts.length; i++) {
      const part = parts[i];
      let found = current.find((node) => node.group.split(":")[i] === part);
      if (!found) {
        found = { ...item };
        current.push(found);
      }

      if (i !== parts.length - 1) {
        found._children = found._children || [];
        current = found._children;
      }
    }
  }

  return result;
}

function formatOriginalBalances(breakdown: AssetBreakdown): string {
  if (!breakdown.originalBalances) return "";
  return breakdown.originalBalances
    .map((balance) => `${balance.currency} ${balance.amount}`)
    .join(", ");
}

function toAssetBalanceExportRow(
  breakdown: AssetBreakdown,
  account: string
): AssetBalanceExportRow {
  return {
    Account: account,
    InvestmentAmount: breakdown.investmentAmount,
    WithdrawalAmount: breakdown.withdrawalAmount,
    BalanceUnits: breakdown.balanceUnits,
    OriginalValue: formatOriginalBalances(breakdown),
    MarketValue: breakdown.marketAmount,
    Change: breakdown.gainAmount,
    XIRR: breakdown.xirr,
    AbsoluteReturn: breakdown.absoluteReturn
  };
}

function flatAccountLabel(group: string): string {
  return group;
}

function hierarchyAccountLabel(group: string, depth: number): string {
  const name = group.split(":").at(-1) || group;
  return `${"  ".repeat(depth)}${name}`;
}

function flattenTreeRows(
  nodes: AssetBreakdownTreeNode[],
  depth = 0,
  accountLabel: (group: string, depth: number) => string
): AssetBalanceExportRow[] {
  const rows: AssetBalanceExportRow[] = [];
  for (const node of nodes) {
    rows.push(toAssetBalanceExportRow(node, accountLabel(node.group, depth)));
    if (node._children) {
      rows.push(...flattenTreeRows(node._children, depth + 1, accountLabel));
    }
  }
  return rows;
}

export function buildAssetBalanceExportRows(
  breakdowns: Record<string, AssetBreakdown>,
  flat: boolean
): AssetBalanceExportRow[] {
  const values = Object.values(breakdowns);
  if (flat) {
    return values
      .sort((a, b) => a.group.localeCompare(b.group))
      .map((breakdown) => toAssetBalanceExportRow(breakdown, flatAccountLabel(breakdown.group)));
  }

  const tree = buildAssetBreakdownTree(values);
  return flattenTreeRows(tree, 0, hierarchyAccountLabel);
}

export function downloadAssetBalanceCSV(breakdowns: Record<string, AssetBreakdown>, flat: boolean) {
  const rows = buildAssetBalanceExportRows(breakdowns, flat);
  const csv = Papa.unparse(rows);
  const downloadLink = document.createElement("a");
  const blob = new Blob([csv], { type: "text/csv;charset=utf-8;" });
  downloadLink.href = window.URL.createObjectURL(blob);
  downloadLink.download = "paisa-asset-balance.csv";
  downloadLink.click();
}

export function downloadAssetBalanceExcel(
  breakdowns: Record<string, AssetBreakdown>,
  flat: boolean
) {
  const rows = buildAssetBalanceExportRows(breakdowns, flat);
  const worksheet = XLSX.utils.json_to_sheet(rows);
  const workbook = XLSX.utils.book_new();
  XLSX.utils.book_append_sheet(workbook, worksheet, "Asset Balance");
  XLSX.writeFile(workbook, "paisa-asset-balance.xlsx");
}
