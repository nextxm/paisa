---
description: "Documentation on how to search and edit transactions in bulk in Paisa, an open source personal finance manager"
---

# Transactions, Filtering & Bulk Edit

Open **Ledger → Transactions** (`/ledger/transaction`) or **Ledger → Postings Register** (Classic: **Ledger → Postings**, `/ledger/posting`) to inspect, filter, export, and bulk-modify journal entries. You can also press `Ctrl+K` anywhere and select **Search Transactions** to jump directly to `/ledger/transaction` with the search bar focused.

## Transaction Filter Bar & Saved Searches

Both the Transactions and Postings views include a filter bar with instant filtering and optional advanced filters:

- **Search bar (`q`)**: Filter by payee or narration text.
- **Expandable Filters** (`Filters` button): Filter by **Min amount** (`amount_min`), **Max amount** (`amount_max`), **From/To date** (`date_from`, `date_to`), **Account** (`account`), and **Commodity** (`commodity`).
- **Shareable URL query parameters**: `/ledger/transaction` reads `q`, `amount_min`, `amount_max`, `account`, `commodity`, `date_from`, and `date_to` from the URL query string on load.
- **Saved Searches** (`/ledger/transaction`): Click the bookmark icon to save the current filter set under a custom name (stored locally in your browser) and reload saved searches from the dropdown.

On **Ledger → Transactions**, the top bar also provides:

- **Download**: Exports balanced postings to CSV (`paisa-transactions.csv`).
- **Force Full Sync**: Re-parses the journal and replaces all postings in the database, bypassing incremental-sync caches.

## Bulk Edit Interface

Paisa's bulk transaction editor helps you review and modify multiple transactions at once:

1. Use the Transaction Filter Bar to narrow down the set of transactions you want to change.
2. Expand **Bulk Edit** to configure a batch operation and click preview to inspect a side-by-side diff before saving.

## Search

#### Plain

Paisa provides a powerful search query interface. Let's start with a
few example queries.

```query
Expenses:Utilities:Electricity
```

This will search for all transactions that have a posting with account
named Expenses:Utilities:Electricity. By default, the search is case
insensitive and will do a substring match. So, `#!query Expenses:Utilities`
will match Expenses:Utilities:Electricity account as well. If you want
to search an account name which has special characters like space in
it, you can use double quotes to enclose it like
`#!query "Expenses:Utilities:Hair Cut"`.

You can also search on transaction date. For example, if you want to
show all the transactions made on 1st Jan 2023, just type
`#!query [2023-01-01]`. If you want to see all made on that month, just leave
out the day part `#!query [2023-01]`. You can do the same with year, `#!query [2023]`
will show all the transactions made in 2023.

There is experimental support for natural language date. You can do
queries like `#!query [last month]`, `#!query [last year]`, `#!query [this month]`, `#!query [last
week]`, `#!query [jan 2023]`, etc.

Let's say you want to search by amount. You can do that by typing
`#!query 42`, it will show all the transactions that have a posting with that
amount.

if you want to search a exact Account, you can do that using Regular
Expression. Just type `#!query /^Assets:Equity:APPLE$/`, you can also do case
insensitive search by using the modifier `i` like
`#!query /^Assets:Equity:APPLE$/i`.

#### Property

You can also search based on properties like account, commodity,
amount, total, filename, note, payee and date.

```query
payee =~ /uber/i
payee = "Advance Tax"
commodity = GOLD
total > 5000
note = "rebate"
account = "Expenses:Utilities:Electricity"
date >= [2023-01-01]
filename = creditcard/2023/jan.ledger
```

The general format is `property operator value`. The property can be
any of the following:

- **account** - posting account
- **commodity** - posting commodity
- **amount** - posting amount
- **total** - transaction total
- **filename** - name of the file the transaction is in
- **note** - posting or transaction note (comment)
- **payee** - transaction payee
- **date** - transaction date

The operator can be any of the following:

- **=** equal
- **=~** regular expression match
- **<** less than
- **<=** less than or equal
- **\>** greater than
- **\>=** greater than or equal

Not all the combinations of property, operator and value would work,
if in doubt, just try it out, the UI will show you an error if the
query is not valid.

In fact, in the previous format we saw, if the property and operator is
not specified and a default set is chosen based on the value type. For
example, `#!query 42` will be treated as `#!query amount = 42`, `#!query Expenses:Utilities`
will be treated as `#!query account = Expenses:Utilities`,
`#!query /Expenses:Utilities/i` will be treated as
`#!query account =~ /Expenses:Utilities/i`, `#!query [2023-01]` will be treated as
`#!query date = [2023-01]`.

#### Conditional

You can combine multiple property based queries using `AND` and `OR`,
you can negate them using `NOT`

```query
account = Expenses:Utilities AND payee =~ /uber/i
commodity = GOLD OR total > 5000
date >= [2023-01-01] AND date < [2023-04-01]
account = Expenses:Utilities AND payee =~ /uber/i AND (total > 5000 OR total < 1000)
[last year] AND (payee =~ /swiggy/i OR payee =~ /phonepe/i)
total < 5000 AND NOT account = Expenses:Utilities
```

If you leave out the conditional operator, it will be treated as
`#!query AND`. Both of the below queries are the same

```query
account = Expenses:Utilities payee =~ /uber/i
account = Expenses:Utilities AND payee =~ /uber/i
```


## Bulk Edit Form

Currently bulk edit form supports only account rename feature. More
will be added later. The preview button allows you to see the changes
before you save them. It will show a side by side diff of the changes.
