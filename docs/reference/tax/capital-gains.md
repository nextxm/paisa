# Capital Gains

Open **Planning → Tax → Capital Gains** (`/planning/tax/capital_gains`; shown when `default_currency` is `INR`) to review financial-year realized capital gains and estimated tax liability across mutual funds and stocks. The underlying lot-matching engine is also used by the [Tax-Aware Drawdown Strategy](../planning.md#4-tax-aware-drawdown-strategy) planner.

Mutual Funds and equities are considered capital assets and capital gains tax
applies on realized withdrawals. The Capital Gains page shows financial
year wise capital gains tax based on your sell/redemption transactions, with
the total summary, account-wise breakdown, and per buy/sell FIFO match details.

Only commodities with `tax_category` (`equity`, `equity65`, `equity35`, `debt`, or `unlisted_equity`) are included. Please check the [Tax
Harvesting](tax-harvesting.md) page for configuration details.

### Gain

The difference between the purchase and sell price.

### Taxable Gain

The gain after applying Indexation or Grandfathering provision.


Based on the duration between the purchase and sell date, an asset
might be classified as Short Term or Long Term.

### Short Term Tax

This would be a percentage of the Taxable Gain. The percentage varies
based on the asset class.

### Long Term Tax

This would be a percentage of the Taxable Gain. The percentage varies
based on the asset class.

### Taxable at Slab Rate

There are cases where the gain would be added to taxable income and
charged **Slab** rate. Since the tax rate would depend on the person,
the whole taxable amount is shown instead of the tax. You can multiply
this with your slab rate to get the tax amount.
