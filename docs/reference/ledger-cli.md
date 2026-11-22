---
description: "Paisa supports all the top 3 plain text account clients namely ledger, hledger and beancount"
---

# Ledger CLI

Paisa is compatible with [ledger](https://www.ledger-cli.org), [hledger](https://hledger.org/) and
[beancount](https://beancount.github.io/). By default paisa will try to use **ledger**, if you
prefer to use **hledger** or **beancount** instead, change the
`ledger_cli` value in [paisa.yaml](./config.md)

```yaml
# OPTIONAL, DEFAULT: ledger, ENUM: ledger, hledger, beancount
ledger_cli: hledger
```

Paisa ships with ledger binary. If you use hledger or beancount, make
sure that the binaries are installed.

## Unavailable Features

Some of the features that are available in Paisa are not supported
by **hledger** and **beancount**

### hledger

* [Recurring](./recurring.md) (partial) - hledger supports automated transactions,
  but it doesn't allow to add **only** metadata to transaction. So
  some of the examples in the recurring section will not work with
  hledger. It's possible to add metadata to transactions manually.


### beancount

* [Budget](./budget.md) - budget relies on periodic transactions, which is not
    supported by beancount.

* [Recurring](./recurring.md) (partial) - beancount doesn't support automated
  transactions. It's possible to add metadata to transactions
  manually. Use lowercase for tag name.

* Search filter in Transactions and Postings pages doesn't support note
  [property](./bulk-edit.md#property).

## Dialect-Aware Features

Paisa adapts output and import formatting automatically to your configured `ledger_cli`:

* **Smart Importer & Quick Add**: When writing entries for `beancount`, Paisa uses `YYYY-MM-DD * "Payee" "Narration"` syntax, `#tag` tags (instead of `; Tag:` comments), and automatically skips `group_by_date` debit consolidation so multi-posting entries match Beancount conventions.
* **Price History Export**: Both the **Commodity Prices** UI export and the `paisa export` CLI support `--format ledger`, `--format hledger`, and `--format beancount` (`P` vs. `price` directives).
