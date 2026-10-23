import Papa from "papaparse";
import * as XLSX from "xlsx";
import _ from "lodash";
import { format } from "./journal";
import { pdf2array } from "./pdf";
import * as XlsxPopulate from "xlsx-populate";

interface Result {
  data: string[][];
  error?: string;
}

export interface StatementTransaction {
  date: string;
  description: string;
  amount: number;
  account: string;
  expenseAccount?: string;
  raw?: Record<string, any>;
}

export interface DuplicateMatchOptions {
  dateWindowHours?: number;
  requirePayee?: boolean;
}

function normalizePayee(value: string): string {
  return _.toString(value || "")
    .trim()
    .toLowerCase()
    .replace(/\s+/g, " ");
}

export function isDuplicateStatementTransaction(
  left: Partial<StatementTransaction>,
  right: Partial<StatementTransaction>,
  options: DuplicateMatchOptions = {}
): boolean {
  const dateWindowHours = options.dateWindowHours ?? 0;
  const requirePayee = options.requirePayee ?? true;

  const sameAccount = _.toString(left.account || "") === _.toString(right.account || "");
  const sameAmount = Math.abs((left.amount ?? 0) - (right.amount ?? 0)) < 0.01;
  if (!sameAccount || !sameAmount) {
    return false;
  }

  const leftPayee = normalizePayee(_.toString(left.description || ""));
  const rightPayee = normalizePayee(_.toString(right.description || ""));
  const payeeMatch = leftPayee !== "" && leftPayee === rightPayee;

  if (requirePayee && !payeeMatch) {
    return false;
  }

  const leftDate = left.date ? new Date(left.date) : null;
  const rightDate = right.date ? new Date(right.date) : null;
  if (
    !leftDate ||
    !rightDate ||
    Number.isNaN(leftDate.getTime()) ||
    Number.isNaN(rightDate.getTime())
  ) {
    return false;
  }

  const diffHours = Math.abs(leftDate.getTime() - rightDate.getTime()) / (1000 * 60 * 60);
  return diffHours <= dateWindowHours;
}

export function deduplicateStatementTransactions(
  transactions: Array<Partial<StatementTransaction>>,
  options: DuplicateMatchOptions = {}
): StatementTransaction[] {
  const deduped: StatementTransaction[] = [];

  for (const tx of transactions) {
    const candidate = {
      date: _.toString(tx.date || ""),
      description: _.toString(tx.description || ""),
      amount: Number.isFinite(tx.amount) ? Number(tx.amount) : 0,
      account: _.toString(tx.account || ""),
      expenseAccount: tx.expenseAccount || "Expenses:Unknown"
    } as StatementTransaction;

    const existing = deduped.find((entry) =>
      isDuplicateStatementTransaction(entry, candidate, options)
    );

    if (existing) {
      existing.amount += candidate.amount;
      if (!existing.expenseAccount || existing.expenseAccount === "Expenses:Unknown") {
        existing.expenseAccount = candidate.expenseAccount;
      }
      continue;
    }

    deduped.push(candidate);
  }

  return deduped;
}

export function consolidateStatementTransactions(
  transactions: Array<Partial<StatementTransaction>>,
  options: { defaultAccount?: string; expensePrefix?: string; dateWindowHours?: number } = {}
): string {
  if (_.isEmpty(transactions)) {
    return "";
  }

  const defaultAccount = options.defaultAccount || "Assets:Unknown";
  const expensePrefix = options.expensePrefix || "Expenses";
  const dateWindowHours = options.dateWindowHours ?? 0;

  const prepared = transactions.map((tx) => ({
    date: _.toString(tx.date || "") || "1970-01-01",
    description: _.toString(tx.description || "Unknown merchant"),
    amount: Number.isFinite(tx.amount) ? Number(tx.amount) : 0,
    account: _.toString(tx.account || defaultAccount),
    expenseAccount:
      tx.expenseAccount ||
      guessExpenseAccount(_.toString(tx.description || "Unknown merchant"), expensePrefix)
  })) as StatementTransaction[];

  const deduped = deduplicateStatementTransactions(prepared, {
    dateWindowHours,
    requirePayee: true
  });
  const groups = new Map<
    string,
    { month: string; expenseAccount: string; amount: number; account: string }
  >();

  for (const tx of deduped) {
    const amount = Math.abs(tx.amount);
    const date = tx.date.includes("-") ? tx.date.slice(0, 7) : "1970-01";
    const key = `${date}|${tx.expenseAccount || "Expenses:Unknown"}`;

    const current = groups.get(key) || {
      month: date,
      expenseAccount: tx.expenseAccount || `${expensePrefix}:Unknown`,
      amount: 0,
      account: tx.account || defaultAccount
    };

    current.amount += amount;
    groups.set(key, current);
  }

  const lines: string[] = [];
  for (const group of Array.from(groups.values()).sort((a, b) => a.month.localeCompare(b.month))) {
    const monthDate = `${group.month}-01`;
    const total = Number(group.amount.toFixed(2));
    lines.push(`${monthDate} * "${group.expenseAccount}"`);
    lines.push(`  ${group.expenseAccount} ${total.toFixed(2)} INR`);
    lines.push(`  ${group.account || defaultAccount} -${total.toFixed(2)} INR`);
  }

  return lines.join("\n");
}

const EXPENSE_KEYWORDS: Array<[string, RegExp[]]> = [
  [
    "Expenses:Groceries",
    [
      /grocery|grocer|mart|supermarket|bigbasket|milk|vegetable|fruit|dairy|reliance fresh|dmart|fresh|foodmart/i
    ]
  ],
  [
    "Expenses:Dining",
    [/restaurant|cafe|coffee|zomato|swiggy|pizza|burger|bistro|food court|canteen|dining/i]
  ],
  [
    "Expenses:Transportation",
    [
      /uber|ola|metro|fuel|petrol|diesel|rail|bus|cab|taxi|parking|toll|travel|auto|transport|indian oil|bharat petrol/i
    ]
  ],
  [
    "Expenses:Shopping",
    [/amazon|flipkart|myntra|shop|shopping|marketplace|retail|ebay|store|paytm|online purchase/i]
  ],
  [
    "Expenses:Utilities",
    [
      /electricity|water|gas|wifi|internet|mobile|phone|recharge|broadband|utility|bill|bse|bescom|airtel|jio|vodafone/i
    ]
  ],
  [
    "Expenses:Health",
    [/doctor|hospital|pharmacy|clinic|apollo|health|medical|medplus|wellness|insurance|optical/i]
  ],
  ["Expenses:Travel", [/flight|hotel|booking|airline|railway|irctc|airbnb|holiday|trip|travel/i]],
  [
    "Expenses:Entertainment",
    [/netflix|spotify|movie|cinema|games|entertainment|youtube|streaming/i]
  ],
  ["Expenses:Education", [/school|college|course|tuition|exam|books|udemy|education|academy/i]],
  ["Expenses:Insurance", [/insurance|premium|policy|lic|mutual|cover|renewal/i]],
  ["Expenses:Investment", [/sip|mutual fund|nps|brokerage|stock|investment|trading|etf/i]]
];

export function guessExpenseAccount(
  description: string,
  prefix: string = "Expenses",
  rules: Array<[string, RegExp[]]> = EXPENSE_KEYWORDS
): string {
  const text = _.toString(description || "").trim();
  if (!text) {
    return `${prefix}:Unknown`;
  }

  const normalized = text.toLowerCase();
  for (const [bucket, patterns] of rules) {
    if (patterns.some((pattern) => pattern.test(normalized))) {
      return `${prefix}:${bucket.replace(/^Expenses:/, "")}`;
    }
  }

  return `${prefix}:Unknown`;
}

export function parse(file: File): Promise<Result> {
  let extension = file.name.split(".").pop();
  extension = extension?.toLowerCase();
  if (extension === "csv" || extension === "txt") {
    return parseCSV(file);
  } else if (extension === "xlsx" || extension === "xls") {
    return parseXLSX(file);
  } else if (extension === "pdf") {
    return parsePDF(file);
  }
  throw new Error(`Unsupported file type ${extension}`);
}

export function asRows(result: Result): Array<Record<string, any>> {
  return _.map(result.data, (row, i) => {
    return _.chain(row)
      .map((cell, j) => {
        return [String.fromCharCode(65 + j), cell];
      })
      .concat([["index", i as any]])
      .fromPairs()
      .value();
  });
}

const COLUMN_REFS = _.chain(_.range(65, 90))
  .map((i) => String.fromCharCode(i))
  .map((a) => [a, a])
  .fromPairs()
  .value();

export function render(
  rows: Array<Record<string, any>>,
  template: Handlebars.TemplateDelegate,
  options: { reverse?: boolean; trim?: boolean } = {}
) {
  const output: string[] = [];
  _.each(rows, (row) => {
    let rendered = template(_.assign({ ROW: row, SHEET: rows }, COLUMN_REFS));
    if (options.trim) {
      rendered = _.trim(rendered);
    }
    if (!_.isEmpty(rendered)) {
      output.push(rendered);
    }
  });
  if (options.reverse) {
    output.reverse();
  }

  if (options.trim) {
    return format(output.join("\n\n"));
  } else {
    return format(output.join(""));
  }
}

function parseCSV(file: File): Promise<Result> {
  return new Promise((resolve, reject) => {
    Papa.parse<string[]>(file, {
      skipEmptyLines: true,
      complete: function (results) {
        resolve(results);
      },
      error: function (error) {
        reject(error);
      },
      delimitersToGuess: [",", "\t", "|", ";", Papa.RECORD_SEP, Papa.UNIT_SEP, "^"]
    });
  });
}

async function parseXLSX(file: File): Promise<Result> {
  const buffer = await readFile(file);
  try {
    const sheet = XLSX.read(buffer, { type: "binary" });
    const json = XLSX.utils.sheet_to_json<string[]>(sheet.Sheets[sheet.SheetNames[0]], {
      header: 1,
      blankrows: false,
      rawNumbers: false
    });
    return { data: json };
  } catch (e) {
    if (/password-protected/.test(e.message)) {
      const password = prompt(
        "Please enter the password to open this XLSX file. Press cancel to exit."
      );
      if (password === null) {
        return { data: [], error: "Password required." };
      }

      try {
        const workbook = await XlsxPopulate.fromDataAsync(buffer, { password });
        const sheet = workbook.sheet(0);
        if (sheet) {
          let json = sheet.usedRange().value();
          json = _.map(json, (row) => {
            return _.map(row, (cell) => {
              if (cell) {
                return cell.toString();
              }
              return "";
            });
          });

          return { data: json };
        }
      } catch (e) {
        // follow through to the error below
      }

      return { data: [], error: "Unable to parse Password protected XLSX" };
    }
    throw e;
  }
}

async function parsePDF(file: File): Promise<Result> {
  const buffer = await readFile(file);
  const array = await pdf2array(buffer);
  return { data: array };
}

function readFile(file: File): Promise<ArrayBuffer> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      resolve(event.target.result as ArrayBuffer);
    };
    reader.onerror = (event) => {
      reject(event);
    };
    reader.readAsArrayBuffer(file);
  });
}
