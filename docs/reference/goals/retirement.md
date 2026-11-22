---
description: "How to configure your retirement goal in Paisa"
---

# Retirement

Paisa helps you plan your retirement and track your progress on **Planning → Financial Goals → Retirement & Savings Goals** (`/planning/goals`). The first step is determining your target retirement corpus from your expected yearly expenses and Safe Withdrawal Rate (`swr`).

Both `swr` and `savings` (along with `name` and `icon`) are required fields when configuring a retirement goal under `goals.retirement` in **Tools → Configuration → Planning → Goals** (`/more/config/goals`) or in `paisa.yaml`.

```yaml
goals:
  retirement:
    - name: Retirement
      icon: mdi:palm-tree
      swr: 3.3
      yearly_expenses: 1100000
      savings:
        - Assets:Equity:*
        - Assets:Debt:*
```

If you track expenses in Paisa, instead of hardcoding `yearly_expenses`, you can specify a list of `expenses` accounts. Paisa will compute the average of your last 3 years of expenses across those accounts:

```yaml
goals:
  retirement:
    - name: Retirement
      icon: mdi:palm-tree
      swr: 2
      priority: 10
      expenses:
        - Expenses:Entertainment
        - Expenses:Gift
        - Expenses:Insurance
        - Expenses:Misc
        - Expenses:Shopping
        - Expenses:Utilities
      savings:
        - Assets:Equity:*
        - Assets:Debt:*
```

Set an optional integer `priority` to control display ordering — higher-priority goals appear first on the Dashboard **Goals** widget and `/planning/goals`.
