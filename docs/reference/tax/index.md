# Tax

Paisa provides Indian tax-planning and reporting views under **Planning → Tax** in both Grouped and Classic navigation.

!!! note "Currency visibility"

    The **Tax** navigation menu is displayed automatically when `default_currency` is set to `INR` in [Configuration](../config.md).

## Available Reports & Tools

- **[Tax Harvesting](./tax-harvesting.md)** (`/planning/tax/harvest`) — Identify equity and debt lots eligible for long-term/short-term gain harvesting.
- **[Capital Gains](./capital-gains.md)** (`/planning/tax/capital_gains`) — Financial-year capital gains breakdown with FIFO lot matching, indexation/grandfathering rules, and slab vs. LTCG/STCG estimates.
- **[Schedule AL](./schedule-al.md)** (`/planning/tax/schedule_al`) — Asset and liability summary mapped to Indian Income Tax Schedule AL sections as of financial year-end.
- **[Tax-Aware Drawdown Strategy](../planning.md#4-tax-aware-drawdown-strategy)** (`/planning/life/drawdown`) — Order retirement withdrawals across priority buckets while estimating capital-gains tax impact.

## Disclaimer

The author of this tool is not an accountant, the calculations are
made based on their interpretation of rules and regulation. This tool
is distributed free hoping it would be useful. The users are advised
to consult a tax accountant before making any decision based on the
information provided here. Check the [LICENSE](https://github.com/nextxm/paisa/blob/main/COPYING) of paisa for more
details.
