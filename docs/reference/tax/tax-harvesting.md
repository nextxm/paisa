# Tax Harvesting

Open **Planning → Tax → Tax Harvesting** (Classic: **Planning → Tax → Harvest**, `/planning/tax/harvest`; shown when `default_currency` is `INR`) to view holdings eligible for tax-loss or tax-gain harvesting.

Gains from asset types like Mutual Funds and Stocks are subject to capital
gains tax. The taxation rate differs based on holding period, whether
the asset is debt or equity, and annual exemption limits, and exit loads may also apply.

Paisa takes a flexible approach and lets you configure when each commodity is
eligible to harvest under **Tools → Configuration → Finance → Commodities & Prices** (`/more/config/commodities`) or in `paisa.yaml`:

1. `harvest` - specifies the number of holding days after which a lot is
   eligible for tax harvesting (for example, based on exit load and LTCG period).

2. `tax_category` - defines how capital gains taxes are computed (applying
   applicable grandfathering, indexation, or slab rules):

    1. `equity` - Listed Indian equity shares and equity-oriented instruments.

    2. `equity65` - Funds with 65% or more invested in Indian equity.

    3. `equity35` - Funds with 35% or more (and less than 65%) invested in Indian equity.

    4. `debt` - Debt mutual funds and fixed-income instruments.

    5. `unlisted_equity` - Unlisted foreign or Indian equity.

```yaml
commodities:
  - name: NIFTY
    type: mutualfund
    price:
      provider: in-mfapi
      code: 120716
    harvest: 365
    tax_category: equity65
  - name: ABCBF
    type: mutualfund
    price:
      provider: in-mfapi
      code: 119533
    harvest: 1095
    tax_category: debt
```

The tax harvest page will show the list of accounts eligible for
harvesting.

## Multiple Folios

Paisa uses FIFO method within an Account. If you have multiple folios,
this might result in incorrect values. The issue can be solved by
using a different Account for each folio.
