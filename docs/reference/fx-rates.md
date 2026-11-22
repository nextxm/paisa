# FX Rates

Open **Ledger → FX Rates** (`/ledger/fx-rates`) in either Grouped or Classic navigation to track foreign currency exchange rates and inspect your portfolio's currency exposure.

Paisa supports tracking multiple currencies and calculating the value of foreign assets in your default currency.

## Features

- **Currency Exposure Breakdown**: A donut chart at the top of the page (and on [Allocation Targets](./allocation-targets.md)) summarizes your holdings by native currency versus your `default_currency`.
- **Daily Exchange Rates & Filters**: Filter rates by **Base currency**, **Quote currency**, **Year**, and **Month**.
- **Direct vs. Derived (Cross) Rates**: Each daily row is tagged as **Direct** (from a provider or journal entry) or **Derived** when Paisa synthesizes a missing pair via a one-hop cross rate through `default_currency` (for example, `USD → INR → EUR`).
- **Provider & Journal Prioritization**: Exchange rates can be sourced from external price providers (such as Yahoo Finance) or directly from your journal entries. When a rate exists in both, the journal's implicit exchange rate takes precedence.
- **Reporting Currency & FX Impact**: Across Balance, Net Worth, Expenses, and Prices views, `report_currency` selectors let you re-denominate reports, while [Net Worth History](./assets.md#net-worth) breaks changes down into organic asset movement vs. FX rate movement (**FX Impact Decomposition**).

## Configuration

To ensure Paisa treats commodities as currencies rather than securities (such as mutual funds or stocks), add them to the `currencies` list under **Tools → Configuration → General → Display & Format** (`/more/config/display-format`) or in `paisa.yaml`:

```yaml
default_currency: INR
currencies:
  - USD
  - EUR
```

Once configured, price entries and multi-currency postings populate the FX Rates page and convert foreign balances into your default currency. If you need to fall back to single-currency price matching only, enable `disable_multi_currency_prices: true` under **Tools → Configuration → Tools → Advanced** (`/more/config/advanced`).
