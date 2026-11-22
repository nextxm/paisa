---
description: "How to configure goals in Paisa to track retirement, savings targets, and milestone life events"
---

# Goals

Goals help you track financial objectives and measure progress against your real ledger balances. Paisa supports three complementary goal systems:

1. **[Retirement](./retirement.md)** — Target corpus calculated from Safe Withdrawal Rate (SWR) and yearly expenses (`/planning/goals`).
2. **[Savings](./savings.md)** — Fixed target amount with either a target date or planned periodic contribution (`/planning/goals`).
3. **Milestone & Recurring Life Goals** — One-off milestones (`milestone`) and recurring cash outflows (`recurring`) modeled inside the Monte Carlo [Life Plan](../planning.md#3-milestone-recurring-life-goals) (`/planning/life/goals`).

## Viewing & Managing Goals

- **Retirement & Savings Goals**: Open **Planning → Financial Goals → Retirement & Savings Goals** (`/planning/goals`; Classic: **Planning → Goals**) to inspect progress, drag to reorder priority, or click into each goal's detailed projection chart. Configure them in **Tools → Configuration → Planning → Goals** (`/more/config/goals`) or directly under `goals:` in `paisa.yaml`.
- **Milestone Life Goals**: Open **Planning → Financial Goals → Milestone Life Goals** (`/planning/life/goals`) to add, edit, reorder, and save life goals directly in the UI.
- **Dashboard Widget**: Configured retirement and savings goals can also be pinned to your [Dashboard](../dashboard.md) (configurable `1–12` items, ordered by descending priority).
