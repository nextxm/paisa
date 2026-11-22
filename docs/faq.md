---
description: "Frequently asked Questions about Paisa: personal finance manager"
hide:
  - navigation
---

# FAQs

## I already use ledger/hledger/beancount. How do I get started?

Go through the [installation](./getting-started/installation.md) docs and get the app working. Go to the
[configuration](./reference/config.md) page (`Tools → Configuration`) and update `journal_path`, `ledger_cli`,
`default_currency`, and `locale`. At this point you should be able to
view your journal data. Read the [accounts](./reference/accounts.md) docs to understand the
account naming conventions followed by Paisa.

## How do I get started from scratch?

You have installed Paisa and tried the **Setup Demo** button on the Dashboard,
and you want to start your own journal without manually typing a decade of history.

Paisa is an accounting application where you can focus on the granularity that
matters most to you. For example, if you do not want to itemize every expense, you can record a single expense summary transaction per
month. Recording just a few entries per month (salary, monthly expenses, investments) already unlocks net worth, savings rate, and projection views—or use the [Smart Importer](./reference/import.md) to ingest bank and credit card statements directly.

## Why is the Tax menu hidden?

The **Tax** submenu under **Planning** (`Tax Harvesting`, `Capital Gains`, `Schedule AL`) is currently shown only when `default_currency` is set to `INR`, because its tax rules and forms (like Schedule AL) are modeled on Indian Income Tax regulations.

## How do I switch between Grouped and Classic navigation menus?

Paisa uses the **Grouped** navigation layout by default (`Cash Flow`, `Net Worth`, `Planning`, `Ledger`, `Tools`). You can toggle between **Grouped** and **Classic** (`Cash Flow`, `Expenses`, `Assets`, `Liabilities`, `Income`, `Planning`, `Ledger`, `More`) at any time using the layout switcher icon in the top-right action bar. Press ++ctrl+k++ (or ++cmd+k++ on Mac) to open the **Command Palette** and jump to any page regardless of layout.

## Can I edit my journal files with VS Code, Vim, or Git outside Paisa?

Yes. Paisa runs a background watcher that checks your main journal and `include`d files on disk. Whenever external edits are detected, the **Sync Journal** icon in the top-right action bar turns amber so you can sync with one click (or run `paisa update --journal`).

## How do I back up all my data?

Open **Tools → Configuration → Tools → Backup & Export** (`/more/config/backup`) and download a `.zip` archive containing your journal files, `paisa.yaml` (with password hashes redacted unless you explicitly opt in), and sheet files. You can ignore `paisa.db` in backups or Git, since it is rebuilt from your journals on sync.

## Mac Desktop app fails with `executable file not found in $PATH` error

Mac Desktop apps are usually started with a different `PATH` than what you
would normally get in a Terminal app. Paisa does a fallback search in
the following folders before throwing the error. Check if your
executable binary is in any of the folders listed below; if not, create
a symbolic link from one of the folders to your executable binary.

```
/bin
/usr/bin
/usr/local/bin
/sbin
/usr/sbin
/opt/homebrew/bin
```
