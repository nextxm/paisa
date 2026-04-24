# FX Rates

Paisa supports tracking multiple currencies and calculating the value of foreign assets in your default currency. The **FX Rates** page helps you visualize the exchange rates between the currencies configured in your ledger.

## Features

- **Daily Exchange Rates**: View the daily closing exchange rates for currency pairs over time.
- **Derived Rates (Cross Rates)**: If a direct pair (e.g., USD to EUR) is missing, Paisa will automatically synthesize it using a one-hop cross rate through the default currency (e.g., USD -> INR -> EUR) if data is available.
- **Provider & Journal Prioritization**: Exchange rates can be sourced from external price providers (like Yahoo Finance) or directly from your journal entries. When a rate exists in both, the journal's implicit exchange rate takes precedence, allowing you to accurately reflect the rate at which you exchanged the money.

## Configuration

To ensure Paisa treats commodities as currencies rather than securities (like mutual funds or stocks), add them to the `currencies` list in `paisa.yaml`:

```yaml
default_currency: INR
currencies:
  - USD
  - EUR
```

Once configured, any price entries or ledger entries using these pairs will populate the FX Rates page and correctly convert your assets to the default currency.
