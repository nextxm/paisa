import _ from "lodash";
import { format } from "./journal";
import type { LedgerFile, Transaction } from "./utils";

interface OperationResult {
  updated: boolean;
  content: string;
  deleted?: boolean;
}

export function applyChanges(
  files: LedgerFile[],
  transactions: Transaction[],
  operation: string,
  args: any
) {
  let updatedTransactionsCount = 0;
  const transactionsGrouped = _.groupBy(transactions, (t) => t.fileName);
  const newFiles = _.map(files, (file) => {
    const fileTransactions = transactionsGrouped[file.name] || [];

    const lines = file.content.replace(/\r/g, "").split("\n");
    const sortedTransactions = _.sortBy(fileTransactions, (t) => t.beginLine);
    let lastLine = 0;
    const newLines: string[] = [];
    for (const transaction of sortedTransactions) {
      newLines.push(...lines.slice(lastLine, transaction.beginLine - 1));
      const oldLines = lines.slice(transaction.beginLine - 1, transaction.endLine);
      const rawText = oldLines.join("\n");

      let result: OperationResult = { updated: false, content: rawText };
      switch (operation) {
        case "rename_account":
          result = renameAccount(rawText, transaction, args);
          break;
        case "add_tag":
          result = addTag(rawText, args);
          break;
        case "remove_tag":
          result = removeTag(rawText, args);
          break;
        case "toggle_status":
          result = toggleStatus(rawText, args);
          break;
        case "rename_payee":
          result = renamePayee(rawText, transaction, args);
          break;
        case "delete":
          result = { updated: true, content: "", deleted: true };
          break;
      }

      if (result.deleted) {
        updatedTransactionsCount++;
        lastLine = transaction.endLine;
        if (lastLine < lines.length && lines[lastLine].trim() === "") {
          lastLine++;
        }
      } else {
        newLines.push(result.content);
        if (result.updated) {
          updatedTransactionsCount++;
        }
        lastLine = transaction.endLine;
      }
    }

    newLines.push(...lines.slice(lastLine));

    return _.merge({}, file, { content: newLines.join("\n") });
  });

  return {
    newFiles,
    updatedTransactionsCount
  };
}

interface RenameAccountArgs {
  oldAccountName: string;
  newAccountName: string;
}

function renameAccount(
  text: string,
  transaction: Transaction,
  args: RenameAccountArgs
): OperationResult {
  if (!args.oldAccountName || !args.newAccountName || args.oldAccountName === args.newAccountName) {
    return { updated: false, content: text };
  }
  const found = _.some(transaction.postings, (p) => p.account === args.oldAccountName);
  if (!found) {
    return { updated: false, content: text };
  }

  const regex = new RegExp(
    `^((?:\t|\\s{2})\\s*)(${escapeRegExp(args.oldAccountName)})((?:\t|\\s{2}).*|\\s*)$`
  );
  const lines = text.replace(/\r/g, "").split("\n");
  const content = format(
    _.map(lines, (line) => {
      return line.replace(regex, `$1${args.newAccountName}$3`);
    }).join("\n")
  );

  return { updated: true, content };
}

function normalizeTag(rawTag: string): string {
  return (rawTag || "")
    .trim()
    .replace(/^[#:]+/, "")
    .replace(/:+$/, "");
}

function isBeancountHeader(header: string): boolean {
  const cli =
    typeof USER_CONFIG !== "undefined"
      ? (USER_CONFIG as { ledger_cli?: string }).ledger_cli
      : undefined;
  if (cli === "beancount") {
    return true;
  }
  return /^\d{4}-\d{2}-\d{2}\s+(?:[*!]|txn)\s+"/.test(header);
}

function addTag(text: string, args: { tag?: string }): OperationResult {
  const tag = normalizeTag(args?.tag || "");
  if (!tag) {
    return { updated: false, content: text };
  }

  const lines = text.replace(/\r/g, "").split("\n");
  if (lines.length === 0) return { updated: false, content: text };

  const beancount = isBeancountHeader(lines[0]);
  const tagPattern = beancount
    ? new RegExp(`(?:^|\\s)#${escapeRegExp(tag)}(?:\\s|$)`)
    : new RegExp(`:${escapeRegExp(tag)}:`);

  if (lines.some((line) => tagPattern.test(line))) {
    return { updated: false, content: text };
  }

  if (beancount) {
    const commentIdx = lines[0].indexOf(";");
    if (commentIdx >= 0) {
      const before = lines[0].slice(0, commentIdx).trimEnd();
      const after = lines[0].slice(commentIdx);
      lines[0] = `${before} #${tag} ${after}`;
    } else {
      lines[0] = `${lines[0].trimEnd()} #${tag}`;
    }
    return { updated: true, content: lines.join("\n") };
  }

  const existingTagLineIdx = lines.findIndex(
    (line, idx) => idx > 0 && /^\s+;\s*:[^:\s]+:/.test(line)
  );
  if (existingTagLineIdx >= 0) {
    lines[existingTagLineIdx] = `${lines[existingTagLineIdx].trimEnd()}${tag}:`;
  } else {
    lines.splice(1, 0, `    ; :${tag}:`);
  }

  return { updated: true, content: lines.join("\n") };
}

function removeTag(text: string, args: { tag?: string }): OperationResult {
  const tag = normalizeTag(args?.tag || "");
  if (!tag) {
    return { updated: false, content: text };
  }

  const lines = text.replace(/\r/g, "").split("\n");
  let changed = false;
  const escaped = escapeRegExp(tag);
  const colonTagRegex = new RegExp(`:${escaped}(?=:)`, "g");
  const singleColonTagRegex = new RegExp(`:${escaped}:`, "g");
  const hashTagRegex = new RegExp(`(^|\\s)#${escaped}(?=\\s|$)`, "g");

  const updatedLines: string[] = [];
  for (let i = 0; i < lines.length; i++) {
    let line = lines[i];
    const orig = line;

    if (singleColonTagRegex.test(line)) {
      // Remove :tag: while preserving shared colons like :a:b: -> :a:
      line = line.replace(colonTagRegex, "");
      if (line === orig) {
        line = line.replace(singleColonTagRegex, "");
      }
      // Collapse leftover lone ":" from ":a:"
      line = line.replace(/^(\s+;\s*):+\s*$/, "$1");
    }

    if (hashTagRegex.test(line)) {
      line = line
        .replace(hashTagRegex, "$1")
        .replace(/\s{2,}/g, (m, offset) => (offset === 0 ? m : " "))
        .trimEnd();
    }

    if (line !== orig) {
      changed = true;
    }

    if (i > 0 && /^\s+;\s*$/.test(line)) {
      changed = true;
      continue;
    }

    updatedLines.push(line);
  }

  if (!changed) {
    return { updated: false, content: text };
  }

  return { updated: true, content: updatedLines.join("\n") };
}

function toggleStatus(text: string, args: { status?: "*" | "!" | "" | "toggle" }): OperationResult {
  const lines = text.replace(/\r/g, "").split("\n");
  if (lines.length === 0) return { updated: false, content: text };

  const match = lines[0].match(
    /^(\d{4}[/-]\d{2}[/-]\d{2}(?:=\d{4}[/-]\d{2}[/-]\d{2})?)\s+(?:([*!])\s+)?(.*)$/
  );
  if (!match) {
    return { updated: false, content: text };
  }

  const [, datePart, currentStatus = "", rest] = match;
  let nextStatus: string;
  if (args?.status === "toggle") {
    if (currentStatus === "*") nextStatus = "!";
    else nextStatus = "*";
  } else {
    nextStatus = args?.status ?? "*";
  }

  if (nextStatus === currentStatus) {
    return { updated: false, content: text };
  }

  lines[0] = nextStatus ? `${datePart} ${nextStatus} ${rest}` : `${datePart} ${rest}`;
  return { updated: true, content: lines.join("\n") };
}

function renamePayee(
  text: string,
  transaction: Transaction,
  args: { oldPayee?: string; newPayee?: string }
): OperationResult {
  const newPayee = (args?.newPayee || "").trim();
  const oldPayee = (args?.oldPayee || "").trim();
  if (!newPayee) {
    return { updated: false, content: text };
  }
  if (oldPayee && transaction.payee !== oldPayee) {
    return { updated: false, content: text };
  }

  const lines = text.replace(/\r/g, "").split("\n");
  if (lines.length === 0) return { updated: false, content: text };

  const header = lines[0];
  if (isBeancountHeader(header)) {
    const bcMatch = header.match(/^(\d{4}-\d{2}-\d{2}\s+(?:[*!]|txn)\s+)"([^"]*)"(.*)$/);
    if (!bcMatch) return { updated: false, content: text };
    const [, prefix, currentPayee, suffix] = bcMatch;
    if (currentPayee === newPayee) return { updated: false, content: text };
    lines[0] = `${prefix}${JSON.stringify(newPayee)}${suffix}`;
    return { updated: true, content: lines.join("\n") };
  }

  const ledgerMatch = header.match(
    /^(\d{4}[/-]\d{2}[/-]\d{2}(?:=\d{4}[/-]\d{2}[/-]\d{2})?\s+(?:[*!]\s+)?(?:\([^)]*\)\s+)?)(.+?)(\s*\|.*|\s*;.*|\s*)$/
  );
  if (!ledgerMatch) {
    return { updated: false, content: text };
  }

  const [, prefix, currentPayee, suffix] = ledgerMatch;
  if (currentPayee.trim() === newPayee) {
    return { updated: false, content: text };
  }

  lines[0] = `${prefix}${newPayee}${suffix}`;
  return { updated: true, content: lines.join("\n") };
}

function escapeRegExp(text: string) {
  return text.replace(/[-[\]{}()*+?.,\\^$|#\s]/g, "\\$&");
}
