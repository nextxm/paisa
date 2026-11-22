---
hide:
  - navigation
  - toc
  - feedback
---

<div class="hero" markdown>
![Logo](./images/logo.svg)
# Paisa
<p class="subtitle">Personal Finance Manager</p>
</div>

<div class="home" markdown>
=== ":material-currency-inr: INR"

    ```ledger
    2022/01/01 Salary
        Income:Salary:Acme     -100,000 INR
        Assets:Checking         100,000 INR

    2022/01/03 Rent
        Assets:Checking         -20,000 INR
        Expenses:Rent

    2022/01/07 Investment
        Assets:Checking         -20,000 INR
        Assets:Equity:NIFTY   168.690 NIFTY @ 118.56 INR
    ```

=== ":material-currency-usd: USD"

    ```ledger
    2022/01/01 Salary
        Income:Salary:Acme      $-5,000
        Assets:Checking          $5,000

    2022/01/03 Rent
        Assets:Checking         $-2,000
        Expenses:Rent

    2022/01/07 Investment
        Assets:Checking         $-1,000
        Assets:Equity:AAPL   6.452 AAPL @ $154.97
    ```

=== ":fontawesome-solid-euro-sign: EURO"

    ```ledger
    commodity €
        format €1.000,00

    commodity AAPL
        format 1.000,00 AAPL

    2022/01/01 Salary
        Income:Salary:Acme      €-5.000
        Assets:Checking          €5.000

    2022/01/03 Rent
        Assets:Checking         €-2.000
        Expenses:Rent

    2022/01/07 Investment
        Assets:Checking      €-1.000,02
        Assets:Equity:AAPL   6,453 AAPL @ €154,97
    ```


<p style="text-align: center; margin-bottom: 3rem">
  <a class="md-button md-button--primary" href="./getting-started/installation/">Install</a>
</p>

<div class="features-container" markdown>
<div class="features" markdown>
- :fontawesome-regular-file-lines: Builds on top of **[ledger](https://www.ledger-cli.org/)**, **[hledger](https://hledger.org/)**, and **[Beancount](https://beancount.github.io/)** double-entry accounting tools.
- :octicons-shield-lock-16: Your financial **data** never leaves your system.
- :simple-git: Your journal and configuration live in **plain text** files for easy version control and full `.zip` **[backup & export](./reference/config.md#backup-export)**.
- :octicons-graph-24: Track market **prices** and **FX rates** across mutual funds, stocks, NPS, metals, and multi-currency holdings.
- :fontawesome-regular-credit-card: Analyze and **budget** your **[expenses](./reference/expenses.md)** with Monthly/Calendar trends, Spending Heatmap, MoM, YoY, and Sankey flows.
- :material-microsoft-excel: **Smart Import** bank and card statements (`CSV`, `TXT`, `XLS`, `XLSX`, `PDF`, `OFX`, `QFX`) with visual column mapping, classification rules, duplicate detection, or custom Handlebars templates.
- :material-chart-pie: Monitor portfolio drift and compute tax-aware cash deployments with **[Target Asset Allocation & Rebalancing](./reference/allocation-targets.md)**.
- :material-stethoscope: Catch accounting errors, missing prices, potential duplicates, and statistical outliers in **[Data Health (Doctor)](./reference/doctor.md)**.
- :fontawesome-solid-calculator: **Calculate** taxes, EMIs, and custom queries using **[Sheets](./reference/sheets.md)** or log entries rapidly with **Natural Language Quick Add** and the **`Ctrl+K` Command Palette**.
- :octicons-goal-16: Track retirement, savings, and milestone **[goals](./reference/goals/index.md)** plus **Dividends & Investment Income**.
- :material-timer-sync: Monitor **[recurring bills](./reference/recurring.md)** and **[credit card](./reference/credit-cards.md)** cycles and due dates.
- :material-beach: Plan retirement with **[Financial DNA, Monte Carlo projections, What-If scenarios, and Tax-Aware Drawdown](./reference/planning.md)**.
- :material-chart-bar: Customize your **[Dashboard](./reference/dashboard.md)** layout, switch between Grouped & Classic navigation, and toggle privacy mode on desktop or mobile.
</div>


</div>


<div class="feature-card-container" markdown>
<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-logo" markdown>
:fontawesome-regular-file-lines:{ .feature-card-icon }
</div>
<div class="feature-card-right" markdown>
# Plain Text

All your financial data is stored in plain text files. Of course, we
will not ask you to draw a picture on blank canvas. We provide
enough guard rails to make data entry easy and error free. The editor
comes with syntax highlighting, auto completion, error checking and
auto formatting.
</div>
</div>


<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-equity" markdown>
:octicons-shield-lock-16:{ .feature-card-icon }
</div>

<div class="feature-card-right" markdown>
# Privacy

Selling your data is not the indirect goal of the app. The app will
never collect or send any data to any server. All your data is stored
in your system. Checkout the [manifesto](./manifesto.md) for more.
</div>
</div>

<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-asset" markdown>
:octicons-graph-24:{ .feature-card-icon }
</div>
<div class="feature-card-right" markdown>
# Price Tracking

Paisa supports various [price](./reference/commodities.md) data providers, so it can keep track of
the latest price of all your assets. It also allows the user to enter
the price manually, so you can use it to revalue your assets like
house, car, land, etc.
</div>
</div>


<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-expense" markdown>
:fontawesome-regular-credit-card:{ .feature-card-icon }
</div>

<div class="feature-card-right" markdown>
# Budget and Expenses

Paisa allows you to track your expenses at the granularity of your
choice. You can customize the categories and subcategories to suit
your needs, even icon is customizable. If you are tight on money,
don't worry, paisa got you covered. You can set a [budget](./reference/budget.md) for each
category and make sure you don't overspend.

</div>
</div>

<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-income" markdown>
:material-microsoft-excel:{ .feature-card-icon }
</div>
<div class="feature-card-right" markdown>
# Data Import

You don't have to manually enter all your transactions. Upload statements from your bank or credit card in `CSV`, `TXT`, `XLS`, `XLSX`, `PDF`, `OFX`, or `QFX` formats. Paisa's **[Smart Importer](./reference/import.md)** auto-detects columns and bank accounts, applies custom **Visual Rule Studio** patterns and TF-IDF suggestions, flags potential duplicates, and writes clean Ledger or Beancount entries—while still supporting extensible **Handlebars templates** for custom workflows.
</div>

</div>


<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-liability" markdown>
:fontawesome-solid-calculator:{ .feature-card-icon }
</div>

<div class="feature-card-right" markdown>
# Sheets

Ever wanted to calculate your taxes, emis, etc? Paisa comes with a
notepad calculator called [sheet](./reference/sheets.md) that can do all the calculations
with the data from your ledger. The sheets are live and interactive,
and will update automatically when you change the data in your ledger.

</div>
</div>

<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-logo" markdown>
:octicons-goal-16:{ .feature-card-icon }
</div>
<div class="feature-card-right" markdown>
# Goals & Financial Planning

Track **[savings and retirement goals](./reference/goals/index.md)** or model your long-term wealth using Paisa's **[Planning suite](./reference/planning.md)**: extract your **Financial DNA**, run **Monte Carlo** simulations with **Milestone Life Goals**, test **What-If scenarios**, and optimize withdrawals with **Tax-Aware Drawdown**.
</div>

</div>

<div class="feature-card" markdown>
<div class="feature-card-left feature-card-icon feature-card-icon-expense" markdown>
:material-timer-sync:{ .feature-card-icon }
</div>

<div class="feature-card-right" markdown>
# Bills

Paisa can help you track your [recurring](./reference/recurring.md) transactions like rent, emi,
[credit card](./reference/credit-cards.md) bills, etc. The calendar view will show you all the bills
that are due in the current month. You can set the recurring period of
the transaction, which is flexible enough to handle any kind of
schedule like weekly, monthly, quarterly, last day of the month, last
day of the quarter, etc.

</div>
</div>

<p></p>
</div>
