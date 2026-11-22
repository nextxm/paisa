import { describe, expect, test } from "bun:test";
import fs from "fs";

import { applyChanges } from "./bulk_edit";
import type { LedgerFile } from "./utils";
import _ from "lodash";

describe("bulk_editor", () => {
  const before = fs.readFileSync("fixture/main.ledger");
  const transactions = JSON.parse(fs.readFileSync("fixture/main.transactions.json").toString());
  fs.readdirSync("fixture/bulk_edit").forEach((dir) => {
    test(dir, () => {
      const files = fs.readdirSync(`fixture/bulk_edit/${dir}`);
      for (const file of files) {
        const [name, extension] = file.split(".");
        if (extension === "ledger") {
          const args = JSON.parse(
            fs.readFileSync(`fixture/bulk_edit/${dir}/${name}.json`).toString()
          );
          const after = fs
            .readFileSync(`fixture/bulk_edit/${dir}/${name}.ledger`)
            .toString()
            .replace(/\r/g, "");
          const ledgerFile: LedgerFile = {
            type: "file",
            name: "main.ledger",
            content: before.toString().replace(/\r/g, ""),
            versions: []
          };
          const {
            newFiles: [newLedgerFile]
          } = applyChanges([ledgerFile], transactions, dir, args);
          expect(_.trim(newLedgerFile.content)).toBe(_.trim(after));
        }
      }
    });
  });

  const sampleContent = [
    "2024/05/01 * Coffee Shop",
    "    Assets:Cash  -5.00 USD",
    "    Expenses:Food  5.00 USD",
    "",
    "2024/05/02 ! Grocery Store | Weekly shop",
    "    Assets:Cash  -20.00 USD",
    "    Expenses:Groceries  20.00 USD"
  ].join("\n");

  const sampleTxns: any[] = [
    {
      id: "1",
      payee: "Coffee Shop",
      beginLine: 1,
      endLine: 3,
      fileName: "main.ledger",
      note: "",
      postings: [{ account: "Assets:Cash" }, { account: "Expenses:Food" }]
    },
    {
      id: "2",
      payee: "Grocery Store",
      beginLine: 5,
      endLine: 7,
      fileName: "main.ledger",
      note: "",
      postings: [{ account: "Assets:Cash" }, { account: "Expenses:Groceries" }]
    }
  ];

  const makeFile = (content = sampleContent): LedgerFile => ({
    type: "file",
    name: "main.ledger",
    content,
    versions: []
  });

  test("add_tag and remove_tag on ledger transactions", () => {
    const added = applyChanges([makeFile()], [sampleTxns[0]], "add_tag", { tag: "#reimbursable" });
    expect(added.updatedTransactionsCount).toBe(1);
    expect(added.newFiles[0].content).toContain("    ; :reimbursable:");

    // Adding again is idempotent
    const addedAgain = applyChanges(added.newFiles, [{ ...sampleTxns[0], endLine: 4 }], "add_tag", {
      tag: "reimbursable"
    });
    expect(addedAgain.updatedTransactionsCount).toBe(0);

    const removed = applyChanges(added.newFiles, [{ ...sampleTxns[0], endLine: 4 }], "remove_tag", {
      tag: "reimbursable"
    });
    expect(removed.updatedTransactionsCount).toBe(1);
    expect(removed.newFiles[0].content).not.toContain(":reimbursable:");
  });

  test("toggle_status switches cleared/pending", () => {
    const toggled = applyChanges([makeFile()], sampleTxns, "toggle_status", { status: "toggle" });
    expect(toggled.updatedTransactionsCount).toBe(2);
    expect(toggled.newFiles[0].content).toContain("2024/05/01 ! Coffee Shop");
    expect(toggled.newFiles[0].content).toContain("2024/05/02 * Grocery Store | Weekly shop");

    const cleared = applyChanges([makeFile()], sampleTxns, "toggle_status", { status: "*" });
    expect(cleared.updatedTransactionsCount).toBe(1);
    expect(cleared.newFiles[0].content).toContain("2024/05/02 * Grocery Store | Weekly shop");
  });

  test("rename_payee updates payee while preserving narration", () => {
    const renamed = applyChanges([makeFile()], sampleTxns, "rename_payee", {
      oldPayee: "Grocery Store",
      newPayee: "Whole Foods"
    });
    expect(renamed.updatedTransactionsCount).toBe(1);
    expect(renamed.newFiles[0].content).toContain("2024/05/02 ! Whole Foods | Weekly shop");
    expect(renamed.newFiles[0].content).toContain("2024/05/01 * Coffee Shop");
  });

  test("delete removes matched transactions cleanly", () => {
    const deleted = applyChanges([makeFile()], [sampleTxns[0]], "delete", {});
    expect(deleted.updatedTransactionsCount).toBe(1);
    expect(deleted.newFiles[0].content).toBe(
      [
        "2024/05/02 ! Grocery Store | Weekly shop",
        "    Assets:Cash  -20.00 USD",
        "    Expenses:Groceries  20.00 USD"
      ].join("\n")
    );
  });
});
