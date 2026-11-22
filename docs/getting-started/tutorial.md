---
description: "Tutorial on how to get started with Paisa, an open source personal finance manager"
---

# Tutorial

This tutorial will introduce all the concepts necessary to get
started. **Paisa** builds on top of the **[ledger](https://ledger-cli.org/)**[^1], a command
line tool that follows the principles of [plain text accounting](https://plaintextaccounting.org/).
**ledger** primarily focuses on command line users and doesn't provide
any graphical user interface. Paisa aims to create a low-friction
graphical user interface on top of **ledger**, thereby making it
accessible to a wider range of users.

As an end user, you should be familiar with the [terms and
concepts](https://github.com/ledger/ledger/blob/master/doc/GLOSSARY.md) used by **ledger**, which we will cover below. Paisa
comes with an embedded **ledger** and you are not required to use
**ledger** via command line unless you want to.

!!! tip

    Even though the tutorial focuses on Indian users, Paisa is
    capable of handling any currency. You can change the default
    currency, locale and financial year starting month etc. Check the
    [configuration](../reference/config.md) reference for more details.


If you are starting with an empty journal, the Dashboard displays a **Setup Demo** button so you can explore a sample journal and configuration with one click, or follow the steps below to build your own journal.

## :fontawesome-regular-file-lines: Journal

A [journal](../reference/journal.md) file captures all your financial transactions. A transaction
may represent a mutual fund purchase, retirement contribution, grocery
purchase and so on. Paisa creates a journal named `main.ledger`, Let's
add our first transaction there. To open the editor, go to `Ledger`
:material-chevron-right: `Editor` (you can also use the `+` **Quick Add** button in the top-right action bar or the [Smart Importer](../reference/import.md) under `Ledger` :material-chevron-right: `Import` to import bank statements).

```ledger
2022/01/01/*(1)!*/ Salary/*(2)!*/
    Income:Salary:Acme/*(3)!*/   -100,000 INR/*(6)!*/
    Assets:Checking/*(4)!*/    /*(5)!*/100,000 INR
```

1. Transaction `Date`
2. Transaction description, also called as `Payee`
3. Credit (`Income`) `Account` — source of funds (negative in Ledger)
4. Debit (`Assets`) `Account` — destination of funds (positive in Ledger)
5. `Amount`
6. `Currency`

**ledger** follows the double-entry accounting system. In simple terms, it
tracks the movement of money from a credit (source) account to a debit
(destination) account. Here `#!ledger Income:Salary:Acme` is the credit account (source of income, shown as negative) and
`#!ledger Assets:Checking` is the debit account (destination receiving the funds, shown as positive). The date at which the
transaction took place and a description of the transaction is written
in the first line followed by the list of postings. Account [naming conventions](../reference/accounts.md) are explained later. The `:` in the account name
represents hierarchy.

```ledger
2022/01/01 Salary
    Income:Salary:Acme      -100,000 INR
    Assets:Checking          100,000 INR

2022/02/01 Salary
    Income:Salary:Acme      -100,000 INR
    Assets:Checking          100,000 INR

2022/03/01 Salary
    Income:Salary:Acme      -100,000 INR
    Assets:Checking          100,000 INR
```

Let's add few more transactions. As you edit your journal file, the
balance of the journal will be shown on the right hand side.

```
         300,000 INR  Assets:Checking
        -300,000 INR  Income:Salary:Acme
--------------------
                   0
```

You would notice zero balance and a checking account with 3 lakhs and
an income account with -3 lakhs. Double-entry accounting will always
results in 0 balance since you have to always enter both the credit
and debit side.


Let's say your company deducts `#!ledger 12,000 INR` and contributes it to EPF,
we could represent it as follows

```ledger
2022/01/01 Salary
    Income:Salary:Acme    -100,000 INR
    Assets:Checking         88,000 INR
    Assets:Debt:EPF         12,000 INR

2022/02/01 Salary
    Income:Salary:Acme    -100,000 INR
    Assets:Checking         88,000 INR
    Assets:Debt:EPF         12,000 INR

2022/03/01 Salary
    Income:Salary:Acme    -100,000 INR
    Assets:Checking         88,000 INR
    Assets:Debt:EPF         12,000 INR
```

You can now see the use of `:` hierarchy in the account name.

```
         300,000 INR  Assets
         264,000 INR    Checking
          36,000 INR    Debt:EPF
        -300,000 INR  Income:Salary:Acme
--------------------
                   0
```

## :material-gold: Commodity

So far we have only dealt with INR. **ledger** can handle commodity as
well. Let's say you are also investing `#!ledger 10,000 INR` in UTI Nifty Index
Fund and `#!ledger 10,000 INR` in ICICI Nifty Next 50 Index Fund every
month.

```ledger
2018/01/01 Investment
    Assets:Checking               -20,000 INR
    Assets:Equity:NIFTY        148.0865 NIFTY @ 67.5281 INR
    Assets:Equity:NIFTY_JR  358.6659 NIFTY_JR @ 27.8811 INR

2018/02/01 Investment
    Assets:Checking               -20,000 INR
    Assets:Equity:NIFTY        140.2870 NIFTY @ 71.2824 INR
    Assets:Equity:NIFTY_JR  363.2242 NIFTY_JR @ 27.5312 INR

2018/03/01 Withdrawal
    Assets:Checking                          6775.49 INR
    Income:CapitalGains:Equity:NIFTY          -22.68 INR
    Assets:Equity:NIFTY          -100 NIFTY {67.5281 INR} [2018/01/01] @ 67.7549 INR
```

Let's consider `#!ledger 148.0865 NIFTY @ 67.5281 INR`. Here `#!ledger
NIFTY` is the name of the commodity and we have bought `#!ledger
148.0865` units at `#!ledger 67.5281 INR` per unit.

The withdrawal transaction is bit more involved. When you buy a
commodity, you buy them at a specific price on a specific date called
lot. When you sell, you usually need to record which lot you are
selling for taxation purpose, usually FIFO. `#!ledger -100 NIFTY
 {67.5281 INR} [2018/01/01] @ 67.7549 INR` means you are selling
`#!ledger NIFTY` at price `#!ledger 67.7549 INR` that was bought on
`#!ledger 2018/01/01` at `#!ledger 67.5281 INR`. The gain or loss
amount comes from the capital gains account `#!ledger
Income:CapitalGains:Equity:NIFTY`

Paisa has support for fetching commodity price history from several
[providers](../reference/commodities.md). Go to `Tools` :material-chevron-right: `Configuration` (or `More` :material-chevron-right: `Configuration` in Classic navigation) and select `Finance` :material-chevron-right: `Commodities & Prices`. Click **Add** (or the :fontawesome-solid-circle-plus: icon) to
add a new commodity, set its name to `#!ledger NIFTY`, choose the price provider details, and save the configuration. Then click the **Update Prices** (`$`) icon in the top-right action bar. If everything is configured correctly, you will see the
latest valuation under `Net Worth` :material-chevron-right: `Net Worth` :material-chevron-right: `Asset Balances` (`Assets` :material-chevron-right: `Balance` in Classic navigation).

## :fontawesome-solid-hand-holding-dollar: Interest

There are many instruments like EPF, FD, etc that pay interest at
regular intervals. We can treat it as just another transaction. Any
income account that has a prefix `#!ledger Income:Interest:` can be
used as the credit account. It's not mandatory to specify the amount on
both sides. If you leave one side blank, **ledger** will deduce it automatically.

```ledger
2022/03/31 EPF Interest
    Income:Interest:EPF     -5,000 INR
    Assets:Debt:EPF
```

```
           5,000 INR  Assets:Debt:EPF
          -5,000 INR  Income:Interest:EPF
--------------------
                   0
```

## Configuration

All the configuration related to paisa is stored in a yaml file named
`paisa.yaml`. The configuration can be edited via the web interface. The
sequence in which it looks for the file is described below

1. `PAISA_CONFIG` environment variable
1. via `--config` flag
1. Current working directory
1. `paisa/paisa.yaml` file inside User Documents folder.

If it can't find the configuration file, it will create a default
configuration file named `paisa/paisa.yaml` inside User Documents
folder. The default configuration is tuned for Indians, users from
other countries would have to change the `default_currency` and
`locale`. Check the [configuration](../reference/config.md) reference for details.

## Update

Paisa fetches the latest price of commodities and re-ingests journal changes when you sync.
Make sure to run `paisa update` (CLI) or click the **Sync Journal** / **Update Prices** icons in the top-right action bar after editing your journal or whenever you want fresh market valuations.
If you edit journal files externally outside Paisa, the background file watcher automatically highlights the **Sync Journal** icon in amber so you know a sync is pending. See [Data Synchronization](../reference/sync.md) for details.

[^1]: [hledger](https://hledger.org) and [beancount](https://beancount.github.io/) are also supported, refer
    [Ledger CLI](../reference/ledger-cli.md) for more information.
