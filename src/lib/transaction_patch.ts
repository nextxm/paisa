import { format } from "./journal";
import type { Transaction } from "./utils";

export interface EditablePosting {
  account: string;
  amount: string;
  commodity: string;
  suffix: string;
  isCommentOnly?: boolean;
  rawLine?: string;
}

export interface EditableTransaction {
  date: string; // YYYY-MM-DD
  status: "" | "*" | "!";
  payee: string;
  narration: string;
  headerSuffix: string;
  commentLines: string[];
  postings: EditablePosting[];
}

function getDialect(explicitDialect?: string): string {
  if (explicitDialect) return explicitDialect;
  const cli =
    typeof USER_CONFIG !== "undefined"
      ? (USER_CONFIG as { ledger_cli?: string }).ledger_cli
      : undefined;
  if (cli) {
    return cli;
  }
  return "ledger";
}

export function parseRawTransaction(raw: string, fallback?: Transaction): EditableTransaction {
  const lines = raw.replace(/\r/g, "").split("\n");
  const header = lines[0] ?? "";

  let date = fallback?.date ? fallback.date.format("YYYY-MM-DD") : "";
  let status: "" | "*" | "!" = "";
  let payee = fallback?.payee ?? "";
  let narration = "";
  let headerSuffix = "";

  const bcMatch = header.match(
    /^(\d{4}-\d{2}-\d{2})\s+(?:([*!])|txn)\s+"([^"]*)"(?:\s+"([^"]*)")?(.*)$/
  );
  if (bcMatch) {
    date = bcMatch[1];
    status = (bcMatch[2] as "*" | "!" | undefined) ?? "*";
    payee = bcMatch[3];
    narration = bcMatch[4] ?? "";
    headerSuffix = bcMatch[5]?.trim() ?? "";
  } else {
    const ledgerMatch = header.match(
      /^(\d{4}[/-]\d{2}[/-]\d{2})(?:=\d{4}[/-]\d{2}[/-]\d{2})?\s+(?:([*!])\s+)?(.*)$/
    );
    if (ledgerMatch) {
      date = ledgerMatch[1].replace(/\//g, "-");
      status = (ledgerMatch[2] as "*" | "!" | undefined) ?? "";
      let rest = ledgerMatch[3].trim();

      const commentIdx = rest.indexOf(";");
      if (commentIdx >= 0) {
        headerSuffix = rest.slice(commentIdx).trim();
        rest = rest.slice(0, commentIdx).trim();
      }

      const pipeIdx = rest.indexOf("|");
      if (pipeIdx >= 0) {
        payee = rest.slice(0, pipeIdx).trim();
        narration = rest.slice(pipeIdx + 1).trim();
      } else {
        payee = rest;
      }
    }
  }

  const commentLines: string[] = [];
  const postings: EditablePosting[] = [];

  for (let i = 1; i < lines.length; i++) {
    const line = lines[i];
    if (!line.trim()) continue;

    if (/^\s+;/.test(line) && postings.length === 0) {
      commentLines.push(line.trim());
      continue;
    }

    if (/^\s+;/.test(line)) {
      postings.push({
        account: "",
        amount: "",
        commodity: "",
        suffix: "",
        isCommentOnly: true,
        rawLine: line
      });
      continue;
    }

    const postingMatch = line.match(
      /^[ \t]+(.+?)(?:[ \t]{2,}|\t)([+-]?[\d.,]+)(?:[ \t]+([^\s;@]+))?(.*)$/
    );
    if (postingMatch) {
      postings.push({
        account: postingMatch[1].trim(),
        amount: postingMatch[2].replace(/,/g, ""),
        commodity: (postingMatch[3] ?? "").trim(),
        suffix: (postingMatch[4] ?? "").trim()
      });
    } else {
      const accountOnlyMatch = line.match(/^[ \t]+([^\s;][^;]*?)(?:\s*(;.*))?$/);
      if (accountOnlyMatch) {
        postings.push({
          account: accountOnlyMatch[1].trim(),
          amount: "",
          commodity: "",
          suffix: (accountOnlyMatch[2] ?? "").trim()
        });
      }
    }
  }

  if (postings.length === 0 && fallback?.postings?.length) {
    for (const p of fallback.postings) {
      postings.push({
        account: p.account,
        amount: String(p.quantity || p.amount),
        commodity: p.commodity,
        suffix: ""
      });
    }
  }

  return {
    date,
    status,
    payee,
    narration,
    headerSuffix,
    commentLines,
    postings
  };
}

export function serializeEditableTransaction(
  tx: EditableTransaction,
  explicitDialect?: string
): string {
  const dialect = getDialect(explicitDialect);
  const lines: string[] = [];

  if (dialect === "beancount") {
    const statusToken = tx.status || "*";
    const payeePart = tx.payee ? ` ${JSON.stringify(tx.payee.trim())}` : ' ""';
    const narrationPart = tx.narration.trim() ? ` ${JSON.stringify(tx.narration.trim())}` : "";
    const suffixPart = tx.headerSuffix.trim() ? ` ${tx.headerSuffix.trim()}` : "";
    lines.push(`${tx.date} ${statusToken}${payeePart}${narrationPart}${suffixPart}`);
  } else {
    const dateStr = tx.date.replace(/-/g, "/");
    const statusPart = tx.status ? ` ${tx.status}` : "";
    const desc =
      tx.payee.trim() && tx.narration.trim()
        ? `${tx.payee.trim()} | ${tx.narration.trim()}`
        : tx.payee.trim() || tx.narration.trim();
    const descPart = desc ? ` ${desc}` : "";
    const suffixPart = tx.headerSuffix.trim() ? ` ${tx.headerSuffix.trim()}` : "";
    lines.push(`${dateStr}${statusPart}${descPart}${suffixPart}`);
  }

  for (const comment of tx.commentLines) {
    if (comment.trim()) {
      lines.push(`    ${comment.trim()}`);
    }
  }

  for (const p of tx.postings) {
    if (p.isCommentOnly && p.rawLine) {
      lines.push(p.rawLine);
      continue;
    }
    const account = p.account.trim();
    if (!account) continue;

    const amount = p.amount.trim();
    const commodity = p.commodity.trim();
    const suffix = p.suffix.trim();

    if (amount) {
      const amountPart = commodity ? `${amount} ${commodity}` : amount;
      const suffixPart = suffix ? ` ${suffix}` : "";
      lines.push(`    ${account}  ${amountPart}${suffixPart}`);
    } else {
      const suffixPart = suffix ? `  ${suffix}` : "";
      lines.push(`    ${account}${suffixPart}`);
    }
  }

  return format(lines.join("\n"));
}
