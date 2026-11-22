# Schedule AL

Open **Planning → Tax → Schedule AL** (`/planning/tax/schedule_al`; shown when `default_currency` is `INR`) to compute your end-of-year asset and liability totals by Schedule AL category. Use the financial year selector in the top navbar to switch between years.

As per Indian Income Tax rules, individuals with total income above ₹50
lakh are required to report Assets and Liabilities under Schedule AL. Paisa computes these totals using your configured account mappings.

| code      | Section        | Details                                                                      |
|-----------|----------------|------------------------------------------------------------------------------|
| immovable | A (1)          | Immovable Assets                                                             |
| metal     | B (1) (i)      | Jewellery, bullion etc.                                                      |
| art       | B (1) (ii)     | Archaeological collections, drawings, painting, sculpture or any work of art |
| vehicle   | B (1) (iii)    | Vehicles, yachts, boats and aircrafts                                        |
| bank      | B (1) (iv) (a) | Financial assets: Bank (including all deposits)                              |
| share     | B (1) (iv) (b) | Financial assets: Shares and securities                                      |
| insurance | B (1) (iv) (c) | Financial assets: Insurance policies                                         |
| loan      | B (1) (iv) (d) | Financial assets: Loans and advances given                                   |
| cash      | B (1) (iv) (e) | Financial assets: Cash in hand                                               |
| liability | C (1)          | Liabilities                                                                  |


All you need to do is map your ledger accounts to each
section under **Tools → Configuration → Finance → Schedule AL** (`/more/config/schedule-al`) or directly in `paisa.yaml`. Paisa computes the total as of the last day of the
selected financial year.


```yaml
schedule_al:
    - code: bank
      accounts:
          - Assets:Checking
    - code: share
      accounts:
          - Assets:Equity:*
          - Assets:Debt:*

```
