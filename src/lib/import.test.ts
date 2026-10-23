import { describe, expect, test } from "bun:test";

import {
  parse,
  render,
  asRows,
  guessExpenseAccount,
  consolidateStatementTransactions,
  isDuplicateStatementTransaction
} from "./spreadsheet";
import fs from "fs";
import helpers from "./template_helpers";
import _ from "lodash";
import Handlebars from "handlebars";
import dayjs from "dayjs";
import customParseFormat from "dayjs/plugin/customParseFormat";
dayjs.extend(customParseFormat);
import isSameOrBefore from "dayjs/plugin/isSameOrBefore";
dayjs.extend(isSameOrBefore);
import utc from "dayjs/plugin/utc";
import timezone from "dayjs/plugin/timezone"; // dependent on utc plugin
dayjs.extend(utc);
dayjs.extend(timezone);
import localeData from "dayjs/plugin/localeData";
dayjs.extend(localeData);
import updateLocale from "dayjs/plugin/updateLocale";
dayjs.extend(updateLocale);

Handlebars.registerHelper(
  _.mapValues(helpers, (helper, name) => {
    return function (...args: any[]) {
      try {
        return helper.apply(this, args);
      } catch (e) {
        console.log("Error in helper", name, args, e);
      }
    };
  })
);

describe("import", () => {
  fs.readdirSync("fixture/import").forEach((dir) => {
    test(dir, async () => {
      const files = fs.readdirSync(`fixture/import/${dir}`);
      for (const file of files) {
        const [name, extension] = file.split(".");
        if (extension === "ledger") {
          const inputFile = _.find(files, (f) => f != file && f.startsWith(name));
          if (inputFile.endsWith(".pdf")) {
            break;
          }
          const input = fs.readFileSync(`fixture/import/${dir}/${inputFile}`);
          const output = fs.readFileSync(`fixture/import/${dir}/${file}`).toString();
          const template = fs
            .readFileSync(`internal/model/template/templates/${dir}.handlebars`)
            .toString();

          const compiled = Handlebars.compile(template);
          const result = await parse(new File([new Uint8Array(input)], inputFile));
          const rows = asRows(result);

          const actual = render(rows, compiled, { trim: true });

          expect(actual.replace(/\r/g, "")).toBe(_.trim(output).replace(/\r/g, ""));
        }
      }
    });
  });
});

describe("template helpers", () => {
  test("acronym", () => {
    expect(helpers.acronym("Foo Bar baz")).toBe("FBB");
    expect(helpers.acronym("foo   the bar")).toBe("FB");
    expect(helpers.acronym("Motital S & P 500")).toBe("MSP");
    expect(helpers.acronym("Axis Liquid Growth Direct Plan")).toBe("AL");
  });
});

describe("statement import helpers", () => {
  test("guessExpenseAccount picks a likely category from common merchants", () => {
    expect(guessExpenseAccount("AMAZON PAY")).toBe("Expenses:Shopping");
    expect(guessExpenseAccount("INDIAN OIL PETROL PUMP")).toBe("Expenses:Transportation");
    expect(guessExpenseAccount("BSES BILL PAYMENT")).toBe("Expenses:Utilities");
    expect(guessExpenseAccount("Unknown merchant")).toBe("Expenses:Unknown");
  });

  test("duplicate matching requires same date and payee instead of amount alone", () => {
    expect(
      isDuplicateStatementTransaction(
        { account: "Expenses:Food", amount: 500, description: "Restaurant A", date: "2024-01-10" },
        { account: "Expenses:Food", amount: 500, description: "Restaurant B", date: "2024-01-11" },
        { dateWindowHours: 48, requirePayee: true }
      )
    ).toBe(false);

    expect(
      isDuplicateStatementTransaction(
        { account: "Expenses:Food", amount: 500, description: "Restaurant A", date: "2024-01-10" },
        { account: "Expenses:Food", amount: 500, description: "Restaurant A", date: "2024-01-11" },
        { dateWindowHours: 48, requirePayee: true }
      )
    ).toBe(true);
  });

  test("consolidateStatementTransactions groups by category and month in beancount format", () => {
    const output = consolidateStatementTransactions(
      [
        {
          date: "2026-03-03",
          description: "AMAZON PAY",
          amount: 1299,
          account: "Liabilities:CreditCard:ICICI",
          expenseAccount: "Expenses:Shopping"
        },
        {
          date: "2026-03-12",
          description: "BIG BASKET",
          amount: 860,
          account: "Liabilities:CreditCard:ICICI",
          expenseAccount: "Expenses:Groceries"
        },
        {
          date: "2026-03-20",
          description: "AMAZON PAY",
          amount: 1349,
          account: "Liabilities:CreditCard:ICICI",
          expenseAccount: "Expenses:Shopping"
        },
        {
          date: "2026-04-02",
          description: "Uber trip",
          amount: 420,
          account: "Liabilities:CreditCard:ICICI",
          expenseAccount: "Expenses:Transportation"
        }
      ],
      { defaultAccount: "Liabilities:CreditCard:ICICI", dateWindowHours: 48 }
    );

    expect(output).toContain('2026-03-01 * "Expenses:Shopping"');
    expect(output).toContain('2026-03-01 * "Expenses:Groceries"');
    expect(output).toContain('2026-04-01 * "Expenses:Transportation"');
    expect(output).toContain("Liabilities:CreditCard:ICICI");
  });
});
