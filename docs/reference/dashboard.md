---
description: "Overview of the Paisa dashboard and its widgets"
---

# Dashboard

The **Dashboard** is the landing page of Paisa.  It provides an
at-a-glance overview of your financial health through a collection of
widgets.

## Recent Transactions

The **Recent Transactions** widget shows the 15 most recent
transactions from your journal.  Each entry is rendered as a
transaction card with the date, description, and posting amounts.

The `GET /api/transaction` endpoint supports optional `limit` and
`offset` query parameters to retrieve a custom page of transactions.
Every response includes all postings for each returned transaction
(pagination is applied at the transaction level, not the posting
level).

## Monthly Cashflow

The **Monthly Cashflow** widget provides a multi-currency breakdown of
income versus expenses for the current month, using the original
ledger quantities (no mark-to-market conversion).  This gives a quick,
currency-accurate net cashflow figure.

## Account Reconciliation

When `enable_reconciliation: true` is set in `paisa.yaml`, the
dashboard shows an **Account Reconciliation** widget with:

- The count of up-to-date accounts (reconciled within their expected
  frequency window).
- The count of overdue accounts.
- Quick-action links to reconcile each overdue account directly from
  the dashboard.

See [Accounts – Account Reconciliation](accounts.md#account-reconciliation)
for full details.

## Account Drill-Down

Clicking on any account balance widget on the dashboard navigates to
`/accounts/[name]/transactions`, which shows a filtered history of all
transactions that touch that account (or any sub-account below it).

## FIRE Metrics

The dashboard features a **FIRE Metrics** level indicator displaying key metrics to track your progress towards financial independence:

- **Years to FIRE**: The projected number of years required to reach your target corpus using the "Expected" net worth projection curve.
- **Target Corpus**: The total asset amount needed to support your annualized expenses based on your Safe Withdrawal Rate (SWR).
- **FIRE Progress**: The percentage of the target corpus you have currently accumulated.

These metrics are calculated dynamically using your historical income, expenses, and investment savings patterns. See [Assets – Projection (FIRE)](assets.md#projection-fire) for the exact calculation logic, formulas, and parameters.
