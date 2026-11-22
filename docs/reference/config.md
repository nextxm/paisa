---
description: "Full list of configuration options supported by Paisa along with their schema, sidebar sections, and Backup & Export"
---

# Configuration

All configuration for Paisa is stored in a YAML file named `paisa.yaml`. You can edit settings through **Tools → Configuration** (`/more/config`; Classic: **More → Configuration**), which organizes options into four sidebar groups:

- **General**: Core Setup, Display & Format, Security & Auth
- **Finance**: Commodities & Prices, Allocation Targets, Credit Cards, Schedule AL
- **Planning**: Goals, Budget
- **Tools**: Import Templates, Accounts, Doctor Rules, Backup & Export, Firefly III, Labs, Advanced

On mobile viewports, the configuration sidebar automatically collapses when a section is selected.

Paisa searches for `paisa.yaml` in the following order:

1. `PAISA_CONFIG` environment variable
2. `--config` CLI flag
3. Current working directory
4. `paisa/paisa.yaml` inside the user's `Documents` folder

If no configuration file is found, Paisa creates a default `paisa/paisa.yaml` inside your `Documents` folder. Because the defaults are tuned for Indian locales (`INR`, `en-IN`, financial year starting April), international users should adjust `default_currency`, `locale`, and `financial_year_starting_month`.

### Account Matching Syntax

In most configuration lists (`checking_accounts`, `inactive_accounts`, `allocation_targets`, `goals`, `schedule_al`), you can specify exact account names (such as `#!ledger Assets:Equity:AAPL`), wildcard patterns (`#!ledger Assets:Equity:*`), or negations (`#!ledger !Expenses:Tax`). When using negation, use negations across the entire list—do not mix positive and negated patterns in the same list.

*(Note: `doctor.*.pattern` uses SQL `LIKE` syntax with `%` wildcards—see [Doctor Rules](#doctor-rules).)*

```yaml
# Path to your main journal file (absolute or relative to paisa.yaml).
# Sub-files loaded via `include` must reside in the same directory or a subdirectory.
# REQUIRED
journal_path: /home/john/Documents/paisa/main.ledger

# Path to a journal file where transactions created via Quick Add,
# Smart Importer, or Firefly III webhooks are appended. Created automatically
# if missing; ensure your main journal `include`s it.
# OPTIONAL
add_journal_path: /home/john/Documents/paisa/added.ledger

# Path to your SQLite cache database file (created automatically if missing).
# REQUIRED
db_path: /home/john/Documents/paisa/paisa.db

# Path to your sheets directory. Defaults to the directory containing journal_path.
# OPTIONAL
sheets_directory: sheets

# Run Paisa in read-only mode (blocks mutating API endpoints).
# OPTIONAL, DEFAULT: false
readonly: false

# Ledger engine to invoke
# OPTIONAL, DEFAULT: ledger, ENUM: "", ledger, hledger, beancount
ledger_cli: ledger

# Default reporting currency. Ensure FX rates to default_currency exist for foreign holdings.
# OPTIONAL, DEFAULT: INR
default_currency: INR

# Additional commodities treated as currencies rather than securities.
# The default_currency is always included automatically.
# OPTIONAL, DEFAULT: []
currencies: []

# Accounts treated as checking accounts on the Dashboard and balance calculations.
# Supports wildcards (e.g., Assets:Checking:*).
# OPTIONAL, DEFAULT: ["Assets:Checking"]
checking_accounts:
  - Assets:Checking

# Accounts treated as inactive (hidden when "Filter Inactive" is enabled).
# Supports wildcards (e.g., Assets:OldBank:*).
# OPTIONAL, DEFAULT: []
inactive_accounts: []

# Decimal precision displayed in the UI (0–4). Does not alter journal entries.
# OPTIONAL, DEFAULT: 0
display_precision: 0

# Column used to align amounts when formatting entries in the Editor (40–100).
# OPTIONAL, DEFAULT: 52
amount_alignment_column: 52

# Browser locale used for number formatting (pattern: ^[a-z]{2}-[A-Z]{2}$).
# OPTIONAL, DEFAULT: en-IN
locale: en-IN

# IANA time zone for parsing and formatting dates (e.g., Asia/Kolkata, America/New_York).
# OPTIONAL, DEFAULT: system local timezone
time_zone: ""

# First month of the financial year (1 = January, 4 = April, etc.).
# OPTIONAL, DEFAULT: 4
financial_year_starting_month: 4

# First day of the week (0 = Sunday, 1 = Monday, ..., 6 = Saturday).
# OPTIONAL, DEFAULT: 0
week_starting_day: 0

# Enforce strict account/commodity declarations (--pedantic in ledger, --strict in hledger).
# Not used for beancount.
# OPTIONAL, ENUM: "", yes, no DEFAULT: no
strict: "no"

# Log outbound HTTP requests/responses to price and reference-data providers.
# Can also be toggled live without saving the full form in Tools -> Configuration -> Tools -> Advanced.
# OPTIONAL, DEFAULT: false
provider_debug_http: false

# Disable pair-aware multi-currency cross-rate resolution and report_currency conversion.
# OPTIONAL, DEFAULT: false
disable_multi_currency_prices: false

# Suppress log warnings when a commodity price is not found in the database.
# OPTIONAL, DEFAULT: false
disable_missing_price_logging: false

## Budget
budget:
  # Rollover unspent budget to the next month
  # OPTIONAL, ENUM: "", yes, no DEFAULT: yes
  rollover: "yes"

## Goals
goals:
  # Retirement goals
  retirement:
      # Goal name (REQUIRED)
    - name: Retirement
      # Goal icon (REQUIRED)
      icon: mdi:palm-tree
      # Safe Withdrawal Rate in % (REQUIRED, 1–10)
      swr: 4
      # Display priority — higher priority appears first (OPTIONAL)
      priority: 10
      # List of expense accounts used to estimate 3-year average annual expenses (OPTIONAL)
      expenses:
        - Expenses:Clothing
        - Expenses:Education
        - Expenses:Entertainment
        - Expenses:Food
        - Expenses:Gift
        - Expenses:Insurance
        - Expenses:Misc
        - Expenses:Restaurant
        - Expenses:Shopping
        - Expenses:Utilities
      # Accounts holding retirement savings (REQUIRED)
      savings:
        - Assets:Equity:*
        - Assets:Debt:*
      # Optional fixed annual expense override (positive integer)
      yearly_expenses: 0

  # Savings goals
  savings:
      # Goal name (REQUIRED)
    - name: House
      # Goal icon (REQUIRED)
      icon: fluent-emoji-high-contrast:house-with-garden
      # Target amount (REQUIRED)
      target: 100000
      # Target date in YYYY-MM-DD (OPTIONAL; requires `rate` when set)
      target_date: "2030-01-01"
      # Expected monthly investment amount (OPTIONAL)
      payment_per_period: 0
      # Expected annual return % (OPTIONAL; required if target_date is set)
      rate: 5
      # Display priority — higher priority appears first (OPTIONAL)
      priority: 5
      # Accounts holding goal savings (REQUIRED)
      accounts:
        - Assets:Equity:*

  # Milestone & recurring life goals (used by Planning -> Life Plan)
  life:
    - name: College Tuition
      icon: mdi:school
      type: recurring # ENUM: milestone, recurring
      start_date: "2032-06" # YYYY-MM (required for recurring)
      end_date: "2036-05"   # YYYY-MM (required for recurring)
      frequency: yearly     # ENUM: monthly, quarterly, yearly
      inflation_rate: 8     # Optional per-goal inflation % override (or null)
      priority: 1
      monthly_allocation: 400000 # Allocation per occurrence (required for recurring)
      funded_by:
        - Assets:Equity:*

## Schedule AL (India Income Tax reporting)
# OPTIONAL, DEFAULT: []
schedule_al:
  # REQUIRED code ENUM: immovable, metal, art, vehicle, bank, share,
  # insurance, loan, cash, liability
  - code: metal
    accounts:
      - Assets:Gold
  - code: bank
    accounts:
      - Assets:Checking
      - Assets:Debt:Cash:FD
  - code: share
    accounts:
      - Assets:Equity:*
  - code: insurance
    accounts:
      - Assets:Debt:Insurance

## Allocation Targets
# OPTIONAL, DEFAULT: []
allocation_targets:
  - name: Debt
    target: 30
    drift: 5 # Optional allowed drift band percentage (0–50)
    accounts:
      - Assets:Debt:*
      - Assets:Checking
  - name: Equity
    target: 60
    drift: 5
    accounts:
      - Assets:Equity:*
    commodities: [] # Optional list of specific commodity tickers
  - name: Others
    target: 10
    accounts:
      - Assets:Gold
      - Assets:RealEstate

## Commodities
# OPTIONAL, DEFAULT: []
commodities:
  - name: NASDAQ
    # REQUIRED ENUM: mutualfund, stock, nps, metal, unknown
    type: mutualfund
    price:
      # REQUIRED ENUM: in-mfapi, com-yahoo, com-purifiedbytes-nps,
      # com-purifiedbytes-metal, co-alphavantage
      provider: in-mfapi
      code: 145552
    harvest: 1095
    # OPTIONAL ENUM: "", debt, equity, equity65, equity35, unlisted_equity
    tax_category: debt
  - name: NIFTY
    type: mutualfund
    price:
      provider: in-mfapi
      code: 120716
    harvest: 365
    tax_category: equity65
  - name: APPLE
    type: stock
    price:
      provider: com-yahoo
      code: AAPL
    harvest: 1095
    tax_category: equity65
  - name: GOLD
    type: metal
    price:
      provider: com-purifiedbytes-metal
      code: gold-999

## Import Templates
# OPTIONAL, DEFAULT: []
import_templates:
  - name: SBI Account Statement
    content: |
      {{#if (isDate ROW.A "D MMM YYYY")}}
        {{date ROW.A "D MMM YYYY"}} {{ROW.C}}
        {{#if (isBlank ROW.F)}}
          {{predictAccount prefix="Expenses"}}      {{amount ROW.E}} INR
          Assets:Checking:SBI
        {{else}}
          Assets:Checking:SBI                       {{amount ROW.F}} INR
          {{predictAccount prefix="Income"}}
        {{/if}}
      {{/if}}

## Accounts: display icons and inactive flag
# OPTIONAL, DEFAULT: []
accounts:
  - name: Liabilities:CreditCard:IDFC
    icon: arcticons:idfc-first-bank
    inactive: false

## User Authentication
# If empty, authentication is disabled.
# OPTIONAL, DEFAULT: []
user_accounts:
  - username: john.doe
    # SHA-256 hex digest prefixed with sha256:
    password: sha256:a96dc73edd639b1c711b006e714bd2ff5bf5c1aecd77d0b3c3370403c66d58e5

# Allow legacy X-Auth: username:password header (disabled by default; not recommended).
# OPTIONAL, DEFAULT: false
allow_legacy_auth: false

## Credit Cards
# OPTIONAL, DEFAULT: []
credit_cards:
  - account: Liabilities:CreditCard:Freedom
    credit_limit: 150000
    statement_end_day: 8
    due_day: 20
    network: visa # ENUM: visa, mastercard, dinersclub, amex, rupay, jcb, discover
    number: "0007"
    expiration_date: "2029-05-01"

## Firefly III Integration
# OPTIONAL
firefly:
  url: https://firefly.example.com
  token: your-personal-access-token
  ignore_accounts:
    - Assets:Cash

## Labs
# Experimental features hidden by default.
# OPTIONAL, DEFAULT: {}
labs:
  firefly_reconcile: false
```

*(Note: Although the `local-json` price provider is implemented in the scraper package and documented in [Commodities](./commodities.md), `local-json` is not yet included in `schema.json`'s provider enum and will currently fail config validation if set in `paisa.yaml`.)*

## Doctor Rules

Automated health diagnostic rules evaluated by [Data Health (Doctor)](./doctor.md):

```yaml
doctor:
  negative_balance:
    enabled: "yes"
    pattern:
      - "Assets:%"
  non_credit_account:
    enabled: "yes"
    pattern:
      - "Income:%"
  non_debit_account:
    enabled: "yes"
    pattern:
      - "Expenses:%"
  exchange_price_missing:
    enabled: "yes"
  unit_price_mismatch:
    enabled: "yes"
  asset_allocation_missing:
    enabled: "yes"
    pattern:
      - "Assets:%"
```

`enabled` accepts `"yes"` or `"no"`, and `pattern` uses SQL `LIKE` `%` wildcards.

## Backup & Export

Export a full `.zip` archive of your financial data via **Tools → Configuration → Tools → Backup & Export** (`/more/config/backup`, `GET /api/backup/export`):

- Includes your main and `include`d journal files (plus `add_journal_path`), `paisa.yaml`, and `.sheet`/`.json`/`.csv` files from `sheets_directory`.
- Excludes `paisa.db` (which is rebuilt from journals on sync).
- **Password Redaction**: By default, `user_accounts[].password` hashes in the exported `paisa.yaml` are replaced with `[REDACTED]`. Check **Include secrets** (`?include_secrets=true`) only if you intentionally want hashes preserved in an encrypted/secure backup.
