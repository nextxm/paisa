---
description: "The Underlying Portfolio page unwraps container commodities such as mutual funds to show underlying securities, industries, and credit ratings."
---

# Underlying Portfolio

Route: `/assets/portfolio` (Grouped: `Net Worth → Investments & Portfolio → Underlying Portfolio`; Classic: `Assets → Portfolio`) *(formerly `Analysis`; `/assets/analysis` redirects here with HTTP 301)*

Ledger represents everything except currency as commodities. A commodity can represent a physical asset like gold or a security like a stock or bond. Container commodities such as mutual funds hold baskets of underlying securities. When you own a mutual fund unit, you indirectly own those underlying holdings.

The **Underlying Portfolio** page unwraps supported container commodities and shows the underlying securities you hold along with their amounts and portfolio percentages, categorized by industry and rating.

!!! example "Alpha / Experimental"
    The underlying holdings data that powers this page comes from external sources and might
    not be 100% accurate. Double-check against an official fund factsheet before making investment decisions.

## Looking for YoY or MoM Expense Analysis?

The **Year-over-Year (YoY)** and **Month-over-Month (MoM)** analysis pages were moved under **Expenses**:

- [Expenses – Year-over-Year (YoY)](./expenses.md#year-over-year-yoy) (`/expense/yoy`; legacy `/analysis/yoy` redirects here)
- [Expenses – Month-over-Month (MoM)](./expenses.md#month-over-month-mom) (`/expense/mom`; legacy `/analysis/mom` redirects here)
