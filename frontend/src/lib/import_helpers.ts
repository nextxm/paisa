import dayjs from "dayjs";
import Handlebars from "handlebars";

export interface ColumnMapping {
  header_row: number;
  date: number;
  description: number;
  debit: number;
  credit: number;
  amount: number;
  dr_cr: number;
  currency: number;
}

export interface StagedTransaction {
  date: string;
  payee: string;
  memo?: string;
  amount: number;
  is_debit?: boolean;
  currency?: string;
  selected_account?: string;
  suggested_account?: string;
  matched_rule_name?: string;
  confidence?: string;
  is_duplicate?: boolean;
  tags?: string[];
  [key: string]: unknown;
}

export interface ImportRule {
  id?: number;
  name: string;
  priority: number;
  enabled: boolean;
  payee_pattern: string;
  payee_match_type: string;
  memo_pattern: string;
  memo_match_type: string;
  tx_type: string;
  target_account: string;
  tags: string[];
  flag_for_review: boolean;
}

export interface ImportTemplate {
  id?: string | number;
  name: string;
  content: string;
  template_type?: "builtin" | "custom" | string;
  base_account?: string;
  delimiter?: string;
}

export interface ImportPreset {
  id?: string | number;
  name: string;
  delimiter?: string;
  column_mappings?: Record<string, string>;
  date_format?: string;
  default_accounts?: Record<string, string>;
}

export interface ImportPreviewRow {
  index: number;
  valid: boolean;
  errors?: string[];
  row: Record<string, string>;
}

export interface AccountTfIdfData {
  tf_idf: Record<string, Record<string, number>>;
  index: {
    docs: Record<string, unknown>;
    tokens: Record<string, unknown[]>;
  };
}

const STOP_WORDS = new Set(["", "fof", "growth", "direct", "plan", "the"]);

export function columnName(index: number): string {
  let name = "";
  let i = index;
  while (i >= 0) {
    name = String.fromCharCode(65 + (i % 26)) + name;
    i = Math.floor(i / 26) - 1;
  }
  return name;
}

function nextChar(key: string): string {
  if (key === "Z") return "AA";
  const last = key.slice(-1);
  const butlast = key.slice(0, -1);
  if (last === "Z") return nextChar(butlast) + "A";
  return butlast + String.fromCharCode(last.charCodeAt(0) + 1);
}

export function parseCSVText(text: string, delimiter = ","): string[][] {
  const rows: string[][] = [];
  let currentRow: string[] = [];
  let currentField = "";
  let inQuotes = false;
  const delim = delimiter || ",";

  for (let i = 0; i < text.length; i++) {
    const ch = text[i];
    if (inQuotes) {
      if (ch === '"') {
        if (text[i + 1] === '"') {
          currentField += '"';
          i++;
        } else {
          inQuotes = false;
        }
      } else {
        currentField += ch;
      }
    } else if (ch === '"') {
      inQuotes = true;
    } else if (ch === delim) {
      currentRow.push(currentField);
      currentField = "";
    } else if (ch === "\r") {
      if (text[i + 1] === "\n") i++;
      currentRow.push(currentField);
      if (currentRow.some((c) => c !== "")) rows.push(currentRow);
      currentRow = [];
      currentField = "";
    } else if (ch === "\n") {
      currentRow.push(currentField);
      if (currentRow.some((c) => c !== "")) rows.push(currentRow);
      currentRow = [];
      currentField = "";
    } else {
      currentField += ch;
    }
  }
  if (currentField !== "" || currentRow.length > 0) {
    currentRow.push(currentField);
    if (currentRow.some((c) => c !== "")) rows.push(currentRow);
  }

  const maxLen = rows.reduce((m, r) => Math.max(m, r.length), 0);
  return rows.map((r) => {
    const padded = [...r];
    while (padded.length < maxLen) padded.push("");
    return padded;
  });
}

export function asRows(data: string[][]): Array<Record<string, string | number>> {
  return data.map((row, i) => {
    const obj: Record<string, string | number> = { index: i };
    row.forEach((cell, idx) => {
      obj[columnName(idx)] = cell ?? "";
    });
    return obj;
  });
}

export function toCSVContent(data: string[][]): string {
  return data
    .map((row) =>
      row
        .map((cell) => {
          const s = String(cell ?? "");
          if (s.includes(",") || s.includes('"') || s.includes("\n")) {
            return `"${s.replaceAll('"', '""')}"`;
          }
          return s;
        })
        .join(",")
    )
    .join("\n");
}

export function cleanPayee(raw?: string): string {
  if (!raw) return "";
  let s = raw.trim();
  s = s.replace(/^(UPI|IMPS|NEFT|RTGS|POS|ATM|ACH|ECS)[/\-\s]+/i, "");
  s = s.replace(/\s+\d{6,}\b/g, "");
  s = s.replace(/\s+/g, " ").trim();
  return s;
}

export function extractShortNote(payee?: string, memo?: string): string {
  return cleanPayee(payee) || cleanPayee(memo) || (payee || memo || "").slice(0, 32).trim();
}

function tokenize(s: string): Record<string, number> {
  const counts: Record<string, number> = {};
  const parts = s
    .split(/[ .()/:]+/)
    .map((p) => p.toLowerCase())
    .filter((p) => p.trim() !== "");
  for (const p of parts) {
    counts[p] = (counts[p] || 0) + 1;
  }
  return counts;
}

export function findMatch(query: string, accountTfIdf: AccountTfIdfData | null): [string, number][] {
  if (!accountTfIdf?.index?.docs || !accountTfIdf.tf_idf) return [];
  const tokens = tokenize(query);
  const totalTokens = Object.keys(tokens).length;
  if (totalTokens === 0) return [];

  const docCount = Object.keys(accountTfIdf.index.docs).length;
  const queryVector: Record<string, number> = {};
  for (const [token, freq] of Object.entries(tokens)) {
    const tf = freq / totalTokens;
    const postings = accountTfIdf.index.tokens?.[token]?.length || 0;
    const idf = Math.log(docCount / (1 + postings)) + 1;
    queryVector[token] = tf * idf;
  }

  const results: [string, number][] = [];
  for (const account of Object.keys(accountTfIdf.index.docs)) {
    const accVec = accountTfIdf.tf_idf[account] || {};
    const allTokens = new Set([...Object.keys(queryVector), ...Object.keys(accVec)]);
    let dot = 0;
    let normQ = 0;
    let normA = 0;
    for (const t of allTokens) {
      const q = queryVector[t] || 0;
      const a = accVec[t] || 0;
      dot += q * a;
      normQ += q * q;
      normA += a * a;
    }
    const denom = Math.sqrt(normQ) * Math.sqrt(normA);
    const sim = denom > 0 ? dot / denom : 0;
    if (sim > 0) results.push([account, sim]);
  }
  results.sort((a, b) => b[1] - a[1]);
  return results;
}

function parseNumericAmount(value: unknown): number | null {
  if (typeof value === "number") return Number.isFinite(value) ? value : null;
  if (typeof value !== "string") return null;
  const cleaned = value.replace(/[^0-9.\-+]/g, "");
  if (!cleaned || cleaned === "-" || cleaned === "+") return null;
  const n = Number(cleaned);
  return Number.isFinite(n) ? n : null;
}

export function renderTemplatePreview(
  templateContent: string,
  rows: Array<Record<string, string | number>>,
  options: { reverse?: boolean; trim?: boolean },
  accountTfIdf: AccountTfIdfData | null
): string {
  if (!templateContent.trim() || rows.length === 0) return "";
  const hb = Handlebars.create();

  hb.registerHelper("eq", (a: unknown, b: unknown) => a === b);
  hb.registerHelper("ne", (a: unknown, b: unknown) => a !== b);
  hb.registerHelper("not", (v: unknown) => !v);
  hb.registerHelper("gte", (a: unknown, b: unknown) => {
    const la = parseNumericAmount(a);
    const rb = parseNumericAmount(b);
    return la !== null && rb !== null && la >= rb;
  });
  hb.registerHelper("gt", (a: unknown, b: unknown) => {
    const la = parseNumericAmount(a);
    const rb = parseNumericAmount(b);
    return la !== null && rb !== null && la > rb;
  });
  hb.registerHelper("lte", (a: unknown, b: unknown) => {
    const la = parseNumericAmount(a);
    const rb = parseNumericAmount(b);
    return la !== null && rb !== null && la <= rb;
  });
  hb.registerHelper("lt", (a: unknown, b: unknown) => {
    const la = parseNumericAmount(a);
    const rb = parseNumericAmount(b);
    return la !== null && rb !== null && la < rb;
  });
  hb.registerHelper("negate", (v: unknown) => {
    const n = parseNumericAmount(v);
    return n !== null ? String(-n) : "";
  });
  hb.registerHelper("round", (v: unknown, opts: Handlebars.HelperOptions) => {
    const n = parseNumericAmount(v);
    if (n === null) return "";
    const p = Math.min(Math.max(Number(opts?.hash?.precision ?? 0), 0), 20);
    return n.toFixed(p);
  });
  hb.registerHelper("and", (...args: unknown[]) => args.slice(0, -1).every(Boolean));
  hb.registerHelper("or", (...args: unknown[]) => {
    for (const arg of args.slice(0, -1)) {
      if (arg) return arg;
    }
    return false;
  });
  hb.registerHelper("isDate", (str: unknown, fmt: string) => {
    if (typeof str !== "string") return false;
    return dayjs(str.trim(), fmt).isValid();
  });
  hb.registerHelper("predictAccount", (...args: unknown[]) => {
    const opts = args.pop() as Handlebars.HelperOptions;
    const query =
      args.length === 0
        ? Object.values((opts.data?.root?.ROW as Record<string, unknown>) || {}).join(" ")
        : args.map((a) => (typeof a === "object" && a !== null ? Object.values(a).join(" ") : String(a ?? ""))).join(" ");
    const prefix = String(opts?.hash?.prefix || "");
    const matches = findMatch(query, accountTfIdf);
    const found = matches.find(([acc]) => acc.startsWith(prefix));
    if (found) return found[0];
    return prefix.endsWith(":") ? `${prefix}Unknown` : `${prefix}:Unknown`;
  });
  hb.registerHelper("isBlank", (str: unknown) => !str || String(str).trim() === "");
  hb.registerHelper("amount", (v: unknown, opts: Handlebars.HelperOptions) => {
    const n = parseNumericAmount(v);
    return n !== null ? String(n) : opts?.hash?.default ?? "";
  });
  hb.registerHelper("date", (str: unknown, _fmt?: string) => {
    const d = dayjs(String(str ?? "").trim());
    return d.isValid() ? d.format("YYYY/MM/DD") : String(str ?? "");
  });
  hb.registerHelper("trim", (str: unknown) => String(str ?? "").trim());
  hb.registerHelper("replace", (str: unknown, search: string, rep: string) =>
    typeof str === "string" ? str.replaceAll(search, rep) : str
  );
  hb.registerHelper("textRange", (fromCol: string, toCol: string, opts: Handlebars.HelperOptions) => {
    const row = (opts.data?.root?.ROW as Record<string, string>) || {};
    const cells: string[] = [];
    let current = fromCol;
    for (let i = 0; i < 1000; i++) {
      cells.push(row[current] || "");
      if (current === toCol) break;
      current = nextChar(current);
    }
    return cells.join(String(opts?.hash?.separator || " "));
  });
  hb.registerHelper("regexpTest", (str: unknown, re: string) =>
    typeof str === "string" ? new RegExp(re).test(str) : false
  );
  hb.registerHelper("regexpMatch", (str: unknown, re: string, opts: Handlebars.HelperOptions) => {
    if (typeof str !== "string") return "";
    const group = Number(opts?.hash?.group ?? 0);
    const m = new RegExp(re).exec(str);
    return m ? m[group] ?? "" : "";
  });
  hb.registerHelper("match", (str: unknown, opts: Handlebars.HelperOptions) => {
    const text = String(str ?? "");
    for (const [val, re] of Object.entries((opts?.hash || {}) as Record<string, string>)) {
      if (new RegExp(re).test(text)) return val;
    }
    return null;
  });
  hb.registerHelper("findAbove", (col: string, opts: Handlebars.HelperOptions) => {
    const re = new RegExp(String(opts?.hash?.regexp || ".+"));
    const sheet = (opts.data?.root?.SHEET as Array<Record<string, string>>) || [];
    let i = Number(opts.data?.root?.ROW?.index ?? 0) - 1;
    while (i >= 0) {
      const cell = sheet[i]?.[col] || "";
      const m = cell.match(re);
      if (m) return opts?.hash?.group ? m[Number(opts.hash.group)] : cell;
      i--;
    }
    return null;
  });
  hb.registerHelper("findBelow", (col: string, opts: Handlebars.HelperOptions) => {
    const re = new RegExp(String(opts?.hash?.regexp || ".+"));
    const sheet = (opts.data?.root?.SHEET as Array<Record<string, string>>) || [];
    let i = Number(opts.data?.root?.ROW?.index ?? 0) + 1;
    while (i < sheet.length) {
      const cell = sheet[i]?.[col] || "";
      const m = cell.match(re);
      if (m) return opts?.hash?.group ? m[Number(opts.hash.group)] : cell;
      i++;
    }
    return null;
  });
  hb.registerHelper("acronym", (str: unknown) =>
    String(str ?? "")
      .replaceAll(/[^a-zA-Z ]/g, "")
      .split(" ")
      .filter((s) => s && !STOP_WORDS.has(s.toLowerCase()))
      .map((s) => s[0].toUpperCase())
      .join("")
  );
  hb.registerHelper("toLowerCase", (str: unknown) => String(str ?? "").toLowerCase());
  hb.registerHelper("toUpperCase", (str: unknown) => String(str ?? "").toUpperCase());
  hb.registerHelper("capitalize", (str: unknown) => {
    const s = String(str ?? "");
    return s ? s[0].toUpperCase() + s.slice(1).toLowerCase() : "";
  });

  const compiled = hb.compile(templateContent);
  const widest = rows.reduce((m, r) => Math.max(m, Object.keys(r).length - 1), 0);
  const cols = Array.from({ length: Math.max(26, widest) }, (_, i) => columnName(i));
  const colRefs = Object.fromEntries(cols.map((c) => [c, c]));

  const output: string[] = [];
  for (const row of rows) {
    let rendered = compiled({ ROW: row, SHEET: rows, ...colRefs });
    if (options.trim) rendered = rendered.trim();
    if (rendered) output.push(rendered);
  }
  if (options.reverse) output.reverse();
  return options.trim ? output.join("\n\n") : output.join("");
}

export function generateLedgerText(
  selected: StagedTransaction[],
  baseAccount: string,
  defaultCurrency: string,
  clubByDate: boolean
): string {
  const lines: string[] = [];

  const writeSingleTx = (tx: StagedTransaction) => {
    const dateFormatted = tx.date.replaceAll("/", "-");
    const shortNote = extractShortNote(tx.payee, tx.memo) || "Transaction";
    const curr = (tx.currency || defaultCurrency || "INR").trim();
    const amtStr = `${Number(tx.amount || 0).toFixed(2)} ${curr}`;
    const tagStr = tx.tags && tx.tags.length > 0 ? `  ; ${tx.tags.join(" ")}` : "";

    lines.push(`${dateFormatted} * ${JSON.stringify(shortNote)}${tagStr}`);
    if (tx.is_debit) {
      lines.push(`    ${baseAccount.padEnd(40)}  -${amtStr}`);
      lines.push(`    ${(tx.selected_account || "Expenses:Unknown").padEnd(40)}   ${amtStr}`);
    } else {
      lines.push(`    ${baseAccount.padEnd(40)}   ${amtStr}`);
      lines.push(`    ${(tx.selected_account || "Income:Unknown").padEnd(40)}  -${amtStr}`);
    }
    lines.push("");
  };

  if (clubByDate) {
    const debitGroups = new Map<string, StagedTransaction[]>();
    const creditTxs: StagedTransaction[] = [];

    for (const tx of selected) {
      const curr = (tx.currency || defaultCurrency || "INR").trim();
      if (tx.is_debit) {
        const key = `${tx.date}__${curr}`;
        if (!debitGroups.has(key)) debitGroups.set(key, []);
        debitGroups.get(key)!.push(tx);
      } else {
        creditTxs.push(tx);
      }
    }

    for (const [key, txs] of debitGroups.entries()) {
      if (txs.length === 1) {
        writeSingleTx(txs[0]);
        continue;
      }
      const [date, curr] = key.split("__");
      lines.push(`${date.replaceAll("/", "-")} * "Consolidated Debits (${txs.length} items)"`);
      let total = 0;
      for (const tx of txs) {
        total += Number(tx.amount || 0);
        const acc = (tx.selected_account || "Expenses:Unknown").padEnd(40);
        const amtStr = `${Number(tx.amount || 0).toFixed(2)} ${curr}`;
        let note = extractShortNote(tx.payee, tx.memo);
        if (tx.tags && tx.tags.length > 0) {
          note = note ? `${note} (${tx.tags.join(" ")})` : tx.tags.join(" ");
        }
        lines.push(`    ${acc}   ${amtStr}${note ? `  ; ${note}` : ""}`);
      }
      lines.push(`    ${baseAccount.padEnd(40)}  -${total.toFixed(2)} ${curr}`);
      lines.push("");
    }

    for (const tx of creditTxs) writeSingleTx(tx);
  } else {
    for (const tx of selected) writeSingleTx(tx);
  }

  return lines.join("\n");
}
