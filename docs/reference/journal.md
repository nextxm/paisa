---
description: "The journal format used by Paisa to capture transactions, metadata, etc."
---

# Journal

All your transactions are stored in plain text files called
journal. You can organize your transactions into multiple journal
files. The [journal_path](./config.md) configuration refers your main journal
file. The main journal file can refer other journal files using
[include](https://ledger-cli.org/doc/ledger3.html#index-include) directive. The `#!ledger include` directive supports
wildcards `*` as well. Transactions are sourced **only** from the main
journal file and other journal files included from the main journal
file.


```ledger
include investments.ledger
include expenses/*.ledger
```

## Editor

Paisa comes with a journal editor. It allows you to edit all the files
with the same file extension as your main journal and in the same or
sub directories as your main journal.


## Backup

Paisa tries its best to keep your journal safe. Every time you save in the editor, it creates a backup of
your journal file with a `.backup.{timestamp}`
extension so you can revert to older versions or delete old backups directly from the editor. When editing journals outside Paisa, `JournalWatcher` automatically monitors your files and flags the navbar **Sync** indicator if a change needs to be re-synced.

You can also download a complete `.zip` archive of your journal files, `paisa.yaml` (with account password hashes redacted by default), and `.sheet` files from **Tools → Configuration → Tools → Backup & Export** (`/more/config/backup`).

!!! warning

    It is recommended to keep your journal files and `paisa.yaml`
    under version control or other backup mechanism. Paisa's backup
    mechanism is not a replacement for a proper backup. You can ignore
    `paisa.db` file from your version control system, the data in db
    file can be recreated from your journal files.

## Syntax

The journal syntax of the features you use normally along with paisa
is documented here. Refer [ledger](https://ledger-cli.org/doc/ledger3.html#Journal-Format) documentation for more details.

##### Transaction

```ledger
2022/01/01 Salary
    Income:Salary:Acme      -100,000 INR
    Assets:Checking          100,000 INR
```

A `transaction` should start with a `date` followed by
`description`. Following that you can have 2 or more `postings`. The
posting line should have at least 2 leading spaces. The `account` name
and the `amount` should be separated by at least 2 spaces.

##### Commodity

```ledger
2022/01/07 Investment
    Assets:Checking         -20,000 INR
    Assets:Equity:NIFTY   168.690 NIFTY @ 118.56 INR
```

`commodity` cost can be specified using the `@` syntax. Here `118.56`
is the per unit cost and `168.690` is the quantity you have bought.

##### Comment

```ledger
2023/07/01 Rent
    ; This is a transaction comment
    Expenses:Rent             15,000 INR
    Assets:Checking   ; This is a posting comment
```

Any text after `;` is treated as a comment. Comment can be at whole
transaction level or individual posting level. Comment is also referred
as **note** in many places, both are same.

##### Tags

```ledger
2023/07/01 Rent
    ; Recurring: Rent
    Expenses:Rent             15,000 INR
    Assets:Checking
```

Transactions can be tagged with extra metadata called tags. Tag has
two parts: tag name and value. In the above example, `Recurring` is
the name and `Rent` is the value. Tag should be inside comment.

##### Include

```ledger
include investments.ledger
include expenses/*.ledger
```

Include directive can be used to include other journal files. It
supports wildcards `*`.

## Natural Language Quick Add

Click the **+** (**Quick Add Transaction**) button in the top-right action strip (or press `Ctrl+K` → **Quick Add Transaction**) to capture new entries from free-form text or a structured form targeting `add_journal_path`. For example:

```text
20 Apr, bought 15$ groceries using bmo cc from no frills
```

The parser extracts date, amount, currency, payee, and account candidates, lets you review confidence scores and candidate pills, and appends a validated journal entry. See the full API contract, request/response fields, and confidence handling in [Natural Language Parser API](./parser.md).
