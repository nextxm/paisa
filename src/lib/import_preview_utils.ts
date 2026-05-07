import Papa from "papaparse";

export function toCSVContent(data: string[][]): string {
  return Papa.unparse(data || []);
}

export function filterSelectedRows<T>(rows: T[], included: boolean[]): T[] {
  return rows.filter((_, index) => included[index] !== false);
}

export function defaultIncludedFromValidation(validRows: Array<{ valid: boolean }>): boolean[] {
  return validRows.map((row) => row.valid);
}
