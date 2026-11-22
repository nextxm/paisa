---
description: "How to import CSV, TXT, OFX, QFX, XLS, XLSX, or PDF statements into Paisa using Smart Ingestion or Handlebars templates"
---

# Import

Route: `/ledger/import` (**Ledger → Import**)

Paisa lets you import bank and credit card statements in **CSV**, **TXT**, **OFX**, **QFX**, **XLS**, **XLSX**, or **PDF** format directly into your journal. The Import page supports two modes:

1. **Smart Ingestion & Reconciliation** *(default)* — automatic column detection, custom column mapping, rule-based and TF-IDF account categorization, duplicate flagging, and direct commit to your journal.
2. **Classic Template Editor** — Handlebars template-based conversion with live preview and file export.

!!! example "Experimental"
    PDF support is in an experimental stage and may not accurately detect rows on complex layouts.

---

## Part 1: Smart Ingestion & Reconciliation (Default Mode)

### 1. Upload Statement

Drag and drop any `.csv`, `.txt`, `.ofx`, `.qfx`, `.xls`, `.xlsx`, or `.pdf` file into the upload dropzone. Paisa parses the statement (`POST /api/import/statement`), extracts candidate transactions, and detects the source bank or card account from header/footer text (including built-in hints for accounts matching `kotak`/`811`, `icici`, `axis`, `hdfc`, and `sbi` as well as your existing chart of accounts).

### 2. Custom Column Mapping (**Map Columns**)

If a CSV, Excel, or PDF statement uses non-standard column headers or multi-row bank banners:

- Click **Map Columns** (this modal also opens automatically if auto-detection finds zero transactions).
- Select the **Header Row** from the first 12 rows of the file.
- Map **Date Column** (required), **Description / Payee Column**, **Amount Format** (**Separate Debit & Credit columns** or **Single Amount column + optional Dr/Cr indicator**), and optional **Currency Column**.
- Click **Apply & Parse** (or **Reset to Auto-Detect** to clear overrides).

### 3. Visual Rule Studio & Account Suggestions

Paisa categorizes each staged transaction in order of confidence:

1. **User Classification Rules (`Rule Studio`)**: Click **Rule Studio** (`GET/POST/DELETE /api/rules`) to define priority-ordered rules matching payee or memo patterns (`contains`, `exact`, or `regex`) and optional transaction direction (`any`, `debit`, `credit`), assigning a target account, tags, and an optional **Flag for Review** marker. You can also save rules inline from any staged transaction row via **Quick Save Rule**.
2. **Historical Payee & Merchant Taxonomy**: Matches normalized descriptions against your past ledger postings and a built-in merchant taxonomy.
3. **TF-IDF Similarity**: Computes cosine similarity in the browser against your existing transactions to suggest the likeliest expense/income/liability account.

### 4. Duplicate Detection

During ingestion, staged rows are scored against existing journal postings. Rows scoring `70` or higher (same amount, close date within 7 days, and matching payee/memo context) are flagged as **Possible Duplicates** and deselected by default so you do not accidentally double-import an already recorded transaction.

### 5. Review & Commit to Ledger

1. Select your **Source / Bank Account** (e.g., `Assets:Checking:ICICI` or `Liabilities:CreditCard:Freedom`).
2. Check or uncheck staged transactions and adjust any target accounts.
3. Click **Review & Commit** to open **Review Generated Ledger Entries**, where you can inspect or edit the generated transactions and toggle **Club debits by date** (groups multiple same-day debits into one consolidated multi-posting transaction for Ledger/hledger; skipped automatically for Beancount).
4. Click **Confirm & Append to Ledger** (`POST /api/import/commit`) to append the entries to `add_journal_path` (falling back to `journal_path` when `add_journal_path` is not configured) and validate the journal. Remember to **Sync Journal** afterward so the newly appended transactions appear across reports.

---

## Part 2: Classic Template Editor

Switch to **Classic Template Editor** at the top of the Import page if you prefer declarative Handlebars templates for full structural control over generated transactions. The classic interface has three components:

1. **File Preview** – Drag and drop CSV/TXT/XLS/XLSX/PDF files to preview raw tabular rows.
2. **Ledger Preview** – Live rendered ledger output.
3. **Template Editor** – Edit the Handlebars template.

Each row in the file is processed against the template:

```handlebars
{{#if (and (isDate ROW.A "DD/MM/YYYY") (isBlank ROW.G))}}
 {{date ROW.A "DD/MM/YYYY"}} {{ROW.C}}
    {{predictAccount prefix="Expenses"}}		{{ROW.F}} INR
    Assets:Checking
{{/if}}
```

This converts a matching row into a transaction such as:

```ledger
2023/03/28 AMAZON HTTP://WWW.AM IN
    Expenses:Shopping		249.00 INR
    Assets:Checking
```

### Template Management

Paisa ships with built-in templates and lets you create custom ones with **Save As** (stored in `paisa.yaml` under `import_templates`).

### Template Data

1. **ROW** – The current row being processed. Columns are keyed by letters (`ROW.A`, `ROW.B`, …) and `ROW.index` holds the 0-based row index.

<details>
  <summary>Example</summary>

```json
{
  "A": "28/03/2023",
  "B": "7357680821",
  "C": "AMAZON HTTP://WWW.AM IN",
  "D": "12",
  "E": "0",
  "F": "249.00",
  "G": "",
  "index": 6
}
```
</details>

2. **SHEET** – The entire parsed sheet as an array of row objects (e.g., `SHEET.5.A`).

<details>
  <summary>Example</summary>

```json
[
    {
        "A": "Accountno:",
        "B": "49493xxx003030",
        "index": 0
    },
    {
        "A": "Customer Name:",
        "B": "MR John Doe",
        "index": 1
    },
    {
        "A": "Address:",
        "B": "1234, ABC Street, XYZ City, 123456",
        "index": 2
    },
    {
        "A": "Transaction Details:",
        "index": 3
    },
    {
        "A": "Date",
        "B": "Sr.No.",
        "C": "Transaction Details",
        "D": "Reward Point Header",
        "E": "Intl.Amount",
        "F": "Amount(in Rs)",
        "G": "BillingAmountSign",
        "index": 4
    },
    {
        "A": "49493xxx003030",
        "index": 5
    },
    {
        "A": "28/03/2023",
        "B": "7357680821",
        "C": "AMAZON HTTP://WWW.AM IN",
        "D": "12",
        "E": "0",
        "F": "249.00",
        "G": "",
        "index": 6
    },
    {
        "A": "28/03/2023",
        "B": "7357821997",
        "C": "AMAZON HTTP://WWW.AM IN",
        "D": "28",
        "E": "0",
        "F": "575.00",
        "G": "",
        "index": 7
    }
]
```
</details>

### Template Helpers

#### `#!typescript eq(a: any, b: any): boolean`
Checks if the two values are equal.

#### `#!typescript not(value: any): boolean`
Negates the given value.

#### `#!typescript and(...args: any[]): boolean`
Returns true if all the arguments are true.

#### `#!typescript or(...args: any[]): boolean`
Returns true if any of the arguments are true.

#### `#!typescript gte(a: string | number, b: string | number): boolean`
Checks if `a` is greater than or equal to `b`.

#### `#!typescript gt(a: string | number, b: string | number): boolean`
Checks if `a` is greater than `b`.

#### `#!typescript lte(a: string | number, b: string | number): boolean`
Checks if `a` is less than or equal to `b`.

#### `#!typescript lt(a: string | number, b: string | number): boolean`
Checks if `a` is less than `b`.

#### `#!typescript negate(value: string | number): number`
Negates the given value. For example, `negate("123.45")` returns `-123.45`.

#### `#!typescript amount(str: string, {default?: string}): string`
Converts the given string to a valid amount. If blank, uses `default`. Examples: `(0.9534)` → `-0.9534`, `₹ 1,234.56` → `1234.56`.

#### `#!typescript round(str: string, {precision?: number}): number`
Rounds the given value to `precision` decimal places (defaults to `0`).

#### `#!typescript isDate(str: string, format: string): boolean`
Checks if `str` is a valid date in `format` (see [Day.js format tokens](https://day.js.org/docs/en/parse/string-format#list-of-all-available-parsing-tokens)).

#### `#!typescript predictAccount(...terms: string[], {prefix?: string}): string`
Finds transactions in your journal with similar descriptions and picks the top matching account. If `terms` are omitted, the entire `ROW` is used. `prefix` filters candidate accounts; returns `Unknown` if no match is found.

```handlebars
{{predictAccount prefix="Income"}}
{{predictAccount ROW.C ROW.F prefix="Income"}}
```

#### `#!typescript isBlank(str: string): boolean`
Checks if the given string is blank.

#### `#!typescript date(str: string, format: string): string`
Parses `str` using `format` and returns `YYYY/MM/DD`.

#### `#!typescript findAbove(column: string, {regexp?: string}): string`
Finds the first non-empty cell above the current row in `column` (optionally matching `regexp`).

```handlebars
{{findAbove B regexp="LIMITED"}}
```

#### `#!typescript findBelow(column: string, {regexp?: string}): string`
Finds the first non-empty cell below the current row in `column` (optionally matching `regexp`).

```handlebars
{{findBelow B regexp="LIMITED"}}
```

#### `#!typescript acronym(str: string): string`
Returns an acronym suitable for a commodity symbol (e.g., `UTI Nifty Next 50 Index Growth Direct Plan` → `UNNI`).

#### `#!typescript trim(str: string): string`
Trims leading and trailing whitespace.

#### `#!typescript replace(str: string, search: string, replace: string): string`
Replaces `search` with `replace` inside `str`.

#### `#!typescript regexpTest(str: string, regexp: string): boolean`
Tests whether `str` matches `regexp`.

#### `#!typescript regexpMatch(str: string, regexp: string, {group?: number}): string`
Extracts a capture group (default `0`): `#!handlebars {{regexpMatch ROW.C "(.*) Growth Direct Plan" group=1}}`.

#### `#!typescript textRange(fromColumn: string, toColumn: string, {separator?: string}): string`
Joins columns from `fromColumn` through `toColumn` with `separator` (defaults to `" "`).

```handlebars
{{textRange A C separator=" "}}
```

#### `#!typescript toLowerCase(str: string): string`
Converts `str` to lowercase.

#### `#!typescript toUpperCase(str: string): string`
Converts `str` to uppercase.

#### `#!typescript capitalize(str: string): string`
Capitalizes `str`.

#### `#!typescript match(str: string, {[string]: string}): string`
Matches `str` against a map of `Account="regex"` pairs:

```handlebars
{{or (match ROW.C Expenses:Shopping="Amazon|Flipkart" Expenses:Groceries="BigBasket")
     "Expenses:Unknown"}}
```

### Import Preview (Template Mode)

Click **Preview** to run `POST /api/import/preview` in dry-run mode. Paisa shows a row-by-row validation table with **valid/invalid** badges, error messages, and per-row selection checkboxes before saving output to a journal file.

### Import Presets

Frequently used delimiter/column configurations can be saved as named presets (`GET/POST/DELETE /api/import/presets`) in Paisa's SQLite database. Built-in presets cannot be deleted:

| Preset Name | Format description |
|---|---|
| `Generic Bank CSV` | Simple date / description / amount CSV |
| `Chase Credit Card CSV` | Chase card statement CSV export |
| `SBI Account Statement CSV` | State Bank of India account statement |
| `ICICI Credit Card CSV` | ICICI Bank credit card statement |
| `Kotak Bank Statement CSV` | Kotak Mahindra Bank account statement |

Custom Handlebars templates can also be configured in `paisa.yaml`:

```yaml
import_templates:
  - name: My Bank Statement
    content: |
      {{#if (isDate ROW.A "DD/MM/YYYY")}}
        {{date ROW.A "DD/MM/YYYY"}} {{ROW.C}}
          {{predictAccount prefix="Expenses"}}    {{amount ROW.F}} INR
          Assets:Checking
      {{/if}}
```
